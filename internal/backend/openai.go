package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/xi0/coderoom-ai/internal/common"
	"github.com/xi0/coderoom-ai/internal/tools"
	"github.com/xi0/coderoom-ai/internal/wire"

	"github.com/sashabaranov/go-openai"
)

type OpenAI struct {
	Settings *Settings
}

type promptData struct {
	modifications bool
	prompt        string
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

	promptChannel := make(chan promptData)
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
				promptChannel <- promptData{
					modifications: message.Modifications,
					prompt:        *message.Prompt,
				}
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

func (be *OpenAI) client() (*openai.Client, *wire.ProviderSettings) {
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
		return nil, nil
	}

	config := openai.DefaultConfig(provider.APIKey)
	config.BaseURL = baseURL
	return openai.NewClientWithConfig(config), provider
}

func (be *OpenAI) agentLoop(writeChannel chan wire.BackendMessage, promptChannel chan promptData, optionChannel chan int, confirmationChannel chan bool) {
	ctx := context.Background()
	toolsList := tools.BuildToolsList(be.Settings.GetBuildProjectTool(), be.Settings.GetRunTestsTool())

	root, err := os.OpenRoot(be.Settings.ProjectDir)
	if err != nil {
		log.Fatalf("Failed to open root: %v", err)
	}
	defer root.Close()

	edits := tools.NewFileEdits(root)

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
	}

	for prompt := range promptChannel {
		messages = append(messages,
			openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt.prompt,
			},
		)

		for {
			client, provider := be.client()

			req := openai.ChatCompletionRequest{
				Model:    provider.ModelID,
				Messages: messages,
				Tools:    toolsList.Get(prompt.modifications),
			}

			resp, err := client.CreateChatCompletion(ctx, req)
			if err != nil {
				errorMessage := fmt.Sprintf("API call failed: %v", err)
				writeChannel <- wire.BackendMessage{
					SystemMessage: &errorMessage,
					WorkDone:      true,
					EnablePrompt:  true,
				}
				break
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
			} else {
				// Handle tool execution requests from the model
				toolOptions := &tools.ToolOptions{
					Modifications:       prompt.modifications,
					Edits:               edits,
					Root:                root,
					WriteChannel:        writeChannel,
					OptionChannel:       optionChannel,
					ConfirmationChannel: confirmationChannel,
				}

				for _, toolCall := range msg.ToolCalls {
					output, err := toolsList.Call(toolCall.Function.Name, toolCall.Function.Arguments, toolOptions)
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
			}
		}
		dump_messages(&messages)
	}
}

func dump_messages(messages *[]openai.ChatCompletionMessage) {
	messagesJSON, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal messages: %v", err)
	} else {
		filename := fmt.Sprintf("/tmp/messages-%d.json", time.Now().Unix())
		err := os.WriteFile(filename, messagesJSON, 0644)
		if err != nil {
			log.Printf("Failed to write messages file: %v", err)
		} else {
			log.Printf("Messages dumped to %s", filename)
		}
	}
}
