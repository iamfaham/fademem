// Command decay-sweep evaluates and prunes JSONL memory records locally.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/iamfaham/decay-library/internal/sweep"
)

type auditLog struct {
	Mode    string `json:"mode"`
	Scanned int    `json:"scanned"`
	Pruned  int    `json:"pruned"`
}

func run(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("decay-sweep", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	inputPath := flags.String("input", "", "JSONL memory-store file")
	archivePath := flags.String("archive", "", "JSONL archive file (required for archive mode)")
	mode := flags.String("mode", "dry-run", "dry-run, archive, or delete")
	nowMillis := flags.Int64("now-ms", 0, "evaluation time as Unix epoch milliseconds")
	halfLifeMillis := flags.Int64("half-life-ms", 0, "exponential half-life in milliseconds")
	threshold := flags.Float64("threshold", 0, "prune scores strictly below this value")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *inputPath == "" {
		return fmt.Errorf("--input is required")
	}
	if *mode != "dry-run" && *mode != "archive" {
		return fmt.Errorf("mode %q is not implemented", *mode)
	}
	if *mode == "archive" && *archivePath == "" {
		return fmt.Errorf("--archive is required for archive mode")
	}
	if *mode == "archive" {
		samePath, err := pathsEqual(*inputPath, *archivePath)
		if err != nil {
			return err
		}
		if samePath {
			return fmt.Errorf("--archive must not refer to --input")
		}
	}

	input, err := os.Open(*inputPath)
	if err != nil {
		return fmt.Errorf("open input JSONL: %w", err)
	}
	policy := sweep.ExponentialPolicy{
		NowMillis:      *nowMillis,
		HalfLifeMillis: *halfLifeMillis,
		Threshold:      *threshold,
	}

	var result sweep.Result
	switch *mode {
	case "dry-run":
		result, err = sweep.ScanJSONL(input, policy)
		closeErr := input.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return fmt.Errorf("close input JSONL: %w", closeErr)
		}
	case "archive":
		if err := input.Close(); err != nil {
			return fmt.Errorf("close input JSONL: %w", err)
		}
		result, err = archiveInput(*inputPath, *archivePath, policy)
		if err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}
	return json.NewEncoder(stdout).Encode(auditLog{
		Mode:    *mode,
		Scanned: result.Scanned,
		Pruned:  result.Pruned,
	})
}

func archiveInput(inputPath, archivePath string, policy sweep.ExponentialPolicy) (sweep.Result, error) {
	input, err := os.Open(inputPath)
	if err != nil {
		return sweep.Result{}, fmt.Errorf("reopen input JSONL for archive: %w", err)
	}
	defer input.Close()

	retained, retainedTempPath, err := createTempOutput(inputPath)
	if err != nil {
		return sweep.Result{}, err
	}
	defer os.Remove(retainedTempPath)
	archived, archiveTempPath, err := createTempOutput(archivePath)
	if err != nil {
		retained.Close()
		return sweep.Result{}, err
	}
	defer os.Remove(archiveTempPath)

	result, err := sweep.ArchiveJSONL(input, retained, archived, policy)
	closeInputErr := input.Close()
	if err != nil {
		retained.Close()
		archived.Close()
		return sweep.Result{}, err
	}
	if closeInputErr != nil {
		retained.Close()
		archived.Close()
		return sweep.Result{}, fmt.Errorf("close input JSONL: %w", closeInputErr)
	}
	if err := retained.Sync(); err != nil {
		retained.Close()
		archived.Close()
		return sweep.Result{}, fmt.Errorf("sync retained JSONL: %w", err)
	}
	if err := archived.Sync(); err != nil {
		retained.Close()
		archived.Close()
		return sweep.Result{}, fmt.Errorf("sync archive JSONL: %w", err)
	}
	if err := retained.Close(); err != nil {
		archived.Close()
		return sweep.Result{}, fmt.Errorf("close retained JSONL: %w", err)
	}
	if err := archived.Close(); err != nil {
		return sweep.Result{}, fmt.Errorf("close archive JSONL: %w", err)
	}
	if err := os.Rename(archiveTempPath, archivePath); err != nil {
		return sweep.Result{}, fmt.Errorf("atomically replace archive JSONL: %w", err)
	}
	if err := os.Rename(retainedTempPath, inputPath); err != nil {
		return sweep.Result{}, fmt.Errorf("atomically replace input JSONL after archiving: %w", err)
	}
	return result, nil
}

func pathsEqual(first, second string) (bool, error) {
	firstAbsolute, err := filepath.Abs(first)
	if err != nil {
		return false, fmt.Errorf("resolve input path: %w", err)
	}
	secondAbsolute, err := filepath.Abs(second)
	if err != nil {
		return false, fmt.Errorf("resolve archive path: %w", err)
	}
	if runtime.GOOS == "windows" {
		return strings.EqualFold(firstAbsolute, secondAbsolute), nil
	}
	return firstAbsolute == secondAbsolute, nil
}

func createTempOutput(targetPath string) (*os.File, string, error) {
	temporary, err := os.CreateTemp(filepath.Dir(targetPath), "."+filepath.Base(targetPath)+".decay-sweep-*")
	if err != nil {
		return nil, "", fmt.Errorf("create temporary output for %q: %w", targetPath, err)
	}
	return temporary, temporary.Name(), nil
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
