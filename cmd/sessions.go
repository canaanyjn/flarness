package cmd

import (
	"github.com/canaanyjn/flarness/internal/daemon"
	"github.com/canaanyjn/flarness/internal/instance"
	"github.com/canaanyjn/flarness/internal/ipc"
	"github.com/canaanyjn/flarness/internal/model"
	"github.com/spf13/cobra"
)

var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "Inspect flarness daemon sessions",
}

var sessionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List known flarness sessions",
	Run: func(cmd *cobra.Command, args []string) {
		metas, err := instance.ListMetas()
		if err != nil {
			printError("failed to list sessions: " + err.Error())
		}

		sessions := make([]map[string]any, 0, len(metas))
		for _, meta := range metas {
			d := daemon.New(meta.Session)
			client := ipc.NewClient(meta.Session)

			if !d.IsRunning() {
				_ = instance.CleanupAll(meta.Session)
				continue
			}

			item := map[string]any{
				"session": meta.Session,
				"project": meta.ProjectPath,
				"device":  meta.Device,
				"running": true,
			}

			if client.IsRunning() {
				resp, err := client.Send(model.Command{Cmd: "status"})
				if err == nil && resp.OK {
					if status, ok := resp.Data.(map[string]any); ok {
						sessions = append(sessions, status)
						continue
					}
				}
			}

			pid, err := d.ReadPID()
			if err == nil {
				item["pid"] = pid
			}
			item["flutter_state"] = "unreachable"
			sessions = append(sessions, item)
		}

		printJSON(map[string]any{
			"status":   "ok",
			"sessions": sessions,
			"count":    len(sessions),
		})
	},
}

func init() {
	sessionsCmd.AddCommand(sessionsListCmd)
	rootCmd.AddCommand(sessionsCmd)
}
