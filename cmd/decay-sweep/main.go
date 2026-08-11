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

	"github.com/iamfaham/fademem/internal/sweep"
)

type auditLog struct {
	Mode    string `json:"mode"`
	Scanned int    `json:"scanned"`
	Pruned  int    `json:"pruned"`
}

type auditEvent struct {
	Mode   string  `json:"mode"`
	ID     string  `json:"id"`
	Score  float64 `json:"score"`
	Action string  `json:"action"`
	Reason string  `json:"reason"`
}

func run(args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("decay-sweep", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	inputPath := flags.String("input", "", "JSONL memory-store file")
	archivePath := flags.String("archive", "", "JSONL archive file (required for archive mode)")
	auditPath := flags.String("audit", "", "JSONL audit output file")
	mode := flags.String("mode", "dry-run", "dry-run, archive, or delete")
	confirmDelete := flags.Bool("confirm-delete", false, "required to delete pruned records without archiving")
	model := flags.String("model", "exponential", "exponential or power-law")
	workers := flags.Int("workers", 1, "bounded concurrent score calculations")
	nowMillis := flags.Int64("now-ms", 0, "evaluation time as Unix epoch milliseconds")
	halfLifeMillis := flags.Int64("half-life-ms", 0, "exponential half-life in milliseconds")
	scaleMillis := flags.Int64("scale-ms", 0, "power-law scale in milliseconds")
	exponent := flags.Float64("exponent", 0, "power-law exponent")
	threshold := flags.Float64("threshold", 0, "prune scores strictly below this value")
	showVersion := flags.Bool("version", false, "print version and exit")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *showVersion {
		return json.NewEncoder(stdout).Encode(map[string]string{"version": "0.1.0"})
	}
	if *model != "exponential" && *model != "power-law" {
		return fmt.Errorf("model %q is not implemented", *model)
	}
	if *inputPath == "" {
		return fmt.Errorf("--input is required")
	}
	if *mode != "dry-run" && *mode != "archive" && *mode != "delete" {
		return fmt.Errorf("mode %q is not implemented", *mode)
	}
	if *mode == "delete" && !*confirmDelete {
		return fmt.Errorf("--confirm-delete is required for delete mode")
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
	if *auditPath != "" {
		samePath, err := pathsEqual(*inputPath, *auditPath)
		if err != nil {
			return err
		}
		if samePath {
			return fmt.Errorf("--audit must not refer to --input")
		}
		if *mode == "archive" {
			samePath, err = pathsEqual(*archivePath, *auditPath)
			if err != nil {
				return err
			}
			if samePath {
				return fmt.Errorf("--audit must not refer to --archive")
			}
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
		Workers:        *workers,
	}
	powerLawPolicy := sweep.PowerLawPolicy{
		NowMillis: *nowMillis, ScaleMillis: *scaleMillis, Exponent: *exponent, Threshold: *threshold, Workers: *workers,
	}

	var result sweep.Result
	switch *mode {
	case "dry-run":
		if *model == "power-law" {
			result, err = dryRunPowerLaw(input, powerLawPolicy, *auditPath, *mode)
		} else {
			result, err = dryRun(input, policy, *auditPath, *mode)
		}
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
		if *model == "power-law" {
			result, err = archiveInputPowerLaw(*inputPath, *archivePath, powerLawPolicy, *auditPath, *mode)
		} else {
			result, err = archiveInput(*inputPath, *archivePath, policy, *auditPath, *mode)
		}
		if err != nil {
			return err
		}
	case "delete":
		if err := input.Close(); err != nil {
			return fmt.Errorf("close input JSONL: %w", err)
		}
		if *model == "power-law" {
			result, err = deleteInputPowerLaw(*inputPath, powerLawPolicy, *auditPath, *mode)
		} else {
			result, err = deleteInput(*inputPath, policy, *auditPath, *mode)
		}
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

func dryRunPowerLaw(input io.Reader, policy sweep.PowerLawPolicy, auditPath, mode string) (sweep.Result, error) {
	var audit *auditOutput
	var err error
	if auditPath != "" {
		audit, err = newAuditOutput(auditPath, mode)
		if err != nil {
			return sweep.Result{}, err
		}
		defer audit.abort()
	}
	summary, err := sweep.ProcessPowerLawJSONL(input, policy, func(record sweep.RecordResult) error {
		if audit == nil {
			return nil
		}
		return audit.write(record.Decision)
	})
	if err != nil {
		return sweep.Result{}, err
	}
	if audit != nil {
		if err := audit.commit(); err != nil {
			return sweep.Result{}, err
		}
	}
	return sweep.Result{Scanned: summary.Scanned, Pruned: summary.Pruned}, nil
}

func dryRun(input io.Reader, policy sweep.ExponentialPolicy, auditPath, mode string) (sweep.Result, error) {
	var audit *auditOutput
	var err error
	if auditPath != "" {
		audit, err = newAuditOutput(auditPath, mode)
		if err != nil {
			return sweep.Result{}, err
		}
		defer audit.abort()
	}

	summary, err := sweep.ProcessJSONL(input, policy, func(record sweep.RecordResult) error {
		if audit == nil {
			return nil
		}
		return audit.write(record.Decision)
	})
	if err != nil {
		return sweep.Result{}, err
	}
	if audit != nil {
		if err := audit.commit(); err != nil {
			return sweep.Result{}, err
		}
	}
	return sweep.Result{Scanned: summary.Scanned, Pruned: summary.Pruned}, nil
}

type auditOutput struct {
	path     string
	tempPath string
	file     *os.File
	mode     string
}

func newAuditOutput(path, mode string) (*auditOutput, error) {
	file, tempPath, err := createTempOutput(path)
	if err != nil {
		return nil, err
	}
	return &auditOutput{path: path, tempPath: tempPath, file: file, mode: mode}, nil
}

func (output *auditOutput) write(decision sweep.Decision) error {
	event := auditEvent{Mode: output.mode, ID: decision.ID, Score: decision.Score, Action: "retained", Reason: "score_at_or_above_threshold"}
	if decision.Prune {
		event.Action = "pruned"
		event.Reason = "score_below_threshold"
	}
	if err := json.NewEncoder(output.file).Encode(event); err != nil {
		return fmt.Errorf("write audit event for record %q: %w", decision.ID, err)
	}
	return nil
}

func (output *auditOutput) commit() error {
	if err := output.file.Sync(); err != nil {
		return fmt.Errorf("sync audit JSONL: %w", err)
	}
	if err := output.file.Close(); err != nil {
		return fmt.Errorf("close audit JSONL: %w", err)
	}
	if err := os.Rename(output.tempPath, output.path); err != nil {
		return fmt.Errorf("atomically replace audit JSONL: %w", err)
	}
	return nil
}

func (output *auditOutput) abort() {
	output.file.Close()
	os.Remove(output.tempPath)
}

type jsonlProcessor func(io.Reader, func(sweep.RecordResult) error) (sweep.Summary, error)

func archiveInput(inputPath, archivePath string, policy sweep.ExponentialPolicy, auditPath, mode string) (sweep.Result, error) {
	return archiveInputWithProcessor(inputPath, archivePath, auditPath, mode, func(reader io.Reader, visit func(sweep.RecordResult) error) (sweep.Summary, error) {
		return sweep.ProcessJSONL(reader, policy, visit)
	})
}

func archiveInputPowerLaw(inputPath, archivePath string, policy sweep.PowerLawPolicy, auditPath, mode string) (sweep.Result, error) {
	return archiveInputWithProcessor(inputPath, archivePath, auditPath, mode, func(reader io.Reader, visit func(sweep.RecordResult) error) (sweep.Summary, error) {
		return sweep.ProcessPowerLawJSONL(reader, policy, visit)
	})
}

func archiveInputWithProcessor(inputPath, archivePath, auditPath, mode string, process jsonlProcessor) (sweep.Result, error) {
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

	var audit *auditOutput
	if auditPath != "" {
		audit, err = newAuditOutput(auditPath, mode)
		if err != nil {
			retained.Close()
			archived.Close()
			return sweep.Result{}, err
		}
		defer audit.abort()
	}

	summary, processErr := process(input, func(record sweep.RecordResult) error {
		writer := retained
		if record.Decision.Prune {
			writer = archived
		}
		if _, err := writer.Write(append(record.Raw, '\n')); err != nil {
			return fmt.Errorf("write record %q: %w", record.Decision.ID, err)
		}
		if audit != nil {
			return audit.write(record.Decision)
		}
		return nil
	})
	result := sweep.Result{Scanned: summary.Scanned, Pruned: summary.Pruned}
	err = processErr
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
	if audit != nil {
		if err := audit.commit(); err != nil {
			return sweep.Result{}, err
		}
	}
	return result, nil
}

func deleteInput(inputPath string, policy sweep.ExponentialPolicy, auditPath, mode string) (sweep.Result, error) {
	return deleteInputWithProcessor(inputPath, auditPath, mode, func(reader io.Reader, visit func(sweep.RecordResult) error) (sweep.Summary, error) {
		return sweep.ProcessJSONL(reader, policy, visit)
	})
}

func deleteInputPowerLaw(inputPath string, policy sweep.PowerLawPolicy, auditPath, mode string) (sweep.Result, error) {
	return deleteInputWithProcessor(inputPath, auditPath, mode, func(reader io.Reader, visit func(sweep.RecordResult) error) (sweep.Summary, error) {
		return sweep.ProcessPowerLawJSONL(reader, policy, visit)
	})
}

func deleteInputWithProcessor(inputPath, auditPath, mode string, process jsonlProcessor) (sweep.Result, error) {
	input, err := os.Open(inputPath)
	if err != nil {
		return sweep.Result{}, fmt.Errorf("reopen input JSONL for delete: %w", err)
	}
	defer input.Close()

	retained, retainedTempPath, err := createTempOutput(inputPath)
	if err != nil {
		return sweep.Result{}, err
	}
	defer os.Remove(retainedTempPath)

	var audit *auditOutput
	if auditPath != "" {
		audit, err = newAuditOutput(auditPath, mode)
		if err != nil {
			retained.Close()
			return sweep.Result{}, err
		}
		defer audit.abort()
	}

	summary, processErr := process(input, func(record sweep.RecordResult) error {
		if !record.Decision.Prune {
			if _, err := retained.Write(append(record.Raw, '\n')); err != nil {
				return fmt.Errorf("write retained record %q: %w", record.Decision.ID, err)
			}
		}
		if audit != nil {
			return audit.write(record.Decision)
		}
		return nil
	})
	result := sweep.Result{Scanned: summary.Scanned, Pruned: summary.Pruned}
	err = processErr
	closeInputErr := input.Close()
	if err != nil {
		retained.Close()
		return sweep.Result{}, err
	}
	if closeInputErr != nil {
		retained.Close()
		return sweep.Result{}, fmt.Errorf("close input JSONL: %w", closeInputErr)
	}
	if err := retained.Sync(); err != nil {
		retained.Close()
		return sweep.Result{}, fmt.Errorf("sync retained JSONL: %w", err)
	}
	if err := retained.Close(); err != nil {
		return sweep.Result{}, fmt.Errorf("close retained JSONL: %w", err)
	}
	if err := os.Rename(retainedTempPath, inputPath); err != nil {
		return sweep.Result{}, fmt.Errorf("atomically replace input JSONL after delete: %w", err)
	}
	if audit != nil {
		if err := audit.commit(); err != nil {
			return sweep.Result{}, err
		}
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
