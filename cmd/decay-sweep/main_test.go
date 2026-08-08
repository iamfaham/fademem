package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDryRunLeavesInputUntouchedAndEmitsAuditLog(t *testing.T) {
	tempDir := "test-output-dry-run"
	if err := os.MkdirAll(tempDir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })
	inputPath := filepath.Join(tempDir, "memories.jsonl")
	input := "{\"id\":\"expired\",\"last_accessed_ms\":-85400000,\"importance\":1}\n" +
		"{\"id\":\"fresh\",\"last_accessed_ms\":87400000,\"importance\":1}\n"
	if err := os.WriteFile(inputPath, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	err := run([]string{
		"--input", inputPath,
		"--mode", "dry-run",
		"--now-ms", "87400000",
		"--half-life-ms", "86400000",
		"--threshold", "0.5",
	}, &stdout)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	gotInput, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotInput) != input {
		t.Fatalf("input changed during dry run: got %q, want %q", gotInput, input)
	}

	var audit map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &audit); err != nil {
		t.Fatalf("audit log is not JSON: %v; output=%q", err, stdout.String())
	}
	if audit["mode"] != "dry-run" || audit["scanned"] != float64(2) || audit["pruned"] != float64(1) {
		t.Fatalf("audit = %#v, want dry-run with 2 scanned and 1 pruned", audit)
	}
}

func TestRunArchiveReplacesInputAndWritesPrunedRecords(t *testing.T) {
	tempDir := "test-output-archive"
	if err := os.MkdirAll(tempDir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })
	inputPath := filepath.Join(tempDir, "memories.jsonl")
	archivePath := filepath.Join(tempDir, "archive.jsonl")
	expired := "{\"id\":\"expired\",\"last_accessed_ms\":-85400000,\"importance\":1}\n"
	fresh := "{\"id\":\"fresh\",\"last_accessed_ms\":87400000,\"importance\":1}\n"
	if err := os.WriteFile(inputPath, []byte(expired+fresh), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	err := run([]string{
		"--input", inputPath,
		"--archive", archivePath,
		"--mode", "archive",
		"--now-ms", "87400000",
		"--half-life-ms", "86400000",
		"--threshold", "0.5",
	}, &stdout)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	gotInput, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotInput) != fresh {
		t.Fatalf("retained input = %q, want %q", gotInput, fresh)
	}
	gotArchive, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotArchive) != expired {
		t.Fatalf("archive output = %q, want %q", gotArchive, expired)
	}
	var audit map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &audit); err != nil {
		t.Fatalf("audit log is not JSON: %v; output=%q", err, stdout.String())
	}
	if audit["mode"] != "archive" || audit["scanned"] != float64(2) || audit["pruned"] != float64(1) {
		t.Fatalf("audit = %#v, want archive with 2 scanned and 1 pruned", audit)
	}
}

func TestRunArchiveRejectsInputAsArchiveDestinationWithoutMutation(t *testing.T) {
	tempDir := "test-output-archive-same-path"
	if err := os.MkdirAll(tempDir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })
	inputPath := filepath.Join(tempDir, "memories.jsonl")
	input := "{\"id\":\"expired\",\"last_accessed_ms\":-85400000,\"importance\":1}\n"
	if err := os.WriteFile(inputPath, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}

	err := run([]string{
		"--input", inputPath,
		"--archive", inputPath,
		"--mode", "archive",
		"--now-ms", "87400000",
		"--half-life-ms", "86400000",
		"--threshold", "0.5",
	}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("run() error = nil, want input/archive path rejection")
	}
	gotInput, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotInput) != input {
		t.Fatalf("input changed after rejected archive target: got %q, want %q", gotInput, input)
	}
}

func TestRunDeleteRequiresExplicitConfirmationWithoutMutation(t *testing.T) {
	tempDir := "test-output-delete-unconfirmed"
	if err := os.MkdirAll(tempDir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })
	inputPath := filepath.Join(tempDir, "memories.jsonl")
	input := "{\"id\":\"expired\",\"last_accessed_ms\":-85400000,\"importance\":1}\n"
	if err := os.WriteFile(inputPath, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}

	err := run([]string{
		"--input", inputPath,
		"--mode", "delete",
		"--now-ms", "87400000",
		"--half-life-ms", "86400000",
		"--threshold", "0.5",
	}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "--confirm-delete") {
		t.Fatalf("run() error = %v, want explicit --confirm-delete requirement", err)
	}
	gotInput, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotInput) != input {
		t.Fatalf("input changed by unconfirmed delete: got %q, want %q", gotInput, input)
	}
}

func TestRunConfirmedDeleteAtomicallyRetainsOnlyUnprunedRecords(t *testing.T) {
	tempDir := "test-output-delete-confirmed"
	if err := os.MkdirAll(tempDir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })
	inputPath := filepath.Join(tempDir, "memories.jsonl")
	expired := "{\"id\":\"expired\",\"last_accessed_ms\":-85400000,\"importance\":1}\n"
	fresh := "{\"id\":\"fresh\",\"last_accessed_ms\":87400000,\"importance\":1}\n"
	if err := os.WriteFile(inputPath, []byte(expired+fresh), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	err := run([]string{
		"--input", inputPath,
		"--mode", "delete",
		"--confirm-delete",
		"--now-ms", "87400000",
		"--half-life-ms", "86400000",
		"--threshold", "0.5",
	}, &stdout)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	gotInput, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotInput) != fresh {
		t.Fatalf("input after confirmed delete = %q, want %q", gotInput, fresh)
	}
	var audit map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &audit); err != nil {
		t.Fatalf("audit log is not JSON: %v; output=%q", err, stdout.String())
	}
	if audit["mode"] != "delete" || audit["scanned"] != float64(2) || audit["pruned"] != float64(1) {
		t.Fatalf("audit = %#v, want delete with 2 scanned and 1 pruned", audit)
	}
}

func TestRunWritesJSONLAuditWithDecisionReasons(t *testing.T) {
	tempDir := "test-output-audit"
	if err := os.MkdirAll(tempDir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })
	inputPath := filepath.Join(tempDir, "memories.jsonl")
	auditPath := filepath.Join(tempDir, "audit.jsonl")
	input := "{\"id\":\"expired\",\"last_accessed_ms\":-85400000,\"importance\":1}\n" +
		"{\"id\":\"fresh\",\"last_accessed_ms\":87400000,\"importance\":1}\n"
	if err := os.WriteFile(inputPath, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}

	err := run([]string{
		"--input", inputPath,
		"--audit", auditPath,
		"--mode", "dry-run",
		"--workers", "4",
		"--now-ms", "87400000",
		"--half-life-ms", "86400000",
		"--threshold", "0.5",
	}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	auditFile, err := os.Open(auditPath)
	if err != nil {
		t.Fatal(err)
	}
	defer auditFile.Close()

	var events []auditEvent
	scanner := bufio.NewScanner(auditFile)
	for scanner.Scan() {
		var event auditEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			t.Fatalf("decode audit event: %v", err)
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("audit events = %d, want 2", len(events))
	}
	if events[0].ID != "expired" || events[0].Action != "pruned" || events[0].Reason != "score_below_threshold" {
		t.Fatalf("first audit event = %+v, want expired pruned for score_below_threshold", events[0])
	}
	if events[1].ID != "fresh" || events[1].Action != "retained" || events[1].Reason != "score_at_or_above_threshold" {
		t.Fatalf("second audit event = %+v, want fresh retained for score_at_or_above_threshold", events[1])
	}
}

func TestRunRejectsAuditPathEqualInputWithoutMutation(t *testing.T) {
	tempDir := "test-output-audit-same-input"
	if err := os.MkdirAll(tempDir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })
	inputPath := filepath.Join(tempDir, "memories.jsonl")
	input := "{\"id\":\"expired\",\"last_accessed_ms\":-85400000,\"importance\":1}\n"
	if err := os.WriteFile(inputPath, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}

	err := run([]string{
		"--input", inputPath,
		"--audit", inputPath,
		"--mode", "dry-run",
		"--now-ms", "87400000",
		"--half-life-ms", "86400000",
		"--threshold", "0.5",
	}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "--audit") {
		t.Fatalf("run() error = %v, want audit/input path rejection", err)
	}
	gotInput, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotInput) != input {
		t.Fatalf("input changed by rejected audit target: got %q, want %q", gotInput, input)
	}
}
