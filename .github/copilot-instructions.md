# Wrist Agent - GitHub Copilot Instructions

This repository contains an Apple Watch to AWS Bedrock integration system that enables voice capture and AI processing through Claude Haiku 4.5.

## Project Overview

**Wrist Agent** is a seamless integration between Apple Watch and AWS Bedrock that allows users to capture voice notes with one tap and have them intelligently processed by Claude AI to create Notes, Reminders, or Calendar events.

### Key Components

- **Apple Watch + Shortcuts**: Voice capture and iOS app integration
- **AWS Lambda (Go)**: Custom runtime processing requests
- **AWS Bedrock**: Claude Haiku 4.5 with Messages API
- **CDK v2 (TypeScript)**: Infrastructure as code
- **GitHub Actions**: OIDC-based CI/CD pipeline
- **Docusaurus**: Documentation site

## Project Structure

```
.
├── cdk/              # AWS CDK v2 (TypeScript) infrastructure
├── lambda/           # Go Lambda function (custom runtime)
├── docs/             # Docusaurus documentation site
├── shortcut/         # Apple Shortcuts recipe and assets
├── .github/
│   └── workflows/    # CI/CD workflows (AWS OIDC deploy, Pages)
├── AGENTS.md         # Detailed repository guidelines
└── CONTRIBUTING.md   # Contribution guidelines
```

## Build, Test, and Development Commands

### Infrastructure (CDK)

```bash
cd cdk
npm ci                                   # Install dependencies
npx cdk synth                           # Synthesize CloudFormation
npx cdk deploy --require-approval never # Deploy to AWS
npx cdk diff                            # Show changes
npx cdk destroy                         # Clean up resources
```

### Lambda (Go)

```bash
cd lambda
go test ./...    # Run all tests
go vet ./...     # Static analysis
go build ./...   # Build locally
```

**Test files**: `*_test.go` with `TestXxx(t *testing.T)` functions

### Documentation

```bash
cd docs
npm ci              # Install dependencies
npm run start       # Development server (http://localhost:3000)
npm run build       # Production build
```

### Invoke Lambda (after deployment)

```bash
curl -X POST "$FUNCTION_URL" \
  -H "Content-Type: application/json" \
  -H "X-Client-Token: <client token>" \
  -d '{"text":"example note","mode":"note","thinkingTokens":0,"maxTokens":800}'
```

## Code Style & Conventions

### TypeScript (cdk/)

- **Indentation**: 2 spaces
- **Naming**: 
  - `camelCase` for variables and functions
  - `PascalCase` for classes, interfaces, and types
- **Formatting**: Follow ESLint + Prettier if configured
- **Best practice**: Use AWS CDK L2 constructs when available

### Go (lambda/)

- **Formatting**: Keep code `gofmt` and `go vet` clean
- **Naming**:
  - Short, lowercase package names
  - Exported identifiers in `CamelCase`
  - Unexported identifiers in `camelCase`
- **Error handling**: Return errors, don't panic (except for truly exceptional cases)
- **Testing**: Table-driven tests preferred

## Testing Guidelines

### Go Tests

- Place tests in `*_test.go` files
- Function signature: `func TestXxx(t *testing.T)`
- Run: `go test ./...` at repo root or package level
- Use `t.Helper()` for test helper functions
- Prefer table-driven tests for multiple cases

### Infrastructure Tests

- Add CDK assertion/snapshot tests as needed
- Review `cdk diff` output before deploying
- Test changes in a non-production environment first

## Commit & Pull Request Guidelines

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat: add Lambda Function URL support`
- `fix: correct SSM parameter name handling`
- `docs: update Apple Shortcut setup guide`
- `chore: update dependencies`
- `refactor: simplify Bedrock API client`
- `test: add unit tests for request validation`

### Pull Requests

- **Description**: Include clear description of changes
- **Linked issues**: Reference related issues (e.g., "Closes #123")
- **Evidence**: Provide test logs or screenshots for docs changes
- **CI**: Ensure all checks pass before requesting review
- **Scope**: Keep PRs focused - separate Lambda, CDK, and docs changes when possible

## Security & Configuration

### Security Best Practices

- **Never commit secrets** to the repository
- **Authentication**: Use header-based auth with SSM Parameter Store
- **Deployment**: GitHub Actions uses OIDC (short-lived credentials) assuming an AWS role
- **Lambda**: Function URL requires `X-Client-Token` header for authentication
- **Token rotation**: Rotate client tokens in SSM periodically
- **IAM**: Follow least privilege principle

### Configuration Management

- Store sensitive values in AWS SSM Parameter Store
- Function URL: `/wrist-agent/client-token`
- Use CDK `StringParameter.valueFromLookup()` for configuration
- Never hardcode AWS account IDs or regions in code

## API Reference

### Request Format

```json
{
  "text": "User input text from voice capture",
  "mode": "note|reminder|event|research",
  "maxTokens": 800,
  "thinkingTokens": 0
}
```

### Response Format

```json
{
  "markdown": "# Title\n\nFormatted content",
  "action": "note|reminder|event",
  "title": "Item title",
  "dueISO": "2025-01-16T14:00:00Z",
  "startISO": "2025-01-16T14:00:00Z",
  "endISO": "2025-01-16T15:00:00Z",
  "location": "Location string",
  "url": "https://example.com",
  "notes": "Additional notes",
  "tags": ["tag1", "tag2"]
}
```

## Common Tasks

### Adding a New Feature

1. Create a feature branch from `main`
2. Update relevant code (Lambda, CDK, or docs)
3. Add or update tests
4. Run tests locally: `go test ./...` or `npm test`
5. Update documentation if needed
6. Create PR with conventional commit message
7. Ensure CI passes

### Modifying Lambda Function

1. Edit code in `lambda/main.go` or related files
2. Add/update tests in `*_test.go` files
3. Run: `go test ./... && go vet ./...`
4. Update CDK if Lambda config changes (memory, timeout, etc.)
5. Test with `curl` after deploying

### Updating Infrastructure

1. Modify CDK stacks in `cdk/lib/`
2. Run: `npx cdk synth` to verify CloudFormation
3. Check changes: `npx cdk diff`
4. Update tests if needed
5. Deploy: `npx cdk deploy`

### Documentation Changes

1. Edit Markdown files in `docs/docs/`
2. Run dev server: `npm run start`
3. Review changes in browser
4. Build: `npm run build` to verify
5. GitHub Actions will auto-deploy to Pages

## CI/CD Pipeline

### GitHub Actions Workflows

- **AWS Deployment**: Triggered on push to `main`
  - Bootstraps CDK (if needed)
  - Deploys infrastructure
  - Uses OIDC for authentication
- **Documentation**: Deploys Docusaurus to GitHub Pages
- **Security Scanning**: Runs Trivy vulnerability scanner

### Environment Variables (GitHub Secrets)

- `AWS_ROLE_ARN`: IAM role for OIDC deployment
- `AWS_REGION`: Target AWS region (default: us-west-2)

## Architecture Notes

### Lambda Function

- **Runtime**: Go 1.22+ with `provided.al2` custom runtime
- **Architecture**: ARM64 for better price/performance
- **Handler**: Processes JSON requests, calls Bedrock, returns structured JSON
- **Timeout**: Configure based on expected Bedrock latency
- **Memory**: 512MB default (adjust based on usage)

### Bedrock Integration

- **Model**: `anthropic.claude-haiku-4-5-20251001-v1:0`
- **API**: Messages API with system prompts for structured output
- **Authentication**: IAM role credentials (no API keys)
- **Region**: us-west-2 (Bedrock availability)

### Function URLs

- **Direct HTTPS endpoint** for Lambda (no API Gateway needed)
- **CORS**: Configured for Apple Shortcuts
- **Auth**: Custom header-based authentication
- **Cost**: ~$0.40 per million requests vs API Gateway

## Troubleshooting

### Common Issues

**401 Unauthorized**
- Verify token in SSM matches header value
- Check header name is exactly `X-Client-Token`

**Bedrock Access Denied**
- Enable Claude Haiku 4.5 in AWS Bedrock console
- Verify Lambda execution role has `bedrock:InvokeModel` permission

**CDK Deployment Fails**
- Ensure CDK is bootstrapped: `npx cdk bootstrap`
- Check AWS credentials and permissions
- Review CloudFormation events in AWS Console

**Tests Failing**
- Run `go mod tidy` to ensure dependencies are up to date
- Check Go version (requires 1.22+)
- Review test output for specific failures

## Resources

- **Main Documentation**: See [docs/docs/](../docs/docs/) for user guides
- **Agent Guidelines**: See [AGENTS.md](../AGENTS.md) for detailed development info
- **Contributing**: See [CONTRIBUTING.md](../CONTRIBUTING.md) for workflow
- **AWS CDK**: https://docs.aws.amazon.com/cdk/
- **Go AWS SDK v2**: https://aws.github.io/aws-sdk-go-v2/
- **Bedrock API**: https://docs.aws.amazon.com/bedrock/

## Tips for Working with This Codebase

1. **Start small**: Make focused, incremental changes
2. **Test locally**: Always run tests before pushing
3. **Review diffs**: Use `cdk diff` before deploying infrastructure changes
4. **Check logs**: Review CloudWatch logs for Lambda debugging
5. **Security first**: Never commit credentials or tokens
6. **Document changes**: Update relevant docs when changing behavior
7. **Follow patterns**: Match existing code style and structure
8. **Ask for help**: Refer to AGENTS.md and CONTRIBUTING.md for guidance

---

For more detailed information, see:
- [AGENTS.md](../AGENTS.md) - Comprehensive repository guidelines
- [CONTRIBUTING.md](../CONTRIBUTING.md) - Contribution workflow
- [docs/](../docs/) - Full documentation site
