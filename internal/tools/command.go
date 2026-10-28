package tools

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/xi0/coderoom-ai/internal/blockingfiles"
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
				ToolMessage: &wire.ToolMessage{Tool: toolString},
			}

			blockingFiles := blockingfiles.BlockingFilesInList(tool.BlockingFiles, options.Edits.Filenames())
			if len(blockingFiles) > 0 {
				filesMessage, edits := options.Edits.Diffs(options.SessionID)
				if len(edits) > 0 {
					if options.AddEdits != nil {
						options.AddEdits(options.SessionID, edits)
					}
				}

				options.WriteChannel <- wire.BackendMessage{
					BlockedMessage: &wire.BlockedMessage{
						Action:    name,
						Filenames: blockingFiles,
					},
					FilesMessage: filesMessage,
					WorkDone:     true,
				}

				confirmed := <-options.ConfirmationChannel

				if !confirmed {
					return "", fmt.Errorf("%q action blocked by user", name)
				}
			}

			cmd := exec.Command("bash", "-c", fmt.Sprintf("cd %q && %s", options.Root.Name(), tool.Command))
			output, err := cmd.CombinedOutput()
			if len(output) == 0 {
				output = []byte("(no output from command)")
			}
			return string(output), err
		},
	}
}
