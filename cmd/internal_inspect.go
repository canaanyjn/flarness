package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/canaanyjn/flarness/internal/inspector"
	"github.com/canaanyjn/flarness/internal/model"
	"github.com/spf13/cobra"
)

var (
	internalInspectDebugURL string
	internalInspectMaxDepth int
)

var internalInspectCmd = &cobra.Command{
	Use:    "_inspect",
	Short:  "Internal: inspect widget tree (subprocess)",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		if internalInspectDebugURL == "" {
			fmt.Fprintf(os.Stderr, "error: --debug-url is required\n")
			os.Exit(1)
		}

		ins := inspector.NewInspector(internalInspectDebugURL)
		result, err := ins.Inspect()
		if err != nil {
			errResp := model.InspectResponse{Status: "error"}
			data, _ := json.Marshal(errResp)
			fmt.Fprintf(os.Stderr, "[_inspect] error: %v\n", err)
			os.Stdout.Write(data)
			os.Stdout.Write([]byte("\n"))
			os.Exit(1)
		}

		var widgetTree any
		if result.Tree != nil {
			if internalInspectMaxDepth > 0 {
				widgetTree = inspector.PruneTree(result.Tree, internalInspectMaxDepth)
			} else {
				widgetTree = result.Tree
			}
		}

		resp := model.InspectResponse{
			Status:     "ok",
			WidgetTree: widgetTree,
			RenderTree: result.RenderTree,
			Summary:    result.Summary,
		}

		data, _ := json.Marshal(resp)
		os.Stdout.Write(data)
		os.Stdout.Write([]byte("\n"))
	},
}

func init() {
	internalInspectCmd.Flags().StringVar(&internalInspectDebugURL, "debug-url", "", "VM Service debug URL")
	internalInspectCmd.Flags().IntVar(&internalInspectMaxDepth, "max-depth", 0, "max depth of widget tree")
	rootCmd.AddCommand(internalInspectCmd)
}
