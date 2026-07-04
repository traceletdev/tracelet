'use strict';

/**
 * Tracelet React Instrumentation
 *
 * Counts component re-renders by installing the React DevTools global hook
 * (window.__REACT_DEVTOOLS_GLOBAL_HOOK__) and inspecting each commit via
 * onCommitFiberRoot. This is the only reliable way to observe renders in a
 * real (bundled / JSX-runtime) app — patching React.createElement does not
 * work because compiled JSX never calls the global React.createElement.
 *
 * HARD REQUIREMENT: this must run BEFORE React initializes, because React
 * only reads __REACT_DEVTOOLS_GLOBAL_HOOK__ once, at module init. Load it via
 * <script src=".../hook.js"> in <head> before your bundle, or as the first
 * import in your app entry (dev only).
 *
 * ponytail: canonical source. Kept byte-identical with
 * packages/tracelet-react/index.js — edit both together.
 */

(function () {
  if (typeof window === 'undefined') return;
  if (window.__traceletReactInstrumentation) return;

  var DEBUG = false; // set window.__TRACELET_DEBUG = true to trace
  function log() {
    if (window.__TRACELET_DEBUG || DEBUG) {
      try { console.log.apply(console, ['[tracelet]'].concat([].slice.call(arguments))); } catch (e) {}
    }
  }

  var idMap = new WeakMap(); // fiber -> instance id (fibers are non-extensible,
                             // so we can't tag them directly — WeakMap works)
  var instances = {};      // instance id -> { name, count }
  var nextId = 1;          // monotonic; never reset (avoids id reuse collisions)

  // Stable per-instance id across the fiber double-buffer: current/alternate
  // swap each commit, so both halves map to the same id.
  function instanceId(fiber) {
    var id = idMap.get(fiber) || (fiber.alternate && idMap.get(fiber.alternate));
    if (!id) id = nextId++;
    idMap.set(fiber, id);
    if (fiber.alternate) idMap.set(fiber.alternate, id);
    return id;
  }

  function getDisplayName(fiber) {
    var type = fiber.type;
    if (typeof type === 'function') return type.displayName || type.name || null;
    if (typeof type === 'string') return null; // host component (div/span/...)
    if (type && typeof type === 'object') {
      // React.memo (type.type) or forwardRef (type.render)
      if (type.displayName) return type.displayName;
      var inner = type.type || type.render;
      if (typeof inner === 'function') return inner.displayName || inner.name || null;
    }
    return null;
  }

  // We walk the whole current tree each commit, so we must distinguish fibers
  // that rendered THIS commit from ones just sitting in a bailed-out subtree.
  // - Updated fibers have an alternate; they rendered iff memoizedProps or
  //   memoizedState changed vs it (a bailout reuses both references). The
  //   PerformedWork flag is unreliable — it stays stale on reused fibers.
  // - Never-updated fibers keep alternate === null forever, so we count their
  //   mount exactly once (rec.seen) rather than on every walk.
  function onCommit(root) {
    if (!root || !root.current) return;
    var stack = [root.current];
    var guard = 0;
    while (stack.length) {
      var fiber = stack.pop();
      if (!fiber) continue;
      var name = getDisplayName(fiber);
      if (name) {
        var id = instanceId(fiber);
        var rec = instances[id] || (instances[id] = { name: name, count: 0, seen: false });
        rec.name = name;
        var alt = fiber.alternate;
        var rendered = alt == null
          ? !rec.seen // mount: count once
          : (alt.memoizedProps !== fiber.memoizedProps || alt.memoizedState !== fiber.memoizedState);
        if (rendered) rec.count++;
        rec.seen = true;
      }
      if (fiber.child) stack.push(fiber.child);
      if (fiber.sibling) stack.push(fiber.sibling);
      if (++guard > 200000) break; // safety against cycles
    }
  }

  function safeCommit(root) {
    try { onCommit(root); } catch (e) { log('commit error', e); }
  }

  // Install (or wrap) the DevTools global hook.
  var existing = window.__REACT_DEVTOOLS_GLOBAL_HOOK__;
  if (existing) {
    // Real DevTools (or another tool) already present — chain onto it so we
    // don't break it. We only see commits from this point forward.
    var origOnCommit = existing.onCommitFiberRoot;
    existing.onCommitFiberRoot = function (id, root) {
      safeCommit(root);
      if (typeof origOnCommit === 'function') return origOnCommit.apply(this, arguments);
    };
    log('wrapped existing DevTools hook');
  } else {
    var renderers = new Map();
    var uid = 0;
    window.__REACT_DEVTOOLS_GLOBAL_HOOK__ = {
      renderers: renderers,
      supportsFiber: true,
      inject: function (renderer) { var id = ++uid; renderers.set(id, renderer); return id; },
      onCommitFiberRoot: function (id, root) { safeCommit(root); },
      onCommitFiberUnmount: function () {},
      on: function () {},
      emit: function () {},
      checkDCE: function () {}
    };
    log('installed minimal DevTools hook');
  }

  // Aggregate per-instance data by display name, keeping the instance count and
  // the hottest single instance — so 1 instance rendering 200x (a real smell) is
  // distinguishable from 40 instances rendering 5x each.
  function aggregate() {
    var byName = {};
    for (var id in instances) {
      var r = instances[id];
      if (r.count === 0) continue; // skip instances that haven't rendered (e.g. post-reset)
      var g = byName[r.name] || (byName[r.name] = { name: r.name, instances: 0, renderCount: 0, maxRenders: 0 });
      g.instances++;
      g.renderCount += r.count;
      if (r.count > g.maxRenders) g.maxRenders = r.count;
    }
    return byName;
  }

  window.__traceletReactInstrumentation = {
    getComponents: function () {
      var byName = aggregate();
      var out = [];
      for (var n in byName) out.push(byName[n]);
      return out;
    },
    getRenderCounts: function () {
      var byName = aggregate();
      var counts = {};
      for (var n in byName) counts[n] = byName[n].renderCount;
      return counts;
    },
    // Zero counts but keep instance records (and their `seen` flags) so already
    // mounted components aren't re-counted as fresh mounts — after reset, only
    // components that actually re-render show up.
    reset: function () { for (var id in instances) instances[id].count = 0; }
  };
})();

// Export for module systems (harmless/no-op in the browser <script> case).
if (typeof module !== 'undefined' && module.exports) {
  module.exports = (typeof window !== 'undefined') ? window.__traceletReactInstrumentation : {};
}
