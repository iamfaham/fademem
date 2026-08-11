// Package main exposes the fixed-width C ABI for the decay engine.
package main

/*
#include <stdint.h>
*/
import "C"

import (
	"github.com/iamfaham/fademem/internal/ffi"
)

const (
	statusOK              int32 = int32(ffi.StatusOK)
	statusInvalidArgument int32 = int32(ffi.StatusInvalidArgument)
	statusNullOutput      int32 = int32(ffi.StatusNullOutput)
)

func scoreExponentialForFFI(lastAccessed, now, halfLifeMillis int64) (int32, float64) {
	status, score := ffi.ScoreExponential(lastAccessed, now, halfLifeMillis)
	return int32(status), score
}

func scorePowerLawForFFI(lastAccessed, now, scaleMillis int64, exponent, importance float64) (int32, float64) {
	status, score := ffi.ScorePowerLaw(lastAccessed, now, scaleMillis, exponent, importance)
	return int32(status), score
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

// DecayScorePowerLaw writes the importance-weighted power-law score to outScore.
// It accepts and returns only fixed-width scalar C values.
//export DecayScorePowerLaw
func DecayScorePowerLaw(
	lastAccessed C.int64_t,
	now C.int64_t,
	scaleMillis C.int64_t,
	exponent C.double,
	importance C.double,
	outScore *C.double,
) C.int32_t {
	if outScore == nil {
		return C.int32_t(statusNullOutput)
	}
	status, score := scorePowerLawForFFI(
		int64(lastAccessed),
		int64(now),
		int64(scaleMillis),
		float64(exponent),
		float64(importance),
	)
	if status != statusOK {
		return C.int32_t(status)
	}
	*outScore = C.double(score)
	return C.int32_t(statusOK)
}

func main() {}
