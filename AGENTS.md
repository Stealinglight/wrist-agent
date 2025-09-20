Here’s the full contents for AGENTS.md:

Repository Guidelines

Project Structure & Module Organization
• cdk/ – AWS CDK v2 (TypeScript) app. Use cdk synth, cdk deploy, cdk diff, cdk destroy. ￼
• lambda/ – Go Lambda (custom runtime). Unit tests live in \*\_test.go. ￼
• docs/ – Docusaurus site (GitHub Pages). Dev: npm run start; build: npm run build. ￼
• shortcut/ – Apple Shortcuts recipe and assets.
• .github/workflows/ – CI/CD (AWS OIDC deploy) and Pages. ￼ ￼

Build, Test, and Development Commands

Infra

cd cdk
npm ci
npx cdk synth
npx cdk deploy --require-approval never

(CDK CLI usage and deploy flow.) ￼

Lambda (Go)

cd lambda
go test ./... # run all package tests
go vet ./... # static checks
go build ./... # local build

(Go tests and patterns.) ￼ ￼

Docs

cd docs
npm ci
npm run start # dev server
npm run build # production build

(Docusaurus CLI.) ￼

Invoke (once deployed)

curl -X POST "$FUNCTION_URL" \
 -H "Content-Type: application/json" \
 -H "X-Agent-Token: <client token>" \
 -d '{"text":"example note","mode":"note","thinkingTokens":0,"maxTokens":800}'

(Function URLs are built-in HTTPS endpoints for Lambda.) ￼

Code Style & Naming
• TypeScript (cdk/): 2-space indent, camelCase for vars, PascalCase for types/classes. Prefer ESLint + Prettier if configured.
• Go (lambda/): keep code gofmt/go vet clean; short, lower-case package names; exported identifiers in CamelCase.

Testing
• Go: tests in *\_test.go, functions TestXxx(t *testing.T). Run go test ./... at repo or package root. ￼
• Infra: add assertion tests as needed (e.g., snapshot/diff) before deploy.

Commit & Pull Request Guidelines
• Commits: follow Conventional Commits (e.g., feat: add Lambda URL, fix: correct SSM param name, docs: update Shortcut steps). ￼
• PRs: include a clear description, linked issue (if any), test evidence (logs/screenshots for docs), and pass CI.

Security & Configuration Tips
• No secrets in repo. Deploy via GitHub Actions with OIDC (short-lived creds) assuming an AWS role. ￼
• Function URL access: restrict with AuthType/resource policies and a shared header; rotate the token in SSM. ￼

⸻

If you want, I can drop this into your repo and add a short “CONTRIBUTING.md” that links back here.

# Repository Guidelines

## Project Structure & Module Organization

- `cdk/` — AWS CDK v2 (TypeScript) app and stacks. Common commands: `cdk synth`, `cdk deploy`, `cdk diff`, `cdk destroy`.
- `lambda/` — Go Lambda (custom runtime). Source in `main.go`; tests live in `*_test.go`.
- `docs/` — Docs site (e.g., Docusaurus) published via GitHub Pages.
- `shortcut/` — Apple Shortcut recipe, JSON export, and screenshots.
- `.github/workflows/` — CI/CD workflows (deploy infra via OIDC; build/publish docs).

## Build, Test, and Development Commands

**Infra**

```bash
cd cdk
npm ci
npx cdk synth
npx cdk deploy --require-approval never
```

**Lambda (Go)**

```bash
cd lambda
go test ./...     # run all tests
go vet ./...      # static checks
go build ./...    # local build
```

**Docs**

```bash
cd docs
npm ci
npm run start     # dev server
npm run build     # production build
```

**Invoke (after deploy)**

```bash
curl -X POST "$FUNCTION_URL" \
  -H "Content-Type: application/json" \
  -H "X-Agent-Token: <client token>" \
  -d '{"text":"example note","mode":"note","thinkingTokens":0,"maxTokens":800}'
```

## Code Style & Naming

- **TypeScript (cdk/):** 2‑space indent; `camelCase` for vars/functions; `PascalCase` for classes/types; prefer ESLint + Prettier if configured.
- **Go (lambda/):** `gofmt` clean; short, lower‑case package names; exported identifiers in `CamelCase`.

## Testing

- **Go:** place tests in `*_test.go` with `TestXxx(t *testing.T)`. Run `go test ./...` at repo or package root.
- **Infra:** add assertion/diff tests as needed before deploy.

## Commit & Pull Request Guidelines

- **Commits:** follow Conventional Commits (e.g., `feat: add Lambda Function URL`, `fix: correct SSM param name`, `docs: update Shortcut steps`).
- **PRs:** include a clear description, linked issue (if any), screenshots or logs when relevant, and ensure CI passes.

## Security & Configuration Tips

- Never commit secrets. Deploy via GitHub Actions OIDC (short‑lived creds) assuming an AWS role.
- Protect the Lambda Function URL: require the shared `X-Agent-Token` header (from SSM) and rotate it periodically. Avoid putting the Lambda in a VPC to prevent NAT costs.

## Agent‑Specific Notes

- When committing as an AI assistant, add a trailer with context (e.g., `Co-authored-by: agent-name`). Keep changes small and atomic; prefer separate PRs for Lambda vs. CDK vs. docs updates.
