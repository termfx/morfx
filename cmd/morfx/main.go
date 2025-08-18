package main

import (
	"fmt"
	"os"

	"github.com/termfx/morfx/internal/cli"
	"github.com/termfx/morfx/internal/config"
	"github.com/termfx/morfx/internal/model"
)

// main is the entry point for morfx, the command-line tool for file transformations.
// It parses command-line flags, builds a configuration, and runs the transformation.
func main() {
	cfg, files, err := config.BuildConfigFromFlags(os.Args[1:])
	exitIf(err, "Error building configuration")

	out := cli.Run(files, cfg)
	handleOutputAndExit(out.Results, out.Error, cfg, out.ExitCode, out.FileErrorCount)
}

func handleOutputAndExit(res []model.Result, err error, cfg *model.Config, exitCode int, fileErrors int) {
	if err != nil {
		config.PrintFatal(err, cfg.JSONOutput)
	}
	if fileErrors > 0 {
		config.PrintFatal(fmt.Errorf("%d files had errors", fileErrors), cfg.JSONOutput)
	}

	for _, r := range res {
		config.PrintResultCLI(&r, cfg)
	}

	config.PrintSummary(res, cfg)

	os.Exit(exitCode)
}

func exitIf(err error, msg string) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", msg, err)
		os.Exit(1)
	}
}
