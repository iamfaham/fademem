package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
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
