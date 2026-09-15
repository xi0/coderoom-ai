package tools

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/xi0/coderoom-ai/internal/wire"

	"github.com/sashabaranov/go-openai"
)

func commandTool(name, description string, tool *wire.ToolSettings) *Tool {
	return &Tool{
		mutating: true,
		definition: &openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        name,
				Description: description,
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {},
					"required": []
				}`),
			},
		},
		handler: func(argsJSON string, options *ToolOptions) (string, error) {
			toolString := fmt.Sprintf("%s()", name)
			options.WriteChannel <- wire.BackendMessage{
				ToolMessage: &toolString,
			}

			// TODO: Check for blocking files

			cmd := exec.Command("bash", "-c", fmt.Sprintf("cd %q && %s", options.Root.Name(), tool.Command))
			output, err := cmd.CombinedOutput()
			return string(output), err
		},
	}
}
