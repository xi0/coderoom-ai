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
	Pong            *PingMessage `json:"pong,omitempty"`
	SystemMessage   *string      `json:"system_message,omitempty"`
	ToolMessage     *string      `json:"tool_message,omitempty"`
	ProposalMessage *string      `json:"proposal_message,omitempty"`
	OptionsMessage  *[]string    `json:"options_message,omitempty"`
}

// Shared messages

type PingMessage struct {
	Seq int64  `json:"seq"`
	TS  string `json:"ts"`
}
