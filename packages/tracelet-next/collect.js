'use strict';

const fs = require('fs');
const path = require('path');
const zlib = require('zlib');

function readJSON(p) {
  try {
    return JSON.parse(fs.readFileSync(p, 'utf8'));
  } catch {
    return null;
  }
}

function gzipSize(p) {
  try {
    const content = fs.readFileSync(p);
    const compressed = zlib.gzipSync(content, { level: 6 });
    return compressed.length;
  } catch {
    return 0;
  }
}

function uniq(arr) {
  return Array.from(new Set(arr));
}

function sumFiles(projectRoot, relPaths) {
  let total = 0;
  let thirdParty = 0;
  for (const rel of uniq(relPaths)) {
    const full = path.join(projectRoot, '.next', rel);
    const size = gzipSize(full);
    total += size;
    // Detect third-party: chunks in node_modules or vendor directories
    // Next.js patterns:
    // - framework-*.js (React, Next.js core)
    // - main-app-*.js (Next.js app router)
    // - webpack-*.js (webpack runtime)
    // - Chunks with hash patterns often contain vendor code
    if (
      rel.includes('/node_modules/') ||
      rel.includes('/webpack/') ||
      rel.match(/^static\/chunks\/framework/) ||
      rel.match(/^static\/chunks\/main-app/) ||
      rel.match(/^static\/chunks\/webpack/) ||
      rel.match(/^static\/chunks\/[0-9a-f]{16}\.js$/)
    ) {
      thirdParty += size;
    }
  }
  return { total, thirdParty };
}

function collectRoutes(projectRoot) {
  const buildManifest = readJSON(path.join(projectRoot, '.next', 'build-manifest.json')) || {
    pages: {},
  };
  const prerenderManifest = readJSON(
    path.join(projectRoot, '.next', 'prerender-manifest.json')
  ) || { routes: {} };
  const appBuild = readJSON(path.join(projectRoot, '.next', 'app-build-manifest.json')) || {
    pages: {},
    rootMainFiles: [],
  };
  const appPathRoutes =
    readJSON(path.join(projectRoot, '.next', 'app-path-routes-manifest.json')) || {};

  const pages = buildManifest.pages || {};
  const appPages = appBuild.pages || {};

  // Build reverse map: route path -> manifest key (e.g., "/[locale]/money2020" -> "/(routes)/[locale]/money2020/page")
  const routePathToManifestKey = {};
  for (const [manifestKey, routePath] of Object.entries(appPathRoutes)) {
    if (manifestKey.endsWith('/page')) {
      routePathToManifestKey[routePath] = manifestKey;
    }
  }

  function normalizeAppRoute(k) {
    // Only count actual page routes; map '/page' → '/', '/foo/page' → '/foo'
    if (!k.endsWith('/page')) return null;
    let r = k.replace(/\/page$/, '') || '/';
    // Drop segment groups like '(routes)'
    r = r.replace(/\([^)]+\)\//g, '/');
    return r;
  }

  function isInternal(route) {
    if (!route) return true;
    if (route.startsWith('/_')) return true; // _app, _error, _not-found
    if (route.includes('/api/')) return true; // API routes
    if (route.endsWith('/route')) return true; // app dir route handlers
    return false;
  }

  function isDynamicRoute(route) {
    if (!route) return false;
    // Skip dynamic route patterns like /[locale], /[...slug], /[id]
    // These are templates, not concrete routes - concrete instances are already tracked
    return /\[/.test(route);
  }

  function isAssetRoute(route) {
    // Filter top-level non-page assets commonly present
    return route === '/favicon.ico' || route === '/robots.txt' || route === '/sitemap.xml';
  }

  function cleanRoute(route) {
    if (!route) return '/';
    let r = route.replace(/\/+/, '/');
    r = r.replace(/\/+/, '/'); // collapse again for edge cases
    r = r.replace(/\/+/g, '/');
    if (!r.startsWith('/')) r = '/' + r;
    if (r.length > 1 && r.endsWith('/')) r = r.slice(0, -1);
    return r;
  }

  const routeToFiles = new Map();
  const dynamicPatternToFiles = new Map(); // Store chunks for dynamic route patterns
  const routesFromDynamicPatterns = new Set(); // Track which routes were matched to dynamic patterns

  // Helper: load nested manifest for routes not in main app-build-manifest
  function loadNestedManifest(manifestKey) {
    // Convert manifest key like "/(routes)/[locale]/money2020/page" to file path
    const relPath = manifestKey.replace(/^\/|\/page$/g, '');
    const manifestPath = path.join(
      projectRoot,
      '.next',
      'server',
      'app',
      relPath,
      'page',
      'app-build-manifest.json'
    );
    const nested = readJSON(manifestPath);
    return nested ? nested.pages || {} : null;
  }

  // App Router pages - collect both concrete routes and dynamic patterns
  // Note: rootMainFiles are shared runtime chunks, NOT counted per route
  for (const k of Object.keys(appPages)) {
    let r = normalizeAppRoute(k);
    if (!r || isInternal(r)) continue;
    r = cleanRoute(r);
    if (isAssetRoute(r)) continue;
    // Only use route-specific chunks, exclude shared rootMainFiles
    const files = appPages[k] || [];
    if (isDynamicRoute(r)) {
      // Store chunks for dynamic route patterns - concrete routes will use these
      dynamicPatternToFiles.set(r, files);
    } else {
      routeToFiles.set(r, (routeToFiles.get(r) || []).concat(files));
    }
  }

  // Check app-path-routes-manifest for routes not in main app-build-manifest
  // These are nested routes like /[locale]/money2020
  for (const [routePath, manifestKey] of Object.entries(routePathToManifestKey)) {
    // Skip if already processed from main app-build-manifest
    if (appPages[manifestKey] || routePath.startsWith('/api/')) continue;
    let r = cleanRoute(routePath);
    if (isInternal(r) || isAssetRoute(r)) continue;

    // Try to load nested manifest
    const nestedPages = loadNestedManifest(manifestKey);
    let files = [];
    if (nestedPages && nestedPages[manifestKey]) {
      files = nestedPages[manifestKey] || [];
    } else if (nestedPages) {
      // If nested manifest exists but key doesn't match, try first page entry
      files = Object.values(nestedPages)[0] || [];
    }

    // If nested route has no chunks, inherit from parent pattern
    // e.g., /[locale]/money2020 inherits from /[locale]
    if (files.length === 0 && isDynamicRoute(r)) {
      const parts = r.split('/').filter(Boolean);
      if (parts.length > 1) {
        const parentPattern = '/' + parts[0]; // /[locale]
        if (dynamicPatternToFiles.has(parentPattern)) {
          files = dynamicPatternToFiles.get(parentPattern);
        }
      }
    }

    if (isDynamicRoute(r)) {
      dynamicPatternToFiles.set(r, files);
    } else {
      routeToFiles.set(r, (routeToFiles.get(r) || []).concat(files));
    }
  }

  // Pages Router fallback
  for (const key of Object.keys(pages)) {
    if (isInternal(key)) continue;
    let r = cleanRoute(key);
    if (isAssetRoute(r)) continue;
    const files = pages[key] || [];
    const runtime = buildManifest.runtime || [];
    if (isDynamicRoute(r)) {
      dynamicPatternToFiles.set(r, files.concat(runtime));
    } else {
      routeToFiles.set(r, (routeToFiles.get(r) || []).concat(files, runtime));
    }
  }

  // Helper: find which dynamic pattern a concrete route matches
  // Only matches routes that are actually from dynamic patterns (not just regex coincidence)
  function findMatchingPattern(concreteRoute) {
    // Check prerender manifest's dynamicRoutes to see if this route is from a dynamic pattern
    const dynamicRoutes = prerenderManifest.dynamicRoutes || {};
    for (const [patternPath] of Object.entries(dynamicRoutes)) {
      // Convert pattern to regex: /[locale] -> /[^/]+
      const regexStr = patternPath
        .replace(/\[\.\.\.(\w+)\]/, '.*') // catch-all: /[...slug] -> .*
        .replace(/\[(\w+)\]/g, '[^/]+'); // single: /[locale] -> [^/]+
      const regex = new RegExp(`^${regexStr}$`);
      if (regex.test(concreteRoute)) {
        // Verify this pattern exists in our dynamic patterns
        const cleanPattern = cleanRoute(patternPath);
        if (dynamicPatternToFiles.has(cleanPattern)) {
          return cleanPattern;
        }
      }
    }
    return null;
  }

  // Prerender routes: map concrete routes to their dynamic pattern chunks
  for (const key of Object.keys(prerenderManifest.routes || {})) {
    if (isInternal(key)) continue;
    let r = cleanRoute(key);
    if (isAssetRoute(r)) continue;
    if (isDynamicRoute(r)) continue; // Skip dynamic route patterns themselves
    if (!routeToFiles.has(r)) {
      // Try to find matching dynamic pattern
      const pattern = findMatchingPattern(r);
      if (pattern && dynamicPatternToFiles.has(pattern)) {
        routeToFiles.set(r, dynamicPatternToFiles.get(pattern));
        routesFromDynamicPatterns.add(r); // Mark as coming from dynamic pattern
      } else {
        routeToFiles.set(r, []); // No chunks found
      }
    }

    // Note: We track route patterns (e.g., /[locale], /[locale]/money2020) in the output,
    // not concrete instances (e.g., /en, /en/money2020). The patterns represent all instances.
  }

  // Add dynamic route patterns to output (e.g., /[locale], /[locale]/money2020)
  // These represent the patterns themselves, not concrete instances
  for (const [pattern, files] of dynamicPatternToFiles.entries()) {
    // Only add if not already in routeToFiles (avoid duplicates)
    if (!routeToFiles.has(pattern)) {
      routeToFiles.set(pattern, files);
    }
  }

  const routes = [];
  for (const [route, files] of routeToFiles.entries()) {
    const { total, thirdParty } = sumFiles(projectRoot, files);
    routes.push({
      path: route,
      jsGzipBytes: total,
      thirdPartyJsBytes: thirdParty,
    });
  }
  return routes;
}

function main() {
  const projectRoot = process.cwd();
  const routes = collectRoutes(projectRoot);
  const outPath = path.join(projectRoot, '.tracelet', 'stats.json');
  fs.mkdirSync(path.dirname(outPath), { recursive: true });
  fs.writeFileSync(outPath, JSON.stringify({ routes }, null, 2));
  console.log(`[tracelet] wrote stats to ${path.relative(projectRoot, outPath)}`);
}

if (require.main === module) {
  main();
}

// Export for programmatic use
module.exports = main;
