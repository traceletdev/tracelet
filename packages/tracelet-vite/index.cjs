'use strict';

const fs = require('fs');
const path = require('path');
const zlib = require('zlib');
const { spawnHud } = require('./hud-spawn.js');

function writeStats(routes, _outDir) {
  const outPath = path.join(process.cwd(), '.tracelet', 'stats.json');
  try {
    fs.mkdirSync(path.dirname(outPath), { recursive: true });
    fs.writeFileSync(outPath, JSON.stringify({ routes }, null, 2));
    console.log(`[tracelet] wrote stats to ${path.relative(process.cwd(), outPath)}`);
  } catch (e) {
    console.warn('[tracelet] failed to write stats:', e.message);
  }
}

/**
 * Tracelet Vite Plugin
 *
 * - **Dev**: auto-starts the HUD server and injects the overlay into every
 *   HTML page — no manual <script> or separate `tracelet hud` process needed.
 * - **Build**: collects per-route JS sizes into .tracelet/stats.json.
 *
 * @param {{ port?: number, configPath?: string }} [opts]
 */
module.exports = function traceletPlugin(opts = {}) {
  const hudPort = opts.port || 3111;
  let isDev = false;

  return {
    name: 'tracelet-plugin',

    // Run in both dev and build modes.
    apply(_config, { command }) {
      return command === 'serve' || command === 'build';
    },

    configResolved(cfg) {
      isDev = cfg.command === 'serve';
      if (isDev) {
        spawnHud({
          cwd: process.cwd(),
          port: hudPort,
          configPath: opts.configPath,
        });
      }
    },

    // Inject overlay script into every HTML response during dev.
    transformIndexHtml: {
      order: 'post',
      handler(html, _ctx) {
        if (!isDev) return html;
        // hook.js must come before the app bundle; overlay.js after.
        const hookTag = `<script src="http://localhost:${hudPort}/hook.js"></script>`;
        const overlayTag = `<script src="http://localhost:${hudPort}/overlay.js" async></script>`;
        // Inject hook before </head> so it precedes React, overlay before </body>.
        return html
          .replace('</head>', `${hookTag}\n</head>`)
          .replace('</body>', `${overlayTag}\n</body>`);
      },
    },

    // ---- Build-time stats collection (unchanged from v1) ----
    generateBundle(_, bundle) {
      if (isDev) return; // stats collection is for builds only
      const routes = [];
      let homeBytes = 0;
      let thirdPartyBytes = 0;
      for (const [fileName, chunk] of Object.entries(bundle)) {
        if (chunk && chunk.type === 'chunk' && chunk.isEntry) {
          const code = chunk.code || '';
          const compressed = zlib.gzipSync(Buffer.from(code), { level: 6 });
          const size = compressed.length;
          homeBytes += size;
          if (
            fileName.includes('vendor') ||
            fileName.includes('chunk-vendor') ||
            (chunk.imports && chunk.imports.some(imp => imp && imp.includes('node_modules')))
          ) {
            thirdPartyBytes += size;
          }
        }
      }
      routes.push({
        path: '/',
        jsGzipBytes: homeBytes,
        thirdPartyJsBytes: thirdPartyBytes,
      });
      writeStats(routes, '');
    },
  };
};
