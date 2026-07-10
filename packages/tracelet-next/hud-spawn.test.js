'use strict';

/**
 * Self-check for probeHud's occupancy/version detection. Run: node hud-spawn.test.js
 * Guards the stale-HUD-on-port footgun: a free port, a current HUD, and an old
 * pre-/healthz HUD must be distinguishable.
 */

const http = require('http');
const assert = require('assert');

const { probeHud } = require('./hud-spawn.js');

// Start a throwaway server with a chosen /healthz behavior, return its port.
function serve(handler) {
  return new Promise(resolve => {
    const srv = http.createServer(handler);
    srv.listen(0, '127.0.0.1', () => resolve({ srv, port: srv.address().port }));
  });
}

(async () => {
  // 1. Free port — nothing listening.
  //    Pick a port then close the server before probing.
  const { srv: tmp, port: freePort } = await serve(() => {});
  await new Promise(r => tmp.close(r));
  assert.deepStrictEqual(await probeHud(freePort), { occupied: false }, 'free port');

  // 2. A current HUD reporting its version.
  const good = await serve((req, res) => {
    res.setHeader('content-type', 'application/json');
    res.end(JSON.stringify({ version: '0.6.0' }));
  });
  assert.deepStrictEqual(
    await probeHud(good.port),
    { occupied: true, version: '0.6.0' },
    'current HUD reports version'
  );
  await new Promise(r => good.srv.close(r));

  // 3. An old pre-/healthz HUD: port occupied, 404 non-JSON body.
  const old = await serve((req, res) => {
    res.statusCode = 404;
    res.end('404 page not found\n');
  });
  assert.deepStrictEqual(
    await probeHud(old.port),
    { occupied: true, version: null },
    'occupied but unrecognized => version null'
  );
  await new Promise(r => old.srv.close(r));

  console.log('ok - probeHud distinguishes free / current / stale');
})().catch(e => {
  console.error(e);
  process.exit(1);
});
