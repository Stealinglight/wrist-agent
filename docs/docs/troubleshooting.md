---
sidebar_position: 6
---

# Troubleshooting

This guide helps you diagnose and resolve common issues with your Wrist Agent deployment.

## Quick Diagnostics

### Test Connectivity

```bash
# Check if Function URL is accessible
curl -I "https://your-function-url"

# Expected: HTTP/2 200 or 401 (authentication required)
# If timeout or connection refused, check deployment
```

### Verify Authentication

```bash
# Get your token
TOKEN=$(aws ssm get-parameter \
  --name "/wrist-agent/client-token" \
  --with-decryption \
  --query 'Parameter.Value' \
  --output text)

# Test with token
curl -X POST "https://your-function-url" \
  -H "Content-Type: application/json" \
  -H "X-Client-Token: $TOKEN" \
  -d '{"text": "test", "mode": "note"}'

# Expected: JSON response with markdown field
```

### Check Lambda Logs

```bash
# View recent logs
aws logs tail /aws/lambda/WristAgentStack-WristAgentHandler --follow

# Search for errors in last hour
aws logs filter-log-events \
  --log-group-name "/aws/lambda/WristAgentStack-WristAgentHandler" \
  --start-time $(date -d '1 hour ago' +%s)000 \
  --filter-pattern "ERROR"
```

## Common Issues

### Authentication Errors

#### 401 Unauthorized

**Symptoms:**
```json
{
  "error": "Unauthorized"
}
```

**Causes:**
1. Missing or incorrect `X-Client-Token` header
2. Token mismatch with SSM parameter
3. Token not set in Apple Shortcut

**Solutions:**

```bash
# Verify token in SSM
aws ssm get-parameter \
  --name "/wrist-agent/client-token" \
  --with-decryption \
  --query 'Parameter.Value' \
  --output text

# Compare with token in your shortcut
# If different, update shortcut with correct token

# Or generate new token and update both
NEW_TOKEN=$(openssl rand -base64 32)
aws ssm put-parameter \
  --name "/wrist-agent/client-token" \
  --value "$NEW_TOKEN" \
  --overwrite
echo "Update your shortcut with: $NEW_TOKEN"
```

### Bedrock Errors

#### Access Denied to Bedrock Model

**Symptoms:**
```
AccessDeniedException: Could not access model
```

**Causes:**
1. Model access not enabled in Bedrock console
2. Wrong region
3. Insufficient IAM permissions

**Solutions:**

```bash
# Enable model access (one-time setup)
# 1. Go to AWS Bedrock Console → Model Access
# 2. Request access to "Anthropic Claude Haiku 4.5"
# 3. Wait for approval (usually instant)

# Verify region matches your deployment
aws cloudformation describe-stacks \
  --stack-name WristAgentStack \
  --query 'Stacks[0].Outputs[?OutputKey==`Region`].OutputValue' \
  --output text

# Check Lambda execution role permissions
aws iam get-role-policy \
  --role-name WristAgentStack-WristAgentHandlerRole \
  --policy-name BedrockAccess
```

#### Model Not Found

**Symptoms:**
```
ResourceNotFoundException: Model not found
```

**Causes:**
1. Model ID incorrect or not available in region
2. Region doesn't support Claude Haiku 4.5

**Solutions:**

```bash
# List available Bedrock models in your region
aws bedrock list-foundation-models \
  --region us-west-2 \
  --query 'modelSummaries[?providerName==`Anthropic`]'

# Update to correct model ID in your deployment
# Edit cdk/lib/config.ts or set environment variable
export BEDROCK_MODEL_ID=anthropic.claude-haiku-4-5-20251001-v1:0
```

### Lambda Function Errors

#### Timeout Errors

**Symptoms:**
```
Task timed out after 30.00 seconds
```

**Causes:**
1. Bedrock response taking too long
2. Large thinking token requests
3. Network latency

**Solutions:**

```bash
# Increase Lambda timeout
# Edit cdk/lib/config.ts
# timeout: cdk.Duration.seconds(60),

# Then redeploy
cd cdk
npx cdk deploy

# For immediate fix, use AWS Console:
# Lambda → Configuration → General → Timeout → 60 seconds
```

#### Out of Memory

**Symptoms:**
```
Runtime exited with error: signal: killed
```

**Causes:**
1. Large response payloads
2. Insufficient memory allocation

**Solutions:**

```bash
# Increase Lambda memory
# Edit cdk/lib/config.ts
# memorySize: 512,

# Then redeploy
cd cdk
npx cdk deploy

# Or use console: Lambda → Configuration → General → Memory
```

#### Cold Start Issues

**Symptoms:**
- First request after idle period is slow
- Intermittent timeouts

**Solutions:**

```bash
# Add provisioned concurrency (increases cost)
# In CDK:
# this.fn.currentVersion.addAlias('live', {
#   provisionedConcurrentExecutions: 1,
# });

# Or accept cold starts as normal behavior
# Typical cold start: 1-3 seconds
```

### Apple Shortcut Issues

#### Shortcut Fails Silently

**Symptoms:**
- Shortcut completes but no output
- No error message shown

**Causes:**
1. Network connectivity issues
2. Invalid JSON in request
3. Response parsing errors

**Solutions:**

```javascript
// In Shortcuts, add "Show Result" action after API call
// to see the raw response

// Add error handling:
// 1. Add "Get Dictionary Value" for "error" key
// 2. If error exists, show notification with error message
// 3. Otherwise, proceed with normal flow
```

#### Voice Dictation Accuracy

**Symptoms:**
- Incorrect transcription
- Missing words or phrases

**Solutions:**

1. **Speak clearly** with pauses between sentences
2. **Use punctuation words**: "comma", "period", "question mark"
3. **Enable research mode** for better processing of unclear text
4. **Review and edit** text before sending to API
5. **Use a quieter environment** to reduce background noise

#### Watch Complication Not Appearing

**Symptoms:**
- Shortcut works on iPhone but not visible on Watch
- Complication slot empty

**Solutions:**

```bash
# On iPhone:
# 1. Open Watch app
# 2. Go to My Watch → Shortcuts
# 3. Verify shortcut is enabled for Watch
# 4. Force sync: Toggle airplane mode on/off

# On Apple Watch:
# 1. Force restart: Hold side button + Digital Crown
# 2. Re-add complication to watch face
```

### Deployment Issues

#### CDK Bootstrap Failure

**Symptoms:**
```
Error: This stack uses assets, so the toolkit stack must be deployed
```

**Solutions:**

```bash
# Bootstrap CDK in your account/region
npx cdk bootstrap aws://ACCOUNT-ID/REGION

# Or with specific profile
npx cdk bootstrap --profile your-profile

# Verify bootstrap stack exists
aws cloudformation describe-stacks \
  --stack-name CDKToolkit
```

#### GitHub Actions Deployment Failure

**Symptoms:**
- Workflow fails with permission errors
- OIDC authentication fails

**Solutions:**

```bash
# Verify OIDC provider exists
aws iam list-open-id-connect-providers

# Check role trust policy
aws iam get-role --role-name WristAgentGitHubActions \
  --query 'Role.AssumeRolePolicyDocument'

# Verify GitHub secret is set correctly
# Settings → Secrets → Actions → AWS_ROLE_ARN
# Should be: arn:aws:iam::ACCOUNT-ID:role/WristAgentGitHubActions
```

#### Stack Update Failures

**Symptoms:**
```
Error: Resource update failed
```

**Solutions:**

```bash
# Check what's being updated
npx cdk diff

# View CloudFormation events
aws cloudformation describe-stack-events \
  --stack-name WristAgentStack \
  --query 'StackEvents[0:10]'

# If stuck, rollback and retry
npx cdk deploy --rollback-configuration RollbackTriggers=[]

# Nuclear option: destroy and redeploy
npx cdk destroy
npx cdk deploy
```

## Performance Issues

### Slow Response Times

**Symptoms:**
- Requests taking >10 seconds
- Inconsistent latency

**Diagnostics:**

```bash
# Measure response time
time curl -X POST "$FUNCTION_URL" \
  -H "Content-Type: application/json" \
  -H "X-Client-Token: $TOKEN" \
  -d '{"text": "test", "mode": "note"}'

# Check Lambda duration metrics
aws cloudwatch get-metric-statistics \
  --namespace AWS/Lambda \
  --metric-name Duration \
  --dimensions Name=FunctionName,Value=WristAgentStack-WristAgentHandler \
  --start-time $(date -u -d '1 hour ago' +%Y-%m-%dT%H:%M:%S) \
  --end-time $(date -u +%Y-%m-%dT%H:%M:%S) \
  --period 300 \
  --statistics Average,Maximum
```

**Solutions:**

1. **Reduce token limits** in request
2. **Use faster mode** (note vs deepthink)
3. **Optimize Lambda memory** (more memory = faster CPU)
4. **Check Bedrock service health** in your region

### High Costs

**Symptoms:**
- Unexpected AWS bill
- Bedrock charges higher than expected

**Solutions:**

```bash
# Check Bedrock usage
aws ce get-cost-and-usage \
  --time-period Start=$(date -d '1 month ago' +%Y-%m-%d),End=$(date +%Y-%m-%d) \
  --granularity MONTHLY \
  --metrics BlendedCost \
  --filter file://<(echo '{"Dimensions":{"Key":"SERVICE","Values":["Amazon Bedrock"]}}')

# Set up billing alarm
aws cloudwatch put-metric-alarm \
  --alarm-name WristAgent-CostAlert \
  --alarm-description "Alert when costs exceed $50" \
  --metric-name EstimatedCharges \
  --namespace AWS/Billing \
  --statistic Maximum \
  --period 86400 \
  --threshold 50 \
  --comparison-operator GreaterThanThreshold

# Optimize usage:
# 1. Reduce maxTokens in requests
# 2. Use thinkingTokens=0 for simple tasks
# 3. Batch multiple items into one request
```

## Debugging Tools

### Enable Debug Logging

```go
// In lambda/main.go, add debug output
log.Printf("Request: %+v\n", request)
log.Printf("Response: %+v\n", response)
```

### Monitor with CloudWatch Insights

```bash
# Query for errors
aws logs start-query \
  --log-group-name "/aws/lambda/WristAgentStack-WristAgentHandler" \
  --start-time $(date -d '1 hour ago' +%s) \
  --end-time $(date +%s) \
  --query-string 'fields @timestamp, @message | filter @message like /ERROR/'

# Query for slow requests
aws logs start-query \
  --log-group-name "/aws/lambda/WristAgentStack-WristAgentHandler" \
  --start-time $(date -d '1 hour ago' +%s) \
  --end-time $(date +%s) \
  --query-string 'fields @timestamp, @duration | filter @duration > 5000'
```

### Test Locally

```bash
# Build Lambda locally
cd lambda
GOOS=linux GOARCH=arm64 go build -o bootstrap main.go

# Run with AWS SAM (optional)
sam local start-api

# Test with curl
curl -X POST http://localhost:3000/ \
  -H "Content-Type: application/json" \
  -H "X-Client-Token: test" \
  -d '{"text": "test", "mode": "note"}'
```

## Getting Help

### Check CloudWatch Logs

Most issues can be diagnosed from Lambda logs:

```bash
# Real-time log monitoring
aws logs tail /aws/lambda/WristAgentStack-WristAgentHandler \
  --follow \
  --format short

# Export logs for analysis
aws logs create-export-task \
  --log-group-name "/aws/lambda/WristAgentStack-WristAgentHandler" \
  --from $(date -d '1 day ago' +%s)000 \
  --to $(date +%s)000 \
  --destination wrist-agent-logs \
  --destination-prefix troubleshooting
```

### Collect Diagnostic Information

When reporting issues, include:

```bash
# System information
echo "AWS Region: $(aws configure get region)"
echo "CDK Version: $(npx cdk --version)"
echo "Go Version: $(go version)"
echo "Node Version: $(node --version)"

# Deployment information
aws cloudformation describe-stacks \
  --stack-name WristAgentStack \
  --query 'Stacks[0].Outputs'

# Recent errors
aws logs filter-log-events \
  --log-group-name "/aws/lambda/WristAgentStack-WristAgentHandler" \
  --start-time $(date -d '1 hour ago' +%s)000 \
  --filter-pattern "ERROR"
```

## Prevention

### Regular Maintenance

```bash
# Weekly checks
1. Review CloudWatch metrics for anomalies
2. Check AWS costs in billing dashboard
3. Verify token security (no unauthorized usage)
4. Update dependencies (npm, go modules)

# Monthly tasks
1. Rotate authentication token
2. Review and clean up CloudWatch logs
3. Test backup/recovery procedures
4. Update to latest Bedrock model if available
```

### Monitoring Setup

```bash
# Create CloudWatch dashboard
aws cloudwatch put-dashboard \
  --dashboard-name WristAgent \
  --dashboard-body file://dashboard.json

# Set up SNS topic for alerts
aws sns create-topic --name wrist-agent-alerts

# Subscribe to alerts
aws sns subscribe \
  --topic-arn arn:aws:sns:REGION:ACCOUNT:wrist-agent-alerts \
  --protocol email \
  --notification-endpoint your-email@example.com
```

## Next Steps

- **[Review Security Best Practices](./security)** - Prevent security issues
- **[Optimize Your Deployment](./deployment)** - Improve performance
- **[Explore API Examples](./examples)** - Learn advanced usage
