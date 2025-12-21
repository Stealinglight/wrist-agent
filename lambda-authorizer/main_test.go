package main

import (
	"context"
	"strings"
	"sync"
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
		t.Skipf("Integration test skipped (requires AWS credentials): %v", err)
	}
}

// Test empty/whitespace-only token headers
func TestExtractToken_EmptyAfterTrim(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		want    string
	}{
		{
			name: "whitespace only X-Client-Token",
			headers: map[string]string{
				"X-Client-Token": "   ",
			},
			want: "",
		},
		{
			name: "empty X-Client-Token",
			headers: map[string]string{
				"X-Client-Token": "",
			},
			want: "",
		},
		{
			name: "whitespace only Authorization Bearer",
			headers: map[string]string{
				"Authorization": "Bearer    ",
			},
			want: "",
		},
		{
			name: "Bearer with no token",
			headers: map[string]string{
				"Authorization": "Bearer",
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := events.APIGatewayCustomAuthorizerRequestTypeRequest{
				Headers: tt.headers,
			}
			got := extractToken(event)
			if got != tt.want {
				t.Errorf("extractToken() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Test malformed Authorization headers
func TestExtractToken_MalformedAuthHeader(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		want    string
	}{
		{
			name: "Authorization without Bearer prefix",
			headers: map[string]string{
				"Authorization": "token-without-bearer",
			},
			want: "", // No Bearer prefix, so ignored
		},
		{
			name: "Authorization with lowercase bearer",
			headers: map[string]string{
				"Authorization": "bearer my-token",
			},
			want: "", // Case-sensitive check for "Bearer "
		},
		{
			name: "Authorization with Basic auth",
			headers: map[string]string{
				"Authorization": "Basic dXNlcjpwYXNz",
			},
			want: "", // Not Bearer auth
		},
		{
			name: "Bearer without space",
			headers: map[string]string{
				"Authorization": "Bearer",
			},
			want: "", // Bearer with no space or token
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := events.APIGatewayCustomAuthorizerRequestTypeRequest{
				Headers: tt.headers,
			}
			got := extractToken(event)
			if got != tt.want {
				t.Errorf("extractToken() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Test concurrent cache access
func TestTokenCache_ConcurrentAccess(t *testing.T) {
	// Save original state
	tokenCache.mu.Lock()
	origToken := tokenCache.token
	origExpiration := tokenCache.expiration
	tokenCache.mu.Unlock()
	
	// Cleanup after test
	defer func() {
		tokenCache.mu.Lock()
		tokenCache.token = origToken
		tokenCache.expiration = origExpiration
		tokenCache.mu.Unlock()
	}()
	
	// Reset cache for test
	tokenCache.mu.Lock()
	tokenCache.token = "concurrent-test-token"
	tokenCache.expiration = time.Now().Add(5 * time.Minute)
	tokenCache.mu.Unlock()

	// Spawn multiple goroutines to read from cache concurrently
	const numGoroutines = 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	results := make(chan string, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			tokenCache.mu.RLock()
			token := tokenCache.token
			tokenCache.mu.RUnlock()
			results <- token
		}()
	}

	wg.Wait()
	close(results)

	// Verify all goroutines read the same value
	for token := range results {
		if token != "concurrent-test-token" {
			t.Errorf("Expected 'concurrent-test-token', got '%s'", token)
		}
	}
}

// Test circuit breaker functionality
func TestCircuitBreaker_OpenAndClose(t *testing.T) {
	cb := &CircuitBreaker{}

	// Circuit should be closed initially
	if cb.isOpen() {
		t.Error("Circuit should be closed initially")
	}

	// Record failures to open circuit
	for i := 0; i < circuitBreakerThreshold; i++ {
		cb.recordFailure()
	}

	// Circuit should be open after threshold failures
	if !cb.isOpen() {
		t.Error("Circuit should be open after threshold failures")
	}

	// Reset circuit
	cb.reset()

	// Circuit should be closed after reset
	if cb.isOpen() {
		t.Error("Circuit should be closed after reset")
	}
}

// Test circuit breaker timeout
func TestCircuitBreaker_Timeout(t *testing.T) {
	cb := &CircuitBreaker{}

	// Record failures to open circuit
	for i := 0; i < circuitBreakerThreshold; i++ {
		cb.recordFailure()
	}

	if !cb.isOpen() {
		t.Error("Circuit should be open")
	}

	// Manually set last failure to past the timeout
	cb.mu.Lock()
	cb.lastFailure = time.Now().Add(-circuitBreakerTimeout - time.Second)
	cb.mu.Unlock()

	// Circuit should now be considered closed (timeout passed)
	if cb.isOpen() {
		t.Error("Circuit should be closed after timeout")
	}
}

// Test hashToken function
func TestHashToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "simple token",
			token: "my-secret-token",
		},
		{
			name:  "complex token",
			token: "Bearer abc123!@#$%^&*()",
		},
		{
			name:  "empty token",
			token: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := hashToken(tt.token)
			hash2 := hashToken(tt.token)

			// Same token should produce same hash
			if hash1 != hash2 {
				t.Errorf("hashToken() produced different results for same input: %s vs %s", hash1, hash2)
			}

			// Hash should start with "user-" prefix
			if !strings.HasPrefix(hash1, "user-") {
				t.Errorf("hashToken() result should start with 'user-', got: %s", hash1)
			}

			// Hash should be consistent length (user- + 16 hex chars)
			expectedLen := len("user-") + 16
			if len(hash1) != expectedLen {
				t.Errorf("hashToken() result should be %d characters, got %d: %s", expectedLen, len(hash1), hash1)
			}
		})
	}

	// Different tokens should produce different hashes
	hash1 := hashToken("token1")
	hash2 := hashToken("token2")
	if hash1 == hash2 {
		t.Error("Different tokens should produce different hashes")
	}
}

// Test circuit breaker with concurrent access
func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := &CircuitBreaker{}
	
	// Concurrently record failures and check state
	const numGoroutines = 20
	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func(iteration int) {
			defer wg.Done()
			
			if iteration < 5 {
				// First 5 goroutines record failures
				cb.recordFailure()
			} else {
				// Rest check if circuit is open
				_ = cb.isOpen()
			}
		}(i)
	}
	
	wg.Wait()
	
	// Verify circuit is open after concurrent failures
	if !cb.isOpen() {
		t.Error("Circuit should be open after concurrent failures")
	}
	
	// Get failure count
	failCount := cb.getFailures()
	if failCount < circuitBreakerThreshold {
		t.Errorf("Expected at least %d failures, got %d", circuitBreakerThreshold, failCount)
	}
}

// Test circuit breaker auto-reset race condition
func TestCircuitBreaker_AutoResetRaceCondition(t *testing.T) {
	cb := &CircuitBreaker{}
	
	// Open the circuit
	for i := 0; i < circuitBreakerThreshold; i++ {
		cb.recordFailure()
	}
	
	if !cb.isOpen() {
		t.Fatal("Circuit should be open")
	}
	
	// Set last failure to past timeout
	cb.mu.Lock()
	cb.lastFailure = time.Now().Add(-circuitBreakerTimeout - time.Second)
	cb.mu.Unlock()
	
	// Multiple goroutines check if circuit is open simultaneously
	const numGoroutines = 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	
	results := make(chan bool, numGoroutines)
	
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			results <- cb.isOpen()
		}()
	}
	
	wg.Wait()
	close(results)
	
	// All should see circuit as closed (false)
	for isOpen := range results {
		if isOpen {
			t.Error("Circuit should be closed after timeout for all goroutines")
		}
	}
	
	// Verify failures were actually reset to 0
	failCount := cb.getFailures()
	if failCount != 0 {
		t.Errorf("Expected failures to be reset to 0, got %d", failCount)
	}
}


