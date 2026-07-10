/**
 * Tracelet HUD dev client entry.
 * Injected as a webpack entry during `next dev` by withTracelet().
 * Dynamically loads the overlay script so the HUD appears without any
 * manual <script> tag in the app.
 */
(function () {
  if (typeof window === 'undefined') return;
  if (window.__traceletHUDInjected) return;
  window.__traceletHUDInjected = true;

  var port = window.__TRACELET_HUD_PORT__ !== undefined ? window.__TRACELET_HUD_PORT__ : 3111;

  var src = 'http://localhost:' + port + '/overlay.js';

  function inject() {
    // Load hook.js first (React instrumentation must precede the app bundle).
    // overlay.js itself also tries to inject hook.js, but loading it here
    // gives it the best chance to run before React initialises. The fetch
    // itself is already async network I/O, so deferring the DOM write below
    // doesn't meaningfully change hook.js's real-world install time.
    var hook = document.createElement('script');
    hook.src = 'http://localhost:' + port + '/hook.js';
    hook.async = false;
    document.head.appendChild(hook);

    var overlay = document.createElement('script');
    overlay.src = src;
    overlay.async = true;
    document.head.appendChild(overlay);
  }

  // ponytail: defer past the synchronous hydration commit (setTimeout hands
  // control back to the event loop after hydrateRoot's initial pass), so
  // appending these <script> tags doesn't shift <head>'s children out from
  // under React mid-diff. Upgrade if a real "hydrated" signal is ever needed.
  setTimeout(inject, 0);
})();
