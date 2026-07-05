#!/usr/bin/env node

'use strict';

const { spawn } = require('child_process');
const fs = require('fs');

// Platform detection
const platform = process.platform; // 'darwin' | 'linux' | 'win32'
const arch = process.arch; // 'arm64' | 'x64'

const supportedPlatforms = new Set(['darwin', 'linux', 'win32']);
const supportedArches = new Set(['arm64', 'x64']);

if (!supportedPlatforms.has(platform) || !supportedArches.has(arch)) {
  console.error(`[tracelet] Unsupported platform: ${platform}-${arch}`);
  process.exit(1);
}

const binaryName = platform === 'win32' ? 'tracelet.exe' : 'tracelet';
const packageName = `@traceletdev/cli-${platform}-${arch}`;

// The binary ships in a per-platform optionalDependency (os/cpu-gated), so npm
// only installs the one matching package. require.resolve finds it without
// needing a postinstall copy/symlink step.
let binaryPath;
try {
  binaryPath = require.resolve(`${packageName}/bin/${binaryName}`);
} catch (err) {
  console.error(`[tracelet] Could not find the platform binary package "${packageName}".`);
  console.error(
    `[tracelet] This usually means npm skipped installing it (platform mismatch) or the install failed. Try reinstalling.`
  );
  process.exit(1);
}

if (platform !== 'win32') {
  try {
    fs.chmodSync(binaryPath, 0o755);
  } catch (e) {
    // Ignore if chmod fails — the tarball should already preserve the bit.
  }
}

// Spawn binary with forwarded args
const args = process.argv.slice(2);
const child = spawn(binaryPath, args, {
  stdio: 'inherit',
  cwd: process.cwd(),
});

child.on('error', (err) => {
  console.error(`[tracelet] Failed to spawn binary:`, err.message);
  process.exit(1);
});

child.on('exit', (code) => {
  process.exit(code === null ? 1 : code);
});
