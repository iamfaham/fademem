// Package ffi translates deterministic core results into C ABI scalar status codes.
package ffi

import "github.com/iamfaham/decay-library/pkg/decay"

type Status int32

const (
	StatusOK Status = iota
	StatusInvalidArgument
	StatusNullOutput
)

func ScoreExponential(lastAccessed, now, halfLifeMillis int64) (Status, float64) {
	score, err := decay.ExponentialScore(lastAccessed, now, halfLifeMillis)
	if err != nil {
		return StatusInvalidArgument, 0
	}
	return StatusOK, score
}

func ScorePowerLaw(lastAccessed, now, scaleMillis int64, exponent, importance float64) (Status, float64) {
	score, err := decay.ImportanceWeightedPowerLawScore(lastAccessed, now, scaleMillis, exponent, importance)
	if err != nil {
		return StatusInvalidArgument, 0
	}
	return StatusOK, score
}
