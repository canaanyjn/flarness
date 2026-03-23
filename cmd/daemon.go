package cmd

import (
	"github.com/canaanyjn/flarness/internal/daemon"
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:    "_daemon",
	Short:  "Internal: run as foreground daemon",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		project, _ := cmd.Flags().GetString("project")
		device, _ := cmd.Flags().GetString("device")
		extraArgs, _ := cmd.Flags().GetStringSlice("extra-args")

		d := daemon.New()
		if err := d.Start(project, device, extraArgs, true); err != nil {
			printError(err.Error())
		}
	},
}

func init() {
	daemonCmd.Flags().String("project", "", "project path")
	daemonCmd.Flags().String("device", "chrome", "target device")
	daemonCmd.Flags().StringSlice("extra-args", nil, "extra flutter run arguments")
	rootCmd.AddCommand(daemonCmd)
}
