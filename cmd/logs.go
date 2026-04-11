package cmd

import (
	"github.com/canaanyjn/flarness/internal/model"
	"github.com/spf13/cobra"
)

var (
	logsLimit  int
	logsSince  string
	logsLevel  string
	logsSource string
	logsGrep   string
	logsAll    bool
	logsFollow bool
	logsList   bool
	logsClean  string
	logsExport string
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Query application logs",
	Long: `Query structured logs from the running Flutter application.

Examples:
  flarness diagnose logs                          # Latest 50 logs
  flarness diagnose logs --limit 200              # Latest 200 logs
  flarness diagnose logs --since 30s              # Logs from last 30 seconds
  flarness diagnose logs --level error            # Only errors
  flarness diagnose logs --grep "overflow"        # Regex search
  flarness diagnose logs --grep "Error" --since 5m --source framework`,
	Run: func(cmd *cobra.Command, args []string) {
		client, _ := sessionClient(cmd)

		queryArgs := map[string]any{}
		if logsLimit > 0 {
			queryArgs["limit"] = logsLimit
		} else {
			queryArgs["limit"] = 50 // default
		}
		if logsSince != "" {
			queryArgs["since"] = logsSince
		}
		if logsLevel != "" {
			queryArgs["level"] = logsLevel
		}
		if logsSource != "" {
			queryArgs["source"] = logsSource
		}
		if logsGrep != "" {
			queryArgs["grep"] = logsGrep
		}
		if logsAll {
			queryArgs["all"] = true
		}

		resp, err := client.Send(model.Command{
			Cmd:  "logs",
			Args: queryArgs,
		})
		if err != nil {
			printError("failed to query logs: " + err.Error())
		}

		if !resp.OK {
			printError(resp.Error)
		}

		printJSON(resp.Data)
	},
}

func init() {
	addSessionFlag(logsCmd)
	logsCmd.Flags().IntVar(&logsLimit, "limit", 0, "number of log entries to return (default: 50)")
	logsCmd.Flags().StringVar(&logsSince, "since", "", "time filter (e.g. 30s, 5m, 1h)")
	logsCmd.Flags().StringVar(&logsLevel, "level", "", "level filter (e.g. error, warning,error)")
	logsCmd.Flags().StringVar(&logsSource, "source", "", "source filter (e.g. app, framework)")
	logsCmd.Flags().StringVar(&logsGrep, "grep", "", "regex pattern to search")
	logsCmd.Flags().BoolVar(&logsAll, "all", false, "search all historical logs")
	logsCmd.Flags().BoolVar(&logsFollow, "follow", false, "stream logs in real-time (non-AI)")
	logsCmd.Flags().BoolVar(&logsList, "list", false, "list all log files")
	logsCmd.Flags().StringVar(&logsClean, "clean", "", "clean old logs (e.g. --clean --keep 7d)")
	logsCmd.Flags().StringVar(&logsExport, "export", "", "export logs to file")
}
