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
		{
			name: "valid reminder request",
			req: Req{
				Text: "Remind me to call mom",
				Mode: "reminder",
			},
			wantErr: false,
		},
		{
			name: "valid event request",
			req: Req{
				Text: "Schedule meeting tomorrow at 3pm",
				Mode: "event",
			},
			wantErr: false,
		},
		{
			name: "valid research request",
			req: Req{
				Text: "Research quantum computing",
				Mode: "research",
			},
			wantErr: false,
		},
		{
			name: "valid deepthink request",
			req: Req{
				Text: "Analyze the implications of AI",
				Mode: "deepthink",
			},
			wantErr: false,
		},
		{
			name: "negative thinking tokens",
			req: Req{
				Text:           "Test",
				Mode:           "note",
				ThinkingTokens: -1,
			},
			wantErr: true,
		},
		{
			name: "excessive thinking tokens",
			req: Req{
				Text:           "Test",
				Mode:           "note",
				ThinkingTokens: 100000,
			},
			wantErr: true,
		},
		{
			name: "excessive max tokens",
			req: Req{
				Text:      "Test",
				Mode:      "note",
				MaxTokens: 10000,
			},
			wantErr: true,
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

func TestApiResponse(t *testing.T) {
	resp := apiResponse(200, map[string]string{"test": "value"})

	if resp.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", resp.StatusCode)
	}

	// Only Content-Type header is set by Lambda
	// CORS headers are handled by API Gateway's defaultCorsPreflightOptions
	if _, exists := resp.Headers["Content-Type"]; !exists {
		t.Errorf("Expected header Content-Type to exist")
	}
	if resp.Headers["Content-Type"] != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", resp.Headers["Content-Type"])
	}

	// Verify CORS headers are NOT set (handled by API Gateway)
	corsHeaders := []string{
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Headers",
		"Access-Control-Allow-Methods",
	}
	for _, header := range corsHeaders {
		if _, exists := resp.Headers[header]; exists {
			t.Errorf("CORS header %s should not be set by Lambda (handled by API Gateway)", header)
		}
	}

	// Verify body is JSON
	var bodyMap map[string]string
	err := json.Unmarshal([]byte(resp.Body), &bodyMap)
	if err != nil {
		t.Errorf("Expected valid JSON body, got error: %v", err)
	}
	if bodyMap["test"] != "value" {
		t.Errorf("Expected body test='value', got '%s'", bodyMap["test"])
	}
}

func TestApiResponse_NilBody(t *testing.T) {
	resp := apiResponse(204, nil)

	if resp.StatusCode != 204 {
		t.Errorf("Expected status code 204, got %d", resp.StatusCode)
	}

	if resp.Body != "" {
		t.Errorf("Expected empty body for nil input, got '%s'", resp.Body)
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
			name:    "h2 markdown header",
			content: "## Project Update\nDetails",
			mode:    "note",
			want:    "Project Update",
		},
		{
			name:    "long title truncation",
			content: "This is a very long title that should be truncated because it exceeds the fifty character limit",
			mode:    "note",
			want:    "This is a very long title that should be trunca...",
		},
		{
			name:    "fallback title for note",
			content: "",
			mode:    "note",
			want:    "Wrist Agent Note",
		},
		{
			name:    "fallback title for reminder",
			content: "",
			mode:    "reminder",
			want:    "Wrist Agent Reminder",
		},
		{
			name:    "skip json content",
			content: "{\"key\": \"value\"}\nActual Title",
			mode:    "note",
			want:    "Actual Title",
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

func TestRequestParsing_WithThinkingTokens(t *testing.T) {
	testJSON := `{"text": "Deep analysis request", "mode": "deepthink", "maxTokens": 2000, "thinkingTokens": 10000}`

	var req Req
	err := json.Unmarshal([]byte(testJSON), &req)
	if err != nil {
		t.Fatalf("Failed to unmarshal test JSON: %v", err)
	}

	if req.ThinkingTokens != 10000 {
		t.Errorf("Expected thinkingTokens 10000, got %d", req.ThinkingTokens)
	}
}

func TestBuildSystemPrompt(t *testing.T) {
	modes := []string{"note", "reminder", "event", "research", "deepthink"}

	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			prompt := buildSystemPrompt(mode)
			if prompt == "" {
				t.Errorf("buildSystemPrompt(%s) returned empty string", mode)
			}
			// Should contain the base JSON format
			if !contains(prompt, "markdown") {
				t.Errorf("buildSystemPrompt(%s) missing 'markdown' field", mode)
			}
			if !contains(prompt, "action") {
				t.Errorf("buildSystemPrompt(%s) missing 'action' field", mode)
			}
		})
	}
}

func TestGetEnv(t *testing.T) {
	// Test default value
	result := getEnv("NON_EXISTENT_VAR_12345", "default")
	if result != "default" {
		t.Errorf("Expected 'default', got '%s'", result)
	}
}

// Mock test for handler with API Gateway event
func TestHandlerEventStructure(t *testing.T) {
	// This test verifies the event structure is correct for API Gateway
	event := events.APIGatewayProxyRequest{
		HTTPMethod: "POST",
		Path:       "/invoke",
		Headers: map[string]string{
			"Content-Type":   "application/json",
			"X-Client-Token": "test-token",
		},
		Body: `{"text": "Test message", "mode": "note"}`,
	}

	// Verify we can access all expected fields
	if event.HTTPMethod != "POST" {
		t.Errorf("Expected POST method, got %s", event.HTTPMethod)
	}
	if event.Path != "/invoke" {
		t.Errorf("Expected /invoke path, got %s", event.Path)
	}
	if event.Headers["X-Client-Token"] != "test-token" {
		t.Errorf("Expected test-token, got %s", event.Headers["X-Client-Token"])
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
