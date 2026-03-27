package cmd

import (
	"github.com/canaanyjn/flarness/internal/ipc"
	"github.com/canaanyjn/flarness/internal/model"
	"github.com/spf13/cobra"
)

var semanticsCmd = &cobra.Command{
	Use:   "semantics",
	Short: "Dump the semantics tree (for debugging UI automation)",
	Long: `Dump the Flutter semantics tree in structured format.
Useful for understanding what elements are available for tap/input/scroll.

Examples:
  flarness semantics`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := ipc.NewClient()
		if !client.IsRunning() {
			printError("daemon is not running — run 'flarness start' first")
			return nil
		}

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
	rootCmd.AddCommand(semanticsCmd)
}
