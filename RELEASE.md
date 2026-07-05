# Release Guide

## Prerequisites

npm publishing uses [Trusted Publishing](https://docs.npmjs.com/trusted-publishers/)
(OIDC) — no token secret. Each of the 10 packages needs a one-time Trusted Publisher
configured on npmjs.com by a `@traceletdev` org member:

- Package Settings → **Trusted Publisher**
  - Provider: **GitHub Actions**
  - Repo: `traceletdev/tracelet`
  - Workflow filename: `release.yml`
  - Allowed actions: `npm publish`

## Release Process

### 1. Update Version

Bump `version` in every `package.json`:

- `package.json` (also bump the `optionalDependencies` pins to match)
- `packages/tracelet-next/package.json`
- `packages/tracelet-vite/package.json`
- `packages/tracelet-react/package.json`
- `packages/cli-{darwin,linux,win32}-{arm64,x64}/package.json`

### 2. Create Release Tag

```bash
git checkout main
git pull origin main
git tag -a v0.5.1 -m "Release v0.5.1"
git push origin v0.5.1
```

### 3. Automated Release

`release.yml`:

1. Builds binaries for all platforms via GoReleaser
2. Packs them into the 6 platform packages (`pack-binaries.js`)
3. Publishes, platform packages first so `@traceletdev/cli`'s
   `optionalDependencies` resolve when it publishes last
4. Creates the GitHub Release

### Binary distribution

`@traceletdev/cli` ships no binaries — just `bin/tracelet.js`, which
`require.resolve()`s the one platform package npm actually installed
(`os`/`cpu` fields mean only one of the 6 ever matches) and spawns it.

### 4. Verify

```bash
npm view @traceletdev/cli version
npm install -D @traceletdev/cli@latest @traceletdev/next@latest
npx tracelet --version
```

## VS Code Extension

Published separately, not part of `release.yml`.

```bash
cd ui/vscode-extension
npm install && npm run compile
npx vsce package   # verify no missing-field errors
npx vsce publish
```

Requires a Marketplace publisher matching `ui/vscode-extension/package.json`'s
`publisher` field, and `vsce login <publisher>` or `VSCE_PAT`.

## Manual Release

```bash
goreleaser release --snapshot --clean
npm run pack:bin
node scripts/publish-all.js 0.5.1 --dry-run
node scripts/publish-all.js 0.5.1
```

## Version Format

Semantic versioning: `MAJOR.MINOR.PATCH`.

## Troubleshooting

**Release workflow fails** — Trusted Publisher config must match this repo +
`release.yml` exactly; check Go version matches `go.mod`; check GoReleaser
built successfully.

**Binaries not found** — `pack:bin` must run after GoReleaser; check `dist/`
matches the expected structure.

**npm publish fails** — check Trusted Publisher config (CI) or your npm
login's org permissions (manual); check the version was actually incremented.
