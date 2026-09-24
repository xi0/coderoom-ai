package tools

import (
	"encoding/json"
	"fmt"

	"github.com/xi0/coderoom-ai/internal/wire"

	"github.com/sashabaranov/go-openai"
)

type PresentOptionsArgs struct {
	Options []string `json:"options"`
	Prompt  string   `json:"prompt"`
}

func presentOptionsTool() *Tool {
	return &Tool{
		mutating: false,
		definition: &openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "present_options",
				Description: "Presents a list of options to the user and returns the textual form of the selected option.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"options": {
							"type": "array",
							"items": {"type": "string"},
							"description": "List of options for the user to choose from"
						},
						"prompt": {
							"type": "string",
							"description": "Optional prompt message to display before the options"
						}
					},
					"required": ["options"]
				}`),
			},
		},
		handler: func(argsJSON string, options *ToolOptions) (string, error) {
			var args PresentOptionsArgs
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("invalid arguments for present_options: %w", err)
			}

			if len(args.Options) == 0 {
				return "", fmt.Errorf("options list cannot be empty")
			}

			options.WriteChannel <- wire.BackendMessage{
				OptionsMessage: &wire.OptionsMessage{
					Description: args.Prompt,
					Options:     args.Options,
				},
				WorkDone: true,
			}

			selection := <-options.OptionChannel

			if selection < 0 || selection >= len(args.Options) {
				return "", fmt.Errorf("User selected option %d is out of range. Should be 0-%d.", selection, len(args.Options)-1)
			}

			selectedOption := args.Options[selection]
			return selectedOption, nil
		},
	}
}
