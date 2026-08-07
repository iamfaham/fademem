// Package main exposes the fixed-width C ABI for the decay engine.
package main

/*
#include <stdint.h>
*/
import "C"

import (
	"github.com/iamfaham/decay-library/pkg/decay"
)

const (
	statusOK              int32 = 0
	statusInvalidArgument int32 = 1
	statusNullOutput      int32 = 2
)

func scoreExponentialForFFI(lastAccessed, now, halfLifeMillis int64) (int32, float64) {
	score, err := decay.ExponentialScore(lastAccessed, now, halfLifeMillis)
	if err != nil {
		return statusInvalidArgument, 0
	}
	return statusOK, score
}

// DecayScoreExponential writes the exponential score to outScore.
// It accepts and returns only fixed-width scalar C values.
//export DecayScoreExponential
func DecayScoreExponential(
	lastAccessed C.int64_t,
	now C.int64_t,
	halfLifeMillis C.int64_t,
	outScore *C.double,
) C.int32_t {
	if outScore == nil {
		return C.int32_t(statusNullOutput)
	}
	status, score := scoreExponentialForFFI(
		int64(lastAccessed),
		int64(now),
		int64(halfLifeMillis),
	)
	if status != statusOK {
		return C.int32_t(status)
	}
	*outScore = C.double(score)
	return C.int32_t(statusOK)
}

func main() {}
