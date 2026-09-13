package tools

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/xi0/coderoom-ai/internal/wire"
)

// reportProgressTestOptions builds a ToolOptions suitable for exercising
// reportProgressTool. The writeChannel is buffered with a single slot so a test
// can read the outgoing progress message without the handler blocking. The tool
// does not touch the filesystem, so root is left nil.
func reportProgressTestOptions() (*ToolOptions, chan wire.BackendMessage) {
	writeChannel := make(chan wire.BackendMessage, 1)

	options := &ToolOptions{
		writeChannel: writeChannel,
	}

	return options, writeChannel
}

func TestReportProgressValidWithPercent(t *testing.T) {
	options, writeChannel := reportProgressTestOptions()

	tool := reportProgressTool()
	result, err := tool.call(`{"description": "Compiling", "percent": 42}`, options)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result != "Progress reported." {
		t.Errorf("Expected result %q, got %q", "Progress reported.", result)
	}

	select {
	case msg := <-writeChannel:
		if msg.UpdateProgress == nil {
			t.Fatal("Expected UpdateProgress to be set")
		}
		if msg.UpdateProgress.Percent != 42 {
			t.Errorf("Expected percent 42, got %d", msg.UpdateProgress.Percent)
		}
		if msg.UpdateProgress.Text != "Compiling" {
			t.Errorf("Expected text %q, got %q", "Compiling", msg.UpdateProgress.Text)
		}
	default:
		t.Error("Expected a progress message to be sent to writeChannel")
	}
}

func TestReportProgressMissingPercent(t *testing.T) {
	options, writeChannel := reportProgressTestOptions()

	tool := reportProgressTool()
	result, err := tool.call(`{"description": "Running tests"}`, options)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result != "Progress reported." {
		t.Errorf("Expected result %q, got %q", "Progress reported.", result)
	}

	select {
	case msg := <-writeChannel:
		if msg.UpdateProgress == nil {
			t.Fatal("Expected UpdateProgress to be set")
		}
		if msg.UpdateProgress.Percent != 0 {
			t.Errorf("Expected percent 0 when omitted, got %d", msg.UpdateProgress.Percent)
		}
		if msg.UpdateProgress.Text != "Running tests" {
			t.Errorf("Expected text %q, got %q", "Running tests", msg.UpdateProgress.Text)
		}
	default:
		t.Error("Expected a progress message to be sent to writeChannel")
	}
}

func TestReportProgressBoundaryPercent(t *testing.T) {
	tests := []struct {
		name    string
		percent int
		want    int
	}{
		{name: "zero", percent: 0, want: 0},
		{name: "hundred", percent: 100, want: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options, writeChannel := reportProgressTestOptions()

			argsJSON, err := json.Marshal(ReportProgressArgs{Description: "boundary", Percent: &tt.percent})
			if err != nil {
				t.Fatalf("json.Marshal(): %v", err)
			}

			tool := reportProgressTool()
			if _, err := tool.call(string(argsJSON), options); err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			select {
			case msg := <-writeChannel:
				if msg.UpdateProgress == nil {
					t.Fatal("Expected UpdateProgress to be set")
				}
				if msg.UpdateProgress.Percent != tt.want {
					t.Errorf("Expected percent %d, got %d", tt.want, msg.UpdateProgress.Percent)
				}
			default:
				t.Error("Expected a progress message to be sent to writeChannel")
			}
		})
	}
}

func TestReportProgressClampsPercent(t *testing.T) {
	tests := []struct {
		name    string
		percent int
		want    int
	}{
		{name: "negative clamped to zero", percent: -1, want: 0},
		{name: "large negative clamped to zero", percent: -999, want: 0},
		{name: "above hundred clamped", percent: 101, want: 100},
		{name: "very large clamped to hundred", percent: 999, want: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options, writeChannel := reportProgressTestOptions()

			argsJSON, err := json.Marshal(ReportProgressArgs{Description: "clamp", Percent: &tt.percent})
			if err != nil {
				t.Fatalf("json.Marshal(): %v", err)
			}

			tool := reportProgressTool()
			if _, err := tool.call(string(argsJSON), options); err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			select {
			case msg := <-writeChannel:
				if msg.UpdateProgress == nil {
					t.Fatal("Expected UpdateProgress to be set")
				}
				if msg.UpdateProgress.Percent != tt.want {
					t.Errorf("Expected clamped percent %d, got %d", tt.want, msg.UpdateProgress.Percent)
				}
			default:
				t.Error("Expected a progress message to be sent to writeChannel")
			}
		})
	}
}

func TestReportProgressInvalidJSON(t *testing.T) {
	options, writeChannel := reportProgressTestOptions()

	tool := reportProgressTool()
	_, err := tool.call(`{invalid json}`, options)

	if err == nil {
		t.Fatal("Expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "invalid arguments") {
		t.Errorf("Expected error to contain 'invalid arguments', got: %v", err)
	}

	if len(writeChannel) != 0 {
		t.Error("Expected no progress message to be sent for invalid JSON")
	}
}

func TestReportProgressEmptyDescription(t *testing.T) {
	options, writeChannel := reportProgressTestOptions()

	tool := reportProgressTool()
	result, err := tool.call(`{"description": ""}`, options)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result != "Progress reported." {
		t.Errorf("Expected result %q, got %q", "Progress reported.", result)
	}

	select {
	case msg := <-writeChannel:
		if msg.UpdateProgress == nil {
			t.Fatal("Expected UpdateProgress to be set")
		}
		if msg.UpdateProgress.Text != "" {
			t.Errorf("Expected empty text, got %q", msg.UpdateProgress.Text)
		}
	default:
		t.Error("Expected a progress message to be sent to writeChannel")
	}
}

func TestReportProgressPreservesDescription(t *testing.T) {
	options, writeChannel := reportProgressTestOptions()

	description := "Build \u4e2d\u6587 \u00e9\u00e8\u00ea with \"quotes\" and \\backslash"
	argsJSON, err := json.Marshal(ReportProgressArgs{Description: description})
	if err != nil {
		t.Fatalf("json.Marshal(): %v", err)
	}

	tool := reportProgressTool()
	if _, err := tool.call(string(argsJSON), options); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	select {
	case msg := <-writeChannel:
		if msg.UpdateProgress == nil {
			t.Fatal("Expected UpdateProgress to be set")
		}
		if msg.UpdateProgress.Text != description {
			t.Errorf("Expected text %q, got %q", description, msg.UpdateProgress.Text)
		}
	default:
		t.Error("Expected a progress message to be sent to writeChannel")
	}
}
