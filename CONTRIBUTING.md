# Contributing

Thanks for helping improve **wrist-agent**!

## How We Work

- **Architecture & guidelines:** See [AGENTS.md](./AGENTS.md) for project layout, commands, style, and security tips.
- Small, focused changes are best. Separate PRs for **Lambda (Go)**, **CDK (TS)**, and **docs** when possible.

## Getting Started

```bash
# Infra
cd cdk && npm ci && npx cdk synth

# Lambda (main handler)
cd ../lambda && go mod tidy && go test ./... && go build ./...

# Lambda (authorizer)
cd ../lambda-authorizer && go mod tidy && go test ./... && go build ./...

# Docs
cd ../docs && npm ci && npm run start
```

## Testing

### Unit Tests

Unit tests run without AWS credentials and cover core logic:

```bash
# Run all Lambda tests (short mode - skips integration tests)
cd lambda && go test -v -short ./...

# Run all authorizer tests (short mode)
cd lambda-authorizer && go test -v -short ./...

# Run with coverage
cd lambda && go test -cover ./...
```

### Integration Tests

Integration tests require AWS credentials and a deployed stack. They are skipped by default when using `-short` flag.

**Prerequisites:**
1. AWS credentials configured (`aws configure` or environment variables)
2. Stack deployed (`npx cdk deploy` in `cdk/` directory)
3. SSM parameter `/wrist-agent/client-token` exists with a valid token

**Running integration tests:**

```bash
# Run ALL tests including integration (requires AWS credentials)
cd lambda-authorizer && go test -v ./...

# The integration test will attempt to:
# - Connect to SSM Parameter Store
# - Retrieve the client token
# - Validate the authorizer logic
```

**Note:** Integration tests may fail if:
- AWS credentials are not configured
- SSM parameter doesn't exist
- IAM permissions are insufficient

### CDK Tests

```bash
cd cdk
npm test                    # Run Jest tests
npx cdk synth --quiet       # Verify synthesis
npx cdk diff                # Preview changes
```

## Pull Request Guidelines

1. **Run all tests** before submitting
2. **Follow Conventional Commits** format (e.g., `feat:`, `fix:`, `docs:`)
3. **Update documentation** if changing behavior
4. **Keep changes focused** - separate PRs for different components

## Code Style

- **Go:** Follow `gofmt` and `go vet` standards
- **TypeScript:** Follow ESLint rules in CDK package
- **Documentation:** Keep markdown clean and consistent
