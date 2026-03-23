package cmd

import (
	"github.com/canaanyjn/flarness/internal/daemon"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the Flarness daemon",
	Run: func(cmd *cobra.Command, args []string) {
		d := daemon.New()
		if !d.IsRunning() {
			printError("daemon is not running")
		}

		if err := d.Stop(); err != nil {
			printError("failed to stop daemon: " + err.Error())
		}

		printJSON(map[string]any{
			"status":  "ok",
			"message": "daemon stopped",
		})
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
