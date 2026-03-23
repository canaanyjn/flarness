package cmd

import (
	"github.com/canaanyjn/flarness/internal/ipc"
	"github.com/canaanyjn/flarness/internal/model"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Flarness daemon status",
	Run: func(cmd *cobra.Command, args []string) {
		client := ipc.NewClient()
		if !client.IsRunning() {
			printJSON(map[string]any{
				"running": false,
			})
			return
		}

		resp, err := client.Send(model.Command{Cmd: "status"})
		if err != nil {
			printError("failed to get status: " + err.Error())
		}

		if !resp.OK {
			printError(resp.Error)
		}

		printJSON(resp.Data)
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
