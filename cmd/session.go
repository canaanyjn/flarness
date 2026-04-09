package cmd

import (
	"fmt"

	"github.com/canaanyjn/flarness/internal/daemon"
	"github.com/canaanyjn/flarness/internal/ipc"
	"github.com/spf13/cobra"
)

const sessionFlagName = "session"

func addSessionFlag(cmd *cobra.Command) {
	cmd.Flags().String(sessionFlagName, "", "target flarness session id")
}

func requireSession(cmd *cobra.Command) string {
	session, _ := cmd.Flags().GetString(sessionFlagName)
	if session == "" {
		printError("missing required --session; run 'flarness sessions list' or use the session returned by 'flarness start'")
	}
	return session
}

func daemonNotRunningError(session string) string {
	return fmt.Sprintf("daemon for session %s is not running", session)
}

func sessionClient(cmd *cobra.Command) (*ipc.Client, string) {
	session := requireSession(cmd)
	client := ipc.NewClient(session)
	if !client.IsRunning() {
		d := daemon.New(session)
		if !d.IsRunning() {
			d.Cleanup()
		}
		printError(daemonNotRunningError(session))
	}
	return client, session
}
