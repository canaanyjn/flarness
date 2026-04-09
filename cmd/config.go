package cmd

import (
	"path/filepath"
	"sort"

	"github.com/canaanyjn/flarness/internal/cliargs"
	"github.com/canaanyjn/flarness/internal/config"
	"github.com/spf13/cobra"
)

var (
	configProjectDevice         string
	configProjectExtraArgs      []string
	configProjectFlutterCommand []string
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage Flarness configuration",
}

var configProjectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage named Flutter projects",
}

var configProjectAddCmd = &cobra.Command{
	Use:   "add <name> <path>",
	Short: "Add or update a named project",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Load()
		if cfg.Projects == nil {
			cfg.Projects = map[string]config.ProjectConfig{}
		}

		path, err := filepath.Abs(args[1])
		if err != nil {
			printError("failed to resolve project path: " + err.Error())
		}

		extraArgs, err := cliargs.NormalizeExtraArgs(configProjectExtraArgs)
		if err != nil {
			printError("invalid --extra-args: " + err.Error())
		}
		flutterCommand, err := cliargs.NormalizeExtraArgs(configProjectFlutterCommand)
		if err != nil {
			printError("invalid --flutter-command: " + err.Error())
		}

		cfg.Projects[args[0]] = config.ProjectConfig{
			Path:           path,
			Device:         configProjectDevice,
			ExtraArgs:      extraArgs,
			FlutterCommand: flutterCommand,
		}
		if err := config.Save(cfg); err != nil {
			printError("failed to save config: " + err.Error())
		}

		printJSON(map[string]any{
			"status":  "ok",
			"message": "project saved",
			"name":    args[0],
			"project": cfg.Projects[args[0]],
		})
	},
}

var configProjectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured projects",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Load()
		names := make([]string, 0, len(cfg.Projects))
		for name := range cfg.Projects {
			names = append(names, name)
		}
		sort.Strings(names)

		projects := make([]map[string]any, 0, len(names))
		for _, name := range names {
			project := cfg.Projects[name]
			projects = append(projects, map[string]any{
				"name":            name,
				"path":            project.Path,
				"device":          project.Device,
				"extra_args":      project.ExtraArgs,
				"flutter_command": project.FlutterCommand,
			})
		}

		printJSON(map[string]any{
			"status":   "ok",
			"projects": projects,
		})
	},
}

var configProjectRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a named project",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Load()
		if _, ok := cfg.Projects[args[0]]; !ok {
			printError("project not found: " + args[0])
		}
		delete(cfg.Projects, args[0])
		if err := config.Save(cfg); err != nil {
			printError("failed to save config: " + err.Error())
		}

		printJSON(map[string]any{
			"status":  "ok",
			"message": "project removed",
			"name":    args[0],
		})
	},
}

func init() {
	configProjectAddCmd.Flags().StringVar(&configProjectDevice, "device", "", "default device for this project")
	configProjectAddCmd.Flags().StringArrayVar(&configProjectExtraArgs, "extra-args", nil, "default extra flutter run arguments for this project")
	configProjectAddCmd.Flags().StringArrayVar(&configProjectFlutterCommand, "flutter-command", nil, "wrapper command to run instead of flutter for this project")

	configProjectCmd.AddCommand(configProjectAddCmd)
	configProjectCmd.AddCommand(configProjectListCmd)
	configProjectCmd.AddCommand(configProjectRemoveCmd)
	configCmd.AddCommand(configProjectCmd)
	rootCmd.AddCommand(configCmd)
}
