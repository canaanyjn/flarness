package cmd

import (
	"github.com/canaanyjn/flarness/internal/ipc"
	"github.com/canaanyjn/flarness/internal/model"
	"github.com/spf13/cobra"
)

var (
	inspectMaxDepth int
)

var inspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "Inspect the Widget tree of the running Flutter app",
	Long: `Inspect the Widget tree of the running Flutter app via VM Service Protocol.

Returns a structured JSON representation of the current Widget hierarchy,
including widget types, properties, and children.

This is useful for AI to understand what's currently rendered on screen
without needing visual interpretation.`,
	Run: func(cmd *cobra.Command, args []string) {
		client := ipc.NewClient()
		if !client.IsRunning() {
			printError("daemon is not running — run 'flarness start' first")
		}

		cmdArgs := map[string]any{}
		if inspectMaxDepth > 0 {
			cmdArgs["max_depth"] = float64(inspectMaxDepth)
		}

		resp, err := client.Send(model.Command{
			Cmd:  "inspect",
			Args: cmdArgs,
		})
		if err != nil {
			printError("inspect failed: " + err.Error())
		}

		if !resp.OK {
			printError(resp.Error)
		}

		printJSON(resp.Data)
	},
}

func init() {
	inspectCmd.Flags().IntVar(&inspectMaxDepth, "max-depth", 0, "max depth of the widget tree (0 = unlimited)")
	rootCmd.AddCommand(inspectCmd)
}
