# Release Guide

This document describes how to release new versions of Tracelet to npm and GitHub.

## Prerequisites

1. **GitHub Token**: Set `GITHUB_TOKEN` (automatically provided in Actions)
2. **NPM Token**: Set `NPM_TOKEN` in GitHub Secrets
   - Packages publish under the `@traceletdev` npm org — the token must belong to
     a member with publish rights on that org.
   - Get token from: `https://www.npmjs.com/settings/<username>/tokens`
   - Create token with "Automation" type
   - Add to GitHub: Settings → Secrets → Actions → New repository secret

## Release Process

### 1. Update Version

Update version in all `package.json` files:

- `package.json`
- `packages/tracelet-next/package.json`
- `packages/tracelet-vite/package.json`
- `packages/tracelet-react/package.json`

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

2. **Package binaries** into npm structure using `pack-binaries.js`

3. **Publish to npm**:
   - `@traceletdev/cli` (main package)
   - `@traceletdev/next` (Next.js plugin)
   - `@traceletdev/vite` (Vite plugin)
   - `@traceletdev/react` (React render instrumentation)

4. **Create GitHub Release** with binaries attached

### 4. Verify Release

```bash
# Check npm packages were published
npm view @traceletdev/cli version
npm view @traceletdev/next version
npm view @traceletdev/vite version
npm view @traceletdev/react version

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

# 2. Pack binaries
npm run pack:bin

# 3. Update versions
node scripts/publish-all.js 0.5.0 --dry-run  # check first
node scripts/publish-all.js 0.5.0              # update versions

# 4. Publish
npm publish
npm publish --workspace=packages/tracelet-next
npm publish --workspace=packages/tracelet-vite
npm publish --workspace=packages/tracelet-react
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

- Check that `NPM_TOKEN` is set in GitHub Secrets
- Verify Go version in workflow matches `go.mod`
- Check that binaries were built successfully in GoReleaser step

### Binaries not found

- Ensure `pack:bin` script runs after GoReleaser
- Check `dist/` directory structure matches expected format
- Verify `pack-binaries.js` is executable

### npm publish fails

- Check npm token has publish permissions
- Verify package names aren't taken on npm
- Ensure version numbers are incremented
