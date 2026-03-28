package cmd

import (
	"github.com/canaanyjn/flarness/internal/daemon"
	"github.com/canaanyjn/flarness/internal/ipc"
	"github.com/canaanyjn/flarness/internal/model"
	"github.com/spf13/cobra"
	"time"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the Flarness daemon",
	Run: func(cmd *cobra.Command, args []string) {
		client := ipc.NewClient()
		if client.IsRunning() {
			resp, err := client.Send(model.Command{Cmd: "stop"})
			if err != nil {
				printError("failed to stop daemon: " + err.Error())
			}
			if !resp.OK {
				printError(resp.Error)
			}
			d := daemon.New()
			deadline := time.Now().Add(3 * time.Second)
			for time.Now().Before(deadline) {
				if !d.IsRunning() {
					printJSON(map[string]any{
						"status":  "ok",
						"message": "daemon stopped",
					})
					return
				}
				time.Sleep(100 * time.Millisecond)
			}
			if d.IsRunning() {
				if err := d.Stop(); err != nil {
					printError("failed to stop daemon: " + err.Error())
				}
			}
			printJSON(map[string]any{
				"status":  "ok",
				"message": "daemon stopped",
			})
			return
		}

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
