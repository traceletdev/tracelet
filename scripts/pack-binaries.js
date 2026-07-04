#!/usr/bin/env node

'use strict';

const fs = require('fs');
const path = require('path');

/**
 * Pack Go binaries into npm package structure
 *
 * Expects binaries to be built and placed in dist/ directory
 * (by GoReleaser or manual build)
 *
 * Structure:
 * dist/
 *   tracelet_darwin_amd64/tracelet
 *   tracelet_darwin_arm64/tracelet
 *   tracelet_linux_amd64/tracelet
 *   tracelet_linux_arm64/tracelet
 *   tracelet_windows_amd64/tracelet.exe
 *   tracelet_windows_arm64/tracelet.exe
 *
 * Output:
 * binaries/
 *   darwin-x64/tracelet
 *   darwin-arm64/tracelet
 *   linux-x64/tracelet
 *   linux-arm64/tracelet
 *   win32-x64/tracelet.exe
 *   win32-arm64/tracelet.exe
 */

const distDir = path.join(__dirname, '..', 'dist');
const binariesDir = path.join(__dirname, '..', 'binaries');

// Map from GoReleaser naming to npm naming
const platformMap = {
  darwin: {
    amd64: { dir: 'darwin-x64', name: 'tracelet' },
    arm64: { dir: 'darwin-arm64', name: 'tracelet' },
  },
  linux: {
    amd64: { dir: 'linux-x64', name: 'tracelet' },
    arm64: { dir: 'linux-arm64', name: 'tracelet' },
  },
  windows: {
    amd64: { dir: 'win32-x64', name: 'tracelet.exe' },
    arm64: { dir: 'win32-arm64', name: 'tracelet.exe' },
  },
};

function packBinaries() {
  if (!fs.existsSync(distDir)) {
    console.error('dist/ directory not found. Run GoReleaser or build binaries first.');
    process.exit(1);
  }

  // Create binaries directory structure
  if (!fs.existsSync(binariesDir)) {
    fs.mkdirSync(binariesDir, { recursive: true });
  }

  let packed = 0;
  let skipped = 0;

  // Find all directories in dist/
  const entries = fs.readdirSync(distDir, { withFileTypes: true });

  for (const entry of entries) {
    if (!entry.isDirectory()) continue;

    const dirName = entry.name;
    // Expected format: tracelet_<os>_<arch> or tracelet_<os>_<arch>v7
    const match = dirName.match(/^tracelet_(darwin|linux|windows)_(amd64|arm64)(?:v\d+)?$/);
    if (!match) {
      console.warn(`Skipping unknown directory: ${dirName}`);
      skipped++;
      continue;
    }

    const [, goos, goarch] = match;
    const mapping = platformMap[goos]?.[goarch];
    if (!mapping) {
      console.warn(`No mapping for ${goos}/${goarch}`);
      skipped++;
      continue;
    }

    const sourceDir = path.join(distDir, dirName);
    const sourceBinary = path.join(sourceDir, mapping.name);
    const targetDir = path.join(binariesDir, mapping.dir);
    const targetBinary = path.join(targetDir, mapping.name);

    if (!fs.existsSync(sourceBinary)) {
      console.warn(`Binary not found: ${sourceBinary}`);
      skipped++;
      continue;
    }

    // Create target directory
    if (!fs.existsSync(targetDir)) {
      fs.mkdirSync(targetDir, { recursive: true });
    }

    // Copy binary
    fs.copyFileSync(sourceBinary, targetBinary);

    // Make executable on Unix
    if (goos !== 'windows') {
      fs.chmodSync(targetBinary, '755');
    }

    console.log(`✓ Packed ${goos}/${goarch} → ${mapping.dir}/${mapping.name}`);
    packed++;
  }

  if (packed === 0 && skipped > 0) {
    console.error('No binaries were packed. Check that dist/ contains built binaries.');
    process.exit(1);
  }

  console.log(`\n✓ Packed ${packed} binaries to binaries/`);
  if (skipped > 0) {
    console.warn(`⚠ Skipped ${skipped} entries`);
  }
}

if (require.main === module) {
  packBinaries();
}

module.exports = packBinaries;
