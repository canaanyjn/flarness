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
		session := requireSession(cmd)
		client := ipc.NewClient(session)
		if client.IsRunning() {
			resp, err := client.Send(model.Command{Cmd: "stop"})
			if err != nil {
				printError("failed to stop daemon: " + err.Error())
			}
			if !resp.OK {
				printError(resp.Error)
			}
			d := daemon.New(session)
			deadline := time.Now().Add(3 * time.Second)
			for time.Now().Before(deadline) {
				if !d.IsRunning() {
					printJSON(map[string]any{
						"status":  "ok",
						"session": session,
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
				"session": session,
				"message": "daemon stopped",
			})
			return
		}

		d := daemon.New(session)
		if !d.IsRunning() {
			d.Cleanup()
			printError(daemonNotRunningError(session))
		}
		if err := d.Stop(); err != nil {
			printError("failed to stop daemon: " + err.Error())
		}

		printJSON(map[string]any{
			"status":  "ok",
			"session": session,
			"message": "daemon stopped",
		})
	},
}

func init() {
	addSessionFlag(stopCmd)
}
