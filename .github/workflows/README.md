# GitHub Actions Workflows

## Workflows

### `test.yml`

Runs on: PR and pushes to `main`/`develop`

- Runs Go tests
- Runs integration tests
- Builds CLI binary
- Verifies binary works

### `tracelet.yml`

Runs on: PR and pushes to `main`

- Builds tracelet CLI
- Runs `tracelet ci --compare` to check against baseline
- Posts markdown report as PR comment
- Updates baseline on `main` branch pushes

### `release.yml`

Runs on: Tag push (e.g., `v0.5.0`)

- Builds multi-platform binaries using GoReleaser
- Packages binaries for npm
- Updates package versions
- Publishes to npm:
  - `@traceletdev/cli` (main package)
  - `@traceletdev/next` (Next.js plugin)
  - `@traceletdev/vite` (Vite plugin)
  - `@traceletdev/react` (React render instrumentation)
- Creates GitHub release with binaries

## Required Secrets

- `NPM_TOKEN` - npm automation token for publishing packages
  - Get from: <https://www.npmjs.com/settings/YOUR_USERNAME/tokens>
  - Type: Automation
  - Add to: Repository Settings → Secrets → Actions

## Release Process

1. Update versions in all `package.json` files
2. Commit and push changes
3. Create and push tag:

   ```bash
   git tag -a v0.5.0 -m "Release v0.5.0"
   git push origin v0.5.0
   ```

4. Workflow automatically builds and publishes
