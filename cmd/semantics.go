package cmd

import (
	"github.com/canaanyjn/flarness/internal/model"
	"github.com/spf13/cobra"
)

var semanticsCmd = &cobra.Command{
	Use:   "semantics",
	Short: "Dump the semantics tree for automation and element targeting",
	Long: `Dump the Flutter semantics tree in structured format.
Useful for understanding what elements are available for tap, type, wait,
scroll, swipe, and long press.

Semantics is the automation-facing view of the UI: it exposes labels, values,
available actions, flags, and element bounds.

Use semantics when you want to locate or interact with UI elements.

Use inspect instead when you want structural debugging information about the
widget tree or render tree.

Examples:
  flarness semantics`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, _ := sessionClient(cmd)

		resp, err := client.Send(model.Command{
			Cmd:  "semantics",
			Args: map[string]any{},
		})
		if err != nil {
			printError(err.Error())
			return nil
		}
		if !resp.OK {
			printError(resp.Error)
			return nil
		}

		printJSON(resp.Data)
		return nil
	},
}

func init() {
	addSessionFlag(semanticsCmd)
	rootCmd.AddCommand(semanticsCmd)
}
