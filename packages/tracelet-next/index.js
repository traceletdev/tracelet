'use strict';

/**
 * Tracelet Next.js Plugin
 *
 * ## Usage
 *
 * ### Dev HUD (recommended)
 * Wrap your Next.js config with `withTracelet` so the HUD server starts
 * automatically when you run `next dev` and the overlay is injected without
 * any manual <script> tag:
 *
 *   // next.config.mjs
 *   import { withTracelet } from '@traceletdev/next';
 *   export default withTracelet({
 *     // ...your normal nextConfig
 *   });
 *
 * ### Build stats collection
 * Add to package.json scripts:
 *   "postbuild": "tracelet-next collect"
 *
 * Or call programmatically:
 *   import collect from '@traceletdev/next';
 *   collect();
 */

const { spawnHud } = require('./hud-spawn.js');
const collect = require('./collect.js');

/**
 * Wraps a Next.js config object to add Tracelet dev-mode support:
 *  - Auto-starts the HUD server when `next dev` runs.
 *  - Injects the overlay script as a webpack entry (no manual <script> needed).
 *
 * @param {import('next').NextConfig} nextConfig
 * @param {{ port?: number, configPath?: string }} [opts]
 * @returns {import('next').NextConfig}
 */
function withTracelet(nextConfig = {}, opts = {}) {
  const hudPort = opts.port || 3111;

  return {
    ...nextConfig,

    webpack(config, context) {
      const { dev, isServer, nextRuntime } = context;

      // Only modify the client-side dev bundle.
      if (dev && !isServer && !nextRuntime) {
        // 1. Spawn the HUD server (idempotent — safe to call on every hot reload).
        spawnHud({
          cwd: process.cwd(),
          port: hudPort,
          configPath: opts.configPath,
        });

        // 2. Inject the port so the dev-entry script knows where to connect.
        const webpack = config.webpack || require('webpack');
        config.plugins.push(
          new webpack.DefinePlugin({
            'window.__TRACELET_HUD_PORT__': JSON.stringify(hudPort),
          })
        );

        // 3. Prepend hud-dev-entry.js to the main client entry so the overlay
        //    loads before the app bundle renders anything.
        const hudEntry = require.resolve('./hud-dev-entry.js');
        const originalEntry = config.entry;
        config.entry = async () => {
          const entries = await (typeof originalEntry === 'function'
            ? originalEntry()
            : Promise.resolve(originalEntry));

          // Next.js App Router uses 'main-app'; Pages Router uses 'main'.
          const targetKey = 'main-app' in entries ? 'main-app' : 'main';
          if (entries[targetKey]) {
            const existing = Array.isArray(entries[targetKey])
              ? entries[targetKey]
              : [entries[targetKey]];
            if (!existing.includes(hudEntry)) {
              entries[targetKey] = [hudEntry, ...existing];
            }
          }

          return entries;
        };
      }

      // Call the user's own webpack config if they have one.
      if (typeof nextConfig.webpack === 'function') {
        return nextConfig.webpack(config, context);
      }
      return config;
    },
  };
}

module.exports = collect;
module.exports.withTracelet = withTracelet;
module.exports.default = withTracelet; // ESM default interop for next.config.mjs
