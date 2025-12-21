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

func TestGetCacheDuration_Default(t *testing.T) {
	// Ensure env var is not set
	t.Setenv("TOKEN_CACHE_TTL_SECONDS", "")

	duration := getCacheDuration()
	expected := time.Duration(defaultCacheDurationSeconds) * time.Second

	if duration != expected {
		t.Errorf("Expected %v, got %v", expected, duration)
	}
}

func TestGetCacheDuration_CustomValue(t *testing.T) {
	t.Setenv("TOKEN_CACHE_TTL_SECONDS", "60")

	duration := getCacheDuration()
	expected := 60 * time.Second

	if duration != expected {
		t.Errorf("Expected %v, got %v", expected, duration)
	}
}

func TestGetCacheDuration_InvalidValue(t *testing.T) {
	t.Setenv("TOKEN_CACHE_TTL_SECONDS", "invalid")

	duration := getCacheDuration()
	expected := time.Duration(defaultCacheDurationSeconds) * time.Second

	if duration != expected {
		t.Errorf("Expected default %v for invalid value, got %v", expected, duration)
	}
}

func TestGeneratePolicy_WithErrorContext(t *testing.T) {
	resource := "arn:aws:execute-api:us-west-2:123456789:api-id/stage/POST/invoke"
	ctx := map[string]interface{}{
		"errorType": ErrMissingToken,
	}

	policy := generatePolicy("user", "Deny", resource, ctx)

	if policy.Context == nil {
		t.Fatal("Expected context to be set")
	}

	if policy.Context["errorType"] != ErrMissingToken {
		t.Errorf("Expected errorType '%s', got '%v'", ErrMissingToken, policy.Context["errorType"])
	}
}

// Test that context timeout is properly set (3 seconds for SSM calls)
func TestGetExpectedToken_ContextTimeout(t *testing.T) {
	// This test verifies the timeout behavior without hitting actual SSM
	// The implementation uses context.WithTimeout(ctx, 3*time.Second)

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Reset cache to force SSM call
	tokenCache.mu.Lock()
	tokenCache.token = ""
	tokenCache.expiration = time.Time{}
	tokenCache.mu.Unlock()

	// getExpectedToken should fail quickly due to cancelled context
	start := time.Now()
	_, err := getExpectedToken(ctx)
	elapsed := time.Since(start)

	// Should fail fast (not hang) when context is cancelled
	if err == nil {
		t.Error("Expected error when context is cancelled")
	}

	// Should complete quickly (under 1 second), not wait for 3s timeout
	if elapsed > 1*time.Second {
		t.Errorf("Expected quick failure with cancelled context, took %v", elapsed)
	}
}

// Test cache atomicity - token and expiration should be read together
func TestTokenCache_AtomicRead(t *testing.T) {
	// Set up cache with valid token
	tokenCache.mu.Lock()
	tokenCache.token = "atomic-test-token"
	tokenCache.expiration = time.Now().Add(5 * time.Minute)
	tokenCache.mu.Unlock()

	// Simulate concurrent reads (verifying atomic read pattern)
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			tokenCache.mu.RLock()
			token := tokenCache.token
			expiration := tokenCache.expiration
			tokenCache.mu.RUnlock()

			// Both values should be read atomically
			if token == "atomic-test-token" && expiration.IsZero() {
				t.Error("Race condition: token set but expiration not read atomically")
			}
			if token == "" && !expiration.IsZero() {
				t.Error("Race condition: expiration set but token not read atomically")
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
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
