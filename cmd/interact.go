package cmd

import "github.com/spf13/cobra"

var interactCmd = &cobra.Command{
	Use:   "interact",
	Short: "UI interaction commands",
	Long: `Grouped UI automation commands.

Use subcommands such as tap, type, wait, scroll, swipe, and longpress.`,
}

func init() {
	rootCmd.AddCommand(interactCmd)
}
