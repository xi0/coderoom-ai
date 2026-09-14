package tools

import (
	"fmt"
	"os"

	"github.com/xi0/coderoom-ai/internal/wire"

	"github.com/sashabaranov/go-openai"
)

type ToolsList struct {
	list []*Tool
}

func BuildToolsList() *ToolsList {
	return &ToolsList{
		list: []*Tool{
			listDirTool(),
			readFileTool(),
			grepTool(),
			writeFileTool(),
			editFileTool(),
			deleteFileTool(),
			proposePlanTool(),
			presentOptionsTool(),
			reportProgressTool(),
		},
	}
}

func (tl *ToolsList) Get(modifications bool) []openai.Tool {
	var result []openai.Tool

	for _, t := range tl.list {
		if modifications || !t.mutating {
			result = append(result, *t.definition)
		}
	}

	return result
}

func (tl *ToolsList) Call(name string, argsJSON string, options *ToolOptions) (string, error) {
	for _, t := range tl.list {
		if name == t.definition.Function.Name && (options.Modifications || !t.mutating) {
			return t.call(argsJSON, options)
		}
	}

	return "", fmt.Errorf("tool does not exists: %s", name)
}

type Tool struct {
	mutating   bool
	definition *openai.Tool
	handler    func(string, *ToolOptions) (string, error)
}

type ToolOptions struct {
	Modifications       bool
	Root                *os.Root
	WriteChannel        chan wire.BackendMessage
	OptionChannel       chan int
	ConfirmationChannel chan bool
}

func (t *Tool) call(argsJSON string, options *ToolOptions) (string, error) {
	return t.handler(argsJSON, options)
}
