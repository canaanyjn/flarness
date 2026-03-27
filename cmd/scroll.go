package cmd

import (
	"github.com/canaanyjn/flarness/internal/ipc"
	"github.com/canaanyjn/flarness/internal/model"
	"github.com/spf13/cobra"
)

var scrollCmd = &cobra.Command{
	Use:   "scroll",
	Short: "Scroll a scrollable UI element",
	Long: `Scroll a Flutter scrollable widget (ListView, ScrollView, etc.).

Examples:
  flarness scroll --text "Todo List" --dy -300
  flarness scroll --text "Todo List" --dy 300
  flarness scroll --type "hasScrollAction" --dx -200`,
	RunE: func(cmd *cobra.Command, args []string) error {
		text, _ := cmd.Flags().GetString("text")
		typ, _ := cmd.Flags().GetString("type")
		dx, _ := cmd.Flags().GetFloat64("dx")
		dy, _ := cmd.Flags().GetFloat64("dy")
		index, _ := cmd.Flags().GetInt("index")

		finderArgs := map[string]any{
			"index": float64(index),
			"dx":    dx,
			"dy":    dy,
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
			Cmd:  "scroll",
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
	scrollCmd.Flags().String("text", "", "find by label text (partial match)")
	scrollCmd.Flags().String("type", "", "find by widget type/flag")
	scrollCmd.Flags().Float64("dx", 0, "horizontal scroll offset (negative=left)")
	scrollCmd.Flags().Float64("dy", 0, "vertical scroll offset (negative=up)")
	scrollCmd.Flags().Int("index", 0, "0-based index when multiple matches")
	rootCmd.AddCommand(scrollCmd)
}
