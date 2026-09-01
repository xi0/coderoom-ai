package wire

// Messages from the frontend

type FrontendMessage struct {
	Modifications bool         `json:"modifications"`
	Ping          *PingMessage `json:"ping,omitempty"`
	Prompt        *string      `json:"prompt,omitempty"`
	ChosenOption  *int         `json:"chosen_option,omitempty"`
	Confirmation  *bool        `json:"confirmation,omitempty"`
}

// Messages from the backend

type BackendMessage struct {
	Init            *InitMessage     `json:"init,omitempty"`
	Pong            *PingMessage     `json:"pong,omitempty"`
	SystemMessage   *string          `json:"system_message,omitempty"`
	ToolMessage     *string          `json:"tool_message,omitempty"`
	ProposalMessage *string          `json:"proposal_message,omitempty"`
	OptionsMessage  *OptionsMessage  `json:"options_message,omitempty"`
	UpdateProgress  *ProgressMessage `json:"update_progress,omitempty"`
	WorkDone        bool             `json:"work_done"`
	EnablePrompt    bool             `json:"enable_prompt"`
}

type InitMessage struct {
	Modifications bool   `json:"modifications"`
	DarkTheme     bool   `json:"dark_theme"`
	ProjectName   string `json:"project_name"`
	ProjectDir    string `json:"project_dir"`
}

type OptionsMessage struct {
	Description string   `json:"description,omitempty"`
	Options     []string `json:"options,omitempty"`
}

type ProgressMessage struct {
	Percent int    `json:"percent"`
	Text    string `json:"text"`
}

// Shared messages

type PingMessage struct {
	Seq int64  `json:"seq"`
	TS  string `json:"ts"`
}

// Settings messages

type GlobalSettings struct {
	Version              int                `json:"version"`
	DefaultModifications bool               `json:"default_modifications"`
	DarkTheme            bool               `json:"dark_theme"`
	AllowedDirs          []string           `json:"allowed_dirs"`
	Providers            []ProviderSettings `json:"providers"`
}

type ProviderSettings struct {
	ProviderID string `json:"provider_id"`
	ModelID    string `json:"model_id"`
	APIKey     string `json:"api_key"`
	Default    bool   `json:"default"`
}

type ProjectSettings struct {
	Version          int           `json:"version"`
	Name             string        `json:"name"`
	buildProjectTool *ToolSettings `json:"build_project_tool"`
	runTestsTool     *ToolSettings `json:"run_tests_tool"`
}

type ToolSettings struct {
	command       string   `json:"command"`
	blockingFiles []string `json:"blocking_files"`
}
