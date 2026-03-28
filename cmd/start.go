package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/canaanyjn/flarness/internal/daemon"
	"github.com/canaanyjn/flarness/internal/ipc"
	"github.com/canaanyjn/flarness/internal/model"
	"github.com/canaanyjn/flarness/internal/platform"
	"github.com/spf13/cobra"
)

var (
	startProject   string
	startDevice    string
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
			device = platform.PickDefaultDevice()
		}

		client := ipc.NewClient()
		if client.IsRunning() {
			resp, err := client.Send(model.Command{Cmd: "status"})
			if err != nil {
				printError("failed to query running daemon: " + err.Error())
			}
			if !resp.OK {
				printError(resp.Error)
			}

			status, ok := resp.Data.(map[string]any)
			if !ok {
				printError("invalid status response from running daemon")
			}

			runningProject, _ := status["project"].(string)
			runningDevice, _ := status["device"].(string)
			if runningProject == project && runningDevice == device {
				printJSON(map[string]any{
					"status":        "ok",
					"device":        device,
					"project":       project,
					"message":       "daemon reused",
					"reused":        true,
					"flutter_state": status["flutter_state"],
					"url":           status["url"],
				})
				return
			}

			printError(fmt.Sprintf(
				"daemon already running for project=%s device=%s; stop it before starting project=%s device=%s",
				runningProject, runningDevice, project, device,
			))
		}

		d := daemon.New()
		if err := d.Start(project, device, startExtraArgs, false); err != nil {
			printError(err.Error())
		}

		printJSON(map[string]any{
			"status":  "ok",
			"device":  device,
			"project": project,
			"message": "daemon started",
			"reused":  false,
		})
	},
}

func init() {
	startCmd.Flags().StringVarP(&startProject, "project", "p", "", "path to Flutter project (default: current directory)")
	startCmd.Flags().StringVarP(&startDevice, "device", "d", "", "target device (default: auto-detect)")
	startCmd.Flags().StringSliceVar(&startExtraArgs, "extra-args", nil, "extra arguments for flutter run")
	rootCmd.AddCommand(startCmd)
}
