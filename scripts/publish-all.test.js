'use strict';

/**
 * Self-check for setVersion's pin-sync. Run: node scripts/publish-all.test.js
 * Guards against the 0.6.0-shipped-with-0.5.1-pins regression.
 */

const fs = require('fs');
const os = require('os');
const path = require('path');
const assert = require('assert');

const { setVersion } = require('./publish-all.js');

const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'tracelet-pin-'));
fs.writeFileSync(
  path.join(dir, 'package.json'),
  JSON.stringify({
    name: '@traceletdev/cli',
    version: '0.5.1',
    optionalDependencies: {
      '@traceletdev/cli-darwin-arm64': '0.5.1',
      '@traceletdev/cli-linux-x64': '0.5.1',
      'some-third-party': '^1.2.3',
    },
  }) + '\n'
);

setVersion(dir, '0.6.0');

const out = JSON.parse(fs.readFileSync(path.join(dir, 'package.json'), 'utf8'));
assert.strictEqual(out.version, '0.6.0', 'version bumped');
assert.strictEqual(
  out.optionalDependencies['@traceletdev/cli-darwin-arm64'],
  '0.6.0',
  'platform pin bumped'
);
assert.strictEqual(
  out.optionalDependencies['@traceletdev/cli-linux-x64'],
  '0.6.0',
  'platform pin bumped'
);
assert.strictEqual(
  out.optionalDependencies['some-third-party'],
  '^1.2.3',
  'unrelated dep untouched'
);

fs.rmSync(dir, { recursive: true, force: true });
console.log('ok - setVersion keeps platform pins in lockstep');
