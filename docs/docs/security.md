---
sidebar_position: 4
---

# Security Configuration

This guide covers the security model, best practices, and configuration for your Wrist Agent deployment.

## Security Architecture

Wrist Agent implements a multi-layered security approach:

```mermaid
graph TD
    A[Apple Watch] --> B[HTTPS Transport]
    B --> C[Function URL]
    C --> D[Header Authentication]
    D --> E[Lambda Execution]
    E --> F[IAM Role]
    F --> G[Bedrock Access]
    F --> H[SSM Parameter]
```

## Authentication Model

### Header-Based Authentication

Wrist Agent uses a simple but effective authentication scheme:

- **Authentication Header**: `X-Client-Token`
- **Token Storage**: AWS SSM Parameter Store
- **Token Type**: Base64-encoded random string (256-bit entropy)
- **Token Scope**: Single shared token for all requests

**Advantages:**

- Simple to implement and manage
- No complex OAuth or JWT flows
- Suitable for personal/single-user deployments
- Easy token rotation

**Considerations:**

- Single token shared across all clients
- Token visible in Apple Shortcuts configuration
- Suitable for personal use, not multi-tenant systems

## Token Management

### Initial Token Generation

The CDK deployment creates a random initial token:

```typescript
// In CDK stack
this.tokenParam = new ssm.StringParameter(this, 'ClientToken', {
  parameterName: config.clientTokenParam,
  stringValue: 'CHANGE_ME_' + Math.random().toString(36).substring(2, 15),
  tier: ssm.ParameterTier.STANDARD,
});
```

### Secure Token Rotation

Regular token rotation is a security best practice:

```bash
# Generate a cryptographically secure token
NEW_TOKEN=$(openssl rand -base64 32)

# Update the SSM parameter
aws ssm put-parameter \
  --name "/wrist-agent/client-token" \
  --value "$NEW_TOKEN" \
  --type "String" \
  --overwrite

# Update your Apple Shortcut with the new token
echo "New token: $NEW_TOKEN"
```

### Automated Rotation (Optional)

Create a Lambda function for automated token rotation:

```go
func rotateToken(ctx context.Context) error {
    // Generate new token
    newToken := generateSecureToken(32)

    // Update SSM parameter
    _, err := ssmClient.PutParameter(ctx, &ssm.PutParameterInput{
        Name:      aws.String("/wrist-agent/client-token"),
        Value:     aws.String(newToken),
        Type:      types.ParameterTypeString,
        Overwrite: aws.Bool(true),
    })

    // Notify about token change (SNS, email, etc.)
    notifyTokenRotation(newToken)

    return err
}
```

## Transport Security

### HTTPS Enforcement

All communication uses HTTPS:

- **Lambda Function URLs**: Native HTTPS endpoints
- **Certificate Management**: Handled by AWS
- **TLS Version**: TLS 1.2+ enforced

### CORS Configuration

The Lambda Function URL is configured with restrictive CORS:

```typescript
cors: {
  allowCredentials: false,
  allowedHeaders: ['Content-Type', 'X-Client-Token'],
  allowedMethods: [lambda.HttpMethod.POST, lambda.HttpMethod.OPTIONS],
  allowedOrigins: ['*'],
  maxAge: cdk.Duration.hours(1),
}
```

**Security Notes:**

- `allowCredentials: false` prevents credential-based attacks
- Limited headers reduce attack surface
- Only POST and OPTIONS methods allowed
- `allowedOrigins: ['*']` required for Apple Shortcuts

## AWS Authentication Options

Wrist Agent supports two authentication methods for Bedrock access:

### Option 1: IAM Role Authentication (Default)

The Lambda function uses least-privilege IAM permissions:

```typescript
// Bedrock model access only
this.fn.addToRolePolicy(
  new iam.PolicyStatement({
    effect: iam.Effect.ALLOW,
    actions: ['bedrock:InvokeModel'],
    resources: [`arn:aws:bedrock:${region}::foundation-model/${modelId}`],
  })
);

// SSM parameter read access only
this.tokenParam.grantRead(this.fn);
```

**Advantages:**

- No long-lived credentials to manage
- Follows AWS security best practices
- Automatic credential rotation via IAM

**Requirements:**

- Lambda execution role must have Bedrock permissions
- Works within AWS environment automatically

### Option 2: Hybrid API Key Authentication (Recommended)

**Best of both worlds: Convenient local development + secure production deployment**

Configure your API key once in `.env` and the system automatically handles both local and production scenarios:

```bash
# Set in your .env file for both local development and production deployment
BEDROCK_API_KEY=your_base64_encoded_api_key_here
```

**How the Hybrid System Works:**

1. **Local Development**: Lambda uses direct environment variable
2. **Production Deployment**: CDK creates encrypted SSM parameter automatically
3. **Runtime**: Lambda intelligently chooses the most secure available method

**Lambda Authentication Logic:**

```go
// Hybrid authentication with automatic fallback
func initializeAWSConfig() (aws.Config, error) {
    // Method 1: Direct environment variable (local development)
    if apiKey := os.Getenv("BEDROCK_API_KEY"); apiKey != "" {
        return configWithStaticCredentials(apiKey)
    }

    // Method 2: Encrypted SSM parameter (production)
    if paramName := os.Getenv("BEDROCK_API_KEY_PARAM"); paramName != "" {
        return configWithEncryptedSSMParameter(paramName)
    }

    // Method 3: IAM role fallback
    return configWithIAMRole()
}
```

**Security Features:**

- **Encrypted at Rest**: Production credentials stored as encrypted SSM SecureString parameters
- **KMS Encryption**: Uses AWS Key Management Service for encryption/decryption
- **Least Privilege**: Lambda only gets read access to specific parameters
- **Audit Logging**: All SSM parameter access logged in CloudTrail
- **Cost Effective**: ~$0.05/month for SSM parameter vs ~$0.40/month for Secrets Manager

**API Key Format:**

- Base64-encoded string
- Decodes to: `AccessKeyId:SecretAccessKey`
- Example: `AKIAIOSFODNN7EXAMPLE:wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY`

**Deployment Process:**

1. **Set API Key**: Configure `BEDROCK_API_KEY` in your `.env` file
2. **Deploy**: Run `npx cdk deploy` - CDK automatically creates encrypted SSM parameter
3. **Runtime**: Lambda loads from encrypted parameter, falls back to direct env var if needed

**Security Considerations:**

- **Development**: API key in local `.env` file (excluded from git)
- **Production**: API key encrypted in SSM Parameter Store with KMS
- **Rotation**: Update SSM parameter value, Lambda picks up change on next cold start
- **Monitoring**: Set up CloudWatch alarms for unusual API usage patterns

**When to Use This Method:**

- **Most deployments**: Balances security and convenience
- **Team environments**: Secure sharing via SSM parameters
- **Compliance requirements**: Encrypted at rest with audit trails
- **Cost optimization**: Much cheaper than AWS Secrets Manager

### Resource-Level Permissions

Permissions are scoped to specific resources:

- **Bedrock**: Only the specified Claude model
- **SSM**: Only the specific token parameter
- **CloudWatch**: Automatic logging permissions

### Deployment Security

The deployment uses OIDC for GitHub Actions:

- **No AWS keys in repository**
- **Temporary credentials only**
- **Scoped to specific repository**

## Data Security

### Data in Transit

- **Encryption**: TLS 1.2+ for all communications
- **Integrity**: HTTPS ensures data integrity
- **No sensitive data in URLs**: All data in request body

### Data at Rest

- **SSM Parameter**: Encrypted at rest by default
- **CloudWatch Logs**: Encrypted with AWS managed keys
- **No persistent storage**: Lambda is stateless

### Data Processing

- **Voice Data**: Processed by Apple, not stored
- **Request Text**: Sent to Bedrock, not permanently stored
- **Responses**: Not logged or stored beyond CloudWatch retention

## Monitoring and Alerting

### CloudWatch Monitoring

Monitor authentication failures and usage patterns:

```bash
# Create alarm for 401 errors
aws cloudwatch put-metric-alarm \
  --alarm-name "WristAgent-AuthFailures" \
  --alarm-description "Alert on authentication failures" \
  --metric-name "4XXError" \
  --namespace "AWS/Lambda" \
  --statistic "Sum" \
  --period 300 \
  --threshold 5 \
  --comparison-operator "GreaterThanThreshold" \
  --evaluation-periods 2
```

### Usage Monitoring

Track API usage for anomaly detection:

```bash
# Monitor invocation count
aws cloudwatch put-metric-alarm \
  --alarm-name "WristAgent-HighUsage" \
  --alarm-description "Alert on unusual usage patterns" \
  --metric-name "Invocations" \
  --namespace "AWS/Lambda" \
  --statistic "Sum" \
  --period 3600 \
  --threshold 100 \
  --comparison-operator "GreaterThanThreshold" \
  --evaluation-periods 1
```

## Security Best Practices

### Deployment Security

1. **Regular Updates**: Keep CDK and dependencies updated
2. **Token Rotation**: Rotate tokens monthly or after any compromise
3. **Access Review**: Regularly review IAM permissions
4. **Monitoring**: Set up alerts for unusual activity

### Apple Shortcut Security

1. **Token Storage**: Tokens are visible in Shortcuts; consider device security
2. **Sharing**: Don't share shortcuts containing tokens
3. **Backup**: Secure backup of shortcuts with credentials
4. **Updates**: Update shortcuts after token rotation

### Operational Security

1. **Least Privilege**: Use minimal AWS permissions for deployment
2. **Separation**: Use separate AWS accounts for dev/prod if needed
3. **Audit Logging**: Enable CloudTrail for API audit logs
4. **Cost Monitoring**: Set up billing alerts to detect abuse

## Incident Response

### Compromised Token

If you suspect token compromise:

```bash
# 1. Immediately rotate the token
NEW_TOKEN=$(openssl rand -base64 32)
aws ssm put-parameter \
  --name "/wrist-agent/client-token" \
  --value "$NEW_TOKEN" \
  --type "String" \
  --overwrite

# 2. Update Apple Shortcuts immediately

# 3. Check CloudWatch logs for suspicious activity
aws logs filter-log-events \
  --log-group-name "/aws/lambda/WristAgentStack-WristAgentHandler" \
  --start-time $(date -d '1 hour ago' +%s)000
```

### Unusual Activity

Signs to monitor:

- **High request volume** from unknown sources
- **Failed authentication attempts** in CloudWatch
- **Unexpected AWS costs** from Bedrock usage
- **Error patterns** indicating attack attempts

## Compliance Considerations

### Data Privacy

- **Personal Data**: Voice transcriptions processed by Apple and Bedrock
- **Retention**: No long-term storage of user data
- **Processing Location**: AWS region configurable (default: us-west-2)

### Regional Compliance

For specific compliance requirements:

```bash
# Deploy to EU region for GDPR
export AWS_REGION=eu-west-1
export BEDROCK_MODEL_ID=anthropic.claude-sonnet-4-20250514-v1:0

cd cdk
npx cdk deploy
```

## Security Checklist

Before going live:

- [ ] **Token rotated** from default value
- [ ] **Monitoring set up** for failures and usage
- [ ] **Access reviewed** - minimal IAM permissions
- [ ] **Apple Shortcuts secured** on trusted devices only
- [ ] **Backup plan** for token recovery
- [ ] **Cost alerts configured** to prevent abuse
- [ ] **CloudTrail enabled** for audit logging
- [ ] **Regular rotation scheduled** for tokens

## Advanced Security Options

### IP Restrictions (Future Enhancement)

For additional security, consider Lambda@Edge for IP filtering:

```typescript
// Example: Restrict to specific IP ranges
const allowedIPs = ['192.168.1.0/24', '10.0.0.0/8'];
```

### Request Signing (Future Enhancement)

For higher security environments, implement request signing:

```typescript
// Example: HMAC-SHA256 request signing
const signature = crypto.createHmac('sha256', secret).update(requestBody).digest('hex');
```

### Multi-Token Support (Future Enhancement)

Support multiple tokens for different devices:

```json
{
  "tokens": {
    "device1": "token1",
    "device2": "token2"
  }
}
```

## Next Steps

- **[Test Your Setup](./examples)** - Verify security configuration
- **[Monitor Usage](./troubleshooting)** - Set up monitoring
- **[Plan Maintenance](./deployment)** - Regular security updates

Your Wrist Agent deployment is now secured with industry best practices. Remember to rotate tokens regularly and monitor for unusual activity.
