package sweep

import (
	"bytes"
	"math"
	"strings"
	"testing"
)

func TestProcessPowerLawJSONLRejectsNullRequiredFields(t *testing.T) {
	for _, input := range []string{
		`{"id":"record","last_accessed_ms":null,"importance":0.5}`,
		`{"id":"record","last_accessed_ms":0,"importance":null}`,
	} {
		_, err := ProcessPowerLawJSONL(strings.NewReader(input), PowerLawPolicy{ScaleMillis: 1000, Exponent: 1, Threshold: 0.5}, func(RecordResult) error { return nil })
		if err == nil {
			t.Fatalf("ProcessPowerLawJSONL(%s) error = nil", input)
		}
	}
}

func TestProcessPowerLawJSONLRejectsNonFiniteThreshold(t *testing.T) {
	_, err := ProcessPowerLawJSONL(strings.NewReader(""), PowerLawPolicy{ScaleMillis: 1000, Exponent: 1, Threshold: math.NaN()}, func(RecordResult) error { return nil })
	if err == nil {
		t.Fatal("ProcessPowerLawJSONL() error = nil, want invalid threshold")
	}
}

func TestProcessPowerLawJSONLRejectsInvalidPolicyBeforeEmptyInput(t *testing.T) {
	_, err := ProcessPowerLawJSONL(strings.NewReader(""), PowerLawPolicy{ScaleMillis: 0, Exponent: 1, Threshold: 0.5}, func(RecordResult) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "scaleMillis") {
		t.Fatalf("ProcessPowerLawJSONL() error = %v, want invalid scale", err)
	}
}

func TestProcessPowerLawJSONLRejectsMissingImportance(t *testing.T) {
	_, err := ProcessPowerLawJSONL(strings.NewReader(`{"id":"record","last_accessed_ms":0}
`), PowerLawPolicy{ScaleMillis: 1000, Exponent: 1, Threshold: 0.5}, func(RecordResult) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "importance") {
		t.Fatalf("ProcessPowerLawJSONL() error = %v, want missing importance", err)
	}
}

func TestScanPowerLawJSONLPrunesOnlyScoresBelowThreshold(t *testing.T) {
	input := strings.NewReader(`{"id":"expired","last_accessed_ms":-171800000,"importance":0.5}
{"id":"boundary","last_accessed_ms":1000000,"importance":0.5}
{"id":"fresh","last_accessed_ms":87400000,"importance":0.5}
`)

	result, err := ScanPowerLawJSONL(input, PowerLawPolicy{
		NowMillis:   87_400_000,
		ScaleMillis: 86_400_000,
		Exponent:    1.0,
		Threshold:   0.25,
	})
	if err != nil {
		t.Fatalf("ScanPowerLawJSONL() error = %v", err)
	}
	if result.Scanned != 3 || result.Pruned != 1 {
		t.Fatalf("result = %+v, want 3 scanned and 1 pruned", result)
	}
	if !result.Decisions[0].Prune || result.Decisions[1].Prune || result.Decisions[2].Prune {
		t.Fatalf("decisions = %+v, want expired pruned and boundary/fresh retained", result.Decisions)
	}
}

func TestScanJSONLPrunesOnlyScoresBelowThreshold(t *testing.T) {
	input := strings.NewReader(`{"id":"expired","last_accessed_ms":-85400000,"importance":1}
{"id":"boundary","last_accessed_ms":1000000,"importance":1}
{"id":"fresh","last_accessed_ms":87400000,"importance":1}
`)

	result, err := ScanJSONL(input, ExponentialPolicy{
		NowMillis:      87_400_000,
		HalfLifeMillis: 86_400_000,
		Threshold:      0.5,
	})
	if err != nil {
		t.Fatalf("ScanJSONL() error = %v", err)
	}
	if result.Scanned != 3 {
		t.Fatalf("Scanned = %d, want 3", result.Scanned)
	}
	if result.Pruned != 1 {
		t.Fatalf("Pruned = %d, want 1", result.Pruned)
	}
	if len(result.Decisions) != 3 {
		t.Fatalf("len(Decisions) = %d, want 3", len(result.Decisions))
	}
	if !result.Decisions[0].Prune || result.Decisions[0].ID != "expired" {
		t.Fatalf("first decision = %+v, want expired record pruned", result.Decisions[0])
	}
	if result.Decisions[1].Prune || result.Decisions[1].ID != "boundary" {
		t.Fatalf("second decision = %+v, want boundary record retained", result.Decisions[1])
	}
	if result.Decisions[2].Prune || result.Decisions[2].ID != "fresh" {
		t.Fatalf("third decision = %+v, want fresh record retained", result.Decisions[2])
	}
}

func TestArchiveJSONLWritesPrunedRecordsSeparately(t *testing.T) {
	input := strings.NewReader(`{"id":"expired","last_accessed_ms":-85400000,"importance":1}
{"id":"fresh","last_accessed_ms":87400000,"importance":1}
`)
	var retained bytes.Buffer
	var archived bytes.Buffer

	result, err := ArchiveJSONL(input, &retained, &archived, ExponentialPolicy{
		NowMillis:      87_400_000,
		HalfLifeMillis: 86_400_000,
		Threshold:      0.5,
	})
	if err != nil {
		t.Fatalf("ArchiveJSONL() error = %v", err)
	}
	if result.Scanned != 2 || result.Pruned != 1 {
		t.Fatalf("result = %+v, want 2 scanned and 1 pruned", result)
	}
	if got, want := retained.String(), "{\"id\":\"fresh\",\"last_accessed_ms\":87400000,\"importance\":1}\n"; got != want {
		t.Fatalf("retained output = %q, want %q", got, want)
	}
	if got, want := archived.String(), "{\"id\":\"expired\",\"last_accessed_ms\":-85400000,\"importance\":1}\n"; got != want {
		t.Fatalf("archived output = %q, want %q", got, want)
	}
}

func TestArchiveJSONLWithWorkersPreservesRecordOrder(t *testing.T) {
	input := strings.NewReader(`{"id":"expired-first","last_accessed_ms":-85400000,"importance":1}
{"id":"fresh-first","last_accessed_ms":87400000,"importance":1}
{"id":"expired-second","last_accessed_ms":-85400000,"importance":1}
{"id":"fresh-second","last_accessed_ms":87400000,"importance":1}
`)
	var retained bytes.Buffer
	var archived bytes.Buffer

	result, err := ArchiveJSONL(input, &retained, &archived, ExponentialPolicy{
		NowMillis:      87_400_000,
		HalfLifeMillis: 86_400_000,
		Threshold:      0.5,
		Workers:        4,
	})
	if err != nil {
		t.Fatalf("ArchiveJSONL() error = %v", err)
	}
	if result.Scanned != 4 || result.Pruned != 2 {
		t.Fatalf("result = %+v, want 4 scanned and 2 pruned", result)
	}
	if got, want := retained.String(), "{\"id\":\"fresh-first\",\"last_accessed_ms\":87400000,\"importance\":1}\n{\"id\":\"fresh-second\",\"last_accessed_ms\":87400000,\"importance\":1}\n"; got != want {
		t.Fatalf("retained order = %q, want %q", got, want)
	}
	if got, want := archived.String(), "{\"id\":\"expired-first\",\"last_accessed_ms\":-85400000,\"importance\":1}\n{\"id\":\"expired-second\",\"last_accessed_ms\":-85400000,\"importance\":1}\n"; got != want {
		t.Fatalf("archive order = %q, want %q", got, want)
	}
}

func TestProcessPowerLawJSONLWithWorkersVisitsRecordsInOrder(t *testing.T) {
	input := strings.NewReader(`{"id":"expired-first","last_accessed_ms":-171800000,"importance":0.5}
{"id":"boundary-first","last_accessed_ms":1000000,"importance":0.5}
{"id":"expired-second","last_accessed_ms":-171800000,"importance":0.5}
{"id":"fresh-second","last_accessed_ms":87400000,"importance":0.5}
`)
	var visited []string
	summary, err := ProcessPowerLawJSONL(input, PowerLawPolicy{
		NowMillis:   87_400_000,
		ScaleMillis: 86_400_000,
		Exponent:    1.0,
		Threshold:   0.25,
		Workers:     4,
	}, func(record RecordResult) error {
		visited = append(visited, record.Decision.ID)
		return nil
	})
	if err != nil {
		t.Fatalf("ProcessPowerLawJSONL() error = %v", err)
	}
	if summary.Scanned != 4 || summary.Pruned != 2 {
		t.Fatalf("summary = %+v, want 4 scanned and 2 pruned", summary)
	}
	if got, want := strings.Join(visited, ","), "expired-first,boundary-first,expired-second,fresh-second"; got != want {
		t.Fatalf("visited order = %q, want %q", got, want)
	}
}

func TestProcessJSONLWithWorkersVisitsRecordsInOrder(t *testing.T) {
	input := strings.NewReader(`{"id":"expired-first","last_accessed_ms":-85400000,"importance":1}
{"id":"fresh-first","last_accessed_ms":87400000,"importance":1}
{"id":"expired-second","last_accessed_ms":-85400000,"importance":1}
{"id":"fresh-second","last_accessed_ms":87400000,"importance":1}
`)
	var visited []string
	summary, err := ProcessJSONL(input, ExponentialPolicy{
		NowMillis:      87_400_000,
		HalfLifeMillis: 86_400_000,
		Threshold:      0.5,
		Workers:        4,
	}, func(record RecordResult) error {
		visited = append(visited, record.Decision.ID)
		return nil
	})
	if err != nil {
		t.Fatalf("ProcessJSONL() error = %v", err)
	}
	if summary.Scanned != 4 || summary.Pruned != 2 {
		t.Fatalf("summary = %+v, want 4 scanned and 2 pruned", summary)
	}
	if got, want := strings.Join(visited, ","), "expired-first,fresh-first,expired-second,fresh-second"; got != want {
		t.Fatalf("visited order = %q, want %q", got, want)
	}
}
