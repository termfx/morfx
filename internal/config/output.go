package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/pflag"

	"github.com/termfx/morfx/internal/model"
	"github.com/termfx/morfx/internal/util"
	"github.com/termfx/morfx/internal/writer"
)

func PrintResultCLI(res *model.Result, cfg *model.Config) {
	if !res.Success {
		var cliErr model.CLIError
		ok := errors.As(res.Error, &cliErr)
		if !ok {
			cliErr = model.CLIError{
				Code:    model.ErrUnknown,
				Message: res.Error.Error(),
			}
		}
		fmt.Fprintf(os.Stderr, "✗ %s: %s (%s)\n", res.File, cliErr, cliErr.Message)
		return
	}

	if cfg.RuleID == "cli-first-arg" {
		// Special case for the first argument rule
		fmt.Printf("✓ %s — No changes made (harmless mode)\n", res.File)
		diff := util.UnifiedDiff("", res.ModifiedContent, res.File, cfg.DiffContext)
		fmt.Print(diff)
		return
	}

	if cfg.Verbose {
		if res.ModifiedCount > 0 {
			fmt.Printf(
				"✓ %s — %d changes (%d bytes diff)\n",
				res.File,
				res.ModifiedCount,
				res.ChangedBytes,
			)
			// For get operations, show the matched content
			if cfg.Operation == model.OpGet {
				for i, change := range res.Changes {
					fmt.Printf(
						"  Match %d: '%s' (lines %d-%d)\n",
						i+1,
						change.New,
						change.LineStart,
						change.LineEnd,
					)
				}
			}
		} else {
			fmt.Printf("✓ %s — No changes\n", res.File)
		}
		return
	}

	if cfg.ShowDiff && res.ModifiedCount > 0 {
		diff := util.UnifiedDiff(
			res.OriginalContent,
			res.ModifiedContent,
			res.File,
			cfg.DiffContext,
		)
		fmt.Print(diff)
		return
	}

	if cfg.StdoutMode {
		fmt.Print(res.ModifiedContent)
		return
	}

	if cfg.JSONOutput {
		jsonOutput, err := json.Marshal(res)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error converting result to JSON: %v\n", err)
			return
		}
		fmt.Println(string(jsonOutput))
		return
	}
}

func PrintFatal(err error, jsonOut bool) {
	if jsonOut {
		var ce model.CLIError
		if errors.As(err, &ce) {
			fmt.Println(ce.JSON())
		} else {
			fmt.Println(model.CLIError{
				Code:    model.ErrUnknown,
				Message: err.Error(),
			}.JSON())
		}
	} else {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
}

func PrintSummary(results []model.Result, cfg *model.Config) {
	// Print writer summary if not in JSON mode and not stdout mode
	if !cfg.JSONOutput && !cfg.StdoutMode {
		var w writer.Writer
		if cfg.Operation == model.OpCommit {
			return
		} else if cfg.Operation == model.OpGet {
			// For get operations, use read-only writer that doesn't show any summary
			w = writer.NewReadOnlyWriter()
		} else if cfg.DryRun {
			w = writer.NewDryRunWriter()
		} else if cfg.Interactive {
			w = writer.NewInteractiveWriter()
		} else {
			// Default to staging writer for non-destructive behavior
			w = writer.NewStagingWriter()
		}
		summary := w.Summary()
		if summary != "" {
			fmt.Fprintf(os.Stderr, "\n%s", summary)
		}
	}
}

func PrintUsage(fs *pflag.FlagSet) {
	fmt.Fprintf(os.Stderr, "\nUsage: morfx [flags] <file1> <file2> ...\n")
	fmt.Fprintf(os.Stderr, "Quick read usage: morfx ./path/to/file1  //Reads a single file\n")
	fmt.Fprintf(os.Stderr, "\nFlags:\n")
	fs.PrintDefaults()
}
