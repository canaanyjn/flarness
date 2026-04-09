package cmd

import (
	"github.com/canaanyjn/flarness/internal/model"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Flarness daemon status",
	Run: func(cmd *cobra.Command, args []string) {
		client, _ := sessionClient(cmd)

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
	addSessionFlag(statusCmd)
	rootCmd.AddCommand(statusCmd)
}
