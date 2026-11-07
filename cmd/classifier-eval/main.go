// Command classifier-eval measures whether the session-report embeddings the
// app already stores (animal_comments.embedding) can predict the comment tags
// volunteers apply — the system "behavior"/"medical" tags and every custom
// tag a group has created. It is a read-only experiment: it runs inside a
// READ ONLY transaction, writes nothing, calls no external API, and prints
// only counts and scores (never comment text).
//
// Connection settings come from the same DB_* environment variables as the
// API (see internal/database.Initialize). Run it against a copy of real data;
// seed data has no real embeddings, so results there are meaningless.
//
//	go run ./cmd/classifier-eval [-folds 5] [-min-positives 20] [-verbose]
//
// System tags ("behavior", "medical") exist in every group, so they are
// pooled across groups by name. A comment is positive when it carries the tag
// or its structured session report has the matching notes field filled in.
// Two experiments are run for each:
//
//   - all: every embedded comment. Optimistic, because a filled-in field also
//     puts a "**Behavior:**" / "**Medical:**" heading into the embedded text,
//     which makes those examples easy.
//   - unstructured: only comments with neither notes field filled in, labeled
//     by tag alone. This is the population a classifier would actually help
//     with, so it is the number to trust.
//
// Custom tags belong to one group, so each is evaluated only on that group's
// comments: a comment in another group could never have received the tag,
// and counting it as a negative would inflate the scores.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/classify"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/database"
)

type commentRow struct {
	ID           uint
	GroupID      uint
	Embedding    pgvector.Vector
	MetaBehavior bool
	MetaMedical  bool
}

type tagRow struct {
	ID        uint
	GroupID   uint
	GroupName string
	Name      string
	IsSystem  bool
}

type assignmentRow struct {
	AnimalCommentID uint
	CommentTagID    uint
}

const commentsSQL = `
SELECT
  c.id,
  a.group_id,
  c.embedding,
  COALESCE(c.metadata->>'behavior_notes', '') <> '' AS meta_behavior,
  COALESCE(c.metadata->>'medical_notes', '') <> '' AS meta_medical
FROM animal_comments c
JOIN animals a ON a.id = c.animal_id
WHERE c.deleted_at IS NULL
  AND a.deleted_at IS NULL
  AND c.embedding IS NOT NULL
ORDER BY c.id`

const tagsSQL = `
SELECT t.id, t.group_id, g.name AS group_name, t.name, t.is_system
FROM comment_tags t
JOIN groups g ON g.id = t.group_id
WHERE t.deleted_at IS NULL AND g.deleted_at IS NULL
ORDER BY t.is_system DESC, g.name, t.name`

const assignmentsSQL = `SELECT animal_comment_id, comment_tag_id FROM animal_comment_tags`

type result struct {
	label     string
	group     string
	examples  int
	positives int
	skipped   string
	auroc     float64
	best      classify.ThresholdMetrics
}

func main() {
	folds := flag.Int("folds", 5, "number of cross-validation folds")
	minPositives := flag.Int("min-positives", 20, "skip a tag with fewer positive examples than this")
	seed := flag.Int64("seed", 42, "random seed for fold assignment")
	verbose := flag.Bool("verbose", false, "print a precision/recall table for every tag, not just the summary")
	flag.Parse()

	if err := run(*folds, *minPositives, *seed, *verbose); err != nil {
		fmt.Fprintln(os.Stderr, "classifier-eval:", err)
		os.Exit(1)
	}
}

func run(folds, minPositives int, seed int64, verbose bool) error {
	db, err := database.Initialize()
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		defer sqlDB.Close()
	}

	var comments []commentRow
	var tags []tagRow
	var assignments []assignmentRow
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET TRANSACTION READ ONLY").Error; err != nil {
			return err
		}
		if err := tx.Raw(commentsSQL).Scan(&comments).Error; err != nil {
			return err
		}
		if err := tx.Raw(tagsSQL).Scan(&tags).Error; err != nil {
			return err
		}
		return tx.Raw(assignmentsSQL).Scan(&assignments).Error
	})
	if err != nil {
		return fmt.Errorf("load data: %w", err)
	}
	if len(comments) == 0 {
		return errors.New("no comments with embeddings found; is SEMANTIC_SEARCH_ENABLED on and has the sweep run?")
	}

	tagsByComment := map[uint]map[uint]bool{}
	for _, a := range assignments {
		if tagsByComment[a.AnimalCommentID] == nil {
			tagsByComment[a.AnimalCommentID] = map[uint]bool{}
		}
		tagsByComment[a.AnimalCommentID][a.CommentTagID] = true
	}
	// System tag IDs by lowercased name, across all groups.
	systemIDs := map[string]map[uint]bool{}
	for _, t := range tags {
		if t.IsSystem {
			key := strings.ToLower(t.Name)
			if systemIDs[key] == nil {
				systemIDs[key] = map[uint]bool{}
			}
			systemIDs[key][t.ID] = true
		}
	}
	hasAny := func(commentID uint, ids map[uint]bool) bool {
		for id := range tagsByComment[commentID] {
			if ids[id] {
				return true
			}
		}
		return false
	}

	ev := evaluator{folds: folds, minPositives: minPositives, seed: seed, verbose: verbose}

	// System tags, pooled across groups.
	for _, name := range sortedKeys(systemIDs) {
		ids := systemIDs[name]
		meta := func(c commentRow) bool { return false }
		switch name {
		case "behavior":
			meta = func(c commentRow) bool { return c.MetaBehavior }
		case "medical":
			meta = func(c commentRow) bool { return c.MetaMedical }
		}
		var allX, unX [][]float32
		var allY, unY []bool
		for _, c := range comments {
			vec := c.Embedding.Slice()
			tagged := hasAny(c.ID, ids)
			allX = append(allX, vec)
			allY = append(allY, tagged || meta(c))
			if !c.MetaBehavior && !c.MetaMedical {
				unX = append(unX, vec)
				unY = append(unY, tagged)
			}
		}
		ev.run(name+" (system, all)", "all groups", allX, allY)
		ev.run(name+" (system, unstructured)", "all groups", unX, unY)
	}

	// Custom tags, each within its own group.
	for _, t := range tags {
		if t.IsSystem {
			continue
		}
		var X [][]float32
		var y []bool
		for _, c := range comments {
			if c.GroupID != t.GroupID {
				continue
			}
			X = append(X, c.Embedding.Slice())
			y = append(y, tagsByComment[c.ID][t.ID])
		}
		ev.run(t.Name, t.GroupName, X, y)
	}

	printSummary(len(comments), minPositives, ev.results)
	return nil
}

type evaluator struct {
	folds, minPositives int
	seed                int64
	verbose             bool
	results             []result
}

func (e *evaluator) run(label, group string, X [][]float32, y []bool) {
	r := result{label: label, group: group, examples: len(y)}
	for _, v := range y {
		if v {
			r.positives++
		}
	}
	defer func() { e.results = append(e.results, r) }()

	if r.positives < e.minPositives {
		r.skipped = fmt.Sprintf("< %d positives", e.minPositives)
		return
	}
	probs, err := classify.CrossValidate(X, y, e.folds, classify.DefaultOptions, e.seed)
	if err != nil {
		r.skipped = err.Error()
		return
	}
	r.auroc = classify.AUROC(probs, y)
	r.best = classify.BestF1(probs, y)

	if e.verbose {
		fmt.Printf("== %s [%s] ==\n", label, group)
		fmt.Printf("examples: %d, positives: %d (%.1f%%), AUROC: %.3f\n", r.examples, r.positives, pct(r.positives, r.examples), r.auroc)
		fmt.Printf("%9s %9s %9s %6s %6s %6s\n", "threshold", "precision", "recall", "TP", "FP", "FN")
		for _, t := range []float64{0.1, 0.3, 0.5, 0.7, 0.9} {
			m := classify.AtThreshold(probs, y, t)
			fmt.Printf("%9.1f %9.3f %9.3f %6d %6d %6d\n", t, m.Precision, m.Recall, m.TruePositives, m.FalsePositives, m.FalseNegatives)
		}
		fmt.Println()
	}
}

func printSummary(totalComments, minPositives int, results []result) {
	fmt.Printf("== Summary: %d embedded comments ==\n", totalComments)
	fmt.Println("AUROC: 0.5 = chance, 1.0 = perfect. P/R/thr = precision, recall and threshold at the best F1.")
	fmt.Println("For behavior/medical, trust the 'unstructured' row; 'all' is inflated by the notes headings.")
	fmt.Println()

	evaluated := []result{}
	skipped := []result{}
	for _, r := range results {
		if r.skipped == "" {
			evaluated = append(evaluated, r)
		} else {
			skipped = append(skipped, r)
		}
	}
	sort.SliceStable(evaluated, func(i, j int) bool { return evaluated[i].auroc > evaluated[j].auroc })

	fmt.Printf("%-34s %-14s %8s %9s %7s %6s %6s %6s\n", "tag", "group", "examples", "positives", "AUROC", "P", "R", "thr")
	for _, r := range evaluated {
		fmt.Printf("%-34s %-14s %8d %9d %7.3f %6.2f %6.2f %6.2f\n",
			trunc(r.label, 34), trunc(r.group, 14), r.examples, r.positives, r.auroc, r.best.Precision, r.best.Recall, r.best.Threshold)
	}
	if len(skipped) > 0 {
		fmt.Printf("\nSkipped (fewer than %d positives, or too few for the fold count):\n", minPositives)
		for _, r := range skipped {
			fmt.Printf("  %-34s %-14s positives: %d\n", trunc(r.label, 34), trunc(r.group, 14), r.positives)
		}
	}
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return 100 * float64(a) / float64(b)
}
