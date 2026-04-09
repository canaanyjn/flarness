package cmd

import (
	"github.com/canaanyjn/flarness/internal/model"
	"github.com/spf13/cobra"
)

var waitCmd = &cobra.Command{
	Use:   "wait",
	Short: "Wait for a UI element to appear",
	Long: `Wait for a Flutter widget to appear in the semantics tree.
Returns when the element is found, or errors on timeout.

Examples:
  flarness interact wait --text "Success"
  flarness interact wait --text "Loading..." --timeout 30`,
	RunE: func(cmd *cobra.Command, args []string) error {
		text, _ := cmd.Flags().GetString("text")
		typ, _ := cmd.Flags().GetString("type")
		timeout, _ := cmd.Flags().GetInt("timeout")
		index, _ := cmd.Flags().GetInt("index")

		finderArgs := map[string]any{
			"index":   float64(index),
			"timeout": float64(timeout),
		}
		if text != "" {
			finderArgs["by"] = "text"
			finderArgs["value"] = text
		} else if typ != "" {
			finderArgs["by"] = "type"
			finderArgs["value"] = typ
		} else {
			printError("one of --text or --type is required")
			return nil
		}

		client, _ := sessionClient(cmd)

		resp, err := client.Send(model.Command{
			Cmd:  "wait",
			Args: finderArgs,
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
	addSessionFlag(waitCmd)
	waitCmd.Flags().String("text", "", "find by label text (partial match)")
	waitCmd.Flags().String("type", "", "find by widget type/flag")
	waitCmd.Flags().Int("timeout", 10, "timeout in seconds")
	waitCmd.Flags().Int("index", 0, "0-based index when multiple matches")
	interactCmd.AddCommand(waitCmd)
}
