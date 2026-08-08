// Package sweep evaluates JSONL memory records against deterministic decay policies.
package sweep

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"

	"github.com/iamfaham/decay-library/pkg/decay"
)

type ExponentialPolicy struct {
	NowMillis      int64
	HalfLifeMillis int64
	Threshold      float64
}

type Decision struct {
	ID    string
	Score float64
	Prune bool
}

type Result struct {
	Scanned   int
	Pruned    int
	Decisions []Decision
}

type memoryRecord struct {
	ID              string  `json:"id"`
	LastAccessedMs  int64   `json:"last_accessed_ms"`
	Importance      float64 `json:"importance"`
}

func ArchiveJSONL(
	reader io.Reader,
	retainedWriter io.Writer,
	archiveWriter io.Writer,
	policy ExponentialPolicy,
) (Result, error) {
	if policy.Threshold < 0 || policy.Threshold > 1 {
		return Result{}, fmt.Errorf("threshold must be between 0 and 1")
	}

	var result Result
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for line := 1; scanner.Scan(); line++ {
		rawRecord := append([]byte(nil), scanner.Bytes()...)
		var record memoryRecord
		if err := json.Unmarshal(rawRecord, &record); err != nil {
			return Result{}, fmt.Errorf("decode JSONL record on line %d: %w", line, err)
		}
		score, err := decay.ExponentialScore(
			record.LastAccessedMs,
			policy.NowMillis,
			policy.HalfLifeMillis,
		)
		if err != nil {
			return Result{}, fmt.Errorf("score record %q: %w", record.ID, err)
		}
		decision := Decision{ID: record.ID, Score: score, Prune: score < policy.Threshold}
		result.Scanned++
		if decision.Prune {
			result.Pruned++
		}
		result.Decisions = append(result.Decisions, decision)

		writer := retainedWriter
		if decision.Prune {
			writer = archiveWriter
		}
		if _, err := writer.Write(append(rawRecord, '\n')); err != nil {
			return Result{}, fmt.Errorf("write record %q: %w", record.ID, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return Result{}, fmt.Errorf("read JSONL records: %w", err)
	}
	return result, nil
}

func ScanJSONL(reader io.Reader, policy ExponentialPolicy) (Result, error) {
	if policy.Threshold < 0 || policy.Threshold > 1 {
		return Result{}, fmt.Errorf("threshold must be between 0 and 1")
	}

	var result Result
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for line := 1; scanner.Scan(); line++ {
		var record memoryRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return Result{}, fmt.Errorf("decode JSONL record on line %d: %w", line, err)
		}
		score, err := decay.ExponentialScore(
			record.LastAccessedMs,
			policy.NowMillis,
			policy.HalfLifeMillis,
		)
		if err != nil {
			return Result{}, fmt.Errorf("score record %q: %w", record.ID, err)
		}
		decision := Decision{
			ID:    record.ID,
			Score: score,
			Prune: score < policy.Threshold,
		}
		result.Scanned++
		if decision.Prune {
			result.Pruned++
		}
		result.Decisions = append(result.Decisions, decision)
	}
	if err := scanner.Err(); err != nil {
		return Result{}, fmt.Errorf("read JSONL records: %w", err)
	}
	return result, nil
}
