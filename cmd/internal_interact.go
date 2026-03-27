package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/canaanyjn/flarness/internal/interaction"
	"github.com/spf13/cobra"
)

var internalInteractCmd = &cobra.Command{
	Use:    "_interact",
	Short:  "Internal: run UI interaction in subprocess",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		debugURL, _ := cmd.Flags().GetString("debug-url")
		action, _ := cmd.Flags().GetString("action")
		argsJSON, _ := cmd.Flags().GetString("args")

		if debugURL == "" || action == "" {
			fmt.Fprintf(os.Stderr, "missing --debug-url or --action\n")
			os.Exit(1)
		}

		var cmdArgs map[string]any
		if argsJSON != "" {
			if err := json.Unmarshal([]byte(argsJSON), &cmdArgs); err != nil {
				fmt.Fprintf(os.Stderr, "invalid args JSON: %v\n", err)
				os.Exit(1)
			}
		}

		it := interaction.NewInteractor(debugURL)

		var result any
		var err error

		switch action {
		case "tap":
			if xv, xok := cmdArgs["x"].(float64); xok {
				if yv, yok := cmdArgs["y"].(float64); yok {
					result, err = it.TapAt(xv, yv)
					break
				}
			}
			finder := parseFinder(cmdArgs)
			result, err = it.Tap(finder)

		case "type":
			typeText := ""
			if v, ok := cmdArgs["text"].(string); ok {
				typeText = v
			}
			clearMode := false
			if v, ok := cmdArgs["clear"].(bool); ok {
				clearMode = v
			}
			appendMode := false
			if v, ok := cmdArgs["append"].(bool); ok {
				appendMode = v
			}
			result, err = it.Type(typeText, clearMode, appendMode)

		case "swipe":
			finder := parseFinder(cmdArgs)
			dx := 0.0
			dy := 0.0
			durationMs := 300
			if v, ok := cmdArgs["dx"].(float64); ok {
				dx = v
			}
			if v, ok := cmdArgs["dy"].(float64); ok {
				dy = v
			}
			if v, ok := cmdArgs["duration"].(float64); ok && v > 0 {
				durationMs = int(v)
			}
			result, err = it.SwipeOn(finder, dx, dy, durationMs)

		case "scroll":
			finder := parseFinder(cmdArgs)
			dx := 0.0
			dy := 0.0
			if v, ok := cmdArgs["dx"].(float64); ok {
				dx = v
			}
			if v, ok := cmdArgs["dy"].(float64); ok {
				dy = v
			}
			result, err = it.Scroll(finder, dx, dy)

		case "longpress":
			finder := parseFinder(cmdArgs)
			durationMs := 500
			if v, ok := cmdArgs["duration"].(float64); ok && v > 0 {
				durationMs = int(v)
			}
			result, err = it.LongPress(finder, durationMs)

		case "wait":
			finder := parseFinder(cmdArgs)
			timeout := 10 * time.Second
			if v, ok := cmdArgs["timeout"].(float64); ok && v > 0 {
				timeout = time.Duration(v) * time.Second
			}
			result, err = it.WaitFor(finder, timeout)

		case "semantics":
			nodes, sErr := it.GetSemanticsTree()
			if sErr != nil {
				err = sErr
			} else {
				result = map[string]any{
					"status": "ok",
					"nodes":  nodes,
					"count":  countSemanticsNodes(nodes),
				}
			}

		default:
			fmt.Fprintf(os.Stderr, "unknown action: %s\n", action)
			os.Exit(1)
		}

		if err != nil {
			errResp := map[string]any{
				"status": "error",
				"error":  err.Error(),
			}
			data, _ := json.MarshalIndent(errResp, "", "  ")
			os.Stderr.Write(data)
			os.Stderr.Write([]byte("\n"))
			os.Exit(1)
		}

		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "json marshal error: %v\n", err)
			os.Exit(1)
		}
		os.Stdout.Write(data)
		os.Stdout.Write([]byte("\n"))

		return nil
	},
}

func parseFinder(args map[string]any) interaction.Finder {
	finder := interaction.Finder{}
	if v, ok := args["by"].(string); ok {
		finder.By = interaction.FinderType(v)
	}
	if v, ok := args["value"].(string); ok {
		finder.Value = v
	}
	if v, ok := args["index"].(float64); ok {
		finder.Index = int(v)
	}
	return finder
}

func countSemanticsNodes(nodes []*interaction.SemanticsNode) int {
	count := len(nodes)
	for _, n := range nodes {
		count += countSemanticsNodes(n.Children)
	}
	return count
}

func init() {
	internalInteractCmd.Flags().String("debug-url", "", "VM Service WebSocket URL")
	internalInteractCmd.Flags().String("action", "", "interaction action (tap/type/scroll/longpress/wait/semantics)")
	internalInteractCmd.Flags().String("args", "{}", "JSON args for the action")
	rootCmd.AddCommand(internalInteractCmd)
}
