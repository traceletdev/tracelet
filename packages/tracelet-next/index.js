'use strict';

/**
 * Tracelet Next.js Plugin
 *
 * This package provides the adapter script to collect Next.js route statistics.
 *
 * Usage:
 * 1. Add to package.json scripts:
 *    "postbuild": "tracelet-next collect"
 *
 * 2. Or call manually after build:
 *    npx tracelet-next collect
 *
 * 3. Or programmatically:
 *    require('@traceletdev/next')()
 *
 * The script reads Next.js build manifests and generates .tracelet/stats.json
 */

// Export the collect function for programmatic use.
module.exports = require('./collect.js');

// Main entry point (can be called directly, e.g. `node index.js`)
if (require.main === module) {
  module.exports();
}
