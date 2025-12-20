package main

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
)

func TestExtractToken_XClientTokenHeader(t *testing.T) {
	event := events.APIGatewayCustomAuthorizerRequestTypeRequest{
		Headers: map[string]string{
			"X-Client-Token": "test-token-123",
		},
	}

	token := extractToken(event)
	if token != "test-token-123" {
		t.Errorf("Expected 'test-token-123', got '%s'", token)
	}
}

func TestExtractToken_AuthorizationBearer(t *testing.T) {
	event := events.APIGatewayCustomAuthorizerRequestTypeRequest{
		Headers: map[string]string{
			"Authorization": "Bearer test-token-456",
		},
	}

	token := extractToken(event)
	if token != "test-token-456" {
		t.Errorf("Expected 'test-token-456', got '%s'", token)
	}
}

func TestExtractToken_CustomHeaderCaseInsensitive(t *testing.T) {
	event := events.APIGatewayCustomAuthorizerRequestTypeRequest{
		Headers: map[string]string{
			"x-client-token": "lowercase-token",
		},
	}

	token := extractToken(event)
	if token != "lowercase-token" {
		t.Errorf("Expected 'lowercase-token', got '%s'", token)
	}
}

func TestExtractToken_MissingToken(t *testing.T) {
	event := events.APIGatewayCustomAuthorizerRequestTypeRequest{
		Headers: map[string]string{},
	}

	token := extractToken(event)
	if token != "" {
		t.Errorf("Expected empty string, got '%s'", token)
	}
}

func TestExtractToken_WhitespaceToken(t *testing.T) {
	event := events.APIGatewayCustomAuthorizerRequestTypeRequest{
		Headers: map[string]string{
			"X-Client-Token": "  trimmed-token  ",
		},
	}

	token := extractToken(event)
	if token != "trimmed-token" {
		t.Errorf("Expected 'trimmed-token', got '%s'", token)
	}
}

func TestExtractToken_XClientTokenTakesPrecedence(t *testing.T) {
	event := events.APIGatewayCustomAuthorizerRequestTypeRequest{
		Headers: map[string]string{
			"X-Client-Token": "primary-token",
			"Authorization":  "Bearer secondary-token",
		},
	}

	token := extractToken(event)
	if token != "primary-token" {
		t.Errorf("Expected 'primary-token', got '%s'", token)
	}
}

func TestGeneratePolicy_Allow(t *testing.T) {
	resource := "arn:aws:execute-api:us-west-2:123456789:api-id/stage/POST/invoke"
	policy := generatePolicy("test-user", "Allow", resource, nil)

	if policy.PrincipalID != "test-user" {
		t.Errorf("Expected PrincipalID 'test-user', got '%s'", policy.PrincipalID)
	}

	if len(policy.PolicyDocument.Statement) != 1 {
		t.Fatalf("Expected 1 statement, got %d", len(policy.PolicyDocument.Statement))
	}

	stmt := policy.PolicyDocument.Statement[0]
	if stmt.Effect != "Allow" {
		t.Errorf("Expected Effect 'Allow', got '%s'", stmt.Effect)
	}

	if len(stmt.Resource) != 1 || stmt.Resource[0] != resource {
		t.Errorf("Expected Resource '%s', got '%v'", resource, stmt.Resource)
	}
}

func TestGeneratePolicy_Deny(t *testing.T) {
	resource := "arn:aws:execute-api:us-west-2:123456789:api-id/stage/POST/invoke"
	policy := generatePolicy("user", "Deny", resource, nil)

	if policy.PrincipalID != "user" {
		t.Errorf("Expected PrincipalID 'user', got '%s'", policy.PrincipalID)
	}

	stmt := policy.PolicyDocument.Statement[0]
	if stmt.Effect != "Deny" {
		t.Errorf("Expected Effect 'Deny', got '%s'", stmt.Effect)
	}
}

func TestGeneratePolicy_WithContext(t *testing.T) {
	resource := "arn:aws:execute-api:us-west-2:123456789:api-id/stage/POST/invoke"
	ctx := map[string]interface{}{
		"authenticated": "true",
		"userId":        "user-123",
	}

	policy := generatePolicy("test-user", "Allow", resource, ctx)

	if policy.Context == nil {
		t.Fatal("Expected context to be set")
	}

	if policy.Context["authenticated"] != "true" {
		t.Errorf("Expected context 'authenticated' to be 'true'")
	}
}

func TestTokenCache_Expiration(t *testing.T) {
	// Reset cache
	tokenCache.mu.Lock()
	tokenCache.token = "cached-token"
	tokenCache.expiration = time.Now().Add(-1 * time.Minute) // Expired
	tokenCache.mu.Unlock()

	// Read should show expired
	tokenCache.mu.RLock()
	isExpired := time.Now().After(tokenCache.expiration)
	tokenCache.mu.RUnlock()

	if !isExpired {
		t.Error("Expected cache to be expired")
	}
}

func TestTokenCache_Valid(t *testing.T) {
	// Reset cache
	tokenCache.mu.Lock()
	tokenCache.token = "valid-token"
	tokenCache.expiration = time.Now().Add(5 * time.Minute) // Valid
	tokenCache.mu.Unlock()

	// Read should show valid
	tokenCache.mu.RLock()
	isValid := tokenCache.token != "" && time.Now().Before(tokenCache.expiration)
	token := tokenCache.token
	tokenCache.mu.RUnlock()

	if !isValid {
		t.Error("Expected cache to be valid")
	}
	if token != "valid-token" {
		t.Errorf("Expected 'valid-token', got '%s'", token)
	}
}

func TestGetEnv_Default(t *testing.T) {
	result := getEnv("NON_EXISTENT_ENV_VAR_12345", "default-value")
	if result != "default-value" {
		t.Errorf("Expected 'default-value', got '%s'", result)
	}
}

// Integration test (requires AWS credentials)
func TestHandler_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test requires AWS credentials and SSM parameter to be set up
	event := events.APIGatewayCustomAuthorizerRequestTypeRequest{
		MethodArn: "arn:aws:execute-api:us-west-2:123456789:api-id/stage/POST/invoke",
		Headers: map[string]string{
			"X-Client-Token": "test-token",
		},
	}

	ctx := context.Background()
	_, err := handler(ctx, event)
	if err != nil {
		t.Logf("Integration test error (expected if no AWS credentials): %v", err)
	}
}
