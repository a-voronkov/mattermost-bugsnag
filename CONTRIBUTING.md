# Contributing to Mattermost Bugsnag Plugin

Thank you for your interest in contributing! This guide covers everything you need to get started.

## Project Structure

```
mattermost-bugsnag/
├── server/                 # Go backend (Mattermost plugin)
│   ├── plugin.go           # Main plugin entry, lifecycle hooks, HTTP routing
│   ├── configuration.go    # Plugin settings and validation
│   ├── webhook.go          # Bugsnag webhook handler
│   ├── actions.go          # Interactive button handlers
│   ├── mm_client.go        # Mattermost API wrapper
│   ├── mappings.go         # User/channel mapping helpers
│   ├── api/                # REST API endpoints
│   ├── bugsnag/            # Bugsnag API client
│   ├── formatter/          # Post/card builder
│   ├── kvkeys/             # KV store key constants
│   ├── scheduler/          # Periodic sync runner
│   └── store/              # KV store abstraction
├── webapp/                 # React frontend (planned)
│   └── src/
├── docs/                   # Documentation
│   ├── local-testing.md    # Local dev environment setup
│   ├── sample-payloads.md  # Example webhook/action payloads
│   └── todo.md             # Implementation checklist
├── plugin.json             # Plugin manifest
└── README.md               # Project overview
```

## Prerequisites

- **Go 1.22+** — server plugin is written in Go
- **Node.js 18+** — for webapp development (when implemented)
- **Docker** — for local Mattermost instance
- **mmctl** — Mattermost CLI for plugin management

## Development Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/a-voronkov/mattermost-bugsnag.git
   cd mattermost-bugsnag
   ```

2. **Install Go dependencies**
   ```bash
   cd server
   go mod tidy
   ```

3. **Run tests**
   ```bash
   go test ./...
   ```

4. **Start local Mattermost** (see `docs/local-testing.md` for details)
   ```bash
   docker run -d --name mm-bugsnag-dev \
     -p 8065:8065 \
     mattermost/mattermost-team-edition:release-9.11
   ```

## Code Style

### Go

- Follow standard [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- Use `gofmt` for formatting
- Run `go vet ./...` before committing
- Keep functions small and focused
- Add comments for exported functions and types
- Error messages should be lowercase without trailing punctuation

### Naming Conventions

- **Files**: lowercase with underscores (`post_builder.go`)
- **Packages**: short, lowercase, single word (`formatter`, `store`)
- **Interfaces**: descriptive names, avoid `-er` suffix when not natural
- **Constants**: use `camelCase` for unexported, `PascalCase` for exported

## Testing

All new code should include tests. Run the test suite from `server/`:

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific package tests
go test ./bugsnag/...

# Run with coverage
go test -cover ./...
```

### Test Files

- Place tests in the same package as the code being tested
- Name test files with `_test.go` suffix
- Use table-driven tests where appropriate
- Mock external dependencies (Mattermost API, Bugsnag API)

## Pull Request Process

1. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** with clear, atomic commits

3. **Ensure tests pass**
   ```bash
   cd server && go test ./...
   ```

4. **Push and create a PR**
   ```bash
   git push origin feature/your-feature-name
   ```

5. **PR requirements**:
   - Clear description of changes
   - Tests for new functionality
   - No breaking changes without discussion
   - All CI checks passing

## Commit Messages

Follow conventional commit format:

```
type(scope): short description

Longer description if needed.
```

**Types**: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`

**Examples**:
- `feat(webhook): add environment filter support`
- `fix(actions): handle missing user mapping gracefully`
- `docs: update local testing instructions`

## Key Interfaces

When contributing, familiarize yourself with these core interfaces:

- `store.KVStore` — abstraction for Mattermost KV storage
- `bugsnag.Client` — Bugsnag API client methods
- `formatter.PostBuilder` — error card construction

## Getting Help

- Check existing issues and PRs
- Review `docs/todo.md` for planned work
- Open an issue for questions or proposals

## License

By contributing, you agree that your contributions will be licensed under the same license as the project.

