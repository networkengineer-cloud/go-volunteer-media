// Command classifier-eval measures whether the embeddings the app already
// stores can predict which animal tags and scripts staff assign to an animal.
// If they can, the app could suggest a tag or script from an animal's recent
// session reports. It is a read-only experiment: it runs inside a READ ONLY
// transaction, writes nothing, calls no external API, and prints only counts
// and scores (never animal or report text).
//
// Connection settings come from the same DB_* environment variables as the
// API (see internal/database.Initialize). Run it against a copy of real data;
// seed data has no real embeddings, so results there are meaningless.
//
//	go run ./cmd/classifier-eval [-recent-reports 20] [-min-positives 10] [-verbose]
//
// Each example is one animal. Every label (an animal tag or a script) belongs
// to one group, so it is evaluated only on that group's animals: an animal in
// another group could never have received it, and counting it as a negative
// would inflate the scores.
//
// Two feature sets are scored for every label:
//
//   - reports: the mean embedding of the animal's most recent session
//     reports. This is the honest test of "do the reports show this?" and is
//     what a suggestion feature would use. Animals with no embedded reports
//     are left out of this experiment.
//   - profile: the animal's own embedding (name, species, breed, description,
//     trainer notes). Likely optimistic, since staff may write the
//     description with the tags in mind; shown for comparison.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"

	"github.com/networkengineer-cloud/go-volunteer-media/internal/classify"
	"github.com/networkengineer-cloud/go-volunteer-media/internal/database"
)

type animalRow struct {
	ID        uint
	GroupID   uint
	Embedding *pgvector.Vector // nil when the animal hasn't been embedded yet
}

type commentRow struct {
	AnimalID  uint
	Embedding pgvector.Vector
}

type labelRow struct {
	Kind      string // "tag" or "script"
	ID        uint
	GroupID   uint
	GroupName string
	Name      string
	Category  string // animal tag category; empty for scripts
}

type linkRow struct {
	Kind     string
	LabelID  uint
	AnimalID uint
}

const animalsSQL = `
SELECT a.id, a.group_id, a.embedding
FROM animals a
JOIN groups g ON g.id = a.group_id
WHERE a.deleted_at IS NULL AND g.deleted_at IS NULL
  AND (? OR a.status <> 'archived')
ORDER BY a.id`

// Newest first, so the first N rows per animal are its most recent reports.
const commentsSQL = `
SELECT c.animal_id, c.embedding
FROM animal_comments c
JOIN animals a ON a.id = c.animal_id
WHERE c.deleted_at IS NULL AND a.deleted_at IS NULL AND c.embedding IS NOT NULL
ORDER BY c.animal_id, c.created_at DESC, c.id DESC`

const labelsSQL = `
SELECT 'tag' AS kind, t.id, t.group_id, g.name AS group_name, t.name, t.category
FROM animal_tags t JOIN groups g ON g.id = t.group_id
WHERE t.deleted_at IS NULL AND g.deleted_at IS NULL
UNION ALL
SELECT 'script', s.id, s.group_id, g.name, s.title, ''
FROM scripts s JOIN groups g ON g.id = s.group_id
WHERE s.deleted_at IS NULL AND g.deleted_at IS NULL
ORDER BY kind, group_name, name`

const linksSQL = `
SELECT 'tag' AS kind, animal_tag_id AS label_id, animal_id FROM animal_animal_tags
UNION ALL
SELECT 'script', script_id, animal_id FROM animal_scripts`

type result struct {
	label     labelRow
	features  string
	examples  int
	positives int
	skipped   string
	auroc     float64
	null95    float64 // 95th percentile AUROC with shuffled labels
	p         float64 // share of shuffles that scored at least as well
	best      classify.ThresholdMetrics
}

type config struct {
	folds, minPositives, recentReports int
	permutations                       int
	includeArchived, verbose           bool
	seed                               int64
}

func main() {
	var cfg config
	flag.IntVar(&cfg.folds, "folds", 5, "number of cross-validation folds")
	flag.IntVar(&cfg.minPositives, "min-positives", 10, "skip a label with fewer positive animals than this")
	flag.IntVar(&cfg.permutations, "permutations", 20, "label shuffles per label for the chance baseline (more = slower, finer p-values)")
	flag.IntVar(&cfg.recentReports, "recent-reports", 20, "how many of an animal's most recent session reports to average")
	flag.BoolVar(&cfg.includeArchived, "include-archived", true, "include archived animals (more examples, but their tags may be stale)")
	flag.BoolVar(&cfg.verbose, "verbose", false, "print a precision/recall table for every label, not just the summary")
	flag.Int64Var(&cfg.seed, "seed", 42, "random seed for fold assignment")
	flag.Parse()

	if err := run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, "classifier-eval:", err)
		os.Exit(1)
	}
}

func run(cfg config) error {
	db, err := database.Initialize()
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		defer sqlDB.Close()
	}

	var animals []animalRow
	var comments []commentRow
	var labels []labelRow
	var links []linkRow
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET TRANSACTION READ ONLY").Error; err != nil {
			return err
		}
		if err := tx.Raw(animalsSQL, cfg.includeArchived).Scan(&animals).Error; err != nil {
			return err
		}
		if err := tx.Raw(commentsSQL).Scan(&comments).Error; err != nil {
			return err
		}
		if err := tx.Raw(labelsSQL).Scan(&labels).Error; err != nil {
			return err
		}
		return tx.Raw(linksSQL).Scan(&links).Error
	})
	if err != nil {
		return fmt.Errorf("load data: %w", err)
	}
	if len(animals) == 0 {
		return errors.New("no animals found")
	}

	reportVec := meanRecentReports(comments, cfg.recentReports)

	type key struct {
		kind string
		id   uint
	}
	linked := map[key]map[uint]bool{}
	for _, l := range links {
		k := key{l.Kind, l.LabelID}
		if linked[k] == nil {
			linked[k] = map[uint]bool{}
		}
		linked[k][l.AnimalID] = true
	}

	withProfile, withReports := 0, 0
	for _, a := range animals {
		if a.Embedding != nil {
			withProfile++
		}
		if reportVec[a.ID] != nil {
			withReports++
		}
	}

	var results []result
	for _, lab := range labels {
		has := linked[key{lab.Kind, lab.ID}]
		var repX, profX [][]float32
		var repY, profY []bool
		for _, a := range animals {
			if a.GroupID != lab.GroupID {
				continue
			}
			if v := reportVec[a.ID]; v != nil {
				repX = append(repX, v)
				repY = append(repY, has[a.ID])
			}
			if a.Embedding != nil {
				profX = append(profX, a.Embedding.Slice())
				profY = append(profY, has[a.ID])
			}
		}
		results = append(results,
			evaluate(cfg, lab, "reports", repX, repY),
			evaluate(cfg, lab, "profile", profX, profY))
	}

	fmt.Printf("== %d animals: %d with a profile embedding, %d with embedded session reports (mean of up to %d most recent) ==\n",
		len(animals), withProfile, withReports, cfg.recentReports)
	printSummary(cfg, results)
	return nil
}

// meanRecentReports averages each animal's first n comment embeddings.
// comments must be ordered by animal, newest first.
func meanRecentReports(comments []commentRow, n int) map[uint][]float32 {
	sums := map[uint][]float64{}
	counts := map[uint]int{}
	for _, c := range comments {
		if counts[c.AnimalID] >= n {
			continue
		}
		vec := c.Embedding.Slice()
		s := sums[c.AnimalID]
		if s == nil {
			s = make([]float64, len(vec))
			sums[c.AnimalID] = s
		}
		if len(vec) != len(s) {
			continue
		}
		for i, v := range vec {
			s[i] += float64(v)
		}
		counts[c.AnimalID]++
	}
	out := make(map[uint][]float32, len(sums))
	for id, s := range sums {
		mean := make([]float32, len(s))
		for i, v := range s {
			mean[i] = float32(v / float64(counts[id]))
		}
		out[id] = mean
	}
	return out
}

func evaluate(cfg config, lab labelRow, features string, X [][]float32, y []bool) result {
	r := result{label: lab, features: features, examples: len(y)}
	for _, v := range y {
		if v {
			r.positives++
		}
	}
	if r.positives < cfg.minPositives {
		r.skipped = fmt.Sprintf("< %d positives", cfg.minPositives)
		return r
	}
	probs, perm, err := classify.PermutationTest(X, y, cfg.folds, classify.DefaultOptions, cfg.seed, cfg.permutations)
	if err != nil {
		r.skipped = err.Error()
		return r
	}
	r.auroc = perm.Observed
	r.null95 = perm.NullQuantile(0.95)
	r.p = perm.P
	r.best = classify.BestF1(probs, y)

	if cfg.verbose {
		fmt.Printf("== %s %q [%s] using %s ==\n", lab.Kind, lab.Name, lab.GroupName, features)
		fmt.Printf("animals: %d, positives: %d (%.1f%%), AUROC: %.3f, chance 95th pct: %.3f, p: %.3f\n",
			r.examples, r.positives, pct(r.positives, r.examples), r.auroc, r.null95, r.p)
		fmt.Printf("%9s %9s %9s %6s %6s %6s\n", "threshold", "precision", "recall", "TP", "FP", "FN")
		for _, t := range []float64{0.1, 0.3, 0.5, 0.7, 0.9} {
			m := classify.AtThreshold(probs, y, t)
			fmt.Printf("%9.1f %9.3f %9.3f %6d %6d %6d\n", t, m.Precision, m.Recall, m.TruePositives, m.FalsePositives, m.FalseNegatives)
		}
		fmt.Println()
	}
	return r
}

func printSummary(cfg config, results []result) {
	fmt.Println("AUROC: 0.5 = chance, 1.0 = perfect. With tens of animals it swings widely by luck, so compare it with")
	fmt.Println("chance95 (the 95th-percentile AUROC after shuffling which animals carry the label) and p (the share of")
	fmt.Printf("%d shuffles that did at least as well). Only p <= 0.05 is worth a second look.\n", cfg.permutations)
	fmt.Println("P/R/thr = precision, recall and threshold at the best F1.")
	fmt.Println("'reports' is the number to trust; 'profile' is likely inflated by descriptions written with the tags in mind.")
	fmt.Println()

	var evaluated, skipped []result
	for _, r := range results {
		if r.skipped == "" {
			evaluated = append(evaluated, r)
		} else {
			skipped = append(skipped, r)
		}
	}
	sort.SliceStable(evaluated, func(i, j int) bool {
		// reports rows first, then most significant, then by AUROC
		if evaluated[i].features != evaluated[j].features {
			return evaluated[i].features == "reports"
		}
		if evaluated[i].p != evaluated[j].p {
			return evaluated[i].p < evaluated[j].p
		}
		return evaluated[i].auroc > evaluated[j].auroc
	})

	fmt.Printf("%-7s %-28s %-12s %-14s %-8s %7s %9s %7s %8s %6s %6s %6s %6s\n",
		"kind", "label", "category", "group", "features", "animals", "positives", "AUROC", "chance95", "p", "P", "R", "thr")
	for _, r := range evaluated {
		fmt.Printf("%-7s %-28s %-12s %-14s %-8s %7d %9d %7.3f %8.3f %6.3f %6.2f %6.2f %6.2f\n",
			r.label.Kind, trunc(r.label.Name, 28), trunc(r.label.Category, 12), trunc(r.label.GroupName, 14), r.features,
			r.examples, r.positives, r.auroc, r.null95, r.p, r.best.Precision, r.best.Recall, r.best.Threshold)
	}
	if len(skipped) > 0 {
		fmt.Printf("\nSkipped (fewer than %d positive animals, or too few for %d folds):\n", cfg.minPositives, cfg.folds)
		for _, r := range skipped {
			fmt.Printf("  %-7s %-28s %-14s %-8s positives: %d\n",
				r.label.Kind, trunc(r.label.Name, 28), trunc(r.label.GroupName, 14), r.features, r.positives)
		}
	}
}

func trunc(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return 100 * float64(a) / float64(b)
}
