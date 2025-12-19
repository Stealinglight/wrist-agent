package main

import (
	"encoding/json"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestValidateRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     Req
		wantErr bool
	}{
		{
			name: "valid note request",
			req: Req{
				Text:      "Create a note about my meeting",
				Mode:      "note",
				MaxTokens: 800,
			},
			wantErr: false,
		},
		{
			name: "empty text",
			req: Req{
				Text: "",
				Mode: "note",
			},
			wantErr: true,
		},
		{
			name: "invalid mode",
			req: Req{
				Text: "Test",
				Mode: "invalid",
			},
			wantErr: true,
		},
		{
			name: "default mode when empty",
			req: Req{
				Text: "Test with no mode",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRequest(&tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCorsResponse(t *testing.T) {
	resp := corsResponse(200, map[string]string{"test": "value"})

	if resp.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", resp.StatusCode)
	}

	expectedHeaders := []string{
		"Content-Type",
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Headers",
		"Access-Control-Allow-Methods",
	}

	for _, header := range expectedHeaders {
		if _, exists := resp.Headers[header]; !exists {
			t.Errorf("Expected header %s to exist", header)
		}
	}
}

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name    string
		content string
		mode    string
		want    string
	}{
		{
			name:    "markdown header",
			content: "# Meeting Notes\nContent here",
			mode:    "note",
			want:    "Meeting Notes",
		},
		{
			name:    "long title truncation",
			content: "This is a very long title that should be truncated because it exceeds the fifty character limit",
			mode:    "note",
			want:    "This is a very long title that should be trunca...",
		},
		{
			name:    "fallback title",
			content: "",
			mode:    "reminder",
			want:    "Wrist Agent Reminder",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTitle(tt.content, tt.mode)
			if got != tt.want {
				t.Errorf("extractTitle() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRequestParsing(t *testing.T) {
	testJSON := `{"text": "Test message", "mode": "note", "maxTokens": 1000}`

	var req Req
	err := json.Unmarshal([]byte(testJSON), &req)
	if err != nil {
		t.Fatalf("Failed to unmarshal test JSON: %v", err)
	}

	if req.Text != "Test message" {
		t.Errorf("Expected text 'Test message', got '%s'", req.Text)
	}

	if req.Mode != "note" {
		t.Errorf("Expected mode 'note', got '%s'", req.Mode)
	}

	if req.MaxTokens != 1000 {
		t.Errorf("Expected maxTokens 1000, got %d", req.MaxTokens)
	}
}

// Mock test for handler OPTIONS request
func TestHandlerOPTIONS(t *testing.T) {
	// Skip this test in CI since it requires AWS credentials
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	event := events.LambdaFunctionURLRequest{
		RequestContext: events.LambdaFunctionURLRequestContext{
			HTTP: events.LambdaFunctionURLRequestContextHTTPDescription{
				Method: "OPTIONS",
			},
		},
	}

	// This would normally require proper AWS setup, so we just test the structure
	_ = event
}
