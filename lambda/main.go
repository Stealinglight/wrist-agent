package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// Request payload structure
type Req struct {
	Text           string `json:"text"`
	Mode           string `json:"mode"`           // note|reminder|event|research|deepthink
	ThinkingTokens int    `json:"thinkingTokens"` // 0..N for extended thinking
	MaxTokens      int    `json:"maxTokens"`      // default 800
}

// Response structure
type Response struct {
	Markdown string   `json:"markdown"`
	Action   string   `json:"action"`
	Title    string   `json:"title"`
	DueISO   *string  `json:"dueISO"`
	Tags     []string `json:"tags"`
}

// Bedrock response structures
type BedrockResponse struct {
	Content []Content `json:"content"`
	Usage   Usage     `json:"usage"`
}

type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Global AWS clients
var (
	ssmClient      *ssm.Client
	bedrockClient  *bedrockruntime.Client
	clientToken    string
	modelID        string
	region         string
	tokenParamName string
)

func init() {
	// Load environment variables
	region = getEnv("BEDROCK_REGION", "us-west-2")
	modelID = getEnv("MODEL_ID", "anthropic.claude-sonnet-4-20250514-v1:0")
	tokenParamName = getEnv("CLIENT_TOKEN_PARAM", "/wrist-agent/client-token")

	// Initialize AWS clients
	cfg, err := initializeAWSConfig()
	if err != nil {
		log.Fatalf("Failed to initialize AWS config: %v", err)
	}

	ssmClient = ssm.NewFromConfig(cfg)
	bedrockClient = bedrockruntime.NewFromConfig(cfg)

	// Load client token from SSM
	if err := loadClientToken(); err != nil {
		log.Fatalf("Failed to load client token: %v", err)
	}

	log.Printf("Initialized Wrist Agent Lambda - Region: %s, Model: %s", region, modelID)
}

// initializeAWSConfig sets up AWS configuration
func initializeAWSConfig() (aws.Config, error) {
	ctx := context.TODO()

	// Option 1: Check for direct BEDROCK_API_KEY environment variable (local development)
	if apiKey := os.Getenv("BEDROCK_API_KEY"); apiKey != "" {
		log.Printf("Using static credentials from BEDROCK_API_KEY environment variable")
		accessKeyID, secretAccessKey, err := parseBedrockAPIKey(apiKey)
		if err != nil {
			return aws.Config{}, fmt.Errorf("failed to parse BEDROCK_API_KEY: %w", err)
		}

		return aws.Config{
			Region:      region,
			Credentials: credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
		}, nil
	}

	// Option 2: Check for BEDROCK_API_KEY_PARAM (production with encrypted SSM parameter)
	if apiKeyParamName := os.Getenv("BEDROCK_API_KEY_PARAM"); apiKeyParamName != "" {
		log.Printf("Attempting to load Bedrock API key from SSM parameter: %s", apiKeyParamName)

		// First, create a basic config to access SSM
		basicCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
		if err != nil {
			return aws.Config{}, fmt.Errorf("failed to load basic AWS config: %w", err)
		}

		// Load API key from SSM parameter
		tempSSMClient := ssm.NewFromConfig(basicCfg)
		result, err := tempSSMClient.GetParameter(ctx, &ssm.GetParameterInput{
			Name:           aws.String(apiKeyParamName),
			WithDecryption: aws.Bool(true), // Decrypt the SecureString parameter
		})
		if err != nil {
			log.Printf("Failed to load API key from SSM parameter %s: %v", apiKeyParamName, err)
			log.Printf("Falling back to IAM role authentication")
		} else {
			// Parse the API key from SSM parameter
			apiKey := *result.Parameter.Value
			accessKeyID, secretAccessKey, err := parseBedrockAPIKey(apiKey)
			if err != nil {
				log.Printf("Failed to parse API key from SSM parameter: %v", err)
				log.Printf("Falling back to IAM role authentication")
			} else {
				log.Printf("Successfully loaded static credentials from SSM parameter")
				return aws.Config{
					Region:      region,
					Credentials: credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
				}, nil
			}
		}
	}

	// Option 3: Fallback to default AWS config (IAM role)
	log.Printf("Using default AWS credentials (IAM role)")
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load default AWS config: %w", err)
	}

	return cfg, nil
}

// parseBedrockAPIKey parses the base64-encoded Bedrock API key
// Expected format after base64 decode: "BedrockAPIKey-{accessKeyId}:{secretAccessKey}"
func parseBedrockAPIKey(encodedKey string) (accessKeyID, secretAccessKey string, err error) {
	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to decode base64 API key: %w", err)
	}

	decodedStr := string(decoded)
	log.Printf("Decoded API key format: %s", decodedStr[:min(20, len(decodedStr))]+"...")

	// Split by colon to separate access key ID and secret access key
	parts := strings.SplitN(decodedStr, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid API key format: expected 'accessKeyId:secretAccessKey'")
	}

	accessKeyID = parts[0]
	secretAccessKey = parts[1]

	if accessKeyID == "" || secretAccessKey == "" {
		return "", "", fmt.Errorf("invalid API key: access key ID or secret access key is empty")
	}

	log.Printf("Parsed credentials - Access Key ID: %s...", accessKeyID[:min(10, len(accessKeyID))])
	return accessKeyID, secretAccessKey, nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func loadClientToken() error {
	result, err := ssmClient.GetParameter(context.TODO(), &ssm.GetParameterInput{
		Name:           aws.String(tokenParamName),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return fmt.Errorf("failed to get SSM parameter %s: %w", tokenParamName, err)
	}

	clientToken = *result.Parameter.Value
	log.Printf("Loaded client token from SSM parameter: %s", tokenParamName)
	return nil
}

func handler(ctx context.Context, event events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	log.Printf("Processing request: %s %s", event.RequestContext.HTTP.Method, event.RawPath)

	// Handle CORS preflight
	if event.RequestContext.HTTP.Method == "OPTIONS" {
		return corsResponse(200, nil), nil
	}

	// Only allow POST requests
	if event.RequestContext.HTTP.Method != "POST" {
		return corsResponse(405, map[string]string{"error": "Method not allowed"}), nil
	}

	// Validate authentication
	authHeader := event.Headers["x-client-token"]
	if authHeader == "" {
		authHeader = event.Headers["X-Client-Token"] // Try capitalized version
	}
	if authHeader != clientToken {
		log.Printf("Authentication failed - invalid or missing token")
		return corsResponse(401, map[string]string{"error": "Invalid or missing authentication token"}), nil
	}

	// Parse request body
	var req Req
	if err := json.Unmarshal([]byte(event.Body), &req); err != nil {
		log.Printf("Failed to parse request body: %v", err)
		return corsResponse(400, map[string]string{"error": "Invalid JSON payload"}), nil
	}

	// Validate request
	if err := validateRequest(&req); err != nil {
		log.Printf("Request validation failed: %v", err)
		return corsResponse(400, map[string]string{"error": err.Error()}), nil
	}

	// Call Bedrock
	response, err := callBedrock(ctx, &req)
	if err != nil {
		log.Printf("Bedrock call failed: %v", err)
		return corsResponse(500, map[string]string{"error": "Failed to process request"}), nil
	}

	log.Printf("Successfully processed request for mode: %s", req.Mode)
	return corsResponse(200, response), nil
}

func validateRequest(req *Req) error {
	if strings.TrimSpace(req.Text) == "" {
		return fmt.Errorf("text field is required")
	}

	validModes := map[string]bool{
		"note": true, "reminder": true, "event": true, "research": true, "deepthink": true,
	}
	if req.Mode == "" {
		req.Mode = "note" // Default mode
	}
	if !validModes[req.Mode] {
		return fmt.Errorf("invalid mode: %s (valid: note, reminder, event, research, deepthink)", req.Mode)
	}

	if req.ThinkingTokens < 0 || req.ThinkingTokens > 65536 {
		return fmt.Errorf("thinkingTokens must be between 0 and 65536")
	}

	if req.MaxTokens <= 0 {
		req.MaxTokens = 800 // Default
	}
	if req.MaxTokens > 4096 {
		return fmt.Errorf("maxTokens cannot exceed 4096")
	}

	return nil
}

func callBedrock(ctx context.Context, req *Req) (*Response, error) {
	// Build system prompt based on mode
	systemPrompt := buildSystemPrompt(req.Mode)

	// Build user message
	userMessage := fmt.Sprintf("Process this request: %s", req.Text)

	// Prepare Bedrock request
	messages := []map[string]interface{}{
		{
			"role": "user",
			"content": []map[string]string{
				{"type": "text", "text": userMessage},
			},
		},
	}

	requestBody := map[string]interface{}{
		"anthropic_version": "bedrock-2023-05-31",
		"system":            systemPrompt,
		"messages":          messages,
		"max_tokens":        req.MaxTokens,
		"temperature":       0.1,
	}

	// Add thinking tokens if specified
	if req.ThinkingTokens > 0 {
		requestBody["thinking"] = map[string]interface{}{
			"max_thinking_tokens": req.ThinkingTokens,
		}
	}

	// Marshal request
	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Bedrock request: %w", err)
	}

	// Call Bedrock
	result, err := bedrockClient.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(modelID),
		ContentType: aws.String("application/json"),
		Body:        requestJSON,
	})
	if err != nil {
		return nil, fmt.Errorf("Bedrock InvokeModel failed: %w", err)
	}

	// Parse Bedrock response
	var bedrockResp BedrockResponse
	if err := json.Unmarshal(result.Body, &bedrockResp); err != nil {
		return nil, fmt.Errorf("failed to parse Bedrock response: %w", err)
	}

	if len(bedrockResp.Content) == 0 {
		return nil, fmt.Errorf("empty response from Bedrock")
	}

	// Extract and parse Claude's structured response
	claudeText := bedrockResp.Content[0].Text

	// Try to parse as JSON first (structured response)
	var structuredResp Response
	if err := json.Unmarshal([]byte(claudeText), &structuredResp); err == nil {
		return &structuredResp, nil
	}

	// Fallback: create response from raw text
	log.Printf("Claude returned unstructured response, creating fallback response")
	return &Response{
		Markdown: claudeText,
		Action:   req.Mode,
		Title:    extractTitle(claudeText, req.Mode),
		Tags:     []string{req.Mode},
	}, nil
}

func buildSystemPrompt(mode string) string {
	basePrompt := `You are a helpful assistant that processes voice-to-text requests from an Apple Watch. Always respond with valid JSON in this exact format:

{
  "markdown": "formatted content here",
  "action": "note|reminder|event|none",
  "title": "extracted or generated title",
  "dueISO": "2025-01-15T09:00:00Z or null",
  "tags": ["tag1", "tag2"]
}

Guidelines:
- Extract clear, actionable titles
- For reminders/events, try to extract dates/times and convert to ISO format
- Use markdown formatting for content
- Keep responses concise but complete`

	switch mode {
	case "reminder":
		return basePrompt + `

Mode: REMINDER
Focus on creating reminders with due dates. Look for time references and convert them to ISO format. Set action to "reminder".`

	case "event":
		return basePrompt + `

Mode: EVENT  
Focus on calendar events with specific dates/times. Extract event details and timing. Set action to "event".`

	case "research":
		return basePrompt + `

Mode: RESEARCH
Provide detailed, well-researched responses. Include sources and comprehensive information. Set action to "note".`

	case "deepthink":
		return basePrompt + `

Mode: DEEP THINKING
Take time to thoroughly analyze the request. Consider multiple perspectives and provide thoughtful insights. Set action to "note".`

	default: // note
		return basePrompt + `

Mode: NOTE
Create clear, well-formatted notes. Extract key information and organize it logically. Set action to "note".`
	}
}

func extractTitle(content string, mode string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "{") {
			// Clean up markdown headers
			line = strings.TrimPrefix(line, "# ")
			line = strings.TrimPrefix(line, "## ")
			if len(line) > 50 {
				line = line[:47] + "..."
			}
			return line
		}
	}
	return fmt.Sprintf("Wrist Agent %s", strings.Title(mode))
}

func corsResponse(statusCode int, body interface{}) events.LambdaFunctionURLResponse {
	var bodyStr string
	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		bodyStr = string(bodyBytes)
	}

	return events.LambdaFunctionURLResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type":                 "application/json",
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Headers": "Content-Type, X-Client-Token",
			"Access-Control-Allow-Methods": "POST, OPTIONS",
			"Access-Control-Max-Age":       "3600",
		},
		Body: bodyStr,
	}
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
