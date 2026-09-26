package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xi0/coderoom-ai/internal/wire"

	"github.com/sashabaranov/go-openai"
)

type ListDirArgs struct {
	RelativePath string `json:"relative_path"`
}

func listDirTool() *Tool {
	return &Tool{
		mutating: false,
		definition: &openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "list_dir",
				Description: "Lists files and directories in a given relative path within the project root.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"relative_path": {
							"type": "string",
							"description": "Relative path from project root (use '' or '.' for root)"
						}
					},
					"required": ["relative_path"]
				}`),
			},
		},
		handler: func(argsJSON string, options *ToolOptions) (string, error) {
			var args ListDirArgs
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("invalid arguments for list_dir: %w", err)
			}

			if args.RelativePath == "" {
				args.RelativePath = "."
			}

			toolString := fmt.Sprintf("list_dir(%q)", args.RelativePath)
			options.WriteChannel <- wire.BackendMessage{
				ToolMessage: &wire.ToolMessage{Tool: toolString},
			}

			dir, err := options.Root.Open(args.RelativePath)
			if err != nil {
				return "", fmt.Errorf("failed to open directory: %w", err)
			}
			defer dir.Close()

			entries, err := dir.ReadDir(0)
			if err != nil {
				return "", fmt.Errorf("failed to read directory: %w", err)
			}

			if len(entries) == 0 {
				return "", fmt.Errorf("directory is empty")
			}

			var result []string
			for _, entry := range entries {
				kind := "file"
				if entry.IsDir() {
					kind = "dir"
				}
				result = append(result, fmt.Sprintf("%s (%s)", entry.Name(), kind))
			}
			return strings.Join(result, "\n"), nil
		},
	}
}
