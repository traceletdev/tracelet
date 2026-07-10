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
const http = require('http');
const path = require('path');
const fs = require('fs');

let _child = null;
let _starting = false;

// Version this spawner expects the HUD to be, read from the installed
// @traceletdev/cli package. Used to spot a leftover HUD from an earlier session
// still holding the port. Null when it can't be determined (dev/monorepo).
function getExpectedVersion(cwd) {
  try {
    const p = path.join(cwd, 'node_modules', '@traceletdev', 'cli', 'package.json');
    return JSON.parse(fs.readFileSync(p, 'utf8')).version || null;
  } catch {
    return null;
  }
}

// Probe port for an existing HUD via /healthz. Resolves:
//   { occupied: false }                  — nothing listening (safe to spawn)
//   { occupied: true, version: '0.6.0' } — a HUD that reports its build
//   { occupied: true, version: null }    — something answered but isn't a HUD
//     we recognize (e.g. an old pre-/healthz HUD — exactly the stale case)
function probeHud(port) {
  return new Promise(resolve => {
    const req = http.get({ host: 'localhost', port, path: '/healthz', timeout: 500 }, res => {
      let body = '';
      res.on('data', d => (body += d));
      res.on('end', () => {
        let version = null;
        try {
          version = JSON.parse(body).version || null;
        } catch {
          /* occupied, but not a version we can read */
        }
        resolve({ occupied: true, version });
      });
    });
    // ECONNREFUSED => port is free. Any other error we treat as free too, since
    // the worst case is a spawn that fails to bind (already handled).
    req.on('error', () => resolve({ occupied: false }));
    req.on('timeout', () => {
      req.destroy();
      resolve({ occupied: false });
    });
  });
}

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
  if (_child || _starting) return; // already running or mid-startup

  const cwd = opts.cwd || process.cwd();
  const port = opts.port || 3111;
  const bin = resolveBin(cwd);

  if (!bin) {
    console.warn(
      '[tracelet] HUD auto-start: could not find tracelet binary — run `tracelet hud` manually.'
    );
    return;
  }

  _starting = true;
  // Check for an existing HUD before spawning: a leftover process from an
  // earlier session can hold this port and silently serve stale code. If one is
  // there but its version doesn't match what we'd spawn, say so loudly instead
  // of quietly reusing it (which is how stale fixes went unnoticed).
  probeHud(port).then(running => {
    if (running.occupied) {
      _starting = false;
      const expected = getExpectedVersion(cwd);
      const kill = `Kill it (lsof -ti:${port} | xargs kill) and restart to pick up the current build.`;
      if (running.version === null) {
        // Occupied, but no readable /healthz — most likely an old HUD from an
        // earlier session (pre-/healthz), the exact stale-serving footgun.
        console.warn(
          `[tracelet] Port ${port} is already in use by a server that didn't report a tracelet version — ` +
            `likely a leftover HUD from an earlier session serving stale code. ${kill}`
        );
      } else if (expected && running.version !== 'dev' && running.version !== expected) {
        console.warn(
          `[tracelet] A HUD already on port ${port} reports version ${running.version}, not ${expected} — ` +
            `likely a leftover from an earlier session serving stale code. ${kill}`
        );
      }
      return; // port is taken; nothing to spawn
    }
    doSpawn(bin, port, cwd, opts);
  });
}

function doSpawn(bin, port, cwd, opts) {
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
  _starting = false;

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

module.exports = { spawnHud, probeHud };
