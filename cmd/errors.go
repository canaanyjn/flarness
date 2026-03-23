package cmd

import (
	"github.com/canaanyjn/flarness/internal/ipc"
	"github.com/canaanyjn/flarness/internal/model"
	"github.com/spf13/cobra"
)

var errorsCmd = &cobra.Command{
	Use:   "errors",
	Short: "Show only error and fatal logs (shortcut for logs --level error,fatal)",
	Run: func(cmd *cobra.Command, args []string) {
		client := ipc.NewClient()
		if !client.IsRunning() {
			printError("daemon is not running — run 'flarness start' first")
		}

		resp, err := client.Send(model.Command{
			Cmd: "logs",
			Args: map[string]any{
				"level": "error,fatal",
				"limit": 50,
			},
		})
		if err != nil {
			printError("failed to query errors: " + err.Error())
		}

		if !resp.OK {
			printError(resp.Error)
		}

		printJSON(resp.Data)
	},
}

func init() {
	rootCmd.AddCommand(errorsCmd)
}
