package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/canaanyjn/flarness/internal/cliargs"
	"github.com/canaanyjn/flarness/internal/config"
	"github.com/canaanyjn/flarness/internal/daemon"
	"github.com/canaanyjn/flarness/internal/instance"
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
		cfg := config.Load()

		project, projectCfg, err := resolveProjectArg(cfg, startProject)
		if err != nil {
			printError(err.Error())
		}

		// Validate: must contain pubspec.yaml.
		if _, err := os.Stat(filepath.Join(project, "pubspec.yaml")); os.IsNotExist(err) {
			printError(fmt.Sprintf("no pubspec.yaml found in %s — is this a Flutter project?", project))
		}

		device := startDevice
		if device == "" {
			if projectCfg.Device != "" {
				device = projectCfg.Device
			} else {
				device = platform.PickDefaultDevice()
			}
		}

		extraArgs, err := cliargs.NormalizeExtraArgs(startExtraArgs)
		if err != nil {
			printError("invalid --extra-args: " + err.Error())
		}
		mergedExtraArgs := append([]string{}, cfg.Defaults.ExtraArgs...)
		mergedExtraArgs = append(mergedExtraArgs, projectCfg.ExtraArgs...)
		mergedExtraArgs = append(mergedExtraArgs, extraArgs...)
		flutterCommand := append([]string{}, cfg.Defaults.FlutterCommand...)
		if len(projectCfg.FlutterCommand) > 0 {
			flutterCommand = append([]string{}, projectCfg.FlutterCommand...)
		}

		session := instance.SessionForProject(project)
		client := ipc.NewClient(session)
		d := daemon.New(session)

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
					"session":       session,
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
				"session %s is already running for project=%s device=%s; stop it before starting project=%s device=%s",
				session, runningProject, runningDevice, project, device,
			))
		}

		if !d.IsRunning() {
			_ = instance.CleanupAll(session)
		}

		if meta, err := instance.LoadMeta(session); err == nil {
			if meta.ProjectPath != project || meta.Device != device {
				printError(fmt.Sprintf(
					"session %s metadata mismatch (project=%s device=%s); clean the stale instance before starting project=%s device=%s",
					session, meta.ProjectPath, meta.Device, project, device,
				))
			}
		}

		if err := d.Start(project, device, mergedExtraArgs, flutterCommand, false); err != nil {
			printError(err.Error())
		}

		printJSON(map[string]any{
			"status":  "ok",
			"session": session,
			"device":  device,
			"project": project,
			"message": "daemon started",
			"reused":  false,
		})
	},
}

func init() {
	startCmd.Flags().StringVarP(&startProject, "project", "p", "", "path to Flutter project or configured project name (default: current directory)")
	startCmd.Flags().StringVarP(&startDevice, "device", "d", "", "target device (default: auto-detect)")
	startCmd.Flags().StringArrayVar(&startExtraArgs, "extra-args", nil, "extra arguments for flutter run; accepts repeated flags or a single JSON array string")
	rootCmd.AddCommand(startCmd)
}
