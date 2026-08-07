package decay

import (
	"fmt"
	"math"
)

func isFinitePositive(value float64) bool {
	return value > 0 && !math.IsInf(value, 0) && !math.IsNaN(value)
}

func isFiniteUnitInterval(value float64) bool {
	return value >= 0 && value <= 1 && !math.IsInf(value, 0) && !math.IsNaN(value)
}

// ImportanceWeightedPowerLawScore returns retention using
// (1 + elapsed/(scaleMillis*(1+importance)))^-exponent.
func ImportanceWeightedPowerLawScore(
	lastAccessed, now, scaleMillis int64,
	exponent, importance float64,
) (float64, error) {
	if scaleMillis <= 0 {
		return 0, fmt.Errorf("scaleMillis must be positive")
	}
	if !isFinitePositive(exponent) {
		return 0, fmt.Errorf("exponent must be finite and positive")
	}
	if !isFiniteUnitInterval(importance) {
		return 0, fmt.Errorf("importance must be finite and in [0, 1]")
	}
	elapsed := now - lastAccessed
	if elapsed < 0 {
		elapsed = 0
	}
	return math.Pow(1+float64(elapsed)/(float64(scaleMillis)*(1+importance)), -exponent), nil
}
