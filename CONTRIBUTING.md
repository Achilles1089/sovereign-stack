# Contributing to Sovereign Stack

Thanks for your interest in Sovereign Stack! Here's how to contribute.

## Quick Start for Contributors

```bash
# Clone
git clone https://github.com/Achilles1089/sovereign-stack.git
cd sovereign-stack

# Build
go build -o sovereign .

# Test
go test ./...

# Lint
go vet ./...
```

## Adding an App

The easiest way to contribute is adding a new app to the marketplace.

1. Open `internal/apps/installer.go`
2. Add a new `AppManifest` to the `BuiltinApps` slice:

```go
{
    Name: "your-app", DisplayName: "Your App", Description: "What it does",
    Category: "category", Version: "1.0", Website: "https://example.com",
    Compose: AppCompose{
        Image: "your/image:tag",
        Ports: []string{"HOST:CONTAINER"},
        Volumes: []string{"app_data:/data"},
    },
    CaddyRoute: &CaddyRoute{Path: "/your-app", Port: HOST_PORT},
},
```

3. Run `go test ./internal/apps/...` — tests auto-validate field completeness
4. Submit a PR

## Code Structure

| Package | Purpose |
|---|---|
| `cmd/` | CLI commands (Cobra) |
| `internal/ai/` | Ollama client, model catalog, system prompt |
| `internal/apps/` | App marketplace (add apps here) |
| `internal/audit/` | JSONL audit logging |
| `internal/backup/` | Restic backup wrapper |
| `internal/config/` | YAML configuration |
| `internal/docker/` | Docker Compose generation |
| `internal/hardware/` | Hardware detection |
| `internal/mesh/` | WireGuard mesh networking |
| `internal/rbac/` | Role-based access control |
| `internal/server/` | REST API + dashboard server |
| `internal/sso/` | Authentik SSO integration |

## Pull Request Guidelines

- Run `go test ./...` and `go vet ./...` before submitting
- One feature per PR
- Include tests for new functionality
- Follow existing code style (gofmt is your friend)

## Reporting Issues

Use [GitHub Issues](https://github.com/Achilles1089/sovereign-stack/issues) with the appropriate template.

## License

By contributing, you agree that your contributions will be licensed under AGPL-3.0.
