// Package decay provides deterministic, local memory-retention score models.
package decay

import (
	"fmt"
	"math"
)

// ExponentialScore returns normalized exponential retention.
func ExponentialScore(lastAccessed, now, halfLifeMillis int64) (float64, error) {
	if halfLifeMillis <= 0 {
		return 0, fmt.Errorf("halfLifeMillis must be positive")
	}
	elapsed := now - lastAccessed
	if elapsed < 0 {
		elapsed = 0
	}
	return math.Exp2(-float64(elapsed) / float64(halfLifeMillis)), nil
}
