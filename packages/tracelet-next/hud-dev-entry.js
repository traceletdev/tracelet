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

  // Load hook.js first (React instrumentation must precede the app bundle).
  // overlay.js itself also tries to inject hook.js, but loading it here
  // gives it the best chance to run before React initialises.
  var hook = document.createElement('script');
  hook.src = 'http://localhost:' + port + '/hook.js';
  hook.async = false;
  document.currentScript
    ? document.currentScript.parentNode.insertBefore(hook, document.currentScript)
    : document.head.appendChild(hook);

  var overlay = document.createElement('script');
  overlay.src = src;
  overlay.async = true;
  document.currentScript
    ? document.currentScript.parentNode.insertBefore(overlay, document.currentScript)
    : document.head.appendChild(overlay);
})();
