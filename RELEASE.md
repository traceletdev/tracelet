# Release Guide

This document describes how to release new versions of Tracelet to npm and GitHub.

## Prerequisites

1. **GitHub Token**: Set `GITHUB_TOKEN` (automatically provided in Actions)
2. **NPM Token**: Set `NPM_TOKEN` in GitHub Secrets
   - Get token from: https://www.npmjs.com/settings/<username>/tokens
   - Create token with "Automation" type
   - Add to GitHub: Settings → Secrets → Actions → New repository secret

## Release Process

### 1. Update Version

Update version in all `package.json` files:
- `package.json`
- `packages/tracelet-next/package.json`
- `packages/tracelet-vite/package.json`

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
   - `tracelet` (main package)
   - `tracelet-next` (Next.js plugin)
   - `tracelet-vite` (Vite plugin)

4. **Create GitHub Release** with binaries attached

### 4. Verify Release

```bash
# Check npm packages were published
npm view tracelet version
npm view tracelet-next version
npm view tracelet-vite version

# Test installation
npm install -D tracelet@latest tracelet-next@latest
npx tracelet --version
```

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

