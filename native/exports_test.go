package main

import (
	"math"
	"testing"
)

func TestScoreExponentialForFFIReturnsScalarStatus(t *testing.T) {
	status, score := scoreExponentialForFFI(1_000_000, 1_000_000, 86_400_000)
	if status != statusOK {
		t.Fatalf("status = %d, want %d", status, statusOK)
	}
	if math.Abs(score-1.0) > 1e-12 {
		t.Fatalf("score = %v, want 1", score)
	}
}

func TestScoreExponentialForFFIMapsInvalidInputToStatus(t *testing.T) {
	status, score := scoreExponentialForFFI(1_000_000, 1_000_000, 999)
	if status != statusInvalidArgument {
		t.Fatalf("status = %d, want %d", status, statusInvalidArgument)
	}
	if score != 0 {
		t.Fatalf("score = %v, want 0", score)
	}
}

func TestScorePowerLawForFFIReturnsScalarStatus(t *testing.T) {
	status, score := scorePowerLawForFFI(1_000_000, 87_400_000, 86_400_000, 1, 0.5)
	if status != statusOK {
		t.Fatalf("status = %d, want %d", status, statusOK)
	}
	if math.Abs(score-0.25) > 1e-12 {
		t.Fatalf("score = %v, want 0.25", score)
	}
}

func TestScorePowerLawForFFIMapsInvalidInputToStatus(t *testing.T) {
	status, score := scorePowerLawForFFI(1_000_000, 1_000_000, 86_400_000, 10.01, 1)
	if status != statusInvalidArgument {
		t.Fatalf("status = %d, want %d", status, statusInvalidArgument)
	}
	if score != 0 {
		t.Fatalf("score = %v, want 0", score)
	}
}
