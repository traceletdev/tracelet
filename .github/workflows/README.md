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
- Packages binaries into the 6 platform packages
- Updates package versions
- Publishes all 10 packages to npm (see RELEASE.md)
- Creates GitHub release with binaries

## npm Publishing

Uses [Trusted Publishing](https://docs.npmjs.com/trusted-publishers/) (OIDC) — no
secret required. See `RELEASE.md` for setup.

## Release Process

1. Update versions in all `package.json` files
2. Commit and push changes
3. Create and push tag:

   ```bash
   git tag -a v0.5.0 -m "Release v0.5.0"
   git push origin v0.5.0
   ```

4. Workflow automatically builds and publishes
