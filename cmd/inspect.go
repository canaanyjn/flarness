package cmd

import (
	"github.com/canaanyjn/flarness/internal/model"
	"github.com/spf13/cobra"
)

var (
	inspectMaxDepth int
)

var inspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "Inspect Flutter UI structure for development debugging",
	Long: `Inspect Flutter UI structure via VM Service Protocol.

Returns a structured JSON representation of the current Widget hierarchy,
including widget types, properties, and children. If widget inspection is not
available on the current target, Flarness falls back to a render tree description.

Use inspect when you need structural debugging information about how the UI is
composed.

Use semantics instead when you need automation-facing data such as labels,
actions, focus, and bounding rectangles.`,
	Run: func(cmd *cobra.Command, args []string) {
		client, _ := sessionClient(cmd)

		cmdArgs := map[string]any{}
		if inspectMaxDepth > 0 {
			cmdArgs["max_depth"] = float64(inspectMaxDepth)
		}

		resp, err := client.Send(model.Command{
			Cmd:  "inspect",
			Args: cmdArgs,
		})
		if err != nil {
			printError("inspect failed: " + err.Error())
		}

		if !resp.OK {
			printError(resp.Error)
		}

		printJSON(resp.Data)
	},
}

func init() {
	addSessionFlag(inspectCmd)
	inspectCmd.Flags().IntVar(&inspectMaxDepth, "max-depth", 0, "max depth of the widget tree (0 = unlimited)")
	rootCmd.AddCommand(inspectCmd)
}
