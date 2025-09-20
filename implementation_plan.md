# Implementation Plan

## [Overview]

Build a complete Apple Watch-to-AWS-Bedrock system that enables one-tap voice capture, processes requests through Claude 4 Sonnet with optional extended thinking, and automatically creates Notes, Reminders, or Calendar events.

This is a full greenfield implementation requiring CDK v2 TypeScript infrastructure with Lambda Function URLs, Go Lambda with Bedrock integration, GitHub Actions with OIDC for secure AWS deployment, Docusaurus documentation site with GitHub Pages, Apple Shortcut with Watch complication support, and complete security model using SSM Parameter Store. The system will be built as a monorepo with clear separation between infrastructure, Lambda code, documentation, and deployment automation.

## [Types]

Define comprehensive type system for request/response handling and AWS resource management.

**Go Lambda Types:**

```go
type Req struct {
    Text           string `json:"text"`
    Mode           string `json:"mode"`            // note|reminder|event|research|deepthink
    ThinkingTokens int    `json:"thinkingTokens"`  // 0..N for extended thinking
    MaxTokens      int    `json:"maxTokens"`       // default 800
}

type BedrockResponse struct {
    Content []Content `json:"content"`
    Usage   Usage     `json:"usage"`
}

type Content struct {
    Type string `json:"type"`
    Text string `json:"text"`
}
```

**Response JSON Structure:**

```json
{
  "markdown": "string - formatted content",
  "action": "note|reminder|event|none",
  "title": "string - extracted title",
  "dueISO": "string|null - ISO date when applicable",
  "tags": ["string array - extracted tags"]
}
```

**TypeScript CDK Types:**

```typescript
interface StackConfig {
  region: string;
  modelId: string;
  clientTokenParam: string;
}

interface LambdaEnvironment {
  BEDROCK_REGION: string;
  MODEL_ID: string;
  CLIENT_TOKEN_PARAM: string;
}
```

## [Files]

Create complete monorepo structure with all necessary configuration and source files.

**New files to be created:**

- `cdk/package.json` - CDK dependencies and scripts
- `cdk/tsconfig.json` - TypeScript configuration for CDK
- `cdk/cdk.json` - CDK app configuration
- `cdk/bin/wrist-agent.ts` - CDK app entry point
- `cdk/lib/wrist-agent-stack.ts` - Main infrastructure stack
- `lambda/go.mod` - Go module definition
- `lambda/main.go` - Go Lambda with Bedrock integration
- `lambda/main_test.go` - Lambda unit tests
- `docs/docusaurus.config.js` - Docusaurus configuration
- `docs/src/pages/index.js` - Documentation homepage
- `docs/docs/setup.md` - Human setup guide
- `docs/docs/apple-shortcut.md` - Apple Shortcut integration
- `docs/docs/security.md` - Security and cost management
- `docs/docs/agent-guide.md` - Agent-specific documentation
- `shortcut/README.md` - Apple Shortcut instructions and screenshots
- `.github/workflows/deploy-infra.yml` - AWS deployment workflow
- `.github/workflows/deploy-pages.yml` - Documentation deployment
- `.gitignore` - Comprehensive ignore patterns

**Existing files to be modified:**

- `README.md` - Add architecture overview and quick start
- `AGENTS.md` - Update with build commands and agent-specific notes

**Configuration updates needed:**

- Add `LICENSE` file (MIT recommended)
- Create `.env.example` for local development

## [Functions]

Implement core Lambda handler and CDK infrastructure functions.

**New Lambda functions in `/lambda/main.go`:**

- `handler(ctx context.Context, e events.LambdaFunctionURLRequest)` - Main HTTP handler
- `init()` - Initialize AWS clients (SSM, Bedrock) and cache token
- `validateRequest(*Req) error` - Input validation and business logic checks
- `callBedrock(ctx context.Context, req *Req) (string, error)` - Bedrock Claude Messages API
- `parseBedrockResponse([]byte) (string, error)` - Extract text from Claude response
- Response helpers: `resp(int, any)`, `respErr(error)` - JSON response formatting

**CDK stack functions in `/cdk/lib/wrist-agent-stack.ts`:**

- `WristAgentStack` constructor - Complete infrastructure provisioning
- Lambda configuration with GoFunction construct
- Function URL setup with CORS and public access
- SSM Parameter creation for client token
- IAM role configuration for Bedrock access

**Utility functions:**

- Environment variable helpers with defaults
- Error handling and logging utilities
- JSON marshaling/unmarshaling helpers

## [Classes]

Define CDK stack class and Go struct implementations.

**CDK Classes:**

- `WristAgentStack` class extending `cdk.Stack` in `/cdk/lib/wrist-agent-stack.ts`
  - Properties: tokenParam, fn, fUrl
  - Methods: GoFunction configuration with ARM64 architecture
  - Function URL with AuthType.NONE and CORS configuration
  - SSM StringParameter for client token storage
  - IAM PolicyStatement for Bedrock invoke permissions

**Go Structures:**

- `Req` struct - Request payload validation and JSON binding
- Response structures for API contract compliance
- AWS client wrapper structures for SSM and Bedrock
- Error types for proper HTTP status code mapping

**Documentation Classes:**

- Docusaurus React components for interactive guides
- Mermaid diagram components for architecture visualization

## [Dependencies]

Install all required packages for CDK, Lambda, and documentation.

**CDK Dependencies (`cdk/package.json`):**

```json
{
  "dependencies": {
    "aws-cdk-lib": "^2.150.0",
    "constructs": "^10.3.0",
    "@aws-cdk/aws-lambda-go-alpha": "^2.198.0-alpha.0"
  },
  "devDependencies": {
    "typescript": "^5.4.0"
  }
}
```

**Lambda Dependencies (`lambda/go.mod`):**

```go
require (
  github.com/aws/aws-lambda-go v1.46.0
  github.com/aws/aws-sdk-go-v2 v1.30.0
  github.com/aws/aws-sdk-go-v2/config v1.27.17
  github.com/aws/aws-sdk-go-v2/service/ssm v1.52.0
  github.com/aws/aws-sdk-go-v2/service/bedrockruntime v1.18.4
)
```

**Documentation Dependencies (`docs/package.json`):**

- `@docusaurus/core` - Documentation framework
- `@docusaurus/preset-classic` - Standard documentation preset
- `@docusaurus/theme-mermaid` - Mermaid diagram support

## [Testing]

Implement comprehensive testing strategy for Lambda, infrastructure, and integration.

**Lambda Testing (`lambda/main_test.go`):**

- `TestHandler` - Unit tests for main handler logic with mock events
- `TestValidateRequest` - Input validation edge cases
- `TestCallBedrock` - Mock Bedrock responses and error handling
- `TestAuthenticiation` - Header validation and SSM parameter checks
- Table-driven tests for different input modes and thinking token levels

**Infrastructure Testing:**

- CDK synthesis validation with `npx cdk synth`
- CloudFormation template snapshot tests
- Resource configuration assertions (Lambda timeout, memory, architecture)
- IAM policy validation for least privilege

**Integration Testing:**

- End-to-end curl examples for all request modes
- Apple Shortcut JSON payload validation
- Function URL CORS and authentication testing
- Performance testing for cold start optimization

## [Implementation Order]

Follow logical dependency order to minimize deployment conflicts and enable iterative testing.

1. **Project Structure Setup**

   - Create directory structure (`cdk/`, `lambda/`, `docs/`, `shortcut/`, `.github/workflows/`)
   - Add `.gitignore` with Node.js, Go, and AWS patterns
   - Create placeholder README updates

2. **CDK Infrastructure Foundation**

   - Initialize CDK app with TypeScript configuration
   - Implement `WristAgentStack` with Lambda, Function URL, SSM Parameter
   - Configure IAM roles and Bedrock permissions
   - Test with `cdk synth` and validate CloudFormation output

3. **Go Lambda Implementation**

   - Set up Go module with AWS SDK v2 dependencies
   - Implement main handler with Bedrock Messages API integration
   - Add extended thinking support with budget_tokens parameter
   - Implement request validation and JSON response formatting

4. **GitHub Actions Workflows**

   - Create `deploy-infra.yml` with OIDC configuration placeholder
   - Set up `deploy-pages.yml` for Docusaurus deployment
   - Configure proper permissions and concurrency controls

5. **Documentation Site (Docusaurus)**

   - Initialize Docusaurus with custom configuration
   - Create setup guides, API documentation, and troubleshooting
   - Add Mermaid diagrams for architecture visualization
   - Include agent-specific implementation notes

6. **Apple Shortcut Integration Guide**

   - Document complete Shortcut creation workflow
   - Provide JSON payload examples for all modes
   - Include Watch complication setup instructions
   - Add troubleshooting for common iOS Shortcuts issues

7. **Security Configuration and Testing**

   - Document SSM Parameter Store setup process
   - Create secure token generation guide
   - Test authentication flow and error handling
   - Validate CORS configuration for Function URL

8. **Integration Testing and Documentation**
   - End-to-end testing with real AWS deployment
   - Performance validation and cost estimation
   - Complete troubleshooting guide with common issues
   - Final documentation review and agent guide completion

Each step includes validation checkpoints and can be deployed incrementally. The Function URL approach eliminates API Gateway complexity while maintaining security through header-based authentication and SSM parameter management.
