package classify

import (
	"errors"
	"math"
	"math/rand"
	"testing"
)

// separable builds n vectors in dim dimensions where positives lean along
// the first axis and negatives against it, with noise elsewhere.
func separable(n, dim int, posRate float64, seed int64) ([][]float32, []bool) {
	rng := rand.New(rand.NewSource(seed))
	X := make([][]float32, n)
	y := make([]bool, n)
	for i := range X {
		y[i] = rng.Float64() < posRate
		x := make([]float32, dim)
		for j := range x {
			x[j] = float32(rng.NormFloat64() * 0.1)
		}
		if y[i] {
			x[0] += 0.5
		} else {
			x[0] -= 0.5
		}
		X[i] = x
	}
	return X, y
}

func TestTrainLearnsSeparableData(t *testing.T) {
	X, y := separable(400, 16, 0.2, 1)
	m, err := Train(X, y, DefaultOptions)
	if err != nil {
		t.Fatal(err)
	}
	correct := 0
	for i, x := range X {
		if (m.Predict(x) >= 0.5) == y[i] {
			correct++
		}
	}
	if acc := float64(correct) / float64(len(X)); acc < 0.95 {
		t.Fatalf("accuracy %.3f, want >= 0.95", acc)
	}
}

func TestTrainRejectsBadInput(t *testing.T) {
	if _, err := Train(nil, nil, DefaultOptions); !errors.Is(err, ErrNoData) {
		t.Fatalf("empty input: got %v", err)
	}
	X := [][]float32{{1, 2}, {3}}
	if _, err := Train(X, []bool{true, false}, DefaultOptions); !errors.Is(err, ErrNoData) {
		t.Fatalf("ragged input: got %v", err)
	}
	X = [][]float32{{1}, {2}}
	if _, err := Train(X, []bool{true, true}, DefaultOptions); !errors.Is(err, ErrOneClass) {
		t.Fatalf("one class: got %v", err)
	}
}

func TestCrossValidateScoresHeldOutData(t *testing.T) {
	X, y := separable(300, 16, 0.1, 2)
	probs, err := CrossValidate(X, y, 5, DefaultOptions, 42)
	if err != nil {
		t.Fatal(err)
	}
	if auc := AUROC(probs, y); auc < 0.95 {
		t.Fatalf("AUROC %.3f, want >= 0.95", auc)
	}
	again, _ := CrossValidate(X, y, 5, DefaultOptions, 42)
	for i := range probs {
		if probs[i] != again[i] {
			t.Fatal("same seed should give identical results")
		}
	}
}

func TestCrossValidateNeedsEnoughPositives(t *testing.T) {
	X, y := separable(50, 4, 0, 3)
	y[0] = true
	if _, err := CrossValidate(X, y, 5, DefaultOptions, 1); err == nil {
		t.Fatal("expected an error with one positive and k=5")
	}
}

func TestAUROC(t *testing.T) {
	y := []bool{false, false, true, true}
	if got := AUROC([]float64{0.1, 0.2, 0.8, 0.9}, y); got != 1 {
		t.Fatalf("perfect ranking: got %v", got)
	}
	if got := AUROC([]float64{0.9, 0.8, 0.2, 0.1}, y); got != 0 {
		t.Fatalf("inverted ranking: got %v", got)
	}
	if got := AUROC([]float64{0.5, 0.5, 0.5, 0.5}, y); got != 0.5 {
		t.Fatalf("all ties: got %v", got)
	}
	if got := AUROC([]float64{0.5, 0.5}, []bool{true, true}); got != 0 {
		t.Fatalf("one class: got %v", got)
	}
}

func TestBestF1(t *testing.T) {
	probs := []float64{0.9, 0.8, 0.6, 0.2}
	y := []bool{true, false, true, true}
	m := BestF1(probs, y)
	// t=0.2 flags all: P=3/4, R=1, F1=0.857 — beats t=0.6 (F1=0.667).
	if m.Threshold != 0.2 || m.TruePositives != 3 || m.FalsePositives != 1 {
		t.Fatalf("got %+v", m)
	}
	if z := BestF1([]float64{0.5}, []bool{false}); z.TruePositives != 0 {
		t.Fatalf("no positives: got %+v", z)
	}
}

func TestAtThreshold(t *testing.T) {
	probs := []float64{0.9, 0.8, 0.6, 0.2}
	y := []bool{true, false, true, true}
	m := AtThreshold(probs, y, 0.5)
	if m.TruePositives != 2 || m.FalsePositives != 1 || m.FalseNegatives != 1 {
		t.Fatalf("counts: %+v", m)
	}
	if math.Abs(m.Precision-2.0/3) > 1e-9 || math.Abs(m.Recall-2.0/3) > 1e-9 {
		t.Fatalf("rates: %+v", m)
	}
}
