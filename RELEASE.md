# Release Guide

This document describes how to release new versions of Tracelet to npm and GitHub.

## Prerequisites

1. **GitHub Token**: Set `GITHUB_TOKEN` (automatically provided in Actions)
2. **npm publishing: Trusted Publishing (OIDC), not a token**. `release.yml` publishes
   via [npm Trusted Publishers](https://docs.npmjs.com/trusted-publishers/) — no
   `NPM_TOKEN` secret is used. This needs one-time setup **per package** (10 total:
   `@traceletdev/{cli,next,vite,react}` + the 6 `@traceletdev/cli-<platform>` binary
   packages), done on npmjs.com by someone with publish rights on the `@traceletdev` org:

   - Go to the package's Settings → **Trusted Publisher** and add:
     - Provider: **GitHub Actions**
     - Organization/repo: `traceletdev/tracelet`
     - Workflow filename: `release.yml` (exact, case-sensitive)
     - Environment: leave blank (this workflow doesn't use one)
     - Allowed actions: `npm publish`

   **Bootstrap caveat:** npm's docs don't confirm whether a Trusted Publisher can be
   registered for a name before its first publish. `@traceletdev/cli`, `@traceletdev/next`,
   `@traceletdev/vite`, and `@traceletdev/react` were bootstrapped this way already (manual
   publish, then configure); the 6 new `@traceletdev/cli-<platform>` packages will need the
   same treatment the first time they're published:
   1. Publish once manually, from a local machine logged in (`npm login`) as a
      `@traceletdev` org member with publish rights —
      `node scripts/publish-all.js <version>` (see "Manual Release" below; requires
      binaries built first via `npm run pack:bin`).
   2. *Then* configure the Trusted Publisher for each now-existing package.
   3. All subsequent releases go through `release.yml`'s OIDC flow with no token.

## Release Process

### 1. Update Version

Update version in all `package.json` files (the automated release does this for you —
manual releases must do it by hand):

- `package.json` — also bump the `0.5.1` pins inside its `optionalDependencies` to match
- `packages/tracelet-next/package.json`
- `packages/tracelet-vite/package.json`
- `packages/tracelet-react/package.json`
- `packages/cli-{darwin,linux,win32}-{arm64,x64}/package.json` (6 files, binary-only)

### 2. Create Release Tag

```bash
# Make sure you're on main and up to date
git checkout main
git pull origin main

# Create and push tag (will trigger release workflow)
git tag -a v0.5.0 -m "Release v0.5.0"
git push origin v0.5.0
```

### 3. Automated Release

The GitHub Actions workflow will:

1. **Build binaries** using GoReleaser for all platforms:
   - darwin (amd64, arm64)
   - linux (amd64, arm64)
   - windows (amd64, arm64)

2. **Package binaries** into the 6 per-platform packages using `pack-binaries.js` (see
   "Binary distribution" below)

3. **Publish to npm**, platform packages first so `@traceletdev/cli`'s
   `optionalDependencies` resolve once it publishes last:
   - `@traceletdev/cli-{darwin,linux,win32}-{arm64,x64}` (6 binary-only packages)
   - `@traceletdev/next` (Next.js plugin)
   - `@traceletdev/vite` (Vite plugin)
   - `@traceletdev/react` (React render instrumentation)
   - `@traceletdev/cli` (main package — thin wrapper, resolves the right platform
     package at runtime)

4. **Create GitHub Release** with binaries attached

### Binary distribution

`@traceletdev/cli` doesn't bundle any binaries itself — it ships only `bin/tracelet.js`,
a thin wrapper that `require.resolve()`s the one platform package that actually got
installed (e.g. `@traceletdev/cli-darwin-arm64/bin/tracelet`) and spawns it. Each platform
package declares `os`/`cpu` fields, so npm only ever installs the single one matching the
machine running `npm install` — not all 6. This keeps a typical install to ~12MB instead
of bundling every platform (~70MB) into one tarball.

### 4. Verify Release

```bash
# Check npm packages were published
npm view @traceletdev/cli version
npm view @traceletdev/next version
npm view @traceletdev/vite version
npm view @traceletdev/react version
npm view @traceletdev/cli-darwin-arm64 version  # spot-check one platform package

# Test installation
npm install -D @traceletdev/cli@latest @traceletdev/next@latest
npx tracelet --version
```

## VS Code Extension (Marketplace)

The extension is published separately (not part of the tag-triggered workflow).

Prerequisites:

- A Marketplace publisher matching `publisher` in `ui/vscode-extension/package.json`
  (currently `traceletdev` — update if your publisher ID differs).
- A Personal Access Token: `vsce login <publisher>` or `VSCE_PAT` env var.

```bash
cd ui/vscode-extension
npm install
npm run compile
npx vsce package        # builds a .vsix — verify no missing-field errors
npx vsce publish        # publishes to the Marketplace
```

An `icon` (128×128 PNG) is optional but recommended before publishing — add it to
`ui/vscode-extension/` and reference it via `"icon"` in package.json.

## Manual Release (Alternative)

If you need to release manually:

```bash
# 1. Build binaries
goreleaser release --snapshot --clean

# 2. Pack binaries into the 6 platform packages (packages/cli-*/bin/)
npm run pack:bin

# 3. Dry-run — bumps versions in every package.json and simulates the publish
node scripts/publish-all.js 0.5.1 --dry-run

# 4. The real thing — bumps versions and publishes all 10 packages, platform
#    packages first, @traceletdev/cli last
node scripts/publish-all.js 0.5.1
```

## Version Format

Use semantic versioning: `MAJOR.MINOR.PATCH`

- `MAJOR`: Breaking changes
- `MINOR`: New features, backwards compatible
- `PATCH`: Bug fixes, backwards compatible

Examples:

- `0.5.0` - Initial npm release
- `0.5.1` - Bug fix
- `0.6.0` - New features
- `1.0.0` - Stable release

## Troubleshooting

### Release workflow fails

- Check that a Trusted Publisher is configured for each package (repo + `release.yml`
  must match exactly — see Prerequisites above)
- Verify Go version in workflow matches `go.mod`
- Check that binaries were built successfully in GoReleaser step

### Binaries not found

- Ensure `pack:bin` script runs after GoReleaser
- Check `dist/` directory structure matches expected format
- Verify `pack-binaries.js` is executable

### npm publish fails

- CI: check the Trusted Publisher config matches this repo/workflow exactly
- Manual: check your local npm login has publish rights on the `@traceletdev` org
- Ensure version numbers are incremented
