package cmd

import (
	"fmt"
	"os"

	"github.com/canaanyjn/flarness/internal/daemon"
	"github.com/canaanyjn/flarness/internal/instance"
	"github.com/canaanyjn/flarness/internal/ipc"
	"github.com/spf13/cobra"
)

const sessionFlagName = "session"

func addSessionFlag(cmd *cobra.Command) {
	cmd.Flags().String(sessionFlagName, "", "target flarness session id (default: the project containing the current directory)")
}

// resolveSession returns the explicit --session when given, otherwise derives
// it from the project root containing the current working directory. This makes
// commands target the worktree you are standing in, so running from inside a
// git worktree never accidentally drives another checkout's daemon.
func resolveSession(cmd *cobra.Command) string {
	session, _ := cmd.Flags().GetString(sessionFlagName)
	if session != "" {
		return session
	}
	cwd, err := os.Getwd()
	if err != nil {
		printError("missing --session and cannot determine current directory: " + err.Error())
	}
	return instance.SessionForProject(resolveProjectRoot(cwd))
}

func daemonNotRunningError(session string) string {
	return fmt.Sprintf("daemon for session %s is not running", session)
}

func sessionClient(cmd *cobra.Command) (*ipc.Client, string) {
	session := resolveSession(cmd)
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
