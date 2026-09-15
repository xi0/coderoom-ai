package tools

import (
	"encoding/json"
	"fmt"

	"github.com/xi0/coderoom-ai/internal/wire"

	"github.com/sashabaranov/go-openai"
)

type ReportProgressArgs struct {
	Description string `json:"description"`
	Percent     *int   `json:"percent,omitempty"`
}

func reportProgressTool() *Tool {
	return &Tool{
		mutating: false,
		definition: &openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "report_progress",
				Description: "Reports the current work in progress with an optional completion percentage.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"description": {
							"type": "string",
							"description": "Description of the work currently in progress"
						},
						"percent": {
							"type": "integer",
							"description": "Optional completion percentage (0-100)"
						}
					},
					"required": ["description"]
				}`),
			},
		},
		handler: func(argsJSON string, options *ToolOptions) (string, error) {
			var args ReportProgressArgs
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("invalid arguments for report_progress: %w", err)
			}

			percent := 0
			if args.Percent != nil {
				percent = *args.Percent
			}

			if percent < 0 {
				percent = 0
			}
			if percent > 100 {
				percent = 100
			}

			options.WriteChannel <- wire.BackendMessage{
				UpdateProgress: &wire.ProgressMessage{
					Percent: percent,
					Text:    args.Description,
				},
			}

			return "Progress reported.", nil
		},
	}
}
