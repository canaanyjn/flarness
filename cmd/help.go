package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// CommandInfo represents a structured command for JSON output
type CommandInfo struct {
	Name  string `json:"name"`
	Use   string `json:"use"`
	Short string `json:"short"`
	Long  string `json:"long,omitempty"`
}

// FullCommandInfo includes flags for detailed command help
type FullCommandInfo struct {
	CommandInfo
	Flags []FlagInfo `json:"flags,omitempty"`
}

// FlagInfo represents a structured flag for JSON output
type FlagInfo struct {
	Name      string `json:"name"`
	Shorthand string `json:"shorthand"`
	Usage     string `json:"usage"`
	DefValue  string `json:"default_value"`
}

func collectFlags(cmd *cobra.Command) []FlagInfo {
	var flags []FlagInfo
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		flags = append(flags, FlagInfo{
			Name:      f.Name,
			Shorthand: f.Shorthand,
			Usage:     f.Usage,
			DefValue:  f.DefValue,
		})
	})
	return flags
}

func buildCommandInfo(cmd *cobra.Command) FullCommandInfo {
	return FullCommandInfo{
		CommandInfo: CommandInfo{
			Name:  cmd.Name(),
			Use:   cmd.Use,
			Short: cmd.Short,
			Long:  cmd.Long,
		},
		Flags: collectFlags(cmd),
	}
}

func printCommandOverview() {
	var commands []CommandInfo
	for _, child := range rootCmd.Commands() {
		if child.Hidden {
			continue
		}
		commands = append(commands, CommandInfo{
			Name:  child.Name(),
			Use:   child.Use,
			Short: child.Short,
		})
	}
	printJSON(map[string]any{
		"status":      "ok",
		"tool":        "flarness",
		"description": rootCmd.Short,
		"commands":    commands,
	})
}

func printCommandDetails(cmd *cobra.Command) {
	printJSON(map[string]any{
		"status":  "ok",
		"command": buildCommandInfo(cmd),
	})
}

var helpCmd = &cobra.Command{
	Use:   "help [command]",
	Short: "Help provides help for any command in structured JSON format",
	Long:  `Returns a JSON representation of all available commands or details for a specified command.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			printCommandOverview()
			return
		}

		target, _, err := rootCmd.Find(args)
		if err != nil || target == nil || target == rootCmd {
			printError(fmt.Sprintf("command '%s' not found", args[0]))
			return
		}
		printCommandDetails(target)
	},
}

func init() {
	rootCmd.SetHelpCommand(helpCmd)
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd == nil || cmd == rootCmd {
			printCommandOverview()
			return
		}
		printCommandDetails(cmd)
	})
}
