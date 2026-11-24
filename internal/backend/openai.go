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

const (
	StateInit     = 0
	StateAllowDir = 1
	StateChat     = 2

	systemPrompt = `You are a helpful software engineering code assistant. Explore and modify the project using available tools to fulfill requests.
If a change touches more than 10 lines or more than one file, propose the plan to the user and get confirmation before proceeding, using the "propose_plan" tool.
If more than one solution makes sense then ask the user for the desired option, using the "present_options" tool.
Report progress every 5-10 seconds if possible, using the "report_progress" tool.`
)

func (be *OpenAI) Run(writeChannel chan wire.BackendMessage, readChannel chan wire.FrontendMessage, sessionID string) {
	log.Printf("OpenAI backend (Session ID: %s)", sessionID)

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

	be.chat(writeChannel, messageChannel, sessionID)
}

func (be *OpenAI) chat(writeChannel chan wire.BackendMessage, readChannel chan wire.FrontendMessage, sessionID string) {
	state := StateInit

	writeChannel <- wire.BackendMessage{
		Init: &wire.InitMessage{
			Modifications: be.Settings.GetDefaultModifications(),
			DarkTheme:     be.Settings.GetDarkTheme(),
			Verbosity:     be.Settings.GetVerbosity(),
			SubmitHotkey:  be.Settings.GetSubmitHotkey(),
			ProjectName:   be.Settings.GetProjectName(),
			ProjectDir:    be.Settings.ProjectDir,
			Providers:     be.Settings.GetProviders(),
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

	promptChannel := make(chan wire.PromptMessage)
	optionChannel := make(chan int)
	confirmationChannel := make(chan bool)
	retryChannel := make(chan struct{}, 1)
	go be.agentLoop(writeChannel, promptChannel, optionChannel, confirmationChannel, retryChannel, sessionID)

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
			if message.Retry != nil && *message.Retry {
				// Wake the agent loop so that it retries the failed request.
				// The send is non-blocking so that a stray retry request cannot
				// stall the message loop when no retry is expected.
				select {
				case retryChannel <- struct{}{}:
				default:
				}
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

func (be *OpenAI) client(providerID, modelID string) (*openai.Client, *wire.ProviderSettings) {
	provider := be.Settings.GetProvider(providerID, modelID)
	if provider == nil {
		provider = be.Settings.GetDefaultProvider()
	}

	p := common.ProvidersMap[provider.ProviderID][provider.ModelID]

	config := openai.DefaultConfig(provider.APIKey)
	config.BaseURL = p.BaseURL
	return openai.NewClientWithConfig(config), provider
}

func (be *OpenAI) agentLoop(writeChannel chan wire.BackendMessage, promptChannel chan wire.PromptMessage, optionChannel chan int, confirmationChannel chan bool, retryChannel chan struct{}, sessionID string) {
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

	// contextUsage holds the most recent context window usage reported by the
	// provider. It is included in every BackendMessage the agent loop sends so
	// that the UI indicator stays up to date. It stays nil until the first
	// response with usage information is received.
	var contextUsage *wire.ContextUsage

	for prompt := range promptChannel {
		messages = append(messages,
			openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt.Prompt,
			},
		)

		for {
			client, provider := be.client(prompt.ProviderID, prompt.ModelID)

			req := openai.ChatCompletionRequest{
				Model:    provider.ModelID,
				Messages: messages,
				Tools:    toolsList.Get(prompt.Modifications),
			}

			resp, err := client.CreateChatCompletion(ctx, req)
			if err != nil {
				errorMessage := fmt.Sprintf("API call failed: %v", err)
				writeChannel <- wire.BackendMessage{
					ErrorMessage: &wire.ErrorMessage{Text: errorMessage},
					ContextUsage: contextUsage,
					WorkDone:     true,
					EnablePrompt: true,
				}

				// Wait for the user to either retry this request or enter a new
				// prompt. Retrying repeats the exact same request; sending a new
				// prompt starts a fresh request with the new prompt appended to
				// the conversation history.
				select {
				case <-retryChannel:
					continue
				case nextPrompt, ok := <-promptChannel:
					if !ok {
						return
					}
					prompt = nextPrompt
					messages = append(messages,
						openai.ChatCompletionMessage{
							Role:    openai.ChatMessageRoleUser,
							Content: prompt.Prompt,
						},
					)
					continue
				}
			}

			// Record the context window usage reported by the provider. The
			// total token count of the response reflects how much of the
			// model's context window is currently occupied. It is only
			// available when the provider reports usage and the model has a
			// known context window.
			if resp.Usage.TotalTokens > 0 {
				if limit := common.ContextWindow(provider.ProviderID, provider.ModelID); limit > 0 {
					contextUsage = &wire.ContextUsage{
						Used:  resp.Usage.TotalTokens,
						Limit: limit,
					}
				}
			}

			msg := resp.Choices[0].Message

			// Reasoning content (e.g. from reasoning-capable models) is shown to
			// the user at high verbosity. It is only meant for display and must
			// not be echoed back to the provider in subsequent requests, so it
			// is cleared before the message is stored in the history.
			if msg.ReasoningContent != "" {
				reasoning := msg.ReasoningContent
				writeChannel <- wire.BackendMessage{
					ReasoningMessage: &reasoning,
					ContextUsage:     contextUsage,
				}
				msg.ReasoningContent = ""
			}

			messages = append(messages, msg)

			// If no tool calls were made, the model has finished its response
			if len(msg.ToolCalls) == 0 {
				filesMessage, edits := edits.Diffs(sessionID)
				if len(edits) > 0 {
					addEdits(sessionID, edits)
				}

				writeChannel <- wire.BackendMessage{
					SystemMessage: &msg.Content,
					FilesMessage:  filesMessage,
					ContextUsage:  contextUsage,
					WorkDone:      true,
					EnablePrompt:  true,
				}
				break
			} else {
				// Handle tool execution requests from the model
				toolOptions := &tools.ToolOptions{
					SessionID:           sessionID,
					Modifications:       prompt.Modifications,
					Edits:               edits,
					Root:                root,
					WriteChannel:        writeChannel,
					OptionChannel:       optionChannel,
					ConfirmationChannel: confirmationChannel,
					AddEdits:            addEdits,
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
