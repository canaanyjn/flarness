package cmd

import (
	"github.com/canaanyjn/flarness/internal/ipc"
	"github.com/canaanyjn/flarness/internal/model"
	"github.com/spf13/cobra"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Run flutter analyze and return structured results",
	Run: func(cmd *cobra.Command, args []string) {
		client := ipc.NewClient()
		if !client.IsRunning() {
			printError("daemon is not running — run 'flarness start' first")
		}

		resp, err := client.Send(model.Command{Cmd: "analyze"})
		if err != nil {
			printError("analyze failed: " + err.Error())
		}

		if !resp.OK {
			printError(resp.Error)
		}

		printJSON(resp.Data)
	},
}

func init() {
	rootCmd.AddCommand(analyzeCmd)
}
