package cmd

import (
	"github.com/canaanyjn/flarness/internal/ipc"
	"github.com/canaanyjn/flarness/internal/model"
	"github.com/spf13/cobra"
)

var tapCmd = &cobra.Command{
	Use:   "tap",
	Short: "Tap on a UI element",
	Long: `Tap on a Flutter widget found by text, type, or coordinates.

Examples:
  flarness interact tap --text "Add Todo"
  flarness interact tap --type "isButton" --index 2
  flarness interact tap --x 400 --y 300`,
	RunE: func(cmd *cobra.Command, args []string) error {
		text, _ := cmd.Flags().GetString("text")
		typ, _ := cmd.Flags().GetString("type")
		index, _ := cmd.Flags().GetInt("index")
		x, _ := cmd.Flags().GetFloat64("x")
		y, _ := cmd.Flags().GetFloat64("y")

		finderArgs := map[string]any{
			"index": float64(index),
		}
		if x >= 0 && y >= 0 {
			finderArgs["x"] = x
			finderArgs["y"] = y
		} else if text != "" {
			finderArgs["by"] = "text"
			finderArgs["value"] = text
		} else if typ != "" {
			finderArgs["by"] = "type"
			finderArgs["value"] = typ
		} else {
			printError("provide --x/--y or one of --text, --type")
			return nil
		}

		client := ipc.NewClient()
		if !client.IsRunning() {
			printError("daemon is not running — run 'flarness start' first")
			return nil
		}

		resp, err := client.Send(model.Command{
			Cmd:  "tap",
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
	tapCmd.Flags().String("text", "", "find by label text (partial match)")
	tapCmd.Flags().String("type", "", "find by widget type/flag")
	tapCmd.Flags().Int("index", 0, "0-based index when multiple matches")
	tapCmd.Flags().Float64("x", -1, "tap by logical x coordinate")
	tapCmd.Flags().Float64("y", -1, "tap by logical y coordinate")
	interactCmd.AddCommand(tapCmd)
}
