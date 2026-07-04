#!/usr/bin/env node

'use strict';

const fs = require('fs');
const path = require('path');
const os = require('os');

// Platform detection
const platform = process.platform;
const arch = process.arch;

// Map Node.js platform/arch to our binary names
let binaryDir;
if (platform === 'darwin') {
  binaryDir = arch === 'arm64' ? 'darwin-arm64' : 'darwin-x64';
} else if (platform === 'linux') {
  binaryDir = arch === 'arm64' ? 'linux-arm64' : 'linux-x64';
} else if (platform === 'win32') {
  binaryDir = arch === 'arm64' ? 'win32-arm64' : 'win32-x64';
} else {
  console.warn(`[tracelet] Unsupported platform: ${platform}-${arch}`);
  process.exit(0);
}

const binaryName = platform === 'win32' ? 'tracelet.exe' : 'tracelet';
const sourcePath = path.join(__dirname, '..', 'binaries', binaryDir, binaryName);
const targetPath = path.join(__dirname, '..', 'node_modules', '.bin', binaryName);

// Check if source binary exists
if (!fs.existsSync(sourcePath)) {
  console.warn(`[tracelet] Binary not found for ${platform}-${arch}, skipping postinstall`);
  console.warn(`[tracelet] You may need to build binaries first`);
  process.exit(0);
}

// Ensure .bin directory exists
const binDir = path.dirname(targetPath);
if (!fs.existsSync(binDir)) {
  fs.mkdirSync(binDir, { recursive: true });
}

// Copy or symlink binary
try {
  // Try symlink first (works on Unix and Windows with proper permissions)
  if (fs.existsSync(targetPath)) {
    fs.unlinkSync(targetPath);
  }

  if (platform === 'win32') {
    // Windows: copy instead of symlink for better compatibility
    fs.copyFileSync(sourcePath, targetPath);
  } else {
    // Unix: use symlink
    fs.symlinkSync(path.relative(binDir, sourcePath), targetPath);
  }

  // Make executable on Unix
  if (platform !== 'win32') {
    fs.chmodSync(targetPath, '755');
  }

  console.log(`[tracelet] Binary linked successfully for ${platform}-${arch}`);
} catch (err) {
  console.warn(`[tracelet] Failed to link binary:`, err.message);
  console.warn(`[tracelet] You can still use 'npx tracelet' or the full path`);
}
