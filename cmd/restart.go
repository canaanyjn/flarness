package cmd

import (
	"github.com/canaanyjn/flarness/internal/ipc"
	"github.com/canaanyjn/flarness/internal/model"
	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Hot restart the Flutter application",
	Run: func(cmd *cobra.Command, args []string) {
		client := ipc.NewClient()
		if !client.IsRunning() {
			printError("daemon is not running — run 'flarness start' first")
		}

		resp, err := client.Send(model.Command{Cmd: "restart"})
		if err != nil {
			printError("restart failed: " + err.Error())
		}

		if !resp.OK {
			printError(resp.Error)
		}

		printJSON(resp.Data)
	},
}

func init() {
	rootCmd.AddCommand(restartCmd)
}
