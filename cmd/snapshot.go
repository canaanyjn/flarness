package cmd

import (
	"github.com/canaanyjn/flarness/internal/ipc"
	"github.com/canaanyjn/flarness/internal/model"
	"github.com/spf13/cobra"
)

var (
	snapshotMaxDepth int
)

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Capture screenshot + Widget tree of the running Flutter app",
	Long: `Capture both a screenshot and the Widget tree of the running Flutter app.

This is the recommended command for AI agents after a hot reload:
- Screenshot: visual representation of the current UI state
- Widget tree: structured data about what's rendered on screen

Together, they give AI a complete understanding of the UI without
needing to run a full test suite.`,
	Run: func(cmd *cobra.Command, args []string) {
		client := ipc.NewClient()
		if !client.IsRunning() {
			printError("daemon is not running — run 'flarness start' first")
		}

		cmdArgs := map[string]any{}
		if snapshotMaxDepth > 0 {
			cmdArgs["max_depth"] = float64(snapshotMaxDepth)
		}

		resp, err := client.Send(model.Command{
			Cmd:  "snapshot",
			Args: cmdArgs,
		})
		if err != nil {
			printError("snapshot failed: " + err.Error())
		}

		if !resp.OK {
			printError(resp.Error)
		}

		printJSON(resp.Data)
	},
}

func init() {
	snapshotCmd.Flags().IntVar(&snapshotMaxDepth, "max-depth", 0, "max depth of the widget tree (0 = unlimited)")
	rootCmd.AddCommand(snapshotCmd)
}
