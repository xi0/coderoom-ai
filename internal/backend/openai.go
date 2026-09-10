package backend

import (
	"context"
	"fmt"
	"log"

	"github.com/xi0/coderoom-ai/internal/common"
	"github.com/xi0/coderoom-ai/internal/tools"
	"github.com/xi0/coderoom-ai/internal/wire"

	"github.com/sashabaranov/go-openai"
)

type OpenAI struct {
	Settings *Settings
}

const (
	StateInit     = 0
	StateAllowDir = 1
	StateChat     = 2

	systemPrompt = `You are a helpful software engineering code assistant. Explore and modify the project using available tools to fulfill requests.
If a change touches more than 10 lines or more than one file, propose the plan to the user and get confirmation before proceeding.
If more than one solution makes sense then ask the user for the desired option.
Report progress every 5-10 seconds if possible.`
)

func (be *OpenAI) Run(writeChannel chan wire.BackendMessage, readChannel chan wire.FrontendMessage) {
	log.Println("OpenAI backend")

	messageChannel := make(chan wire.FrontendMessage)

	go func() {
		for message := range readChannel {
			if message.Ping != nil {
				writeChannel <- wire.BackendMessage{
					Pong: message.Ping,
				}
			} else {
				messageChannel <- message
			}
		}
		close(messageChannel)
	}()

	be.chat(writeChannel, messageChannel)
}

func (be *OpenAI) chat(writeChannel chan wire.BackendMessage, readChannel chan wire.FrontendMessage) {
	state := StateInit

	writeChannel <- wire.BackendMessage{
		Init: &wire.InitMessage{
			Modifications: be.Settings.GetDefaultModifications(),
			DarkTheme:     be.Settings.GetDarkTheme(),
			ProjectName:   be.Settings.GetProjectName(),
			ProjectDir:    be.Settings.ProjectDir,
		},
	}

	if be.Settings.DirAllowed() {
		state = StateChat
		be.sendGreeting(writeChannel)
	} else {
		state = StateAllowDir

		writeChannel <- wire.BackendMessage{
			AllowDirMessage: &be.Settings.ProjectDir,
			WorkDone:        true,
		}
	}

	promptChannel := make(chan string)
	optionChannel := make(chan int)
	confirmationChannel := make(chan bool)
	go be.agentLoop(writeChannel, promptChannel, optionChannel, confirmationChannel)

	for message := range readChannel {
		switch state {
		case StateAllowDir:
			if message.AllowDir != nil {
				if *message.AllowDir {
					be.Settings.AllowDir()
					be.Settings.saveGlobal()

					systemMessage := "Thank you! The directory is now allowed."

					writeChannel <- wire.BackendMessage{
						SystemMessage: &systemMessage,
					}

					state = StateChat
					be.sendGreeting(writeChannel)
				} else {
					systemMessage := "Goodbye!"

					writeChannel <- wire.BackendMessage{
						SystemMessage: &systemMessage,
						WorkDone:      true,
					}
				}
			}
		case StateChat:
			if message.Prompt != nil {
				promptChannel <- *message.Prompt
			}
			if message.ChosenOption != nil {
				optionChannel <- *message.ChosenOption
			}
			if message.Confirmation != nil {
				confirmationChannel <- *message.Confirmation
			}
		}
	}
}

func (be *OpenAI) sendGreeting(writeChannel chan wire.BackendMessage) {
	systemMessage := "Welcome to Coderoom AI. How can I help you with your code today?"

	writeChannel <- wire.BackendMessage{
		SystemMessage: &systemMessage,
		WorkDone:      true,
		EnablePrompt:  true,
	}
}

func (be *OpenAI) client() *openai.Client {
	provider := be.Settings.GetDefaultProvider()
	baseURL := ""
	for _, p := range common.Providers {
		if p.ProviderID == provider.ProviderID {
			for _, m := range p.Models {
				if m.ModelID == provider.ModelID {
					baseURL = p.BaseURL
				}
			}
		}
	}

	if baseURL == "" {
		return nil
	}

	config := openai.DefaultConfig(provider.APIKey)
	config.BaseURL = baseURL
	return openai.NewClientWithConfig(config)
}

func (be *OpenAI) agentLoop(writeChannel chan wire.BackendMessage, promptChannel chan string, optionChannel chan int, confirmationChannel chan bool) {
	ctx := context.Background()
	toolsList := tools.BuildToolsList()

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
	}

	for prompt := range promptChannel {
		// TODO: Get modifications bool along with the prompt
		modifications := true

		systemMessage := fmt.Sprintf("Got prompt:\n\n%s", prompt)

		writeChannel <- wire.BackendMessage{
			SystemMessage: &systemMessage,
			WorkDone:      true,
			EnablePrompt:  true,
		}

		messages = append(messages,
			openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		)

		req := openai.ChatCompletionRequest{
			Model:    "code",
			Messages: messages,
			Tools:    toolsList.Get(modifications),
		}

		client := be.client()
		resp, err := client.CreateChatCompletion(ctx, req)
		if err != nil {
			log.Fatalf("API call failed: %v", err)
		}

		msg := resp.Choices[0].Message
		messages = append(messages, msg)

		// If no tool calls were made, the model has finished its response
		if len(msg.ToolCalls) == 0 {
			writeChannel <- wire.BackendMessage{
				SystemMessage: &msg.Content,
				WorkDone:      true,
				EnablePrompt:  true,
			}
			break
		}

		// Handle tool execution requests from the model
		/*
			for _, toolCall := range msg.ToolCalls {
				truncatedArguments := toolCall.Function.Arguments
				if len(truncatedArguments) > 30 {
					truncatedArguments = truncatedArguments[0:30] + "..."
				}
				fmt.Printf("[Tool Call] Running %s with args: %s\n", toolCall.Function.Name, truncatedArguments)

				output, err := run_tool(toolCall.Function.Name, toolCall.Function.Arguments, targetDir, nil, nil)
				if err != nil {
					output = fmt.Sprintf("Error executing tool: %v", err)
				}

				// Append tool execution result back to conversation history
				messages = append(messages, openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					Content:    output,
					ToolCallID: toolCall.ID,
				})
			}
		*/
	}
}
