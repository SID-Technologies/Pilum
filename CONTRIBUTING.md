# Contributing to Pilum

Thank you for your interest in contributing to Pilum! This guide will help you get started.

## Development Setup

### Prerequisites

- Go 1.23+
- [golangci-lint](https://golangci-lint.run/) v2.6+
- [pre-commit](https://pre-commit.com/)

### Getting Started

```bash
# Clone the repository
git clone https://github.com/sid-technologies/pilum.git
cd pilum

# Install development tools
make install-tools

# Build
make build

# Run tests
go test ./...

# Run linter
make lint
```

### Pre-commit Hooks

We use pre-commit hooks to ensure code quality. Install them with:

```bash
pre-commit install
```

The hooks run:
- `go mod tidy` - Ensures go.mod is clean
- `golangci-lint` - Linting with auto-fix
- Conventional commit validation

## Code Style

### Error Handling

Use the custom `lib/errors` package, not `fmt.Errorf`:

```go
// Good
import "github.com/sid-technologies/pilum/lib/errors"

if err != nil {
    return errors.Wrap(err, "failed to load service: %s", serviceName)
}

// Bad - will fail linting
import "fmt"

if err != nil {
    return fmt.Errorf("failed to load service: %w", err)
}
```

### Logging

Use the `lib/output` package for CLI output:

```go
import "github.com/sid-technologies/pilum/lib/output"

output.Info("Starting deployment...")
output.Success("Deployed %s", serviceName)
output.Error("Failed to deploy: %v", err)
output.Debug("Verbose info here")  // Only shown with --debug
```

Never use `log` or raw `fmt.Print` in library code.

### Linting

We use strict linting. Key rules:
- No unused variables or imports
- All errors must be handled
- No `log` package (use `lib/output`)
- No `fmt.Errorf` (use `lib/errors`)

Run the linter before submitting:

```bash
make lint
```

## Project Structure

The codebase follows standard Go project conventions:

- `cmd/` - CLI commands (Cobra)
- `lib/` - Core libraries (errors, output, orchestrator, etc.)
- `ingredients/` - Cloud provider implementations
- `recepies/` - Deployment workflow definitions

See [CLAUDE.md](CLAUDE.md) for detailed architecture documentation.

## Adding a New Provider

This is the most common contribution. See [docs/adding-a-provider.md](docs/adding-a-provider.md) for a complete walkthrough.

**Quick overview:**

1. **Create the recipe YAML** in `recepies/<provider>-recepie.yaml`
2. **Register handlers** in `lib/registry/commands.go` (if needed)
3. **Create ingredient package** in `ingredients/<provider>/` (if needed)

Most providers can be added with just a recipe YAML file using explicit commands.

## Pull Request Process

### Before Submitting

1. **Run tests**: `go test ./...`
2. **Run linter**: `make lint`
3. **Test your changes**: `./dist/pilum deploy --dry-run`

### Commit Messages

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add AWS ECS Fargate support
fix: correct region validation for GCP
docs: add troubleshooting guide
chore: update dependencies
```

Types:
- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation only
- `chore` - Maintenance (deps, CI, etc.)
- `refactor` - Code change that neither fixes a bug nor adds a feature
- `test` - Adding or updating tests

### PR Guidelines

- Keep PRs focused - one feature or fix per PR
- Update documentation if adding new features
- Add tests for new functionality
- Ensure CI passes before requesting review

## Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./lib/recepie/...

# Run with race detection
go test -race ./...
```

### Writing Tests

Place tests in `*_test.go` files alongside the code:

```go
// lib/recepie/loader_test.go
package recepie

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestLoadRecipe(t *testing.T) {
    recipe, err := LoadRecipe("testdata/valid-recipe.yaml")
    assert.NoError(t, err)
    assert.Equal(t, "my-recipe", recipe.Name)
}
```

Use `testdata/` directories for test fixtures.

### E2E Tests

End-to-end tests are in `test/e2e/`. Run with:

```bash
./test/e2e/run_tests.sh
```

## Getting Help

- Open an issue for bugs or feature requests
- Check existing issues before creating new ones
- For questions, open a discussion on GitHub

## License

By contributing, you agree that your contributions will be licensed under the BSL 1.1 license (converting to Apache 2.0 on December 3, 2028).
