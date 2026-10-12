package backend

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/xi0/coderoom-ai/internal/wire"
	"znkr.io/diff/textdiff"
)

var (
	Options = []string{
		"Reticulate the splines",
		"Frob the foos with the bars",
		"Refactor everything",
		"I don't really care",
	}

	confirmationFunc func(chan wire.BackendMessage, string, bool)
)

type TestBackend struct {
	Settings *Settings
}

func (be *TestBackend) Run(writeChannel chan wire.BackendMessage, readChannel chan wire.FrontendMessage, sessionID string) {
	log.Printf("Test backend (Session ID: %s)", sessionID)

	confirmationFunc = be.confirmation

	go func() {
		for message := range readChannel {
			if message.Ping != nil {
				writeChannel <- wire.BackendMessage{
					Pong: message.Ping,
				}
			} else {
				be.handleMessage(message, writeChannel, sessionID)
			}
		}
	}()

	systemMessage := `The backend is running in test mode.

You can use the following keywords to test different message types in the UI.

* system
* reasoning
* tool
* proposal
* options
* blocked`

	writeChannel <- wire.BackendMessage{
		Init: &wire.InitMessage{
			Modifications: be.Settings.GetDefaultModifications(),
			DarkTheme:     be.Settings.GetDarkTheme(),
			Verbosity:     be.Settings.GetVerbosity(),
			ProjectName:   be.Settings.GetProjectName(),
			ProjectDir:    be.Settings.ProjectDir,
			Providers:     be.Settings.GetProviders(),
		},
		SystemMessage: &systemMessage,
		WorkDone:      true,
		EnablePrompt:  true,
	}

	select {}
}

func (be *TestBackend) handleMessage(message wire.FrontendMessage, writeChannel chan wire.BackendMessage, sessionID string) {
	if message.Prompt != nil {
		log.Printf("Prompt: %q\n", message.Prompt.Prompt)

		if strings.Contains(message.Prompt.Prompt, "system") {
			go be.systemMessage(writeChannel)
		} else if strings.Contains(message.Prompt.Prompt, "reasoning") {
			go be.reasoningMessage(writeChannel)
		} else if strings.Contains(message.Prompt.Prompt, "tool") {
			go be.toolMessage(writeChannel)
		} else if strings.Contains(message.Prompt.Prompt, "proposal") {
			confirmationFunc = be.confirmation
			go be.proposalMessage(writeChannel)
		} else if strings.Contains(message.Prompt.Prompt, "options") {
			go be.optionsMessage(writeChannel)
		} else if strings.Contains(message.Prompt.Prompt, "blocked") {
			confirmationFunc = be.blockedConfirmation
			go be.blockedMessage(writeChannel)
		} else {
			go be.unknownMessage(writeChannel)
		}
	}

	if message.ChosenOption != nil {
		log.Printf("Chosen option: %d\n", *message.ChosenOption)
		be.chosenOption(writeChannel, *message.ChosenOption)
	}

	if message.Confirmation != nil {
		log.Printf("Confirmation: %t\n", *message.Confirmation)
		confirmationFunc(writeChannel, sessionID, *message.Confirmation)
	}
}

// =============================================

func (be *TestBackend) systemMessage(writeChannel chan wire.BackendMessage) {
	time.Sleep(2 * time.Second)

	systemMessage := `This is a test message.

### This is a heading

This is some more text.`

	writeChannel <- wire.BackendMessage{
		SystemMessage: &systemMessage,
		WorkDone:      true,
		EnablePrompt:  true,
	}
}

func (be *TestBackend) reasoningMessage(writeChannel chan wire.BackendMessage) {
	time.Sleep(2 * time.Second)

	reasoningMessage := `Let me think about this step by step.

1. First, I consider the available tools.
2. Then, I decide that *this* is the best approach.`

	writeChannel <- wire.BackendMessage{
		ReasoningMessage: &reasoningMessage,
		WorkDone:         true,
		EnablePrompt:     true,
	}
}

func (be *TestBackend) toolMessage(writeChannel chan wire.BackendMessage) {
	time.Sleep(2 * time.Second)

	toolMessage := "read_file(\"README.md\")"

	writeChannel <- wire.BackendMessage{
		ToolMessage:  &wire.ToolMessage{Tool: toolMessage},
		WorkDone:     true,
		EnablePrompt: true,
	}
}

func (be *TestBackend) proposalMessage(writeChannel chan wire.BackendMessage) {
	time.Sleep(2 * time.Second)

	proposalMessage := `### Overview
I propose implementing a new feature to enhance the user experience. This plan outlines the key steps and considerations.

### Key Steps
* **Phase 1:** Analyze current codebase structure</li>
* **Phase 2:** Design the new component architecture</li>
* **Phase 3:** Implement core functionality</li>
* **Phase 4:** Add unit tests and documentation</li>

### Expected Outcomes
* Improved performance by 30%
* Better code maintainability
* Enhanced user interface responsiveness
`

	writeChannel <- wire.BackendMessage{
		ProposalMessage: &proposalMessage,
		WorkDone:        true,
	}
}

func (be *TestBackend) confirmation(writeChannel chan wire.BackendMessage, sessionID string, confirmation bool) {
	if !confirmation {
		systemMessage := `I will not go ahead with the proposed plan.`

		writeChannel <- wire.BackendMessage{
			SystemMessage: &systemMessage,
			WorkDone:      true,
			EnablePrompt:  true,
		}

		return
	}

	time.Sleep(2 * time.Second)

	writeChannel <- wire.BackendMessage{
		UpdateProgress: &wire.ProgressMessage{
			Percent: 20,
			Text:    "Analyzing current codebase structure",
		},
	}

	time.Sleep(2 * time.Second)

	for _, f := range []string{"src/core.go", "src/foo.go", "src/bar.go"} {
		toolMessage := fmt.Sprintf("read_file(%q)", f)

		writeChannel <- wire.BackendMessage{
			ToolMessage: &wire.ToolMessage{
				Tool: toolMessage,
			},
		}

		time.Sleep(1 * time.Second)
	}

	writeChannel <- wire.BackendMessage{
		UpdateProgress: &wire.ProgressMessage{
			Percent: 40,
			Text:    "Designing the new component architecture",
		},
	}

	time.Sleep(2 * time.Second)

	writeChannel <- wire.BackendMessage{
		UpdateProgress: &wire.ProgressMessage{
			Percent: 60,
			Text:    "Implementing core functionality",
		},
	}

	time.Sleep(2 * time.Second)

	for i := 0; i < 3; i++ {
		toolMessage := "edit_file(\"src/core.go\")"
		multipleFmt := "%d edits"

		writeChannel <- wire.BackendMessage{
			ToolMessage: &wire.ToolMessage{
				Tool:        toolMessage,
				MultipleFmt: &multipleFmt,
			},
		}

		time.Sleep(2 * time.Second)
	}

	time.Sleep(2 * time.Second)

	writeChannel <- wire.BackendMessage{
		UpdateProgress: &wire.ProgressMessage{
			Percent: 80,
			Text:    "Adding unit tests and documentation",
		},
	}

	time.Sleep(2 * time.Second)

	for i := 0; i < 2; i++ {
		toolMessage := "edit_file(\"src/core_test.go\")"
		multipleFmt := "%d edits"

		writeChannel <- wire.BackendMessage{
			ToolMessage: &wire.ToolMessage{
				Tool:        toolMessage,
				MultipleFmt: &multipleFmt,
			},
		}

		time.Sleep(2 * time.Second)
	}

	be.addEdits(sessionID)

	systemMessage := `I have succesfully completed the outlined steps. 🎉`
	filesMessage := &wire.FilesMessage{
		Files: []wire.File{
			wire.File{
				Filename: "src/core.go",
				Action:   wire.FileActionModifiedFile,
				URI:      fmt.Sprintf("/edits/%s/src/core.go", sessionID),
			},
			wire.File{
				Filename: "src/core_test.go",
				Action:   wire.FileActionModifiedFile,
				URI:      fmt.Sprintf("/edits/%s/src/core_test.go", sessionID),
			},
		},
	}

	writeChannel <- wire.BackendMessage{
		SystemMessage: &systemMessage,
		FilesMessage:  filesMessage,
		WorkDone:      true,
		EnablePrompt:  true,
	}
}

func (be *TestBackend) optionsMessage(writeChannel chan wire.BackendMessage) {
	time.Sleep(2 * time.Second)

	writeChannel <- wire.BackendMessage{
		OptionsMessage: &wire.OptionsMessage{
			Description: "Please choose how you want to proceed.",
			Options:     Options,
		},
		WorkDone: true,
	}
}

func (be *TestBackend) chosenOption(writeChannel chan wire.BackendMessage, option int) {
	time.Sleep(2 * time.Second)

	systemMessage := fmt.Sprintf("I will proceed with the option %q.", Options[option])

	writeChannel <- wire.BackendMessage{
		SystemMessage: &systemMessage,
		WorkDone:      true,
		EnablePrompt:  true,
	}
}

func (be *TestBackend) blockedMessage(writeChannel chan wire.BackendMessage) {
	time.Sleep(2 * time.Second)

	writeChannel <- wire.BackendMessage{
		BlockedMessage: &wire.BlockedMessage{
			Action: "Build project",
			Filenames: []string{
				"/foo/bar",
				"/foo/baz",
				"/test/party",
			},
		},
		WorkDone: true,
	}
}

func (be *TestBackend) blockedConfirmation(writeChannel chan wire.BackendMessage, sessionID string, confirmation bool) {
	if !confirmation {
		systemMessage := `I will not run the action.`

		writeChannel <- wire.BackendMessage{
			SystemMessage: &systemMessage,
			WorkDone:      true,
			EnablePrompt:  true,
		}

		return
	}

	toolMessage := "build_project()"

	writeChannel <- wire.BackendMessage{
		ToolMessage: &wire.ToolMessage{Tool: toolMessage},
	}

	time.Sleep(2 * time.Second)

	systemMessage := `I successfully ran the action.`

	writeChannel <- wire.BackendMessage{
		SystemMessage: &systemMessage,
		WorkDone:      true,
		EnablePrompt:  true,
	}
}

func (be *TestBackend) unknownMessage(writeChannel chan wire.BackendMessage) {
	systemMessage := `I did not find a matching keyword in your prompt.`

	writeChannel <- wire.BackendMessage{
		SystemMessage: &systemMessage,
		WorkDone:      true,
		EnablePrompt:  true,
	}
}

func (be *TestBackend) addEdits(sessionID string) {
	files := make(map[string]*wire.EditResponse)

	fileNames := []string{"src/core.go", "src/core_test.go"}
	for _, fn := range fileNames {
		var lines1, lines2 []string
		for i := 0; i < 100; i++ {
			line1 := fmt.Sprintf("%s line %d", fn, i)
			line2 := fmt.Sprintf("%s line %d", fn, i)

			if i == 75 {
				line2 += " (something added)"
			}

			if i != 37 {
				lines1 = append(lines1, line1)
			}
			if i != 45 && i != 46 {
				lines2 = append(lines2, line2)
			}
		}

		// Calculate and add the unified diff
		files[fn] = &wire.EditResponse{
			Action: wire.FileActionModifiedFile,
			Content: textdiff.Unified(
				strings.Join(lines1, "\n"),
				strings.Join(lines2, "\n"),
			),
		}

	}

	addEdits(sessionID, files)
}
