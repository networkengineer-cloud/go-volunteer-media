// Package classify trains and evaluates small binary classifiers over the
// embedding vectors the app already stores (animal_comments.embedding). It
// has no network or database dependencies: callers load vectors and labels,
// and this package does the math.
//
// The model is L2-regularized logistic regression fit by full-batch gradient
// descent. Embedding vectors are already dense and roughly unit-scale, so no
// feature standardization is applied. Positive examples are up-weighted by
// the negative/positive ratio so a rare label (e.g. medical) isn't drowned
// out by the majority class.
package classify

import (
	"errors"
	"math"
)

// Options controls training. Zero values fall back to DefaultOptions.
type Options struct {
	Epochs       int     // full passes of gradient descent
	LearningRate float64 // step size
	L2           float64 // ridge penalty on weights (not the bias)
}

// DefaultOptions are reasonable for 1024-dim Voyage vectors and a few
// thousand examples.
var DefaultOptions = Options{Epochs: 300, LearningRate: 0.5, L2: 1e-3}

// Model is a trained logistic-regression classifier.
type Model struct {
	Weights []float64
	Bias    float64
}

// ErrNoData is returned when training input is empty or mismatched.
var ErrNoData = errors.New("classify: need equal-length, non-empty X and y with consistent dimensions")

// ErrOneClass is returned when the training labels are all the same value.
var ErrOneClass = errors.New("classify: training labels contain only one class")

// Train fits a model on vectors X with binary labels y.
func Train(X [][]float32, y []bool, opts Options) (*Model, error) {
	if len(X) == 0 || len(X) != len(y) || len(X[0]) == 0 {
		return nil, ErrNoData
	}
	dim := len(X[0])
	for _, x := range X {
		if len(x) != dim {
			return nil, ErrNoData
		}
	}
	if opts.Epochs <= 0 {
		opts.Epochs = DefaultOptions.Epochs
	}
	if opts.LearningRate <= 0 {
		opts.LearningRate = DefaultOptions.LearningRate
	}
	if opts.L2 < 0 {
		opts.L2 = DefaultOptions.L2
	}

	nPos := 0
	for _, label := range y {
		if label {
			nPos++
		}
	}
	nNeg := len(y) - nPos
	if nPos == 0 || nNeg == 0 {
		return nil, ErrOneClass
	}
	posWeight := float64(nNeg) / float64(nPos)
	// Normalize so the total sample weight equals len(y).
	norm := float64(len(y)) / (float64(nPos)*posWeight + float64(nNeg))

	m := &Model{Weights: make([]float64, dim)}
	grad := make([]float64, dim)
	n := float64(len(y))
	for epoch := 0; epoch < opts.Epochs; epoch++ {
		for j := range grad {
			grad[j] = 0
		}
		var gradBias float64
		for i, x := range X {
			p := m.Predict(x)
			target := 0.0
			w := norm
			if y[i] {
				target = 1
				w *= posWeight
			}
			diff := w * (p - target)
			for j, v := range x {
				grad[j] += diff * float64(v)
			}
			gradBias += diff
		}
		for j := range m.Weights {
			m.Weights[j] -= opts.LearningRate * (grad[j]/n + opts.L2*m.Weights[j])
		}
		m.Bias -= opts.LearningRate * gradBias / n
	}
	return m, nil
}

// Predict returns the model's probability that x is positive.
func (m *Model) Predict(x []float32) float64 {
	z := m.Bias
	for j, v := range x {
		z += m.Weights[j] * float64(v)
	}
	return sigmoid(z)
}

func sigmoid(z float64) float64 {
	if z >= 0 {
		return 1 / (1 + math.Exp(-z))
	}
	e := math.Exp(z)
	return e / (1 + e)
}
