package tools

import (
	"encoding/json"
	"fmt"

	"github.com/xi0/coderoom-ai/internal/wire"

	"github.com/sashabaranov/go-openai"
)

type DeleteFileArgs struct {
	RelativePath string `json:"relative_path"`
}

func deleteFileTool() *Tool {
	return &Tool{
		mutating: true,
		definition: &openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "delete_file",
				Description: "Deletes the specified file relative to the project root.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"relative_path": {
							"type": "string",
							"description": "Path to the file relative to the project root"
						}
					},
					"required": ["relative_path"]
				}`),
			},
		},
		handler: func(argsJSON string, options *ToolOptions) (string, error) {
			var args DeleteFileArgs
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("invalid arguments for delete_file: %w", err)
			}

			toolString := fmt.Sprintf("delete_file(%q)", args.RelativePath)
			options.WriteChannel <- wire.BackendMessage{
				ToolMessage: &toolString,
			}

			// Capture the original state of the file before it is deleted so
			// that the recorded edit references the deleted contents.
			if options.Edits != nil {
				options.Edits.registerFile(args.RelativePath)
			}

			err := options.Root.Remove(args.RelativePath)
			if err != nil {
				return "", fmt.Errorf("failed to delete file %q: %w", args.RelativePath, err)
			}

			if options.Edits != nil {
				options.Edits.DeleteFile(args.RelativePath)
			}

			return fmt.Sprintf("File deleted successfully:\n%s", args.RelativePath), nil
		},
	}
}
