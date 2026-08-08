// Package sweep evaluates JSONL memory records against deterministic decay policies.
package sweep

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/iamfaham/decay-library/pkg/decay"
)

type ExponentialPolicy struct {
	NowMillis      int64
	HalfLifeMillis int64
	Threshold      float64
	// Workers is the bounded number of concurrent score calculations. Zero uses one worker.
	Workers int
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
	ID             string  `json:"id"`
	LastAccessedMs int64   `json:"last_accessed_ms"`
	Importance     float64 `json:"importance"`
}

type workItem struct {
	line   int
	raw    []byte
	record memoryRecord
}

type evaluatedItem struct {
	item     workItem
	decision Decision
	err      error
}

// ArchiveJSONL writes retained and pruned records in their original input order.
// Score calculations may run concurrently according to policy.Workers.
func ArchiveJSONL(
	reader io.Reader,
	retainedWriter io.Writer,
	archiveWriter io.Writer,
	policy ExponentialPolicy,
) (Result, error) {
	evaluated, err := evaluateJSONL(reader, policy)
	if err != nil {
		return Result{}, err
	}

	result := resultFromEvaluated(evaluated)
	for _, item := range evaluated {
		writer := retainedWriter
		if item.decision.Prune {
			writer = archiveWriter
		}
		if _, err := writer.Write(append(item.item.raw, '\n')); err != nil {
			return Result{}, fmt.Errorf("write record %q: %w", item.decision.ID, err)
		}
	}
	return result, nil
}

// ScanJSONL evaluates records without writing retained or archived JSONL output.
func ScanJSONL(reader io.Reader, policy ExponentialPolicy) (Result, error) {
	evaluated, err := evaluateJSONL(reader, policy)
	if err != nil {
		return Result{}, err
	}
	return resultFromEvaluated(evaluated), nil
}

func evaluateJSONL(reader io.Reader, policy ExponentialPolicy) ([]evaluatedItem, error) {
	if policy.Threshold < 0 || policy.Threshold > 1 {
		return nil, fmt.Errorf("threshold must be between 0 and 1")
	}
	if policy.Workers < 0 {
		return nil, fmt.Errorf("workers must be positive when specified")
	}

	items, err := readJSONL(reader)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}

	workers := policy.Workers
	if workers == 0 {
		workers = 1
	}
	if workers > len(items) {
		workers = len(items)
	}

	jobs := make(chan workItem)
	results := make(chan evaluatedItem, workers)
	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			for item := range jobs {
				score, err := decay.ExponentialScore(
					item.record.LastAccessedMs,
					policy.NowMillis,
					policy.HalfLifeMillis,
				)
				result := evaluatedItem{item: item, err: err}
				if err == nil {
					result.decision = Decision{
						ID:    item.record.ID,
						Score: score,
						Prune: score < policy.Threshold,
					}
				}
				results <- result
			}
		}()
	}
	go func() {
		for _, item := range items {
			jobs <- item
		}
		close(jobs)
		group.Wait()
		close(results)
	}()

	evaluated := make([]evaluatedItem, len(items))
	for result := range results {
		evaluated[result.item.line-1] = result
	}
	for _, result := range evaluated {
		if result.err != nil {
			return nil, fmt.Errorf("score record %q: %w", result.item.record.ID, result.err)
		}
	}
	return evaluated, nil
}

func readJSONL(reader io.Reader) ([]workItem, error) {
	var items []workItem
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for line := 1; scanner.Scan(); line++ {
		rawRecord := append([]byte(nil), scanner.Bytes()...)
		var record memoryRecord
		if err := json.Unmarshal(rawRecord, &record); err != nil {
			return nil, fmt.Errorf("decode JSONL record on line %d: %w", line, err)
		}
		items = append(items, workItem{line: line, raw: rawRecord, record: record})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read JSONL records: %w", err)
	}
	return items, nil
}

func resultFromEvaluated(evaluated []evaluatedItem) Result {
	result := Result{Decisions: make([]Decision, 0, len(evaluated))}
	for _, item := range evaluated {
		result.Scanned++
		if item.decision.Prune {
			result.Pruned++
		}
		result.Decisions = append(result.Decisions, item.decision)
	}
	return result
}
