package cmd

import (
	"github.com/canaanyjn/flarness/internal/ipc"
	"github.com/canaanyjn/flarness/internal/model"
	"github.com/spf13/cobra"
)

var swipeCmd = &cobra.Command{
	Use:   "swipe",
	Short: "Swipe on a UI element",
	Long: `Swipe a Flutter widget found by text or type.
Useful for Dismissible (swipe-to-delete), drawers, etc.

Examples:
  flarness interact swipe --text "Todo Item" --dx -400
  flarness interact swipe --text "Todo Item" --dx 300
  flarness interact swipe --text "Photo" --dy -200`,
	RunE: func(cmd *cobra.Command, args []string) error {
		text, _ := cmd.Flags().GetString("text")
		typ, _ := cmd.Flags().GetString("type")
		dx, _ := cmd.Flags().GetFloat64("dx")
		dy, _ := cmd.Flags().GetFloat64("dy")
		duration, _ := cmd.Flags().GetInt("duration")
		index, _ := cmd.Flags().GetInt("index")

		finderArgs := map[string]any{
			"index":    float64(index),
			"dx":       dx,
			"dy":       dy,
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

		client := ipc.NewClient()
		if !client.IsRunning() {
			printError("daemon is not running — run 'flarness start' first")
			return nil
		}

		resp, err := client.Send(model.Command{
			Cmd:  "swipe",
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
	swipeCmd.Flags().String("text", "", "find by label text (partial match)")
	swipeCmd.Flags().String("type", "", "find by widget type/flag")
	swipeCmd.Flags().Float64("dx", 0, "horizontal swipe distance")
	swipeCmd.Flags().Float64("dy", 0, "vertical swipe distance")
	swipeCmd.Flags().Int("duration", 300, "swipe duration in milliseconds")
	swipeCmd.Flags().Int("index", 0, "0-based index when multiple matches")
	interactCmd.AddCommand(swipeCmd)
}
