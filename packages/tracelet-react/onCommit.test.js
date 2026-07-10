'use strict';

/**
 * Self-check for the render-counter's bailout handling. Run: node onCommit.test.js
 *
 * Reproduces the double-buffer scenario that used to inflate counts: a fiber
 * that rendered a few times (so its two work buffers hold divergent
 * memoizedProps) and is then reused-untouched on unrelated commits. The walker
 * revisits it every commit and must NOT keep counting it.
 */

const assert = require('assert');

// The module bails out unless `window` exists; give it one, then load it.
global.window = {};
require('./index.js');

const inst = global.window.__traceletReactInstrumentation;
const hook = global.window.__REACT_DEVTOOLS_GLOBAL_HOOK__;

function Comp() {}
function commit(fiber) {
  hook.onCommitFiberRoot(1, { current: fiber });
}
function countOf(name) {
  const c = inst.getComponents().find(x => x.name === name);
  return c ? c.renderCount : 0;
}

// Two work buffers for one logical instance, alternates of each other.
const fA = { type: Comp, memoizedProps: null, memoizedState: null, child: null, sibling: null };
const fB = { type: Comp, memoizedProps: null, memoizedState: null, child: null, sibling: null };
fA.alternate = fB;
fB.alternate = fA;

// Three real renders, ping-ponging buffers with fresh props each time.
fA.memoizedProps = { v: 1 };
commit(fA); // mount
fB.memoizedProps = { v: 2 };
commit(fB); // re-render
fA.memoizedProps = { v: 3 };
commit(fA); // re-render
assert.strictEqual(countOf('Comp'), 3, 'three real renders should count 3');

// 50 foreign commits: Comp bailed out and was reused untouched (current stays
// fA, memoizedProps unchanged). The buffers still hold {v:3} vs {v:2}.
for (let i = 0; i < 50; i++) commit(fA);
assert.strictEqual(countOf('Comp'), 3, 'bailed-out reuse must not inflate the count');

// A genuine re-render after the idle period still counts.
fB.memoizedProps = { v: 4 };
commit(fB);
assert.strictEqual(countOf('Comp'), 4, 'a real render after idle should count');

// reset() zeros counts; a subsequent reused-fiber commit stays at 0.
inst.reset();
commit(fB);
assert.strictEqual(countOf('Comp'), 0, 'reset then reuse should stay 0');

console.log('ok - onCommit bailout handling');
