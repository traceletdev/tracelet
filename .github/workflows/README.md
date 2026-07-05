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

## npm Publishing

Authenticates via [npm Trusted Publishing](https://docs.npmjs.com/trusted-publishers/)
(OIDC) — no secret required. Each of the 4 packages needs a Trusted Publisher configured
on npmjs.com pointing at this repo and `release.yml`. See `RELEASE.md` for setup
(including the bootstrap step for packages that have never been published).

## Release Process

1. Update versions in all `package.json` files
2. Commit and push changes
3. Create and push tag:

   ```bash
   git tag -a v0.5.0 -m "Release v0.5.0"
   git push origin v0.5.0
   ```

4. Workflow automatically builds and publishes
