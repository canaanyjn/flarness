package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// CommandInfo represents a structured command for JSON output
type CommandInfo struct {
	Name      string     `json:"name"`
	Use       string     `json:"use"`
	Short     string     `json:"short"`
	Long      string     `json:"long,omitempty"`
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

var helpCmd = &cobra.Command{
	Use:   "help [command]",
	Short: "Help provides help for any command in structured JSON format",
	Long:  `Returns a JSON representation of all available commands or details for a specified command.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			// Return all commands overview
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
			return
		}

		// Return specific command details
		target, _, err := rootCmd.Find(args)
		if err != nil || target == nil || target == rootCmd {
			printError(fmt.Sprintf("command '%s' not found", args[0]))
			return
		}

		var flags []FlagInfo
		target.Flags().VisitAll(func(f *pflag.Flag) {
			flags = append(flags, FlagInfo{
				Name:      f.Name,
				Shorthand: f.Shorthand,
				Usage:     f.Usage,
				DefValue:  f.DefValue,
			})
		})

		info := FullCommandInfo{
			CommandInfo: CommandInfo{
				Name:  target.Name(),
				Use:   target.Use,
				Short: target.Short,
				Long:  target.Long,
			},
			Flags: flags,
		}

		printJSON(map[string]any{
			"status":  "ok",
			"command": info,
		})
	},
}

func init() {
	// Replacing default help command with our JSON version
	rootCmd.SetHelpCommand(helpCmd)
}
