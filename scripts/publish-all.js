#!/usr/bin/env node

'use strict';

/**
 * Publish all packages to npm
 *
 * Ensures version sync across packages and publishes:
 * - tracelet (main package)
 * - packages/tracelet-next
 * - packages/tracelet-vite
 *
 * Usage:
 *   node scripts/publish-all.js [version]
 *
 * If version is provided, updates all package.json files
 * If not, publishes current versions
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const packages = [
  { name: 'tracelet', path: '.' },
  { name: 'tracelet-next', path: 'packages/tracelet-next' },
  { name: 'tracelet-vite', path: 'packages/tracelet-vite' },
  { name: 'tracelet-react', path: 'packages/tracelet-react' },
];

function getVersion(pkgPath) {
  const pkgJsonPath = path.join(pkgPath, 'package.json');
  if (!fs.existsSync(pkgJsonPath)) {
    return null;
  }
  const pkg = JSON.parse(fs.readFileSync(pkgJsonPath, 'utf8'));
  return pkg.version;
}

function setVersion(pkgPath, version) {
  const pkgJsonPath = path.join(pkgPath, 'package.json');
  const pkg = JSON.parse(fs.readFileSync(pkgJsonPath, 'utf8'));
  pkg.version = version;
  fs.writeFileSync(pkgJsonPath, JSON.stringify(pkg, null, 2) + '\n');
  console.log(`✓ Updated ${pkg.name} to ${version}`);
}

function publishPackage(pkgPath, dryRun = false) {
  const cmd = dryRun ? 'npm publish --dry-run' : 'npm publish';
  console.log(`\n📦 Publishing ${path.basename(pkgPath)}...`);
  try {
    execSync(cmd, { cwd: pkgPath, stdio: 'inherit' });
    console.log(`✓ Published ${path.basename(pkgPath)}`);
  } catch (err) {
    console.error(`✗ Failed to publish ${path.basename(pkgPath)}`);
    throw err;
  }
}

function main() {
  const args = process.argv.slice(2);
  const dryRun = args.includes('--dry-run');
  let newVersion = args.find(arg => !arg.startsWith('--'));

  // Check if binaries are built
  const binariesDir = path.join(process.cwd(), 'binaries');
  if (!fs.existsSync(binariesDir)) {
    console.warn('⚠ binaries/ directory not found. Run "npm run pack:bin" first.');
    console.warn('⚠ Publishing without binaries will result in broken package.');
    if (!dryRun) {
      console.error('✗ Aborting publish. Build binaries first.');
      process.exit(1);
    }
  }

  // Get current versions
  const versions = packages.map(pkg => ({
    ...pkg,
    version: getVersion(pkg.path)
  })).filter(pkg => pkg.version !== null);

  if (versions.length === 0) {
    console.error('No packages found with package.json');
    process.exit(1);
  }

  const currentVersion = versions[0].version;
  console.log(`Current version: ${currentVersion}`);

  // Update versions if new version provided
  if (newVersion) {
    if (!/^\d+\.\d+\.\d+/.test(newVersion)) {
      console.error('Invalid version format. Use semantic versioning (e.g., 0.5.0)');
      process.exit(1);
    }
    console.log(`\n📝 Updating all packages to ${newVersion}...`);
    versions.forEach(pkg => setVersion(pkg.path, newVersion));
  }

  // Publish packages
  console.log(`\n🚀 Publishing packages${dryRun ? ' (dry run)' : ''}...`);
  versions.forEach(pkg => {
    publishPackage(pkg.path, dryRun);
  });

  console.log('\n✅ All packages published successfully!');
}

if (require.main === module) {
  main();
}

module.exports = { getVersion, setVersion, publishPackage };

