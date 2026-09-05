package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/xi0/coderoom-ai/internal/browser"
	"github.com/xi0/coderoom-ai/internal/common"
	"github.com/xi0/coderoom-ai/internal/wire"
)

var (
	webSocket          *browser.WebSocket
	onlyWhiteSpace     = regexp.MustCompile("^\\s*$")
	providerModelMatch = regexp.MustCompile("^([0-9]+)/([0-9]+)$")
)

func butterbar(message string, isSuccess bool) {
	doc := browser.Document()
	bar := doc.GetElementByID("butterbar")

	bar.RemoveClass("hidden")
	bar.RemoveClass("success")
	bar.RemoveClass("error")

	if isSuccess {
		bar.AddClass("success")
	} else {
		bar.AddClass("error")
	}

	bar.TextContent(message)

	// Hide the butterbar after 3 seconds
	go func() {
		time.Sleep(3 * time.Second)
		bar.AddClass("hidden")
	}()
}

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

	saveGlobalSettingsButton := doc.GetElementByID("save-global-settings-btn")
	saveGlobalSettingsButton.AddClickHandler(saveGlobalSettings)

	addProviderBtn := doc.GetElementByID("add-provider-btn")
	addProviderBtn.AddClickHandler(addProvider)

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
	doc := browser.Document()
	dialog := doc.GetElementByID("global-settings")
	dialog.ShowModal()

	go loadGlobalSettings()

	return nil
}

func loadGlobalSettings() {
	doc := browser.Document()

	location := browser.Document().Location()
	url := fmt.Sprintf("%s//%s/settings/global", location.Protocol, location.Host)

	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		fmt.Printf("Get(): %v\n", err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Get from URL %q failed with status %q\n", url, resp.Status)
		return
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("io.ReadAll(): %v", err)
		return
	}

	var settings wire.GlobalSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		fmt.Printf("JSON unmarshal error: %v\n", err)
		return
	}

	// Set default modifications checkbox
	defaultModCheckbox := doc.GetElementByID("default-modifications")
	if settings.DefaultModifications {
		defaultModCheckbox.SetChecked(true)
	} else {
		defaultModCheckbox.SetChecked(false)
	}

	// Render providers
	renderProviders(settings.Providers)

	// Render allowed directories
	renderAllowedDirs(settings.AllowedDirs)
}

func saveGlobalSettings(this, e *browser.Object) any {
	doc := browser.Document()
	dialog := doc.GetElementByID("global-settings")

	// Get default modifications setting
	defaultModCheckbox := doc.GetElementByID("default-modifications")
	defaultModifications := defaultModCheckbox.GetChecked()

	// Get dark theme from document element
	documentElement := browser.DocumentElement()
	theme := documentElement.GetAttribute("data-theme")
	darkTheme := theme == "dark"

	// Collect providers
	providers, err := collectProviders()
	if err != nil {
		butterbar("Global settings NOT saved", false)
		return nil
	}

	// Collect allowed directories
	allowedDirs := collectAllowedDirs()

	settings := wire.GlobalSettings{
		DefaultModifications: defaultModifications,
		DarkTheme:            darkTheme,
		Providers:            providers,
		AllowedDirs:          allowedDirs,
	}

	jsonData, err := json.Marshal(settings)
	if err != nil {
		fmt.Printf("JSON marshal error: %v\n", err)
		butterbar("Global settings NOT saved", false)
		return nil
	}

	location := browser.Document().Location()
	url := fmt.Sprintf("%s//%s/settings/global", location.Protocol, location.Host)

	go func() {
		resp, err := http.DefaultClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Printf("Post(): %v\n", err)
			butterbar("Global settings NOT saved", false)
			return
		}

		if resp.StatusCode == http.StatusOK {
			butterbar("Global settings saved", true)
			dialog.Close()
		} else {
			butterbar("Global settings NOT saved", false)
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

// =============================================
// Global Settings Helper Functions
// =============================================

func renderProviders(providers []wire.ProviderSettings) {
	doc := browser.Document()
	providersList := doc.GetElementByID("providers-list")
	providersList.RemoveChildren()

	for i, provider := range providers {
		providerItem := createProviderElement(provider, i)
		providersList.Append(providerItem)
	}
}

func createProviderElement(provider wire.ProviderSettings, index int) *browser.Object {
	var providerInfoProvider, providerInfoModel string
	var providerOptions []browser.SelectOption

	if provider.ProviderID == "" && provider.ModelID == "" {
		providerOptions = append(
			providerOptions,
			browser.SelectOption{
				Value: "",
				Text:  "Please select Provider / Model",
			},
		)
	}

	for i, p := range common.Providers {
		for j, m := range p.Models {
			if (provider.ProviderID == "" && provider.ModelID == "") || (provider.ProviderID == p.ProviderID && provider.ModelID == m.ModelID) {
				providerOptions = append(
					providerOptions,
					browser.SelectOption{
						Value: fmt.Sprintf("%d/%d", i, j),
						Text:  fmt.Sprintf("%s / %s", p.Name, m.Name),
					},
				)
			}
			if provider.ProviderID == p.ProviderID && provider.ModelID == m.ModelID {
				providerInfoProvider = p.Name
				providerInfoModel = m.Name
			}
		}
	}

	providerInfo := browser.Div(
		[]string{"provider-info"},
		browser.Div(
			[]string{"provider-name"},
			browser.Text(providerInfoProvider),
		),
		browser.Div(
			[]string{"provider-model"},
			browser.Text(providerInfoModel),
		),
	)

	if provider.Default {
		providerInfo.Append(
			browser.Span(
				[]string{"provider-default-badge"},
				browser.Text("Default"),
			),
		)
	}

	editBtn := browser.Button(
		nil,
		browser.Text("Edit"),
	)
	editBtn.AddClickHandler(toggleProviderEdit)

	providerActions := browser.Div(
		[]string{"provider-actions"},
		editBtn,
	)

	providerHeader := browser.Div(
		[]string{"provider-header"},
		providerInfo,
		providerActions,
	)

	providerDropdown := browser.Div(
		[]string{"form-group"},
		browser.Label(
			nil,
			browser.Text("Provider / Model"),
		),
		browser.Select(
			[]string{"provider-model-dropdown"},
			providerOptions,
		),
	)

	defaultCheckbox := browser.Input(
		[]string{"provider-default-checkbox"},
		browser.InputTypeCheckbox,
		fmt.Sprintf("%d", index),
	)
	if provider.Default {
		defaultCheckbox.SetChecked(true)
	}

	cancelBtn := browser.Button(
		[]string{"secondary-btn"},
		browser.Text("Cancel"),
	)
	cancelBtn.AddClickHandler(toggleProviderEdit)

	saveBtn := browser.Button(
		nil,
		browser.Text("Save"),
	)
	saveBtn.AddClickHandler(saveProvider)

	deleteBtn := browser.Button(
		[]string{"delete-btn"},
		browser.Text("Delete"),
	)
	deleteBtn.AddClickHandler(deleteProvider)

	// Create API key input with toggle button
	apiKeyInput := browser.Input(
		[]string{"api-key-input"},
		browser.InputTypePassword,
		provider.APIKey,
	)

	togglePasswordBtn := browser.Button(
		[]string{"toggle-password-btn"},
		browser.HTML(`<svg class="eye-icon eye-open" viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg><svg class="eye-icon eye-closed" viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2"><path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19m-6.72-1.07a3 3 0 1 1-4.24-4.24"/><line x1="1" y1="1" x2="23" y2="23"/></svg>`),
	)
	togglePasswordBtn.AddClickHandler(togglePasswordVisibility)

	apiKeyWrapper := browser.Div(
		[]string{"api-key-wrapper"},
		apiKeyInput,
		togglePasswordBtn,
	)

	editSection := browser.Div(
		[]string{"provider-edit-section"},
		providerDropdown,
		browser.Div(
			[]string{"form-group"},
			browser.Label(
				nil,
				browser.Text("API Key"),
			),
			apiKeyWrapper,
		),
		browser.Label(
			[]string{"checkbox-label"},
			defaultCheckbox,
			browser.Span(
				nil,
				browser.Text(" Set as default provider"),
			),
		),
		browser.Div(
			[]string{"provider-edit-actions"},
			cancelBtn,
			saveBtn,
			deleteBtn,
		),
	)
	editSection.SetAttribute("data-index", fmt.Sprintf("%d", index))

	providerItem := browser.Div(
		[]string{"provider-item"},
		providerHeader,
		editSection,
	)
	providerItem.SetAttribute("data-index", fmt.Sprintf("%d", index))

	return providerItem
}

func toggleProviderEdit(this, e *browser.Object) any {
	doc := browser.Document()

	providerItem := this.ClosestByClassName("provider-item")
	if providerItem == nil {
		return nil
	}

	index := providerItem.GetAttribute("data-index")

	providersList := doc.GetElementByID("providers-list")

	var editSection *browser.Object

	editSections := providersList.GetElementsByClassName("provider-edit-section")
	for _, es := range editSections {
		if es.GetAttribute("data-index") == index {
			editSection = es
		}
	}
	if editSection == nil {
		return nil
	}

	providerActions := providerItem.GetElementsByClassName("provider-actions")[0]
	buttons := providerActions.GetElementsByTagName("button")

	providerEditActions := editSection.GetElementsByClassName("provider-edit-actions")[0]
	buttons = append(buttons, providerEditActions.GetElementsByTagName("button")[0])

	inputs := providerItem.GetElementsByTagName("input")

	expand := false
	if !editSection.HasClass("expanded") {
		expand = true
		editSection.AddClass("expanded")
	} else {
		if providerItem.GetAttribute("data-new") == "true" {
			providerItem.Remove()
			return nil
		}

		editSection.RemoveClass("expanded")
	}

	for _, input := range inputs {
		if expand {
			saveInputValue(input)
		} else {
			restoreInputValue(input)
		}
	}

	for _, button := range buttons {
		if expand {
			button.TextContent("Cancel")
		} else {
			button.TextContent("Edit")
		}
	}

	return nil
}

func togglePasswordVisibility(this, e *browser.Object) any {
	// Find the api-key-input in the same wrapper
	apiKeyWrapper := this.ClosestByClassName("api-key-wrapper")
	if apiKeyWrapper == nil {
		return nil
	}

	apiKeyInputs := apiKeyWrapper.GetElementsByClassName("api-key-input")
	if len(apiKeyInputs) == 0 {
		return nil
	}

	apiKeyInput := apiKeyInputs[0]
	currentType := apiKeyInput.GetAttribute("type")

	// Toggle between password and text
	if currentType == browser.InputTypePassword {
		apiKeyInput.SetAttribute("type", browser.InputTypeText)
		this.AddClass("showing")
	} else {
		apiKeyInput.SetAttribute("type", browser.InputTypePassword)
		this.RemoveClass("showing")
	}

	return nil
}

func saveInputValue(input *browser.Object) {
	switch input.GetAttribute("type") {
	case browser.InputTypeText:
		input.SetAttribute("data-saved", input.GetValue())
	case browser.InputTypePassword:
		input.SetAttribute("data-saved", input.GetValue())
	case browser.InputTypeCheckbox:
		checked := "false"
		if input.GetChecked() {
			checked = "true"
		}
		input.SetAttribute("data-saved", checked)
	default:
		fmt.Printf("saveInputValue() does not handle input type %q\n", input.GetAttribute("type"))
	}
}

func restoreInputValue(input *browser.Object) {
	switch input.GetAttribute("type") {
	case browser.InputTypeText:
		input.SetValue(input.GetAttribute("data-saved"))
	case browser.InputTypePassword:
		input.SetValue(input.GetAttribute("data-saved"))
	case browser.InputTypeCheckbox:
		checked := false
		if input.GetAttribute("data-saved") == "true" {
			checked = true
		}
		input.SetChecked(checked)
	default:
		fmt.Printf("saveInputValue() does not handle input type %q\n", input.GetAttribute("type"))
	}
}

func saveProvider(this, e *browser.Object) any {
	doc := browser.Document()
	providerItem := this.ClosestByClassName("provider-item")
	if providerItem == nil {
		return nil
	}

	providerItem.RemoveAttribute("data-new")

	// Get input values

	providerModelDropdown := providerItem.GetElementsByClassName("provider-model-dropdown")[0]
	defaultCheckbox := providerItem.GetElementsByClassName("provider-default-checkbox")[0]

	m := providerModelMatch.FindStringSubmatch(providerModelDropdown.GetValue())
	if m == nil {
		browser.Alert("No provider selected!")
		return nil
	}
	providerIndex, err := strconv.Atoi(m[1])
	if err != nil {
		fmt.Printf("Int could not be parsed: %q\n", m[1])
		return nil
	}
	provider := common.Providers[providerIndex]

	modelIndex, err := strconv.Atoi(m[2])
	if err != nil {
		fmt.Printf("Int could not be parsed: %q\n", m[2])
		return nil
	}
	model := provider.Models[modelIndex]

	/*
		providerID := providerIDInput.GetValue()
		modelID := modelIDInput.GetValue()
	*/
	isDefault := defaultCheckbox.GetChecked()

	// Update provider info display
	providerName := providerItem.GetElementsByClassName("provider-name")[0]
	providerModel := providerItem.GetElementsByClassName("provider-model")[0]
	providerName.TextContent(provider.Name)
	providerModel.TextContent(model.Name)

	// Remove existing default badge
	existingBadges := providerItem.GetElementsByClassName("provider-default-badge")
	if len(existingBadges) > 0 {
		existingBadges[0].Remove()
	}

	// Add default badge if applicable
	if isDefault {
		// Remove default from all other providers
		allDefaultCheckboxes := doc.GetElementsByClassName("provider-default-checkbox")
		for i := 0; i < len(allDefaultCheckboxes); i++ {
			checkbox := allDefaultCheckboxes[i]
			if checkbox.GetValue() != defaultCheckbox.GetValue() {
				checkbox.SetChecked(false)
			}
		}

		// Remove all existing default badges
		allBadges := doc.GetElementsByClassName("provider-default-badge")
		for i := 0; i < len(allBadges); i++ {
			allBadges[i].Remove()
		}

		// Add badge to this provider
		providerInfo := providerItem.GetElementsByClassName("provider-info")[0]
		providerInfo.Append(
			browser.Span(
				[]string{"provider-default-badge"},
				browser.Text("Default"),
			),
		)
	}

	// Close edit section
	editSection := providerItem.GetElementsByClassName("provider-edit-section")[0]
	editSection.RemoveClass("expanded")

	// Update edit button text
	providerActions := providerItem.GetElementsByClassName("provider-actions")[0]
	editBtn := providerActions.GetElementsByTagName("button")[0]
	if editBtn != nil {
		editBtn.TextContent("Edit")
	}

	return nil
}

func deleteProvider(this, e *browser.Object) any {
	providerItem := this.ClosestByClassName("provider-item")
	if providerItem == nil {
		return nil
	}

	providerName := providerItem.GetElementsByClassName("provider-name")[0]
	providerID := providerName.GetTextContent()

	if !browser.Confirm(fmt.Sprintf("Are you sure you want to delete provider \"%s\"?", providerID)) {
		return nil
	}

	providerItem.Remove()
	return nil
}

func addProvider(this, e *browser.Object) any {
	doc := browser.Document()
	providersList := doc.GetElementByID("providers-list")

	// Create a new empty provider
	newProvider := wire.ProviderSettings{
		ProviderID: "",
		ModelID:    "",
		APIKey:     "",
		Default:    false,
	}

	// Count existing providers to get next index
	existingProviders := providersList.GetElementsByClassName("provider-item")
	index := len(existingProviders)

	providerItem := createProviderElement(newProvider, index)
	providerItem.SetAttribute("data-new", "true")
	providersList.Append(providerItem)

	// Automatically expand the edit section for the new provider
	editSection := providerItem.GetElementsByClassName("provider-edit-section")[0]
	if editSection != nil {
		editSection.AddClass("expanded")
	}

	return nil
}

func renderAllowedDirs(dirs []string) {
	doc := browser.Document()
	dirsList := doc.GetElementByID("allowed-dirs-list")
	dirsList.RemoveChildren()

	for _, dir := range dirs {
		dirItem := createDirElement(dir)
		dirsList.Append(dirItem)
	}
}

func createDirElement(dir string) *browser.Object {
	deleteBtn := browser.Button(
		nil,
		browser.Text("Delete"),
	)
	deleteBtn.AddClickHandler(deleteDirectory)

	return browser.Div(
		[]string{"dir-item"},
		browser.Div(
			[]string{"dir-name"},
			browser.Text(dir),
		),
		deleteBtn,
	)
}

func deleteDirectory(this, e *browser.Object) any {
	e.PreventDefault()
	dirItem := this.ClosestByClassName("dir-item")
	if dirItem == nil {
		return nil
	}

	dirName := dirItem.GetElementsByClassName("dir-name")[0]
	dirPath := dirName.GetTextContent()

	if !browser.Confirm(fmt.Sprintf("Are you sure you want to delete directory \"%s\"?", dirPath)) {
		return nil
	}

	dirItem.Remove()
	return nil
}

func collectProviders() ([]wire.ProviderSettings, error) {
	doc := browser.Document()
	providersList := doc.GetElementByID("providers-list")
	providerItems := providersList.GetElementsByClassName("provider-item")

	var providers []wire.ProviderSettings

	for i := 0; i < len(providerItems); i++ {
		item := providerItems[i]

		providerModelDropdown := item.GetElementsByClassName("provider-model-dropdown")[0]
		m := providerModelMatch.FindStringSubmatch(providerModelDropdown.GetValue())
		if m == nil {
			browser.Alert("No provider selected!")
			return nil, fmt.Errorf("no provider selected (%d)", i)
		}

		providerIndex, err := strconv.Atoi(m[1])
		if err != nil {
			fmt.Printf("Int could not be parsed: %q\n", m[1])
			return nil, fmt.Errorf("int could not be parsed: %q", m[1])
		}
		provider := common.Providers[providerIndex]

		modelIndex, err := strconv.Atoi(m[2])
		if err != nil {
			fmt.Printf("Int could not be parsed: %q\n", m[2])
			return nil, fmt.Errorf("int could not be parsed: %q", m[2])
		}
		model := provider.Models[modelIndex]

		apiKeyInput := item.GetElementsByClassName("api-key-input")[0]
		defaultCheckbox := item.GetElementsByClassName("provider-default-checkbox")[0]

		wireProvider := wire.ProviderSettings{
			ProviderID: provider.ProviderID,
			ModelID:    model.ModelID,
			APIKey:     apiKeyInput.GetValue(),
			Default:    defaultCheckbox.GetChecked(),
		}
		providers = append(providers, wireProvider)
	}

	return providers, nil
}

func collectAllowedDirs() []string {
	doc := browser.Document()
	dirsList := doc.GetElementByID("allowed-dirs-list")
	dirItems := dirsList.GetElementsByClassName("dir-item")

	var dirs []string

	for i := 0; i < len(dirItems); i++ {
		item := dirItems[i]
		dirName := item.GetElementsByClassName("dir-name")[0]
		dirs = append(dirs, dirName.GetTextContent())
	}

	return dirs
}
