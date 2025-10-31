'use strict';

const fs = require('fs');
const path = require('path');
const zlib = require('zlib');

function writeStats(routes, outDir) {
  const outPath = path.join(process.cwd(), '.tracelet', 'stats.json');
  try {
    fs.mkdirSync(path.dirname(outPath), { recursive: true });
    fs.writeFileSync(outPath, JSON.stringify({ routes }, null, 2));
    // eslint-disable-next-line no-console
    console.log(`[tracelet] wrote stats to ${path.relative(process.cwd(), outPath)}`);
  } catch (e) {
    // eslint-disable-next-line no-console
    console.warn('[tracelet] failed to write stats:', e.message);
  }
}

module.exports = function traceletPlugin() {
  let resolvedOutDir = '';
  return {
    name: 'tracelet-plugin',
    apply: 'build',
    configResolved(cfg) {
      resolvedOutDir = (cfg.build && cfg.build.outDir) || 'dist';
    },
    generateBundle(_, bundle) {
      // Find entry chunks and sum sizes as a homepage surrogate
      const routes = [];
      let homeBytes = 0;
      let thirdPartyBytes = 0;
      for (const [fileName, chunk] of Object.entries(bundle)) {
        if (chunk && chunk.type === 'chunk' && chunk.isEntry) {
          // Compress chunk code directly (files not written yet in generateBundle)
          const code = chunk.code || '';
          const compressed = zlib.gzipSync(Buffer.from(code), { level: 6 });
          const size = compressed.length;
          homeBytes += size;
          // Detect third-party: Vite often names vendor chunks as "vendor-*.js" or includes node_modules imports
          if (fileName.includes('vendor') || fileName.includes('chunk-vendor') ||
              (chunk.imports && chunk.imports.some(imp => imp && imp.includes('node_modules')))) {
            thirdPartyBytes += size;
          }
        }
      }
      routes.push({
        path: '/',
        jsGzipBytes: homeBytes,
        thirdPartyJsBytes: thirdPartyBytes
      });
      writeStats(routes, resolvedOutDir);
    },
  };
};


