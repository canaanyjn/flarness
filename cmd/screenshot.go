package cmd

import (
	"github.com/canaanyjn/flarness/internal/model"
	"github.com/spf13/cobra"
)

var screenshotCmd = &cobra.Command{
	Use:   "screenshot",
	Short: "Capture a screenshot of the running Flutter app",
	Long: `Capture a screenshot of the running Flutter app.

For Web platform: uses Chrome DevTools Protocol (CDP) for instant capture.
For macOS debug apps with flarness_plugin initialized: captures Flutter-rendered
content through a VM service extension.
For other platforms: uses flutter screenshot first, then falls back to the same
VM service extension when available.

The screenshot is saved to ~/.flarness/screenshots/ by default.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, _ := sessionClient(cmd)

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
	addSessionFlag(screenshotCmd)
}
