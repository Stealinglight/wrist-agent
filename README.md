# Wrist Agent

A seamless Apple Watch to AWS Bedrock integration system that enables one-tap voice capture, intelligent processing through Claude Haiku 4.5, and automatic creation of Notes, Reminders, or Calendar events.

## 🚀 Quick Start

1. **[Deploy Infrastructure](docs/docs/setup.md)** - AWS CDK deployment to us-west-2
2. **[Configure Security](docs/docs/security.md)** - Set up authentication tokens
3. **[Install Shortcut](docs/docs/apple-shortcut.md)** - Add to Apple Watch

## ✨ Key Features

- **🎙️ One-Tap Voice Capture**: Direct from Apple Watch with complication support
- **🧠 AI Processing**: Claude Haiku 4.5 with optional extended thinking (up to 65K thinking tokens)
- **📱 Native Integration**: Seamlessly creates Notes, Reminders, and Calendar events
- **🔒 Secure**: API Gateway with Lambda Authorizer and SSM Parameter Store
- **💰 Cost-Optimized**: Efficient architecture (~$16-32/month)
- **⚡ Fast**: ARM64 Lambda with sub-second response times
- **🚀 CI/CD Ready**: GitHub Actions with OIDC deployment

## 🏗️ Architecture

```mermaid
graph LR
    A[Apple Watch] --> B[Apple Shortcut]
    B --> C[API Gateway]
    C --> D[Lambda Authorizer]
    D --> E{Valid Token?}
    E -->|Yes| F[Go Lambda]
    E -->|No| G[401 Denied]
    F --> H[Bedrock Claude 4]
    H --> I[Structured Response]
    I --> J[iOS Apps]

    K[SSM Parameter] --> D
    L[CDK TypeScript] --> C
    M[GitHub Actions] --> L
```

**Components:**

- **Apple Watch + Shortcuts**: Voice capture and iOS app integration
- **API Gateway**: REST API with rate limiting and request validation
- **Lambda Authorizer**: Token validation with 5-minute caching
- **AWS Lambda**: Go 1.22+ runtime (ARM64)
- **AWS Bedrock**: Claude Haiku 4.5 with Messages API
- **CDK v2**: TypeScript infrastructure as code
- **GitHub Actions**: OIDC-based CI/CD pipeline
- **Docusaurus**: Dual documentation (human + agent guides)

## 🎯 Use Cases

### 📝 Note Taking

Convert voice recordings into well-formatted notes with titles and tags.

### ⏰ Reminders

Create time-based reminders with automatic date/time extraction.

### 📅 Calendar Events

Schedule events with intelligent date/time parsing.

### 🔍 Research Mode

Get detailed, well-researched responses with sources.

### 🤔 Deep Thinking

Enable extended reasoning with up to 65K thinking tokens for complex queries.

## 🛠️ Technology Stack

### AWS Infrastructure

- **Lambda**: Go 1.22+ with provided.al2 runtime
- **Function URLs**: Direct HTTPS endpoints with CORS
- **Bedrock**: Claude Haiku 4.5 (anthropic.claude-haiku-4-5-20251001-v1:0)
- **SSM Parameter Store**: Secure token management
- **IAM**: Least privilege access control

### Development Tools

- **CDK v2**: TypeScript infrastructure
- **Go**: Lambda runtime with AWS SDK v2
- **GitHub Actions**: OIDC deployment
- **Docusaurus**: Documentation site

### Apple Integration

- **Shortcuts**: Custom shortcut with Watch complication
- **Voice Recognition**: Native iOS dictation
- **App Integration**: Notes, Reminders, Calendar

## 📋 Prerequisites

- AWS Account with Bedrock access
- AWS CLI configured
- Node.js 18+ and npm
- Go 1.22+
- Apple Developer Account (free tier sufficient)
- Apple Watch with watchOS 6+

## 🔧 Installation

### 1. Clone Repository

```bash
git clone https://github.com/your-username/wrist-agent.git
cd wrist-agent
```

### 2. Deploy Infrastructure

```bash
# Install CDK dependencies
cd cdk && npm install

# Install Go dependencies
cd ../lambda && go mod tidy

# Deploy to AWS
cd ../cdk
npx cdk bootstrap  # First time only
npx cdk deploy
```

### 3. Configure Security

```bash
# Generate secure token
TOKEN=$(openssl rand -base64 32)

# Update SSM parameter
aws ssm put-parameter \
  --name "/wrist-agent/client-token" \
  --value "$TOKEN" \
  --type "SecureString" \
  --overwrite

echo "Your token: $TOKEN"
```

### 4. Setup Apple Shortcut

1. Get API Endpoint from CDK output
2. Create Apple Shortcut with HTTP POST action
3. Configure headers: `X-Client-Token` and `Content-Type: application/json`
4. Add Watch complication

**Full setup guide: [docs/docs/setup.md](docs/docs/setup.md)**

## 🧪 Testing

### Test the API

```bash
# Get your configuration
API_ENDPOINT=$(aws cloudformation describe-stacks \
  --stack-name WristAgentStack \
  --query 'Stacks[0].Outputs[?OutputKey==`InvokeEndpoint`].OutputValue' \
  --output text)

TOKEN=$(aws ssm get-parameter \
  --name "/wrist-agent/client-token" \
  --with-decryption \
  --query 'Parameter.Value' \
  --output text)

# Test API
curl -X POST "$API_ENDPOINT" \
  -H "Content-Type: application/json" \
  -H "X-Client-Token: $TOKEN" \
  -d '{"text": "Create a note about testing", "mode": "note"}'
```

### Run Unit Tests

```bash
# Go Lambda tests
cd lambda && go test -v

# CDK tests
cd ../cdk && npm test
```

## 📊 Request/Response Format

### Request

```json
{
  "text": "Create a reminder to call John tomorrow at 2pm",
  "mode": "reminder",
  "maxTokens": 800,
  "thinkingTokens": 0
}
```

### Response

```json
{
  "markdown": "# Call John\n\nReminder to call John tomorrow at 2pm",
  "action": "reminder",
  "title": "Call John",
  "dueISO": "2025-01-16T14:00:00Z",
  "startISO": null,
  "endISO": null,
  "location": null,
  "url": null,
  "notes": null,
  "tags": ["reminder", "call"]
}
```

Event responses include `startISO`, `endISO`, `location`, `url`, and `notes` when available.

## 🔒 Security

- **API Gateway**: Request validation and rate limiting (10 req/sec, 20 burst)
- **Lambda Authorizer**: Token validation with 5-minute caching
- **Authentication**: Shared token in SSM Parameter Store (SecureString)
- **Transport**: HTTPS with TLS 1.2+
- **IAM**: Least privilege permissions
- **Bedrock Auth**: IAM role credentials only (no API keys)
- **No persistent storage**: Stateless Lambda execution

**Security guide: [docs/docs/security.md](docs/docs/security.md)**

## 💰 Cost Estimation

Monthly costs for moderate usage (100 requests/day):

| Service            | Cost              |
| ------------------ | ----------------- |
| API Gateway        | ~$0.35            |
| Lambda Invocations | ~$0.50            |
| Bedrock Claude 4   | ~$15-30           |
| SSM Parameter      | ~$0.05            |
| **Total**          | **~$16-32/month** |

## 📚 Documentation

- **[Getting Started](docs/docs/setup.md)** - Complete setup guide
- **[Apple Shortcut](docs/docs/apple-shortcut.md)** - iOS integration
- **[Security](docs/docs/security.md)** - Authentication and best practices
- **[Agent Guide](docs/docs/agent-guide.md)** - Implementation details for AI agents
- **[API Reference](docs/docs/api.md)** - Complete API documentation

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Make your changes and test thoroughly
4. Submit a pull request

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.

## 🔄 CI/CD

GitHub Actions automatically:

- **Tests**: Run Go and CDK tests on PRs
- **Bootstrap**: Run `cdk bootstrap` to prepare AWS environment
- **Deploys**: Deploy infrastructure on main branch pushes
- **Security**: Scan for vulnerabilities with Trivy
- **Documentation**: Deploy docs to GitHub Pages

Uses OIDC for secure AWS deployment (no long-lived credentials).

## 🐛 Troubleshooting

### Common Issues

**401 Unauthorized**

- Verify token matches SSM parameter value
- Check header name is exactly `X-Client-Token`
- Ensure API Gateway authorizer is deployed

**Bedrock Access Denied**

- Enable Claude Haiku 4.5 in Bedrock console
- Verify IAM permissions

**CDK Deployment Fails**

- Ensure CDK is bootstrapped: `npx cdk bootstrap` (GitHub Actions handles this automatically)
- Check AWS credentials and permissions

**More help: [docs/docs/troubleshooting.md](docs/docs/troubleshooting.md)**

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

---

**Ready to transform your Apple Watch into an AI-powered productivity tool?**

Start with the [Setup Guide](docs/docs/setup.md) and begin capturing ideas with a tap on your wrist! 🚀
