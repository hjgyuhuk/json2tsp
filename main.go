package main

import (
	"log/slog"
	"os"

	"json2tsp/cmd"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	rootCmd := cmd.NewRootCommand()
	if err := rootCmd.Execute(); err != nil {
		logger.Error("json2tsp failed", "error", err)
		os.Exit(1)
	}
}
