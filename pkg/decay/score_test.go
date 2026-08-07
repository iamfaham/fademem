package decay

import (
	"math"
	"testing"
)

const oneDayMillis int64 = 86_400_000

func TestExponentialScoreIsOneAtLastAccess(t *testing.T) {
	score, err := ExponentialScore(1_000_000, 1_000_000, oneDayMillis)
	if err != nil {
		t.Fatalf("ExponentialScore() error = %v", err)
	}
	if math.Abs(score-1.0) > 1e-12 {
		t.Fatalf("ExponentialScore() = %v, want 1", score)
	}
}

func TestExponentialScoreHalvesAtHalfLife(t *testing.T) {
	score, err := ExponentialScore(1_000_000, 87_400_000, oneDayMillis)
	if err != nil {
		t.Fatalf("ExponentialScore() error = %v", err)
	}
	if math.Abs(score-0.5) > 1e-12 {
		t.Fatalf("ExponentialScore() = %v, want 0.5", score)
	}
}

func TestExponentialScoreClampsFutureAccessToOne(t *testing.T) {
	score, err := ExponentialScore(87_400_000, 1_000_000, oneDayMillis)
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

func TestExponentialScoreRejectsOutOfRangeHalfLife(t *testing.T) {
	for _, halfLifeMillis := range []int64{999, 315_576_000_001} {
		_, err := ExponentialScore(1_000_000, 1_000_000, halfLifeMillis)
		if err == nil {
			t.Fatalf("ExponentialScore(%d) error = nil, want error", halfLifeMillis)
		}
	}
}

func TestImportanceWeightedPowerLawScoreFollowsConfiguredCurve(t *testing.T) {
	cases := []struct {
		name       string
		lastAccess int64
		now        int64
		importance float64
		want       float64
	}{
		{name: "at access", lastAccess: 1_000_000, now: 1_000_000, importance: 1, want: 1},
		{name: "base scale", lastAccess: 1_000_000, now: 87_400_000, importance: 1, want: 0.5},
		{name: "importance scales score", lastAccess: 1_000_000, now: 87_400_000, importance: 0.5, want: 0.25},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			score, err := ImportanceWeightedPowerLawScore(tc.lastAccess, tc.now, oneDayMillis, 1, tc.importance)
			if err != nil {
				t.Fatalf("ImportanceWeightedPowerLawScore() error = %v", err)
			}
			if math.Abs(score-tc.want) > 1e-12 {
				t.Fatalf("ImportanceWeightedPowerLawScore() = %v, want %v", score, tc.want)
			}
		})
	}
}

func TestImportanceWeightedPowerLawScoreClampsFutureAccessToOne(t *testing.T) {
	score, err := ImportanceWeightedPowerLawScore(87_400_000, 1_000_000, oneDayMillis, 1, 1)
	if err != nil {
		t.Fatalf("ImportanceWeightedPowerLawScore() error = %v", err)
	}
	if math.Abs(score-1.0) > 1e-12 {
		t.Fatalf("ImportanceWeightedPowerLawScore() = %v, want 1", score)
	}
}

func TestImportanceWeightedPowerLawScoreRejectsNonPositiveScale(t *testing.T) {
	_, err := ImportanceWeightedPowerLawScore(1_000_000, 1_000_000, 0, 1, 0)
	if err == nil {
		t.Fatal("ImportanceWeightedPowerLawScore() error = nil, want error")
	}
}

func TestImportanceWeightedPowerLawScoreRejectsOutOfRangeParameters(t *testing.T) {
	for _, scaleMillis := range []int64{999, 315_576_000_001} {
		_, err := ImportanceWeightedPowerLawScore(1_000_000, 1_000_000, scaleMillis, 1, 1)
		if err == nil {
			t.Fatalf("scaleMillis %d error = nil, want error", scaleMillis)
		}
	}
	for _, exponent := range []float64{0.09, 10.01} {
		_, err := ImportanceWeightedPowerLawScore(1_000_000, 1_000_000, oneDayMillis, exponent, 1)
		if err == nil {
			t.Fatalf("exponent %v error = nil, want error", exponent)
		}
	}
}

func TestImportanceWeightedPowerLawScoreRejectsInvalidFloatParameters(t *testing.T) {
	cases := []struct {
		name       string
		exponent   float64
		importance float64
	}{
		{name: "zero exponent", exponent: 0, importance: 0},
		{name: "negative exponent", exponent: -1, importance: 0},
		{name: "nonfinite exponent", exponent: math.NaN(), importance: 0},
		{name: "negative importance", exponent: 1, importance: -0.1},
		{name: "importance over one", exponent: 1, importance: 1.1},
		{name: "nonfinite importance", exponent: 1, importance: math.Inf(1)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ImportanceWeightedPowerLawScore(1_000_000, 1_000_000, oneDayMillis, tc.exponent, tc.importance)
			if err == nil {
				t.Fatal("ImportanceWeightedPowerLawScore() error = nil, want error")
			}
		})
	}
}
