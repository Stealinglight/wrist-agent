---
sidebar_position: 2
---

# Setup Guide

This guide will help you deploy the complete Wrist Agent infrastructure on AWS and configure it for use with your Apple Watch.

## Prerequisites

Before you begin, ensure you have:

- **AWS Account** with appropriate permissions
- **AWS CLI** configured with your credentials
- **Node.js** 18+ and npm installed
- **Go** 1.22+ installed
- **Git** for cloning the repository
- **Apple Developer Account** (free tier sufficient)

## Step 1: Clone and Setup Repository

```bash
# Clone the repository
git clone https://github.com/your-username/wrist-agent.git
cd wrist-agent

# Install CDK dependencies
cd cdk
npm install

# Install Go dependencies
cd ../lambda
go mod tidy

# Return to project root
cd ..
```

## Step 2: Configure Environment

Copy the environment template and configure your settings:

```bash
cp .env.example .env
```

Edit `.env` with your preferred settings:

```bash
# AWS Configuration
AWS_REGION=us-west-2
AWS_PROFILE=default

# Bedrock Configuration
BEDROCK_MODEL_ID=anthropic.claude-sonnet-4-20250514-v1:0

# Security Configuration
CLIENT_TOKEN_PARAM_NAME=/wrist-agent/client-token

# Lambda Configuration
LAMBDA_TIMEOUT=30
LAMBDA_MEMORY=256
```

## Step 3: Deploy Infrastructure

### Option A: Local Deployment

```bash
# Navigate to CDK directory
cd cdk

# Bootstrap CDK (first time only)
npx cdk bootstrap

# Build and deploy
npm run build
npx cdk deploy

# Note the Function URL from the output
```

### Option B: GitHub Actions Deployment

1. **Fork the repository** to your GitHub account

2. **Configure GitHub Secrets**:

   - Go to Repository Settings → Secrets and variables → Actions
   - Add `AWS_ROLE_ARN` with your OIDC role ARN

3. **Setup OIDC Role** (if not already configured):

```bash
# Create OIDC identity provider (one-time setup)
aws iam create-open-id-connect-provider \
  --url https://token.actions.githubusercontent.com \
  --client-id-list sts.amazonaws.com \
  --thumbprint-list 6938fd4d98bab03faadb97b34396831e3780aea1

# Create role with trust policy for your repo
# Replace YOUR_GITHUB_USERNAME and YOUR_REPO_NAME
cat > trust-policy.json << EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::$(aws sts get-caller-identity --query Account --output text):oidc-provider/token.actions.githubusercontent.com"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "token.actions.githubusercontent.com:aud": "sts.amazonaws.com"
        },
        "StringLike": {
          "token.actions.githubusercontent.com:sub": "repo:YOUR_GITHUB_USERNAME/YOUR_REPO_NAME:*"
        }
      }
    }
  ]
}
EOF

aws iam create-role \
  --role-name WristAgentGitHubActions \
  --assume-role-policy-document file://trust-policy.json

aws iam attach-role-policy \
  --role-name WristAgentGitHubActions \
  --policy-arn arn:aws:iam::aws:policy/PowerUserAccess
```

4. **Push to main branch** to trigger deployment

   > **Note:** The GitHub Actions workflow automatically runs `cdk bootstrap` before deployment, so no manual bootstrap is required.

## Step 4: Verify Deployment

After deployment completes, you should see outputs similar to:

```
Outputs:
WristAgentStack.FunctionUrl = https://abc123def456.lambda-url.us-west-2.on.aws/
WristAgentStack.TokenParameterName = /wrist-agent/client-token
```

Test the Function URL:

```bash
# Get the client token
TOKEN=$(aws ssm get-parameter --name "/wrist-agent/client-token" --with-decryption --query 'Parameter.Value' --output text)

# Test the endpoint
curl -X POST "https://YOUR_FUNCTION_URL" \
  -H "Content-Type: application/json" \
  -H "X-Client-Token: $TOKEN" \
  -d '{"text": "Create a note about testing the Wrist Agent system", "mode": "note"}'
```

Expected response:

```json
{
  "markdown": "# Testing the Wrist Agent System\n\n✅ Successfully tested the Wrist Agent integration...",
  "action": "note",
  "title": "Testing the Wrist Agent System",
  "dueISO": null,
  "tags": ["note", "testing"]
}
```

## Step 5: Configure Authentication Token

For security, update the default authentication token:

```bash
# Generate a secure token
TOKEN=$(openssl rand -base64 32)

# Update the SSM parameter
aws ssm put-parameter \
  --name "/wrist-agent/client-token" \
  --value "$TOKEN" \
  --type "String" \
  --overwrite

# Save the token securely - you'll need it for the Apple Shortcut
echo "Your Wrist Agent Token: $TOKEN"
```

## Step 6: Enable Bedrock Model Access

Ensure Claude Sonnet 4 is enabled in your AWS region:

1. Go to **AWS Bedrock Console** → **Model Access**
2. Request access to **Anthropic Claude Sonnet 4**
3. Wait for approval (usually instant for Claude models)

## Troubleshooting

### Common Issues

**Function URL returns 401 Unauthorized**

- Verify the token matches the SSM parameter
- Check the header name is exactly `X-Client-Token`

**Bedrock access denied errors**

- Ensure model access is enabled in Bedrock console
- Verify IAM permissions include `bedrock:InvokeModel`

**CDK deployment fails**

- Check AWS credentials and permissions
- Ensure CDK is bootstrapped: `npx cdk bootstrap`

**Lambda timeout errors**

- Increase timeout in `.env`: `LAMBDA_TIMEOUT=60`
- Redeploy with: `npx cdk deploy`

### Resource Cleanup

To remove all AWS resources:

```bash
cd cdk
npx cdk destroy
```

**Warning**: This will delete all resources including the SSM parameter with your token.

## Next Steps

Now that your infrastructure is deployed:

1. **[Configure Apple Shortcut](./apple-shortcut)** - Set up the iOS integration
2. **[Review Security Settings](./security)** - Understand the security model
3. **[Explore API Examples](./examples)** - Test different modes and features

## Cost Monitoring

Set up billing alerts to monitor costs:

```bash
# Create a CloudWatch alarm for estimated charges
aws cloudwatch put-metric-alarm \
  --alarm-name "WristAgentCosts" \
  --alarm-description "Alert when Wrist Agent costs exceed $50" \
  --metric-name EstimatedCharges \
  --namespace AWS/Billing \
  --statistic Maximum \
  --period 86400 \
  --threshold 50 \
  --comparison-operator GreaterThanThreshold \
  --dimensions Name=Currency,Value=USD \
  --evaluation-periods 1
```

Your Wrist Agent system is now ready! Head to the [Apple Shortcut guide](./apple-shortcut) to complete the integration.
