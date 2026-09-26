package tools

import (
	"encoding/json"
	"fmt"

	"github.com/xi0/coderoom-ai/internal/wire"

	"github.com/sashabaranov/go-openai"
)

type ReadFileArgs struct {
	RelativePath string `json:"relative_path"`
}

func readFileTool() *Tool {
	return &Tool{
		mutating: false,
		definition: &openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "read_file",
				Description: "Reads the content of a specified file relative to the project root.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"relative_path": {
							"type": "string",
							"description": "Relative path to the file from project root"
						}
					},
					"required": ["relative_path"]
				}`),
			},
		},
		handler: func(argsJSON string, options *ToolOptions) (string, error) {
			var args ReadFileArgs
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("invalid arguments for read_file: %w", err)
			}

			toolString := fmt.Sprintf("read_file(%q)", args.RelativePath)
			options.WriteChannel <- wire.BackendMessage{
				ToolMessage: &wire.ToolMessage{Tool: toolString},
			}

			content, err := options.Root.ReadFile(args.RelativePath)
			if err != nil {
				return "", fmt.Errorf("failed to read file: %w", err)
			}

			if len(content) == 0 {
				return "", fmt.Errorf("file is empty")
			}

			return string(content), nil
		},
	}
}
