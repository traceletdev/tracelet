# Contributing to Tracelet

Thank you for your interest in contributing to Tracelet!

## Development Setup

1. Clone the repository
2. Build the CLI: `go build -o tracelet ./cmd/tracelet`
3. Run tests: `go test ./tests/integration -v`

## Project Structure

- `cmd/tracelet/` - CLI entrypoint
- `internal/` - Core Go packages
- `adapters/` - Framework-specific collectors
- `ui/vscode-extension/` - VS Code extension
- `tests/` - Integration tests and fixtures
- `docs/` - Documentation

## Git Workflow

We use Git Flow:
- `main` - Stable releases
- `develop` - Active development
- Feature branches - `feature/description`

## Commit Messages

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add new rule for duplicate chunks
fix: handle missing stats file gracefully
docs: update config reference
```

## Adding New Rules

1. Create rule function in `internal/lint/rules/`
2. Register in `internal/lint/lint.go`
3. Add to config schema if needed
4. Write tests in `tests/integration/`
5. Document in `docs/rules.md`

## Adding Adapters

1. Create adapter file in `adapters/`
2. Output `.tracelet/stats.json` with unified schema
3. Test with example project in `tests/fixtures/`
4. Document usage

## Testing

- Write integration tests in `tests/integration/`
- Use fixtures in `tests/fixtures/`
- Run: `go test ./tests/integration -v`

## Documentation

- Update relevant docs in `docs/`
- Keep README.md up to date
- Add examples where helpful

## Code Style

- Go: Follow standard `gofmt` style
- TypeScript: Use default VS Code settings
- Comments: Document public APIs

## Questions?

Open an issue or start a discussion!

