package tools

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/xi0/coderoom-ai/internal/wire"

	"github.com/sashabaranov/go-openai"
)

type EditFileArgs struct {
	RelativePath string `json:"relative_path"`
	OldString    string `json:"old_string"`
	NewString    string `json:"new_string"`
}

func editFileTool() *Tool {
	return &Tool{
		mutating: true,
		definition: &openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "edit_file",
				Description: "Edit a file by replacing one block of content with a new block. The first occurrence of the block is replaced.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"relative_path": {
							"type": "string",
							"description": "Path to the file relative to the project root"
						},
						"old_string": {
							"type": "string",
							"description": "The old string to be replaced"
						},
						"new_string": {
							"type": "string",
							"description": "The new string to replace the old one"
						}
					},
					"required": ["relative_path", "old_string", "new_string"]
				}`),
			},
		},
		handler: func(argsJSON string, options *ToolOptions) (string, error) {
			var args EditFileArgs
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("invalid arguments for edit_file: %w", err)
			}

			toolString := fmt.Sprintf("edit_file(%q)", args.RelativePath)
			options.writeChannel <- wire.BackendMessage{
				ToolMessage: &toolString,
			}

			content, err := options.root.ReadFile(args.RelativePath)
			if err != nil {
				return "", fmt.Errorf("failed to read file %q: %w", args.RelativePath, err)
			}

			if bytes.Index(content, []byte(args.OldString)) == -1 {
				return "", fmt.Errorf("the content in old_string is not found in the file")
			}

			content = bytes.Replace(content, []byte(args.OldString), []byte(args.NewString), 1)

			err = options.root.WriteFile(args.RelativePath, []byte(content), 0644)
			if err != nil {
				return "", fmt.Errorf("failed to write file: %w", err)
			}

			return fmt.Sprintf("File edited successfully:\n%s", args.RelativePath), nil
		},
	}
}
