package tools

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/xi0/coderoom-ai/internal/wire"
)

// proposePlanTestOptions builds a ToolOptions suitable for exercising
// proposePlanTool. Both channels are buffered with a single slot so a test can
// read the proposal message and queue the user's confirmation without the
// handler blocking. The tool does not touch the filesystem, so root is left nil.
func proposePlanTestOptions() (*ToolOptions, chan wire.BackendMessage, chan bool) {
	writeChannel := make(chan wire.BackendMessage, 1)
	confirmationChannel := make(chan bool, 1)

	options := &ToolOptions{
		Modifications:       false,
		WriteChannel:        writeChannel,
		ConfirmationChannel: confirmationChannel,
	}

	return options, writeChannel, confirmationChannel
}

func TestProposePlanConfirmed(t *testing.T) {
	options, writeChannel, confirmationChannel := proposePlanTestOptions()
	confirmationChannel <- true

	tool := proposePlanTool()
	plan := "Step 1: do a thing\nStep 2: do another thing"
	argsJSON := `{"plan": "Step 1: do a thing\nStep 2: do another thing"}`

	result, err := tool.call(argsJSON, options)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result != "true" {
		t.Errorf("Expected result %q, got %q", "true", result)
	}

	// Verify the proposal was forwarded exactly once and correctly.
	select {
	case msg := <-writeChannel:
		if msg.ProposalMessage == nil {
			t.Fatal("Expected ProposalMessage to be set")
		}
		if *msg.ProposalMessage != plan {
			t.Errorf("Expected proposal %q, got %q", plan, *msg.ProposalMessage)
		}
		if msg.ToolMessage != nil {
			t.Errorf("Expected ToolMessage to be nil, got %q", *msg.ToolMessage)
		}
	default:
		t.Error("Expected a proposal message to be sent to writeChannel")
	}
}

func TestProposePlanRejected(t *testing.T) {
	options, writeChannel, confirmationChannel := proposePlanTestOptions()
	confirmationChannel <- false

	tool := proposePlanTool()
	plan := "A plan the user will reject"
	argsJSON := `{"plan": "A plan the user will reject"}`

	result, err := tool.call(argsJSON, options)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result != "false" {
		t.Errorf("Expected result %q, got %q", "false", result)
	}

	// The proposal must still be sent even when the user rejects it.
	select {
	case msg := <-writeChannel:
		if msg.ProposalMessage == nil {
			t.Fatal("Expected ProposalMessage to be set")
		}
		if *msg.ProposalMessage != plan {
			t.Errorf("Expected proposal %q, got %q", plan, *msg.ProposalMessage)
		}
	default:
		t.Error("Expected a proposal message to be sent to writeChannel")
	}
}

func TestProposePlanInvalidJSON(t *testing.T) {
	options, writeChannel, _ := proposePlanTestOptions()

	tool := proposePlanTool()
	_, err := tool.call(`{invalid json}`, options)

	if err == nil {
		t.Fatal("Expected error for invalid JSON, got nil")
	}

	if !strings.Contains(err.Error(), "invalid arguments") {
		t.Errorf("Expected error to contain 'invalid arguments', got: %v", err)
	}

	// No proposal should be forwarded when parsing fails.
	if len(writeChannel) != 0 {
		t.Error("Expected no proposal message to be sent for invalid JSON")
	}
}

func TestProposePlanMissingPlanField(t *testing.T) {
	options, writeChannel, confirmationChannel := proposePlanTestOptions()
	confirmationChannel <- true

	tool := proposePlanTool()
	result, err := tool.call(`{}`, options)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result != "true" {
		t.Errorf("Expected result %q, got %q", "true", result)
	}

	// A missing plan defaults to the empty string and is still forwarded.
	select {
	case msg := <-writeChannel:
		if msg.ProposalMessage == nil {
			t.Fatal("Expected ProposalMessage to be set")
		}
		if *msg.ProposalMessage != "" {
			t.Errorf("Expected empty proposal, got %q", *msg.ProposalMessage)
		}
	default:
		t.Error("Expected a proposal message to be sent to writeChannel")
	}
}

func TestProposePlanPreservesMarkdown(t *testing.T) {
	options, writeChannel, confirmationChannel := proposePlanTestOptions()
	confirmationChannel <- true

	plan := "# Title\n\n- item 1\n- item 2\n\n```go\nfmt.Println(\"hi\")\n```\nUnicode: \u4e2d\u6587 \u00e9\u00e8\u00ea\n\ttabbed"

	argsJSON, err := json.Marshal(ProposePlanArgs{Plan: plan})
	if err != nil {
		t.Fatalf("json.Marshal(): %v", err)
	}

	tool := proposePlanTool()
	result, err := tool.call(string(argsJSON), options)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result != "true" {
		t.Errorf("Expected result %q, got %q", "true", result)
	}

	select {
	case msg := <-writeChannel:
		if msg.ProposalMessage == nil {
			t.Fatal("Expected ProposalMessage to be set")
		}
		if *msg.ProposalMessage != plan {
			t.Errorf("Expected proposal %q, got %q", plan, *msg.ProposalMessage)
		}
	default:
		t.Error("Expected a proposal message to be sent to writeChannel")
	}
}

// TestProposePlanSendsProposalBeforeConfirmation asserts that the tool forwards
// the proposal to the frontend before blocking while awaiting the user's
// decision. It uses unbuffered channels to make the ordering observable.
func TestProposePlanSendsProposalBeforeConfirmation(t *testing.T) {
	writeChannel := make(chan wire.BackendMessage)
	confirmationChannel := make(chan bool)

	options := &ToolOptions{
		WriteChannel:        writeChannel,
		ConfirmationChannel: confirmationChannel,
	}

	type callResult struct {
		out string
		err error
	}
	resultChannel := make(chan callResult, 1)

	tool := proposePlanTool()
	go func() {
		out, err := tool.call(`{"plan": "my plan"}`, options)
		resultChannel <- callResult{out: out, err: err}
	}()

	// The handler must send the proposal before it blocks on confirmation.
	select {
	case msg := <-writeChannel:
		if msg.ProposalMessage == nil {
			t.Fatal("Expected ProposalMessage to be set")
		}
		if *msg.ProposalMessage != "my plan" {
			t.Errorf("Expected proposal %q, got %q", "my plan", *msg.ProposalMessage)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for the proposal message")
	}

	// Provide the user's decision and wait for the handler to return.
	confirmationChannel <- true

	select {
	case res := <-resultChannel:
		if res.err != nil {
			t.Fatalf("Expected no error, got: %v", res.err)
		}
		if res.out != "true" {
			t.Errorf("Expected result %q, got %q", "true", res.out)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for the tool to return")
	}
}
