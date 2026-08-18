package ui

import (
	"fmt"
	"time"

	"github.com/xi0/coderoom-ai/internal/browser"
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

	go initDone()
}

func initDone() {
	time.Sleep(2 * time.Second)

	doc := browser.Document()

	addMessage(systemMessage("Hello! I am Coderoom AI. How can I assist you with your code today?"))
	setProgress(14)
	time.Sleep(2 * time.Second)

	addMessage(userMessage("Can you help me set up a responsive UI header and layout?"))
	setProgress(29)
	time.Sleep(2 * time.Second)

	addMessage(systemMessage("Certainly! Here is a clean CSS Flexbox structure designed to stretch across the full viewport, feature a fixed header and expandable auto-resizing input box at the bottom."))
	setProgress(43)
	time.Sleep(2 * time.Second)

	addMessage(toolMessage("read_file(\"html/some_interesting_file.html\")"))
	setProgress(57)
	time.Sleep(2 * time.Second)

	addMessage(proposalMessage(`### Overview
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
`))
	setProgress(71)
	time.Sleep(2 * time.Second)

	addMessage(optionsMessage(
		[]string{
			"This is a much longer placeholder text option that is approximately five times longer than the original short option label",
			"Another extended placeholder text option that provides more descriptive context for the user to understand what they are selecting",
			"A third lengthy placeholder option with detailed information to help guide the user toward making an informed decision",
			"The fourth and final extended placeholder option that maintains consistency with the other verbose option descriptions",
		},
	))
	setProgress(86)
	time.Sleep(2 * time.Second)

	workingMessage := doc.GetElementByID("working-message")
	workingMessage.AddClass("hidden")

	focusPrompt()
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

func setProgress(percent int) {
	doc := browser.Document()

	progress := doc.GetElementByID("working-progress")
	progress.Style().Width(fmt.Sprintf("%d%%", percent))
}

func focusPrompt() {
	doc := browser.Document()

	prompt := doc.GetElementByID("prompt-input")
	prompt.Focus()
}
