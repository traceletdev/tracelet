#!/usr/bin/env node

'use strict';

const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');

// Platform detection
const platform = process.platform;
const arch = process.arch;

// Map Node.js platform/arch to our binary names
const platformMap = {
  'darwin': platform === 'darwin' ? (arch === 'arm64' ? 'darwin-arm64' : 'darwin-x64') : null,
  'linux': platform === 'linux' ? (arch === 'arm64' ? 'linux-arm64' : 'linux-x64') : null,
  'win32': platform === 'win32' ? (arch === 'arm64' ? 'win32-arm64' : 'win32-x64') : null,
};

const binaryDir = platformMap[platform] || platformMap[process.platform];
if (!binaryDir) {
  console.error(`[tracelet] Unsupported platform: ${platform}-${arch}`);
  process.exit(1);
}

const binaryName = platform === 'win32' ? 'tracelet.exe' : 'tracelet';
const binaryPath = path.join(__dirname, '..', 'binaries', binaryDir, binaryName);

// Check if binary exists
if (!fs.existsSync(binaryPath)) {
  console.error(`[tracelet] Binary not found at ${binaryPath}`);
  console.error(`[tracelet] Please run 'npm run postinstall' or reinstall the package`);
  process.exit(1);
}

// Make binary executable on Unix-like systems
if (platform !== 'win32') {
  try {
    fs.chmodSync(binaryPath, '755');
  } catch (e) {
    // Ignore if chmod fails
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
  process.exit(code || 0);
});

