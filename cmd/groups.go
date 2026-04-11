package cmd

import "github.com/spf13/cobra"

var appCmd = &cobra.Command{
	Use:   "app",
	Short: "Manage the running Flutter app and daemon lifecycle",
}

var observeCmd = &cobra.Command{
	Use:   "observe",
	Short: "Observe the current UI through screenshots, structure, and semantics",
}

var diagnoseCmd = &cobra.Command{
	Use:   "diagnose",
	Short: "Diagnose runtime and static issues",
}

func init() {
	appCmd.AddCommand(startCmd)
	appCmd.AddCommand(stopCmd)
	appCmd.AddCommand(statusCmd)
	appCmd.AddCommand(reloadCmd)
	appCmd.AddCommand(restartCmd)

	observeCmd.AddCommand(screenshotCmd)
	observeCmd.AddCommand(inspectCmd)
	observeCmd.AddCommand(semanticsCmd)

	diagnoseCmd.AddCommand(logsCmd)
	diagnoseCmd.AddCommand(analyzeCmd)

	rootCmd.AddCommand(appCmd)
	rootCmd.AddCommand(observeCmd)
	rootCmd.AddCommand(diagnoseCmd)
}
