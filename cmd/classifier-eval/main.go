// Command classifier-eval measures whether the session-report embeddings the
// app already stores (animal_comments.embedding) can predict the
// "behavior" and "medical" labels volunteers apply. It is a read-only
// experiment: it runs inside a READ ONLY transaction, writes nothing, and
// calls no external API. Output is a report on stdout.
//
// Connection settings come from the same DB_* environment variables as the
// API (see internal/database.Initialize). Run it against a copy of real data;
// seed data has no real embeddings, so results there are meaningless.
//
//	go run ./cmd/classifier-eval [-folds 5] [-min-positives 20]
//
// Labels. A comment is positive for a label when it carries the system
// comment tag of that name, or its structured session report has that notes
// field filled in. Two experiments are run per label:
//
//   - all: every embedded comment. Optimistic, because a filled-in field also
//     puts a "**Behavior:**" / "**Medical:**" heading into the embedded text,
//     which makes those examples easy.
//   - unstructured: only comments with neither notes field filled in, labeled
//     by tag alone. This is the population a classifier would actually help
//     with (reports where the volunteer didn't use the field), so it is the
//     number to trust.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/classify"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/database"
)

type row struct {
	ID            uint
	GroupID       uint
	Embedding     pgvector.Vector
	TagBehavior   bool
	TagMedical    bool
	MetaBehavior  bool
	MetaMedical   bool
	HasStructured bool
}

const loadSQL = `
SELECT
  c.id,
  a.group_id,
  c.embedding,
  EXISTS (SELECT 1 FROM animal_comment_tags act
          JOIN comment_tags t ON t.id = act.comment_tag_id
          WHERE act.animal_comment_id = c.id AND t.deleted_at IS NULL
            AND t.is_system AND lower(t.name) = 'behavior') AS tag_behavior,
  EXISTS (SELECT 1 FROM animal_comment_tags act
          JOIN comment_tags t ON t.id = act.comment_tag_id
          WHERE act.animal_comment_id = c.id AND t.deleted_at IS NULL
            AND t.is_system AND lower(t.name) = 'medical') AS tag_medical,
  COALESCE(c.metadata->>'behavior_notes', '') <> '' AS meta_behavior,
  COALESCE(c.metadata->>'medical_notes', '') <> '' AS meta_medical,
  c.metadata IS NOT NULL AS has_structured
FROM animal_comments c
JOIN animals a ON a.id = c.animal_id
WHERE c.deleted_at IS NULL
  AND a.deleted_at IS NULL
  AND c.embedding IS NOT NULL
ORDER BY c.id`

func main() {
	folds := flag.Int("folds", 5, "number of cross-validation folds")
	minPositives := flag.Int("min-positives", 20, "skip an experiment with fewer positive examples than this")
	seed := flag.Int64("seed", 42, "random seed for fold assignment")
	flag.Parse()

	if err := run(*folds, *minPositives, *seed); err != nil {
		fmt.Fprintln(os.Stderr, "classifier-eval:", err)
		os.Exit(1)
	}
}

func run(folds, minPositives int, seed int64) error {
	db, err := database.Initialize()
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		defer sqlDB.Close()
	}

	var rows []row
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET TRANSACTION READ ONLY").Error; err != nil {
			return err
		}
		return tx.Raw(loadSQL).Scan(&rows).Error
	})
	if err != nil {
		return fmt.Errorf("load comments: %w", err)
	}
	if len(rows) == 0 {
		return errors.New("no comments with embeddings found; is SEMANTIC_SEARCH_ENABLED on and has the sweep run?")
	}

	printCounts(rows)

	labels := []struct {
		name string
		tag  func(row) bool
		meta func(row) bool
	}{
		{"behavior", func(r row) bool { return r.TagBehavior }, func(r row) bool { return r.MetaBehavior }},
		{"medical", func(r row) bool { return r.TagMedical }, func(r row) bool { return r.MetaMedical }},
	}
	for _, l := range labels {
		var allX, unX [][]float32
		var allY, unY []bool
		for _, r := range rows {
			vec := r.Embedding.Slice()
			allX = append(allX, vec)
			allY = append(allY, l.tag(r) || l.meta(r))
			if !r.MetaBehavior && !r.MetaMedical {
				unX = append(unX, vec)
				unY = append(unY, l.tag(r))
			}
		}
		experiment(l.name+" / all", allX, allY, folds, minPositives, seed)
		experiment(l.name+" / unstructured (trust this one)", unX, unY, folds, minPositives, seed)
	}
	return nil
}

func printCounts(rows []row) {
	type counts struct{ total, tagB, tagM, metaB, metaM, structured int }
	byGroup := map[uint]*counts{}
	var all counts
	for _, r := range rows {
		c := byGroup[r.GroupID]
		if c == nil {
			c = &counts{}
			byGroup[r.GroupID] = c
		}
		for _, x := range []*counts{c, &all} {
			x.total++
			if r.TagBehavior {
				x.tagB++
			}
			if r.TagMedical {
				x.tagM++
			}
			if r.MetaBehavior {
				x.metaB++
			}
			if r.MetaMedical {
				x.metaM++
			}
			if r.HasStructured {
				x.structured++
			}
		}
	}
	fmt.Println("== Label counts (embedded, non-deleted comments) ==")
	fmt.Printf("%-8s %7s %10s %10s %11s %11s %10s\n", "group", "total", "tag:behav", "tag:med", "field:behav", "field:med", "structured")
	line := func(name string, c counts) {
		fmt.Printf("%-8s %7d %10d %10d %11d %11d %10d\n", name, c.total, c.tagB, c.tagM, c.metaB, c.metaM, c.structured)
	}
	for id, c := range byGroup {
		line(fmt.Sprint(id), *c)
	}
	line("ALL", all)
	fmt.Println()
}

func experiment(name string, X [][]float32, y []bool, folds, minPositives int, seed int64) {
	pos := 0
	for _, v := range y {
		if v {
			pos++
		}
	}
	fmt.Printf("== %s ==\n", name)
	fmt.Printf("examples: %d, positives: %d (%.1f%%)\n", len(y), pos, pct(pos, len(y)))
	if pos < minPositives {
		fmt.Printf("SKIPPED: fewer than %d positives — not enough labeled data to judge.\n\n", minPositives)
		return
	}
	probs, err := classify.CrossValidate(X, y, folds, classify.DefaultOptions, seed)
	if err != nil {
		fmt.Printf("SKIPPED: %v\n\n", err)
		return
	}
	fmt.Printf("AUROC: %.3f  (0.5 = chance, 1.0 = perfect)\n", classify.AUROC(probs, y))
	fmt.Printf("%9s %9s %9s %6s %6s %6s\n", "threshold", "precision", "recall", "TP", "FP", "FN")
	for _, t := range []float64{0.3, 0.5, 0.7, 0.9} {
		m := classify.AtThreshold(probs, y, t)
		fmt.Printf("%9.1f %9.3f %9.3f %6d %6d %6d\n", t, m.Precision, m.Recall, m.TruePositives, m.FalsePositives, m.FalseNegatives)
	}
	fmt.Println()
}

func pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return 100 * float64(a) / float64(b)
}
