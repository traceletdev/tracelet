'use strict';

/**
 * Tracelet Next.js Plugin
 *
 * This package provides the adapter script to collect Next.js route statistics.
 *
 * Usage:
 * 1. Add to package.json scripts:
 *    "postbuild": "node node_modules/tracelet-next/collect.js"
 *
 * 2. Or call manually after build:
 *    node node_modules/tracelet-next/collect.js
 *
 * The script reads Next.js build manifests and generates .tracelet/stats.json
 */

// Export the collect function for programmatic use
module.exports = function collect() {
  require('./collect.js');
};

// Main entry point (can be called directly)
if (require.main === module) {
  module.exports();
}

