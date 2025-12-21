package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// Authorization error types for debugging (returned in policy context)
// These help identify the reason for authorization failures without leaking sensitive data
const (
	ErrMissingToken  = "missing_token"
	ErrInvalidToken  = "invalid_token"
	ErrTokenMismatch = "token_mismatch"
	ErrSSMFailure    = "ssm_failure"
)

// Default cache duration in seconds (can be overridden by TOKEN_CACHE_TTL_SECONDS env var)
const defaultCacheDurationSeconds = 300 // 5 minutes

// Circuit breaker configuration
const (
	circuitBreakerThreshold = 3                // Number of failures before opening circuit
	circuitBreakerTimeout   = 30 * time.Second // How long to wait before trying again
)

// TokenCache holds cached token with expiration
type TokenCache struct {
	token      string
	expiration time.Time
	mu         sync.RWMutex
}

// CircuitBreaker tracks SSM failures to prevent cascading failures
type CircuitBreaker struct {
	failures    int
	lastFailure time.Time
	mu          sync.RWMutex
}

var (
	ssmClient      *ssm.Client
	tokenParamName string
	region         string
	tokenCache     = &TokenCache{}
	circuitBreaker = &CircuitBreaker{}
	cacheDuration  time.Duration
)

// getCacheDuration reads cache TTL from environment or returns default
func getCacheDuration() time.Duration {
	if env := os.Getenv("TOKEN_CACHE_TTL_SECONDS"); env != "" {
		if seconds, err := strconv.Atoi(env); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
		log.Printf("Invalid TOKEN_CACHE_TTL_SECONDS value: %s, using default", env)
	}
	return time.Duration(defaultCacheDurationSeconds) * time.Second
}

func init() {
	region = getEnv("AWS_REGION", "us-west-2")
	tokenParamName = strings.TrimSpace(getEnv("CLIENT_TOKEN_PARAM_NAME", "/wrist-agent/client-token"))
	cacheDuration = getCacheDuration()

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	ssmClient = ssm.NewFromConfig(cfg)
	log.Printf("Lambda Authorizer initialized - Region: %s, TokenParam: %s, CacheTTL: %v", region, tokenParamName, cacheDuration)
}

func handler(ctx context.Context, event events.APIGatewayCustomAuthorizerRequestTypeRequest) (events.APIGatewayCustomAuthorizerResponse, error) {
	log.Printf("Authorizer invoked for method: %s", event.MethodArn)

	// Extract token from header
	token := extractToken(event)
	if token == "" {
		log.Printf("Authorization denied: missing token")
		return generatePolicy("user", "Deny", event.MethodArn, map[string]interface{}{
			"errorType": ErrMissingToken,
		}), nil
	}

	// Get expected token from SSM (with caching)
	expectedToken, err := getExpectedToken(ctx)
	if err != nil {
		log.Printf("Authorization error: failed to retrieve expected token: %v", err)
		return generatePolicy("user", "Deny", event.MethodArn, map[string]interface{}{
			"errorType": ErrSSMFailure,
		}), nil
	}

	// Validate token
	// SECURITY: Never log actual token values - only metadata about the validation result
	if token != expectedToken {
		log.Printf("Authorization denied: token mismatch")
		return generatePolicy("user", "Deny", event.MethodArn, map[string]interface{}{
			"errorType": ErrTokenMismatch,
		}), nil
	}

	// Use hashed token as principal ID for audit trail
	principalID := hashToken(token)
	log.Printf("Authorization granted for principal: %s", principalID)
	return generatePolicy(principalID, "Allow", event.MethodArn, map[string]interface{}{
		"authenticated": "true",
	}), nil
}

// extractToken gets the token from request headers
func extractToken(event events.APIGatewayCustomAuthorizerRequestTypeRequest) string {
	// Check X-Client-Token header (case-insensitive)
	for key, value := range event.Headers {
		if strings.EqualFold(key, "X-Client-Token") {
			token := strings.TrimSpace(value)
			if token != "" {
				return token
			}
		}
	}

	// Fallback to Authorization header with Bearer prefix
	if auth, ok := event.Headers["Authorization"]; ok {
		// Check if it starts with "Bearer " (with space)
		if strings.HasPrefix(auth, "Bearer ") {
			token := strings.TrimSpace(auth[7:]) // Skip "Bearer "
			if token != "" {
				return token
			}
		}
	}

	return ""
}

// hashToken creates a SHA-256 hash of the token for use as principal ID
// This allows distinguishing users in audit logs without exposing the actual token
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return "user-" + hex.EncodeToString(hash[:8]) // Use first 8 bytes (16 hex chars) for readability
}

// isOpen checks if the circuit breaker is open
// After timeout expires, the circuit transitions to "half-open" state where the next
// SSM call will be attempted. If it succeeds, reset() is called. If it fails, failures
// are incremented and circuit re-opens.
func (cb *CircuitBreaker) isOpen() bool {
	cb.mu.RLock()
	if cb.failures < circuitBreakerThreshold {
		cb.mu.RUnlock()
		return false
	}

	// Check if timeout has passed - capture time once to avoid drift
	lastFailureTime := cb.lastFailure
	cb.mu.RUnlock()

	timeSinceFailure := time.Since(lastFailureTime)
	if timeSinceFailure < circuitBreakerTimeout {
		return true
	}

	// Timeout passed - upgrade to write lock and reset
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Double-check after acquiring write lock to avoid race condition
	// Recalculate time to ensure timeout truly passed (guards against concurrent updates)
	if cb.failures >= circuitBreakerThreshold && time.Since(cb.lastFailure) >= circuitBreakerTimeout {
		cb.failures = 0
		log.Printf("Circuit breaker RESET after timeout")
	}
	return false
}

// recordFailure increments the failure count
func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	wasOpen := cb.failures >= circuitBreakerThreshold
	cb.failures++
	cb.lastFailure = time.Now()

	// Log when circuit opens
	if !wasOpen && cb.failures >= circuitBreakerThreshold {
		log.Printf("Circuit breaker OPENED after %d failures", cb.failures)
	}
}

// getFailures returns the current failure count safely
func (cb *CircuitBreaker) getFailures() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failures
}

// reset resets the circuit breaker
func (cb *CircuitBreaker) reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.failures > 0 {
		log.Printf("Circuit breaker CLOSED (manual reset from %d failures)", cb.failures)
	}
	cb.failures = 0
}

// getExpectedToken retrieves and caches the expected token from SSM
func getExpectedToken(ctx context.Context) (string, error) {
	// Capture current time once for consistency across checks
	now := time.Now()
	
	// Read token and expiration atomically to avoid race condition
	tokenCache.mu.RLock()
	token := tokenCache.token
	expiration := tokenCache.expiration
	tokenCache.mu.RUnlock()

	if token != "" && now.Before(expiration) {
		return token, nil
	}

	// Check circuit breaker before attempting SSM call
	if circuitBreaker.isOpen() {
		// Circuit is open, try to use cached token even if expired
		tokenCache.mu.RLock()
		cachedToken := tokenCache.token
		tokenCache.mu.RUnlock()

		if cachedToken != "" {
			log.Printf("Circuit breaker open, using stale cached token")
			return cachedToken, nil
		}
		return "", fmt.Errorf("circuit breaker open and no cached token available")
	}

	// Cache miss or expired - fetch from SSM
	tokenCache.mu.Lock()
	defer tokenCache.mu.Unlock()

	// Double-check after acquiring write lock (read atomically again)
	// Reuse the same 'now' timestamp to avoid time drift between checks
	if tokenCache.token != "" && now.Before(tokenCache.expiration) {
		return tokenCache.token, nil
	}

	// Add timeout to prevent indefinite blocking on SSM call
	ssmCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	output, err := ssmClient.GetParameter(ssmCtx, &ssm.GetParameterInput{
		Name:           aws.String(tokenParamName),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		circuitBreaker.recordFailure()
		failureCount := circuitBreaker.getFailures()
		log.Printf("SSM GetParameter failed (failures: %d): %v", failureCount, err)

		// Try to return stale cache if available
		if tokenCache.token != "" {
			log.Printf("Returning stale cached token due to SSM failure")
			return tokenCache.token, nil
		}
		return "", fmt.Errorf("failed to get SSM parameter %s: %w", tokenParamName, err)
	}

	// Success - reset circuit breaker
	circuitBreaker.reset()

	// SECURITY: Never log token values - only log metadata about the cache operation
	token = strings.TrimSpace(aws.ToString(output.Parameter.Value))
	if token == "" {
		return "", fmt.Errorf("SSM parameter %s returned empty value", tokenParamName)
	}
	

	tokenCache.token = token
	tokenCache.expiration = time.Now().Add(cacheDuration)

	log.Printf("Token refreshed from SSM, cached for %v", cacheDuration)
	return token, nil
}

// generatePolicy creates an IAM policy document for API Gateway
func generatePolicy(principalID, effect, resource string, context map[string]interface{}) events.APIGatewayCustomAuthorizerResponse {
	authResponse := events.APIGatewayCustomAuthorizerResponse{
		PrincipalID: principalID,
	}

	if effect != "" && resource != "" {
		authResponse.PolicyDocument = events.APIGatewayCustomAuthorizerPolicy{
			Version: "2012-10-17",
			Statement: []events.IAMPolicyStatement{
				{
					Action:   []string{"execute-api:Invoke"},
					Effect:   effect,
					Resource: []string{resource},
				},
			},
		}
	}

	if context != nil {
		authResponse.Context = context
	}

	return authResponse
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	lambda.Start(handler)
}
