# Contributing

Thanks for helping improve **wrist-agent**!

## How We Work

- **Architecture & guidelines:** See [AGENTS.md](./AGENTS.md) for project layout, commands, style, and security tips.
- Small, focused changes are best. Separate PRs for **Lambda (Go)**, **CDK (TS)**, and **docs** when possible.

## Getting Started

```bash
# Infra
cd cdk && npm ci && npx cdk synth

# Lambda
cd ../lambda && go test ./... && go build ./...

# Docs
cd ../docs && npm ci && npm run start
```
