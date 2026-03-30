package cmd

import (
	"github.com/canaanyjn/flarness/internal/cliargs"
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
		rawExtraArgs, _ := cmd.Flags().GetStringArray("extra-args")
		extraArgs, err := cliargs.NormalizeExtraArgs(rawExtraArgs)
		if err != nil {
			printError("invalid --extra-args: " + err.Error())
		}

		d := daemon.New()
		if err := d.Start(project, device, extraArgs, true); err != nil {
			printError(err.Error())
		}
	},
}

func init() {
	daemonCmd.Flags().String("project", "", "project path")
	daemonCmd.Flags().String("device", "chrome", "target device")
	daemonCmd.Flags().StringArray("extra-args", nil, "extra flutter run arguments")
	rootCmd.AddCommand(daemonCmd)
}
