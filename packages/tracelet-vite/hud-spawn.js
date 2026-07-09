'use strict';

/**
 * Spawns the tracelet HUD server as a child process.
 * Used by both the Next.js and Vite plugins during dev.
 *
 * Resolves the tracelet binary from @traceletdev/cli (the installed npm
 * package), falling back to a locally-built binary if the package isn't
 * found (e.g. in the monorepo itself).
 */

const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');

let _child = null;

/**
 * Find the tracelet binary, preferring the installed npm package.
 * @param {string} cwd  Project root (used to resolve the package).
 * @returns {string|null} Absolute path to the binary, or null if not found.
 */
function resolveBin(cwd) {
  // 1. Try the locally-installed @traceletdev/cli package.
  const candidates = [
    path.join(cwd, 'node_modules', '.bin', 'tracelet'),
    path.join(cwd, 'node_modules', '@traceletdev', 'cli', 'bin', 'tracelet.js'),
  ];
  for (const p of candidates) {
    if (fs.existsSync(p)) return p;
  }

  // 2. Monorepo / development fallback: compiled Go binary next to this file.
  const repoBin = path.resolve(__dirname, '..', '..', 'tracelet');
  if (fs.existsSync(repoBin)) return repoBin;

  return null;
}

/**
 * Start the HUD server.  Safe to call multiple times — a second call is a
 * no-op when the server is already running.
 *
 * @param {{ cwd?: string, port?: number, configPath?: string }} [opts]
 */
function spawnHud(opts = {}) {
  if (_child) return; // already running

  const cwd = opts.cwd || process.cwd();
  const port = opts.port || 3111;
  const bin = resolveBin(cwd);

  if (!bin) {
    console.warn(
      '[tracelet] HUD auto-start: could not find tracelet binary — run `tracelet hud` manually.'
    );
    return;
  }

  const args = ['hud', '--port', String(port)];
  if (opts.configPath) args.push('--config', opts.configPath);

  // Use 'node' to run .js scripts, or exec directly for native binaries.
  const isScript = bin.endsWith('.js');
  const cmd = isScript ? process.execPath : bin;
  const cmdArgs = isScript ? [bin, ...args] : args;

  _child = spawn(cmd, cmdArgs, {
    cwd,
    stdio: 'pipe',
    detached: false,
  });

  _child.stdout.on('data', d => {
    const line = d.toString().trim();
    if (line) console.log(`[tracelet] ${line}`);
  });
  _child.stderr.on('data', d => {
    const line = d.toString().trim();
    // Suppress "address already in use" — another HUD is already running.
    if (line && !line.includes('address already in use')) {
      console.warn(`[tracelet] ${line}`);
    }
  });
  _child.on('exit', code => {
    _child = null;
    if (code !== 0 && code !== null) {
      console.warn(`[tracelet] HUD server exited with code ${code}`);
    }
  });

  // Clean up when the parent process exits.
  const cleanup = () => {
    if (_child) {
      try {
        _child.kill();
      } catch {}
      _child = null;
    }
  };
  process.on('exit', cleanup);
  process.on('SIGINT', cleanup);
  process.on('SIGTERM', cleanup);
}

module.exports = { spawnHud };
