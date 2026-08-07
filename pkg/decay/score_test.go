package decay

import (
	"math"
	"testing"
)

func TestExponentialScoreIsOneAtLastAccess(t *testing.T) {
	score, err := ExponentialScore(1_000_000, 1_000_000, 86_400)
	if err != nil {
		t.Fatalf("ExponentialScore() error = %v", err)
	}
	if math.Abs(score-1.0) > 1e-12 {
		t.Fatalf("ExponentialScore() = %v, want 1", score)
	}
}

func TestExponentialScoreHalvesAtHalfLife(t *testing.T) {
	score, err := ExponentialScore(1_000_000, 1_086_400, 86_400)
	if err != nil {
		t.Fatalf("ExponentialScore() error = %v", err)
	}
	if math.Abs(score-0.5) > 1e-12 {
		t.Fatalf("ExponentialScore() = %v, want 0.5", score)
	}
}

func TestExponentialScoreClampsFutureAccessToOne(t *testing.T) {
	score, err := ExponentialScore(1_086_400, 1_000_000, 86_400)
	if err != nil {
		t.Fatalf("ExponentialScore() error = %v", err)
	}
	if math.Abs(score-1.0) > 1e-12 {
		t.Fatalf("ExponentialScore() = %v, want 1", score)
	}
}

func TestExponentialScoreRejectsNonPositiveHalfLife(t *testing.T) {
	_, err := ExponentialScore(1_000_000, 1_000_000, 0)
	if err == nil {
		t.Fatal("ExponentialScore() error = nil, want error")
	}
}
