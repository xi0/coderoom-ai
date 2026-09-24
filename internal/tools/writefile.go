package tools

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/xi0/coderoom-ai/internal/wire"

	"github.com/sashabaranov/go-openai"
)

type WriteFileArgs struct {
	RelativePath string `json:"relative_path"`
	Content      string `json:"content"`
}

func writeFileTool() *Tool {
	return &Tool{
		mutating: true,
		definition: &openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "write_file",
				Description: "Creates a new file or overwrites an existing file with specified content.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"relative_path": {
							"type": "string",
							"description": "Path to the file relative to the project root"
						},
						"content": {
							"type": "string",
							"description": "Text content to write to the file"
						}
					},
					"required": ["relative_path", "content"]
				}`),
			},
		},
		handler: func(argsJSON string, options *ToolOptions) (string, error) {
			var args WriteFileArgs
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("invalid arguments for write_file: %w", err)
			}

			toolString := fmt.Sprintf("write_file(%q)", args.RelativePath)
			options.WriteChannel <- wire.BackendMessage{
				ToolMessage: &toolString,
			}

			// Ensure target directory exists before writing
			if err := options.Root.MkdirAll(filepath.Dir(args.RelativePath), 0755); err != nil {
				return "", fmt.Errorf("failed to create parent directory: %w", err)
			}

			content := []byte(args.Content)
			created := false

			_, err := options.Root.Stat(args.RelativePath)
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					created = true
				} else {
					return "", fmt.Errorf("failed to stat file: %w", err)
				}
			}

			// Capture the original state of the file before it is modified so
			// that the recorded edit references the pre-write contents.
			if options.Edits != nil {
				options.Edits.registerFile(args.RelativePath)
			}

			err = options.Root.WriteFile(args.RelativePath, content, 0644)
			if err != nil {
				return "", fmt.Errorf("failed to write file: %w", err)
			}

			if options.Edits != nil {
				options.Edits.WriteFile(args.RelativePath, content, created)
			}

			return fmt.Sprintf("Successfully wrote file: %s", args.RelativePath), nil
		},
	}
}
