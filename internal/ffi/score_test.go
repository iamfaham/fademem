package ffi

import (
	"math"
	"testing"
)

func TestScoreExponentialMapsCoreResultToCABIStatus(t *testing.T) {
	status, score := ScoreExponential(1_000_000, 87_400_000, 86_400_000)
	if status != StatusOK {
		t.Fatalf("status = %d, want %d", status, StatusOK)
	}
	if math.Abs(score-0.5) > 1e-12 {
		t.Fatalf("score = %v, want 0.5", score)
	}
}

func TestScoreExponentialMapsInvalidInputToCABIStatus(t *testing.T) {
	status, score := ScoreExponential(1_000_000, 1_000_000, 999)
	if status != StatusInvalidArgument {
		t.Fatalf("status = %d, want %d", status, StatusInvalidArgument)
	}
	if score != 0 {
		t.Fatalf("score = %v, want 0", score)
	}
}

func TestScorePowerLawMapsCoreResultToCABIStatus(t *testing.T) {
	status, score := ScorePowerLaw(1_000_000, 87_400_000, 86_400_000, 1, 0.5)
	if status != StatusOK {
		t.Fatalf("status = %d, want %d", status, StatusOK)
	}
	if math.Abs(score-0.25) > 1e-12 {
		t.Fatalf("score = %v, want 0.25", score)
	}
}

func TestScorePowerLawMapsInvalidInputToCABIStatus(t *testing.T) {
	status, score := ScorePowerLaw(1_000_000, 1_000_000, 86_400_000, 10.01, 1)
	if status != StatusInvalidArgument {
		t.Fatalf("status = %d, want %d", status, StatusInvalidArgument)
	}
	if score != 0 {
		t.Fatalf("score = %v, want 0", score)
	}
}
