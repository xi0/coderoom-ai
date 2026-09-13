package tools

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/xi0/coderoom-ai/internal/wire"
)

// presentOptionsTestOptions builds a ToolOptions suitable for exercising
// presentOptionsTool. Both channels are buffered with a single slot so a test
// can read the outgoing options message and queue the user's selection without
// the handler blocking. The tool does not touch the filesystem, so root is left
// nil.
func presentOptionsTestOptions() (*ToolOptions, chan wire.BackendMessage, chan int) {
	writeChannel := make(chan wire.BackendMessage, 1)
	optionChannel := make(chan int, 1)

	options := &ToolOptions{
		modifications: false,
		writeChannel:  writeChannel,
		optionChannel: optionChannel,
	}

	return options, writeChannel, optionChannel
}

func TestPresentOptionsValidSelection(t *testing.T) {
	options, writeChannel, optionChannel := presentOptionsTestOptions()
	optionChannel <- 1

	tool := presentOptionsTool()
	argsJSON := `{"options": ["apple", "banana", "cherry"], "prompt": "Pick a fruit"}`

	result, err := tool.call(argsJSON, options)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result != "banana" {
		t.Errorf("Expected result %q, got %q", "banana", result)
	}

	// Verify the options message was forwarded exactly once and correctly.
	select {
	case msg := <-writeChannel:
		if msg.OptionsMessage == nil {
			t.Fatal("Expected OptionsMessage to be set")
		}
		if msg.OptionsMessage.Description != "Pick a fruit" {
			t.Errorf("Expected description %q, got %q", "Pick a fruit", msg.OptionsMessage.Description)
		}
		want := []string{"apple", "banana", "cherry"}
		if len(msg.OptionsMessage.Options) != len(want) {
			t.Fatalf("Expected %d options, got %d", len(want), len(msg.OptionsMessage.Options))
		}
		for i := range want {
			if msg.OptionsMessage.Options[i] != want[i] {
				t.Errorf("Option %d: expected %q, got %q", i, want[i], msg.OptionsMessage.Options[i])
			}
		}
		if msg.ToolMessage != nil {
			t.Errorf("Expected ToolMessage to be nil, got %q", *msg.ToolMessage)
		}
	default:
		t.Error("Expected an options message to be sent to writeChannel")
	}
}

func TestPresentOptionsBoundarySelections(t *testing.T) {
	tests := []struct {
		name      string
		selection int
		want      string
	}{
		{name: "first option", selection: 0, want: "one"},
		{name: "last option", selection: 2, want: "three"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options, _, optionChannel := presentOptionsTestOptions()
			optionChannel <- tt.selection

			tool := presentOptionsTool()
			argsJSON := `{"options": ["one", "two", "three"]}`

			result, err := tool.call(argsJSON, options)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}
			if result != tt.want {
				t.Errorf("Expected result %q, got %q", tt.want, result)
			}
		})
	}
}

func TestPresentOptionsEmptyOptions(t *testing.T) {
	options, writeChannel, _ := presentOptionsTestOptions()

	tool := presentOptionsTool()
	_, err := tool.call(`{"options": []}`, options)

	if err == nil {
		t.Fatal("Expected error for empty options list, got nil")
	}
	if !strings.Contains(err.Error(), "options list cannot be empty") {
		t.Errorf("Expected error to contain 'options list cannot be empty', got: %v", err)
	}

	// No message should be forwarded for an empty options list.
	if len(writeChannel) != 0 {
		t.Error("Expected no options message to be sent for an empty options list")
	}
}

func TestPresentOptionsInvalidJSON(t *testing.T) {
	options, writeChannel, _ := presentOptionsTestOptions()

	tool := presentOptionsTool()
	_, err := tool.call(`{invalid json}`, options)

	if err == nil {
		t.Fatal("Expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "invalid arguments") {
		t.Errorf("Expected error to contain 'invalid arguments', got: %v", err)
	}

	// No message should be forwarded when parsing fails.
	if len(writeChannel) != 0 {
		t.Error("Expected no options message to be sent for invalid JSON")
	}
}

func TestPresentOptionsSelectionOutOfRange(t *testing.T) {
	tests := []struct {
		name      string
		selection int
	}{
		{name: "negative", selection: -1},
		{name: "equal to length", selection: 3},
		{name: "greater than length", selection: 42},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options, writeChannel, optionChannel := presentOptionsTestOptions()
			optionChannel <- tt.selection

			tool := presentOptionsTool()
			argsJSON := `{"options": ["a", "b", "c"]}`

			_, err := tool.call(argsJSON, options)
			if err == nil {
				t.Fatal("Expected error for out-of-range selection, got nil")
			}
			if !strings.Contains(err.Error(), "out of range") {
				t.Errorf("Expected error to contain 'out of range', got: %v", err)
			}

			// The options message is still forwarded before the selection is
			// validated, since the user must see the options to choose.
			select {
			case msg := <-writeChannel:
				if msg.OptionsMessage == nil {
					t.Fatal("Expected OptionsMessage to be set")
				}
			default:
				t.Error("Expected an options message to be sent to writeChannel")
			}
		})
	}
}

func TestPresentOptionsMissingPrompt(t *testing.T) {
	options, writeChannel, optionChannel := presentOptionsTestOptions()
	optionChannel <- 0

	tool := presentOptionsTool()
	result, err := tool.call(`{"options": ["yes", "no"]}`, options)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result != "yes" {
		t.Errorf("Expected result %q, got %q", "yes", result)
	}

	select {
	case msg := <-writeChannel:
		if msg.OptionsMessage == nil {
			t.Fatal("Expected OptionsMessage to be set")
		}
		if msg.OptionsMessage.Description != "" {
			t.Errorf("Expected empty description, got %q", msg.OptionsMessage.Description)
		}
	default:
		t.Error("Expected an options message to be sent to writeChannel")
	}
}

func TestPresentOptionsPreservesValues(t *testing.T) {
	options, writeChannel, optionChannel := presentOptionsTestOptions()
	optionChannel <- 1

	prompt := "Choose \u4e2d\u6587 \u00e9\u00e8\u00ea"
	values := []string{
		"# Markdown *option*",
		"Unicode: \u4e2d\u6587 \u00e9\u00e8\u00ea",
		"with \"quotes\" and \\backslash",
	}

	argsJSON, err := json.Marshal(PresentOptionsArgs{Options: values, Prompt: prompt})
	if err != nil {
		t.Fatalf("json.Marshal(): %v", err)
	}

	tool := presentOptionsTool()
	result, err := tool.call(string(argsJSON), options)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result != values[1] {
		t.Errorf("Expected result %q, got %q", values[1], result)
	}

	select {
	case msg := <-writeChannel:
		if msg.OptionsMessage == nil {
			t.Fatal("Expected OptionsMessage to be set")
		}
		if msg.OptionsMessage.Description != prompt {
			t.Errorf("Expected description %q, got %q", prompt, msg.OptionsMessage.Description)
		}
		for i := range values {
			if msg.OptionsMessage.Options[i] != values[i] {
				t.Errorf("Option %d: expected %q, got %q", i, values[i], msg.OptionsMessage.Options[i])
			}
		}
	default:
		t.Error("Expected an options message to be sent to writeChannel")
	}
}

// TestPresentOptionsSendsMessageBeforeSelection asserts that the tool forwards
// the options to the frontend before blocking while awaiting the user's
// selection. It uses unbuffered channels to make the ordering observable.
func TestPresentOptionsSendsMessageBeforeSelection(t *testing.T) {
	writeChannel := make(chan wire.BackendMessage)
	optionChannel := make(chan int)

	options := &ToolOptions{
		writeChannel:  writeChannel,
		optionChannel: optionChannel,
	}

	type callResult struct {
		out string
		err error
	}
	resultChannel := make(chan callResult, 1)

	tool := presentOptionsTool()
	go func() {
		out, err := tool.call(`{"options": ["red", "green"], "prompt": "Pick a color"}`, options)
		resultChannel <- callResult{out: out, err: err}
	}()

	// The handler must send the options before it blocks on the selection.
	select {
	case msg := <-writeChannel:
		if msg.OptionsMessage == nil {
			t.Fatal("Expected OptionsMessage to be set")
		}
		if msg.OptionsMessage.Description != "Pick a color" {
			t.Errorf("Expected description %q, got %q", "Pick a color", msg.OptionsMessage.Description)
		}
		if len(msg.OptionsMessage.Options) != 2 {
			t.Errorf("Expected 2 options, got %d", len(msg.OptionsMessage.Options))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for the options message")
	}

	// Provide the user's selection and wait for the handler to return.
	optionChannel <- 1

	select {
	case res := <-resultChannel:
		if res.err != nil {
			t.Fatalf("Expected no error, got: %v", res.err)
		}
		if res.out != "green" {
			t.Errorf("Expected result %q, got %q", "green", res.out)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for the tool to return")
	}
}
