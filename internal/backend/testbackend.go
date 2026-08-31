package backend

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/xi0/coderoom-ai/internal/wire"
)

var (
	Options = []string{
		"Reticulate the splines",
		"Frob the foos with the bars",
		"Refactor everything",
		"I don't really care",
	}
)

type TestBackend struct {
	Settings *Settings
}

func (be *TestBackend) Run(writeChannel chan wire.BackendMessage, readChannel chan wire.FrontendMessage) {
	log.Println("Test backend")

	go func() {
		for message := range readChannel {
			if message.Ping != nil {
				writeChannel <- wire.BackendMessage{
					Pong: message.Ping,
				}
			} else {
				handleMessage(message, writeChannel)
			}
		}
	}()

	systemMessage := `The backend is running in test mode.

You can use the following keywords to test different message types in the UI.

* system
* tool
* proposal
* options`

	writeChannel <- wire.BackendMessage{
		Init: &wire.InitMessage{
			Modifications: be.Settings.GetDefaultModifications(),
			DarkTheme:     be.Settings.GetDarkTheme(),
			ProjectName:   "Coderoom AI",
			ProjectDir:    "/home/xi/projects/coderoom-ai",
		},
		SystemMessage: &systemMessage,
		WorkDone:      true,
		EnablePrompt:  true,
	}

	select {}
}

func handleMessage(message wire.FrontendMessage, writeChannel chan wire.BackendMessage) {
	if message.Prompt != nil {
		log.Printf("Prompt: %q\n", *message.Prompt)

		if strings.Contains(*message.Prompt, "system") {
			go systemMessage(writeChannel)
		} else if strings.Contains(*message.Prompt, "tool") {
			go toolMessage(writeChannel)
		} else if strings.Contains(*message.Prompt, "proposal") {
			go proposalMessage(writeChannel)
		} else if strings.Contains(*message.Prompt, "options") {
			go optionsMessage(writeChannel)
		} else {
			go unknownMessage(writeChannel)
		}
	}

	if message.ChosenOption != nil {
		log.Printf("Chosen option: %d\n", *message.ChosenOption)
		chosenOption(writeChannel, *message.ChosenOption)
	}

	if message.Confirmation != nil {
		log.Printf("Confirmation: %t\n", *message.Confirmation)
		confirmation(writeChannel, *message.Confirmation)
	}
}

// =============================================

func systemMessage(writeChannel chan wire.BackendMessage) {
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

func toolMessage(writeChannel chan wire.BackendMessage) {
	time.Sleep(2 * time.Second)

	toolMessage := "read_file(\"README.md\")"

	writeChannel <- wire.BackendMessage{
		ToolMessage:  &toolMessage,
		WorkDone:     true,
		EnablePrompt: true,
	}
}

func proposalMessage(writeChannel chan wire.BackendMessage) {
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

func confirmation(writeChannel chan wire.BackendMessage, confirmation bool) {
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

	writeChannel <- wire.BackendMessage{
		UpdateProgress: &wire.ProgressMessage{
			Percent: 80,
			Text:    "Adding unit tests and documentation",
		},
	}

	time.Sleep(2 * time.Second)

	systemMessage := `I have succesfully completed the outlined steps. 🎉`

	writeChannel <- wire.BackendMessage{
		SystemMessage: &systemMessage,
		WorkDone:      true,
		EnablePrompt:  true,
	}
}

func optionsMessage(writeChannel chan wire.BackendMessage) {
	time.Sleep(2 * time.Second)

	writeChannel <- wire.BackendMessage{
		OptionsMessage: &wire.OptionsMessage{
			Description: "Please choose how you want to proceed.",
			Options:     Options,
		},
		WorkDone: true,
	}
}

func chosenOption(writeChannel chan wire.BackendMessage, option int) {
	time.Sleep(2 * time.Second)

	systemMessage := fmt.Sprintf("I will proceed with the option %q.", Options[option])

	writeChannel <- wire.BackendMessage{
		SystemMessage: &systemMessage,
		WorkDone:      true,
		EnablePrompt:  true,
	}
}

func unknownMessage(writeChannel chan wire.BackendMessage) {
	systemMessage := `I did not find a matching keyword in your prompt.`

	writeChannel <- wire.BackendMessage{
		SystemMessage: &systemMessage,
		WorkDone:      true,
		EnablePrompt:  true,
	}
}
