package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	appVersion string
	verbose    bool
)

var rootCmd = &cobra.Command{
	Use:   "flarness",
	Short: "AI-friendly Flutter development harness",
	Long: `Flarness — Flutter AI Harness

An AI-friendly tool that lets AI Agents drive the complete Flutter development loop.
	Provides structured JSON responses for every operation.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// SetVersion sets the version string (injected from main.go).
func SetVersion(v string) {
	appVersion = v
	rootCmd.Version = v
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "enable verbose output")
}

// printJSON marshals v to JSON and prints to stdout.
func printJSON(v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "json error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}

// printError prints a JSON error response and exits with code 1.
func printError(msg string) {
	printJSON(map[string]any{
		"status":  "error",
		"message": msg,
	})
	os.Exit(1)
}
