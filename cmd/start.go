package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/canaanyjn/flarness/internal/daemon"
	"github.com/spf13/cobra"
)

var (
	startProject  string
	startDevice   string
	startExtraArgs []string
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Flarness daemon and launch Flutter",
	Long:  `Starts the Flarness background daemon which manages the Flutter process.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Resolve project path.
		project := startProject
		if project == "" {
			var err error
			project, err = os.Getwd()
			if err != nil {
				printError("cannot determine project path: " + err.Error())
			}
		}
		project, _ = filepath.Abs(project)

		// Validate: must contain pubspec.yaml.
		if _, err := os.Stat(filepath.Join(project, "pubspec.yaml")); os.IsNotExist(err) {
			printError(fmt.Sprintf("no pubspec.yaml found in %s — is this a Flutter project?", project))
		}

		device := startDevice
		if device == "" {
			device = "chrome"
		}

		d := daemon.New()
		if err := d.Start(project, device, startExtraArgs, false); err != nil {
			printError(err.Error())
		}

		printJSON(map[string]any{
			"status": "ok",
			"device": device,
			"project": project,
			"message": "daemon started",
		})
	},
}

func init() {
	startCmd.Flags().StringVarP(&startProject, "project", "p", "", "path to Flutter project (default: current directory)")
	startCmd.Flags().StringVarP(&startDevice, "device", "d", "", "target device (default: chrome)")
	startCmd.Flags().StringSliceVar(&startExtraArgs, "extra-args", nil, "extra arguments for flutter run")
	rootCmd.AddCommand(startCmd)
}
