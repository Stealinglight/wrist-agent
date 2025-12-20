package main

import (
	"context"
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

// TokenCache holds cached token with expiration
type TokenCache struct {
	token      string
	expiration time.Time
	mu         sync.RWMutex
}

var (
	ssmClient      *ssm.Client
	tokenParamName string
	region         string
	tokenCache     = &TokenCache{}
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

	log.Printf("Authorization granted")
	return generatePolicy("wrist-agent-user", "Allow", event.MethodArn, map[string]interface{}{
		"authenticated": "true",
	}), nil
}

// extractToken gets the token from request headers
func extractToken(event events.APIGatewayCustomAuthorizerRequestTypeRequest) string {
	// Check X-Client-Token header (case-insensitive)
	for key, value := range event.Headers {
		if strings.EqualFold(key, "X-Client-Token") {
			return strings.TrimSpace(value)
		}
	}

	// Fallback to Authorization header with Bearer prefix
	if auth, ok := event.Headers["Authorization"]; ok {
		token := strings.TrimPrefix(auth, "Bearer ")
		return strings.TrimSpace(token)
	}

	return ""
}

// getExpectedToken retrieves and caches the expected token from SSM
func getExpectedToken(ctx context.Context) (string, error) {
	tokenCache.mu.RLock()
	if tokenCache.token != "" && time.Now().Before(tokenCache.expiration) {
		token := tokenCache.token
		tokenCache.mu.RUnlock()
		return token, nil
	}
	tokenCache.mu.RUnlock()

	// Cache miss or expired - fetch from SSM
	tokenCache.mu.Lock()
	defer tokenCache.mu.Unlock()

	// Double-check after acquiring write lock
	if tokenCache.token != "" && time.Now().Before(tokenCache.expiration) {
		return tokenCache.token, nil
	}

	output, err := ssmClient.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           aws.String(tokenParamName),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get SSM parameter %s: %w", tokenParamName, err)
	}

	// SECURITY: Never log token values - only log metadata about the cache operation
	token := strings.TrimSpace(aws.ToString(output.Parameter.Value))
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
