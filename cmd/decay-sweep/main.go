// Command decay-sweep evaluates and prunes JSONL memory records locally.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

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
	if *mode != "dry-run" {
		return fmt.Errorf("mode %q is not implemented", *mode)
	}

	input, err := os.Open(*inputPath)
	if err != nil {
		return fmt.Errorf("open input JSONL: %w", err)
	}
	defer input.Close()

	result, err := sweep.ScanJSONL(input, sweep.ExponentialPolicy{
		NowMillis:      *nowMillis,
		HalfLifeMillis: *halfLifeMillis,
		Threshold:      *threshold,
	})
	if err != nil {
		return err
	}
	return json.NewEncoder(stdout).Encode(auditLog{
		Mode:    *mode,
		Scanned: result.Scanned,
		Pruned:  result.Pruned,
	})
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
