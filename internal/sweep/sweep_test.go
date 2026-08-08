package sweep

import (
	"bytes"
	"strings"
	"testing"
)

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
