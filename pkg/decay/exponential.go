// Package decay provides deterministic, local memory-retention score models.
package decay

import (
	"fmt"
	"math"
)

const (
	minDurationMillis int64 = 1_000
	maxDurationMillis int64 = 315_576_000_000
)

func isValidDurationMillis(value int64) bool {
	return value >= minDurationMillis && value <= maxDurationMillis
}

// ExponentialScore returns normalized exponential retention.
func ExponentialScore(lastAccessed, now, halfLifeMillis int64) (float64, error) {
	if halfLifeMillis <= 0 {
		return 0, fmt.Errorf("halfLifeMillis must be positive")
	}
	if !isValidDurationMillis(halfLifeMillis) {
		return 0, fmt.Errorf("halfLifeMillis must be between %d and %d", minDurationMillis, maxDurationMillis)
	}
	elapsed := now - lastAccessed
	if elapsed < 0 {
		elapsed = 0
	}
	return math.Exp2(-float64(elapsed) / float64(halfLifeMillis)), nil
}
