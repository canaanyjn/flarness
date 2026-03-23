package cmd

import (
	"github.com/canaanyjn/flarness/internal/ipc"
	"github.com/canaanyjn/flarness/internal/model"
	"github.com/spf13/cobra"
)

var screenshotCmd = &cobra.Command{
	Use:   "screenshot",
	Short: "Capture a screenshot of the running Flutter app",
	Long: `Capture a screenshot of the running Flutter app.

For Web platform: uses Chrome DevTools Protocol (CDP) for instant capture.
For other platforms: uses flutter screenshot command.

The screenshot is saved to ~/.flarness/screenshots/ by default.`,
	Run: func(cmd *cobra.Command, args []string) {
		client := ipc.NewClient()
		if !client.IsRunning() {
			printError("daemon is not running — run 'flarness start' first")
		}

		resp, err := client.Send(model.Command{Cmd: "screenshot"})
		if err != nil {
			printError("screenshot failed: " + err.Error())
		}

		if !resp.OK {
			printError(resp.Error)
		}

		printJSON(resp.Data)
	},
}

func init() {
	rootCmd.AddCommand(screenshotCmd)
}
