package classify

import (
	"fmt"
	"math/rand"
	"sort"
)

// CrossValidate runs stratified k-fold cross-validation and returns an
// out-of-fold probability for every example: each example is scored by a
// model that never saw it in training. seed makes fold assignment
// deterministic so repeated runs are comparable.
func CrossValidate(X [][]float32, y []bool, k int, opts Options, seed int64) ([]float64, error) {
	if len(X) != len(y) || len(X) == 0 {
		return nil, ErrNoData
	}
	var pos, neg []int
	for i, label := range y {
		if label {
			pos = append(pos, i)
		} else {
			neg = append(neg, i)
		}
	}
	if k < 2 || len(pos) < k || len(neg) < k {
		return nil, fmt.Errorf("classify: need at least %d examples of each class for %d-fold CV (have %d positive, %d negative)", k, k, len(pos), len(neg))
	}

	// #nosec G404 -- deterministic fold shuffling, not security-sensitive.
	rng := rand.New(rand.NewSource(seed))
	rng.Shuffle(len(pos), func(i, j int) { pos[i], pos[j] = pos[j], pos[i] })
	rng.Shuffle(len(neg), func(i, j int) { neg[i], neg[j] = neg[j], neg[i] })
	fold := make([]int, len(y))
	for n, i := range pos {
		fold[i] = n % k
	}
	for n, i := range neg {
		fold[i] = n % k
	}

	probs := make([]float64, len(y))
	for f := 0; f < k; f++ {
		var trainX [][]float32
		var trainY []bool
		for i := range X {
			if fold[i] != f {
				trainX = append(trainX, X[i])
				trainY = append(trainY, y[i])
			}
		}
		m, err := Train(trainX, trainY, opts)
		if err != nil {
			return nil, fmt.Errorf("fold %d: %w", f, err)
		}
		for i := range X {
			if fold[i] == f {
				probs[i] = m.Predict(X[i])
			}
		}
	}
	return probs, nil
}

// ThresholdMetrics is precision/recall at one decision threshold.
type ThresholdMetrics struct {
	Threshold      float64
	TruePositives  int
	FalsePositives int
	FalseNegatives int
	Precision      float64 // of the flagged examples, how many were really positive
	Recall         float64 // of the real positives, how many were flagged
}

// AtThreshold computes precision and recall when flagging probs >= t.
func AtThreshold(probs []float64, y []bool, t float64) ThresholdMetrics {
	m := ThresholdMetrics{Threshold: t}
	for i, p := range probs {
		flagged := p >= t
		switch {
		case flagged && y[i]:
			m.TruePositives++
		case flagged && !y[i]:
			m.FalsePositives++
		case !flagged && y[i]:
			m.FalseNegatives++
		}
	}
	if d := m.TruePositives + m.FalsePositives; d > 0 {
		m.Precision = float64(m.TruePositives) / float64(d)
	}
	if d := m.TruePositives + m.FalseNegatives; d > 0 {
		m.Recall = float64(m.TruePositives) / float64(d)
	}
	return m
}

// BestF1 returns the threshold, among the distinct predicted probabilities,
// that maximizes F1 (the harmonic mean of precision and recall), along with
// its metrics. It is a convenience for summarizing many labels in one line;
// a real cutoff should be chosen from the precision/recall trade-off the
// feature can tolerate. Returns a zero value if there are no positives.
func BestF1(probs []float64, y []bool) ThresholdMetrics {
	var best ThresholdMetrics
	bestF1 := -1.0
	seen := map[float64]bool{}
	for _, t := range probs {
		if seen[t] {
			continue
		}
		seen[t] = true
		m := AtThreshold(probs, y, t)
		if m.TruePositives == 0 {
			continue
		}
		f1 := 2 * m.Precision * m.Recall / (m.Precision + m.Recall)
		if f1 > bestF1 || (f1 == bestF1 && t > best.Threshold) {
			bestF1, best = f1, m
		}
	}
	return best
}

// AUROC is the probability that a randomly chosen positive is scored above a
// randomly chosen negative (ties count half). 0.5 is chance, 1.0 is perfect.
// Returns 0 if either class is absent.
func AUROC(probs []float64, y []bool) float64 {
	type scored struct {
		p   float64
		pos bool
	}
	items := make([]scored, len(probs))
	nPos := 0
	for i, p := range probs {
		items[i] = scored{p, y[i]}
		if y[i] {
			nPos++
		}
	}
	nNeg := len(items) - nPos
	if nPos == 0 || nNeg == 0 {
		return 0
	}
	sort.Slice(items, func(i, j int) bool { return items[i].p < items[j].p })

	// Mann-Whitney U with average ranks for ties.
	var rankSumPos float64
	for i := 0; i < len(items); {
		j := i
		for j < len(items) && items[j].p == items[i].p {
			j++
		}
		avgRank := float64(i+j+1) / 2 // ranks are 1-based: (i+1 + j) / 2
		for t := i; t < j; t++ {
			if items[t].pos {
				rankSumPos += avgRank
			}
		}
		i = j
	}
	u := rankSumPos - float64(nPos)*float64(nPos+1)/2
	return u / (float64(nPos) * float64(nNeg))
}
