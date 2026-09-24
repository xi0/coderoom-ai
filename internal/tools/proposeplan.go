package tools

import (
	"encoding/json"
	"fmt"

	"github.com/xi0/coderoom-ai/internal/wire"

	"github.com/sashabaranov/go-openai"
)

type ProposePlanArgs struct {
	Plan string `json:"plan"`
}

func proposePlanTool() *Tool {
	return &Tool{
		mutating: false,
		definition: &openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "propose_plan",
				Description: "Proposes a plan to the user and returns a boolean indicating whether the user wishes to proceed with the plan or not.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"plan": {
							"type": "string",
							"description": "The plan as markdown text"
						}
					},
					"required": ["plan"]
				}`),
			},
		},
		handler: func(argsJSON string, options *ToolOptions) (string, error) {
			var args ProposePlanArgs
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("invalid arguments for read_file: %w", err)
			}

			options.WriteChannel <- wire.BackendMessage{
				ProposalMessage: &args.Plan,
				WorkDone:        true,
			}

			confirmed := <-options.ConfirmationChannel

			// Return boolean as string indicating user's decision
			return fmt.Sprintf("%t", confirmed), nil
		},
	}
}
