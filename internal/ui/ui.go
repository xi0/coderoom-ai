package ui

import (
	"fmt"
	"regexp"

	"github.com/xi0/coderoom-ai/internal/browser"
	"github.com/xi0/coderoom-ai/internal/wire"
)

var (
	webSocket      *browser.WebSocket
	onlyWhiteSpace = regexp.MustCompile("^\\s*$")
)

func init() {
	doc := browser.Document()

	modeToggle := doc.GetElementByID("mode-toggle")
	modeToggle.AddClickHandler(toggleMode)

	themeToggle := doc.GetElementByID("theme-toggle")
	themeToggle.AddClickHandler(toggleTheme)

	settingsToggle := doc.GetElementByID("settings-toggle")
	settingsToggle.AddClickHandler(toggleSettings)
	doc.AddClickHandler(hideSettings)

	promptInput := doc.GetElementByID("prompt-input")
	promptInput.AddInputHandler(adjustPromptHeight)

	sendButton := doc.GetElementByID("send-btn")
	sendButton.AddClickHandler(submitPrompt)

	go initDone()
}

func initDone() {
	webSocket = browser.NewWebSocket(
		browser.MessageHandlers{
			Init:            handleInit,
			SystemMessage:   handleSystemMessage,
			ToolMessage:     handleToolMessage,
			ProposalMessage: handleProposalMessage,
			OptionsMessage:  handleOptionsMessage,
			UpdateProgress:  handleUpdateProgress,
			WorkDone:        handleWorkDone,
			EnablePrompt:    handleEnablePrompt,
		},
	)
}

func toggleMode(this, e *browser.Object) any {
	e.PreventDefault()

	mode := this.GetAttribute("data-mode")
	var modeLabel string

	if mode == "modify" {
		mode = "inspect"
		modeLabel = "Inspection only"
	} else {
		mode = "modify"
		modeLabel = "Modifications allowed"
	}

	this.SetAttribute("data-mode", mode)
	this.SetAttribute("aria-label", fmt.Sprintf("Modification mode: %s", modeLabel))
	this.SetAttribute("title", fmt.Sprintf("Modification mode: %s", modeLabel))

	return nil
}

func toggleTheme(this, e *browser.Object) any {
	e.PreventDefault()

	documentElement := browser.DocumentElement()
	theme := documentElement.GetAttribute("data-theme")

	if theme == "dark" {
		theme = "light"
	} else {
		theme = "dark"
	}

	documentElement.SetAttribute("data-theme", theme)

	return nil
}

func toggleSettings(this, e *browser.Object) any {
	e.StopPropagation()

	doc := browser.Document()
	menu := doc.GetElementByID("settings-menu")
	menu.ToggleClass("hidden")

	return nil
}

func hideSettings(this, e *browser.Object) any {
	doc := browser.Document()
	menu := doc.GetElementByID("settings-menu")

	if !menu.HasClass("hidden") {
		menu.AddClass("hidden")
	}

	return nil
}

func adjustPromptHeight(this, e *browser.Object) any {
	this.Style().Height("auto")
	this.Style().Height(fmt.Sprintf("%dpx", min(this.ScrollHeight(), 200)))

	doc := browser.Document()
	workingMessage := doc.GetElementByID("working-message")
	workingMessage.ScrollIntoView()

	return nil
}

// =============================================

func submitPrompt(this, e *browser.Object) any {
	e.PreventDefault()

	doc := browser.Document()
	prompt := doc.GetElementByID("prompt-input")

	text := prompt.GetValue()
	if onlyWhiteSpace.MatchString(text) {
		return nil
	}

	prompt.Disabled(true)
	addMessage(userMessage(text))
	prompt.SetValue("")

	wrapper := doc.GetElementsByClassName("input-wrapper")[0]
	wrapper.RemoveClass("focus")

	modeToggle := doc.GetElementByID("mode-toggle")
	mode := modeToggle.GetAttribute("data-mode")

	modifications := false
	if mode == "modify" {
		modifications = true
	}

	message := wire.FrontendMessage{
		Modifications: modifications,
		Prompt:        &text,
	}

	err := webSocket.Send(message)
	if err != nil {
		fmt.Printf("webSocket.Send(): %v", err)
	}

	progress := doc.GetElementByID("working-progress")
	progress.Style().Width(fmt.Sprintf("%d%%", 0))

	workingMessage := doc.GetElementByID("working-message")
	textElement := workingMessage.GetElementsByClassName("message-text")[0]
	textElement.TextContent("Working...")

	workingMessage.RemoveClass("hidden")
	workingMessage.ScrollIntoView()

	return nil
}

// =============================================

func handleInit(message *wire.InitMessage) {
	doc := browser.Document()
	button := doc.GetElementByID("mode-toggle")
	mode := button.GetAttribute("data-mode")
	var modeLabel string

	if message.Modifications {
		mode = "modify"
		modeLabel = "Modifications allowed"
	} else {
		mode = "inspect"
		modeLabel = "Inspection only"
	}

	button.SetAttribute("data-mode", mode)
	button.SetAttribute("aria-label", fmt.Sprintf("Modification mode: %s", modeLabel))
	button.SetAttribute("title", fmt.Sprintf("Modification mode: %s", modeLabel))

	name := doc.GetElementByID("project-name")
	name.TextContent(message.ProjectName)

	dir := doc.GetElementByID("project-dir")
	dir.TextContent(message.ProjectDir)
}

func handleSystemMessage(markdown string) {
	fmt.Println("handleSystemMessage")

	addMessage(systemMessage(markdown))
}

func handleToolMessage(text string) {
	fmt.Println("handleToolMessage")

	addMessage(toolMessage(text))
}

func handleProposalMessage(markdown string) {
	fmt.Println("handleProposalMessage")

	addMessage(proposalMessage(markdown, sendConfirmation))
}

func sendConfirmation(confirmation bool) {
	fmt.Printf("sendConfirmation: %t\n", confirmation)

	webSocket.Send(wire.FrontendMessage{
		Confirmation: &confirmation,
	})
}

func handleOptionsMessage(message *wire.OptionsMessage) {
	fmt.Println("handleOptionsMessage")

	addMessage(optionsMessage(message.Description, message.Options, sendChosenOption))
}

func sendChosenOption(option int) {
	fmt.Printf("sendChosenOption: %d\n", option)

	webSocket.Send(wire.FrontendMessage{
		ChosenOption: &option,
	})
}

func handleUpdateProgress(message *wire.ProgressMessage) {
	doc := browser.Document()

	progress := doc.GetElementByID("working-progress")
	progress.Style().Width(fmt.Sprintf("%d%%", message.Percent))

	workingMessage := doc.GetElementByID("working-message")
	text := workingMessage.GetElementsByClassName("message-text")[0]
	text.TextContent(message.Text)
}

func handleWorkDone() {
	doc := browser.Document()

	workingMessage := doc.GetElementByID("working-message")
	workingMessage.AddClass("hidden")
}

func handleEnablePrompt() {
	fmt.Println("handleEnablePrompt")

	doc := browser.Document()

	wrapper := doc.GetElementsByClassName("input-wrapper")[0]
	wrapper.AddClass("focus")

	prompt := doc.GetElementByID("prompt-input")
	prompt.Disabled(false)
	prompt.Focus()
}
