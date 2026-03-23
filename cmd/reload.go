package cmd

import (
	"github.com/canaanyjn/flarness/internal/ipc"
	"github.com/canaanyjn/flarness/internal/model"
	"github.com/spf13/cobra"
)

var reloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Hot reload the Flutter application",
	Run: func(cmd *cobra.Command, args []string) {
		client := ipc.NewClient()
		if !client.IsRunning() {
			printError("daemon is not running — run 'flarness start' first")
		}

		resp, err := client.Send(model.Command{Cmd: "reload"})
		if err != nil {
			printError("reload failed: " + err.Error())
		}

		if !resp.OK {
			printError(resp.Error)
		}

		printJSON(resp.Data)
	},
}

func init() {
	rootCmd.AddCommand(reloadCmd)
}
