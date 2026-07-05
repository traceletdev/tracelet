# GitHub Setup Guide

Follow these steps to set up GitHub and npm publishing for Tracelet.

## 1. Initialize Git Repository

```bash
# Initialize git (if not already done)
git init

# Add remote (replace with your repo URL)
git remote add origin https://github.com/yourusername/tracelet.git

# Add all files
git add .

# Initial commit
git commit -m "Initial commit: Tracelet npm package setup"

# Push to GitHub
git branch -M main
git push -u origin main
```

## 2. Set Up GitHub Secrets

### NPM Token

1. Go to <https://www.npmjs.com/settings/YOUR_USERNAME/tokens>
2. Click "Generate New Token"
3. Select "Automation" type
4. Copy the token
5. In GitHub repo: Settings → Secrets and variables → Actions
6. Click "New repository secret"
7. Name: `NPM_TOKEN`
8. Value: paste your npm token
9. Click "Add secret"

## 3. Create First Release

```bash
# Update versions in package.json files first
# Then create and push a tag

git tag -a v0.5.0 -m "Release v0.5.0"
git push origin v0.5.0
```

The release workflow will automatically:

- Build binaries for all platforms
- Package them for npm
- Publish to npm registry
- Create GitHub release

## 4. Verify

Check that packages were published:

```bash
npm view @traceletdev/cli
npm view @traceletdev/next
npm view @traceletdev/vite
```

Test installation:

```bash
npm install -D @traceletdev/cli@latest @traceletdev/next@latest
npx tracelet --version
```

## Workflow Files

- `.github/workflows/test.yml` - Runs tests on PR/push
- `.github/workflows/tracelet.yml` - CI checks with PR comments
- `.github/workflows/release.yml` - Automated releases on tag push

## Next Steps

- Set up branch protection for `main` branch
- Configure required status checks for PRs
- Add CODEOWNERS file if needed
- Set up dependabot for security updates
