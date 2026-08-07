// Package decay provides deterministic, local memory-retention score models.
package decay

import (
	"fmt"
	"math"
)

// ExponentialScore returns normalized exponential retention.
func ExponentialScore(lastAccessed, now, halfLifeSeconds int64) (float64, error) {
	if halfLifeSeconds <= 0 {
		return 0, fmt.Errorf("halfLifeSeconds must be positive")
	}
	elapsed := now - lastAccessed
	if elapsed < 0 {
		elapsed = 0
	}
	return math.Exp2(-float64(elapsed) / float64(halfLifeSeconds)), nil
}
