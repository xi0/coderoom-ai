package ui

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

	// Add handlers for main controls

	modeToggle := doc.GetElementByID("mode-toggle")
	modeToggle.AddClickHandler(toggleMode)

	resetSessionButton := doc.GetElementByID("reset-session-btn")
	resetSessionButton.AddClickHandler(resetSession)

	themeToggle := doc.GetElementByID("theme-toggle")
	themeToggle.AddClickHandler(toggleTheme)

	settingsToggle := doc.GetElementByID("settings-toggle")
	settingsToggle.AddClickHandler(toggleSettings)
	doc.AddClickHandler(hideSettings)

	settingsMenu := doc.GetElementByID("settings-menu")
	menuButtons := settingsMenu.GetElementsByTagName("button")
	menuButtons[0].AddClickHandler(globalSettings)
	menuButtons[1].AddClickHandler(projectSettings)

	// Add handlers for prompt input

	promptInput := doc.GetElementByID("prompt-input")
	promptInput.AddInputHandler(adjustPromptHeight)

	sendButton := doc.GetElementByID("send-btn")
	sendButton.AddClickHandler(submitPrompt)

	// Add handlers for settings dialogs

	dialog := doc.GetElementByID("global-settings")
	saveGlobalSettingsButton := dialog.GetElementsByTagName("button")[0]
	saveGlobalSettingsButton.AddClickHandler(saveGlobalSettings)

	go initDone()
}

func initDone() {
	webSocket = browser.NewWebSocket(
		browser.MessageHandlers{
			Close:           handleClose,
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

func resetSession(this, e *browser.Object) any {
	if browser.Confirm("Are you sure that you want to reset the session?") {
		browser.Document().Location().Reload()
	}

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

	go saveTheme(theme)

	return nil
}

func saveTheme(theme string) {
	values := url.Values{}
	if theme == "dark" {
		values.Set("dark_theme", "1")
	} else {
		values.Set("dark_theme", "0")
	}

	location := browser.Document().Location()
	url := fmt.Sprintf("%s//%s/settings/global/theme", location.Protocol, location.Host)

	resp, err := http.DefaultClient.PostForm(url, values)

	if err != nil {
		fmt.Printf("PostForm(): %v\n", err)
		return
	}

	if resp.StatusCode == http.StatusOK {
		return
	}

	fmt.Printf("Post to URL %q failed with status %q\n", url, resp.Status)
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

func globalSettings(this, e *browser.Object) any {
	fmt.Println("globalSettings")

	doc := browser.Document()
	dialog := doc.GetElementByID("global-settings")
	dialog.ShowModal()

	go loadGlobalSettings()

	return nil
}

func loadGlobalSettings() {
	doc := browser.Document()
	dialog := doc.GetElementByID("global-settings")

	location := browser.Document().Location()
	url := fmt.Sprintf("%s//%s/settings/global", location.Protocol, location.Host)

	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		fmt.Printf("Get(): %v\n", err)
		return
	}

	if resp.StatusCode == http.StatusOK {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("io.ReadAll(): %v", err)
			return
		}

		textArea := dialog.GetElementsByTagName("textarea")[0]
		textArea.RemoveChildren()
		textArea.Append(browser.Text(string(data)))
		return
	}

	fmt.Printf("Post to URL %q failed with status %q\n", url, resp.Status)
}

func saveGlobalSettings(this, e *browser.Object) any {
	doc := browser.Document()
	dialog := doc.GetElementByID("global-settings")

	fmt.Println("saveGlobalSettings")

	location := browser.Document().Location()
	url := fmt.Sprintf("%s//%s/settings/global", location.Protocol, location.Host)

	textArea := dialog.GetElementsByTagName("textarea")[0]
	data := []byte(textArea.GetValue())

	go func() {
		resp, err := http.DefaultClient.Post(url, "application/json", bytes.NewBuffer(data))
		if err != nil {
			fmt.Printf("Post(): %v\n", err)
			return
		}

		if resp.StatusCode == http.StatusOK {
			browser.Alert("Global settings saved")
		} else {
			browser.Alert("Global settings NOT saved")
		}
	}()

	return nil
}

func projectSettings(this, e *browser.Object) any {
	fmt.Println("projectSettings")
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

	adjustPromptHeight(prompt, e)

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

func handleClose(wasError bool) {
	doc := browser.Document()
	prompt := doc.GetElementByID("prompt-input")
	prompt.Disabled(true)
	wrapper := doc.GetElementsByClassName("input-wrapper")[0]
	wrapper.RemoveClass("focus")

	text := "Connection lost"
	if wasError {
		text += " due to error"
	}

	browser.Alert(text)
}

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

	if !message.DarkTheme {
		documentElement := browser.DocumentElement()
		theme := documentElement.GetAttribute("data-theme")
		theme = "light"
		documentElement.SetAttribute("data-theme", theme)
	}

	name := doc.GetElementByID("project-name")
	name.TextContent(message.ProjectName)

	dir := doc.GetElementByID("project-dir")
	dir.TextContent(message.ProjectDir)
}

func handleSystemMessage(markdown string) {
	addMessage(systemMessage(markdown))
}

func handleToolMessage(text string) {
	addMessage(toolMessage(text))
}

func handleProposalMessage(markdown string) {
	addMessage(proposalMessage(markdown, sendConfirmation))
}

func sendConfirmation(confirmation bool) {
	webSocket.Send(wire.FrontendMessage{
		Confirmation: &confirmation,
	})
}

func handleOptionsMessage(message *wire.OptionsMessage) {
	addMessage(optionsMessage(message.Description, message.Options, sendChosenOption))
}

func sendChosenOption(option int) {
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
	doc := browser.Document()

	wrapper := doc.GetElementsByClassName("input-wrapper")[0]
	wrapper.AddClass("focus")

	prompt := doc.GetElementByID("prompt-input")
	prompt.Disabled(false)
	prompt.Focus()
}
