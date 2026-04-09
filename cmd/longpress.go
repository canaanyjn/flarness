package cmd

import (
	"github.com/canaanyjn/flarness/internal/model"
	"github.com/spf13/cobra"
)

var longpressCmd = &cobra.Command{
	Use:   "longpress",
	Short: "Long press on a UI element",
	Long: `Long press on a Flutter widget found by text or type.

Examples:
  flarness interact longpress --text "Todo Item"
  flarness interact longpress --text "Delete" --duration 1000`,
	RunE: func(cmd *cobra.Command, args []string) error {
		text, _ := cmd.Flags().GetString("text")
		typ, _ := cmd.Flags().GetString("type")
		duration, _ := cmd.Flags().GetInt("duration")
		index, _ := cmd.Flags().GetInt("index")

		finderArgs := map[string]any{
			"index":    float64(index),
			"duration": float64(duration),
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
			Cmd:  "longpress",
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
	addSessionFlag(longpressCmd)
	longpressCmd.Flags().String("text", "", "find by label text (partial match)")
	longpressCmd.Flags().String("type", "", "find by widget type/flag")
	longpressCmd.Flags().Int("duration", 500, "long press duration in milliseconds")
	longpressCmd.Flags().Int("index", 0, "0-based index when multiple matches")
	interactCmd.AddCommand(longpressCmd)
}
