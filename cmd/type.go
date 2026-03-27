package cmd

import (
	"github.com/canaanyjn/flarness/internal/ipc"
	"github.com/canaanyjn/flarness/internal/model"
	"github.com/spf13/cobra"
)

var typeCmd = &cobra.Command{
	Use:   "type",
	Short: "Type text into the currently focused text field",
	Long: `Type text into whichever Flutter text field currently has focus.
No element finder is needed — this command writes to the active input.

Use 'flarness tap' first to focus the desired text field, then 'flarness type'.

Examples:
  flarness tap --text "Search"
  flarness type --value "buy milk"

  flarness type --value " and eggs" --append
  flarness type --clear`,
	RunE: func(cmd *cobra.Command, args []string) error {
		value, _ := cmd.Flags().GetString("value")
		clear, _ := cmd.Flags().GetBool("clear")
		appendMode, _ := cmd.Flags().GetBool("append")

		if value == "" && !clear {
			printError("--value is required (or use --clear to clear the field)")
			return nil
		}

		typeArgs := map[string]any{
			"text": value,
		}
		if clear {
			typeArgs["clear"] = true
		}
		if appendMode {
			typeArgs["append"] = true
		}

		client := ipc.NewClient()
		if !client.IsRunning() {
			printError("daemon is not running — run 'flarness start' first")
			return nil
		}

		resp, err := client.Send(model.Command{
			Cmd:  "type",
			Args: typeArgs,
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
	typeCmd.Flags().String("value", "", "text to type into the focused field")
	typeCmd.Flags().Bool("clear", false, "clear the focused field")
	typeCmd.Flags().Bool("append", false, "append text instead of replacing")
	rootCmd.AddCommand(typeCmd)
}
