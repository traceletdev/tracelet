package hud

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"

	"tracelet/internal/config"
	"tracelet/internal/hud/frameworks"
	"tracelet/internal/lint"
)

type Event struct {
	Type   string      `json:"type"`
	Data   interface{} `json:"data"`
	SentAt string      `json:"sentAt"`
}

type RouteMetrics struct {
	TTFBms int     `json:"ttfb"`
	FCPms  int     `json:"fcp"`
	LCPms  int     `json:"lcp"`
	CLS    float64 `json:"cls"`
	TBTms  int     `json:"tbt"`
	FSIms  int     `json:"fsi"`
}

type Hub struct {
	mu         sync.Mutex
	conns      map[net.Conn]struct{}
	lastMsg    []byte
	metrics    map[string]RouteMetrics // route path -> metrics
	components []frameworks.Component  // current component tree
}

func NewHub() *Hub {
	return &Hub{
		conns:      make(map[net.Conn]struct{}),
		metrics:    make(map[string]RouteMetrics),
		components: []frameworks.Component{},
	}
}

func (h *Hub) Add(conn net.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.conns[conn] = struct{}{}
	if len(h.lastMsg) > 0 {
		_ = wsutil.WriteServerText(conn, h.lastMsg)
	}
}

func (h *Hub) Remove(conn net.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.conns, conn)
}

func (h *Hub) Broadcast(v interface{}) {
	b, _ := json.Marshal(v)
	h.mu.Lock()
	h.lastMsg = b
	defer h.mu.Unlock()
	for c := range h.conns {
		_ = wsutil.WriteServerText(c, b)
	}
}

func Start(port int, cfgPath string, version string) error {
	hub := NewHub()

	// If a config path is provided, change working directory to its directory
	if cfgPath != "" {
		abs, err := filepath.Abs(cfgPath)
		if err == nil {
			if dir := filepath.Dir(abs); dir != "" {
				_ = os.Chdir(dir)
			}
		}
	}

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, _, _, err := ws.UpgradeHTTP(r, w)
		if err != nil {
			return
		}
		hub.Add(conn)
		go func() {
			defer conn.Close()
			defer hub.Remove(conn)
			// Read messages from client (metrics updates)
			for {
				data, _, err := wsutil.ReadClientData(conn)
				if err != nil {
					log.Printf("[HUD] WebSocket read error: %v", err)
					return
				}
				log.Printf("[HUD] Received WebSocket message (length: %d bytes)", len(data))
				// Try to parse as metrics or components message
				var msg map[string]interface{}
				if err := json.Unmarshal(data, &msg); err == nil {
					if msgType, ok := msg["type"].(string); ok {
						if msgType == "components" {
							// Handle component tree updates
							log.Printf("[HUD] Received components message (raw data length: %d)", len(data))
							if componentsData, ok := msg["components"].([]interface{}); ok {
								log.Printf("[HUD] Parsing %d components", len(componentsData))
								var components []frameworks.Component
								for _, compData := range componentsData {
									if compMap, ok := compData.(map[string]interface{}); ok {
										var comp frameworks.Component
										if name, ok := compMap["name"].(string); ok {
											comp.Name = name
										}
										// Handle renderCount which might be float64 (from JSON) or already int
										if count, ok := compMap["renderCount"].(float64); ok {
											comp.RenderCount = int(count)
										} else if count, ok := compMap["renderCount"].(int); ok {
											comp.RenderCount = count
										}
										if v, ok := compMap["instances"].(float64); ok {
											comp.Instances = int(v)
										}
										if v, ok := compMap["maxRenders"].(float64); ok {
											comp.MaxRenders = int(v)
										}
										components = append(components, comp)
									}
								}
								log.Printf("[HUD] Storing %d components in hub", len(components))
								hub.mu.Lock()
								hub.components = components
								hub.mu.Unlock()
								// Broadcast to all clients
								evt := map[string]interface{}{
									"type":       "components",
									"components": components,
									"sentAt":     time.Now().UTC().Format(time.RFC3339),
								}
								log.Printf("[HUD] Broadcasting %d components to clients", len(components))
								hub.Broadcast(evt)
							} else {
								log.Printf("[HUD] Failed to parse components data: componentsData type assertion failed")
							}
							continue
						}
						if msgType == "metrics" {
							if route, ok := msg["route"].(string); ok {
								if metricsData, ok := msg["metrics"].(map[string]interface{}); ok {
									var m RouteMetrics
									if ttfb, ok := metricsData["ttfb"].(float64); ok {
										m.TTFBms = int(ttfb)
									}
									if fcp, ok := metricsData["fcp"].(float64); ok {
										m.FCPms = int(fcp)
									}
									if lcp, ok := metricsData["lcp"].(float64); ok {
										m.LCPms = int(lcp)
									}
									if cls, ok := metricsData["cls"].(float64); ok {
										m.CLS = cls
									}
									if tbt, ok := metricsData["tbt"].(float64); ok {
										m.TBTms = int(tbt)
									}
									if fsi, ok := metricsData["fsi"].(float64); ok {
										m.FSIms = int(fsi)
									}
									hub.mu.Lock()
									hub.metrics[route] = m
									hub.mu.Unlock()
									// Broadcast updated metrics to all clients
									evt := map[string]interface{}{
										"type":    "metrics",
										"route":   route,
										"metrics": m,
										"sentAt":  time.Now().UTC().Format(time.RFC3339),
									}
									hub.Broadcast(evt)
								}
							}
						}
					}
				}
			}
		}()
	})

	http.HandleFunc("/overlay.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		fmt.Fprint(w, overlayJS(port))
	})

	// /hook.js installs the React DevTools hook. It MUST load in <head> before
	// the app's React bundle so render tracking works (see overlay Components tab).
	http.HandleFunc("/hook.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		fmt.Fprint(w, getReactInstrumentationCode())
	})

	// /healthz reports which build is actually serving, so a spawner can tell a
	// leftover HUD from a previous session apart from the current one instead of
	// silently reusing whatever already holds the port.
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"version": version})
	})

	// periodic lint and broadcast
	go func() {
		for {
			cfg, err := config.Load(cfgPath)
			if err == nil {
				results, st := lint.Run(lint.Request{Scope: "all", Config: cfg})
				hub.mu.Lock()
				metricsCopy := make(map[string]RouteMetrics)
				for k, v := range hub.metrics {
					metricsCopy[k] = v
				}
				componentsCopy := make([]frameworks.Component, len(hub.components))
				copy(componentsCopy, hub.components)
				hub.mu.Unlock()
				if len(componentsCopy) > 0 {
					log.Printf("[HUD] Lint broadcast includes %d components", len(componentsCopy))
				}
				evt := map[string]interface{}{
					"type":       "lint",
					"results":    results,
					"stats":      st,
					"metrics":    metricsCopy,
					"components": componentsCopy,
					"sentAt":     time.Now().UTC().Format(time.RFC3339),
				}
				hub.Broadcast(evt)
			}
			time.Sleep(2 * time.Second)
		}
	}()

	addr := fmt.Sprintf(":%d", port)
	log.Printf("tracelet HUD listening on http://localhost%s (ws at /ws)\n", addr)
	return http.ListenAndServe(addr, nil)
}

func overlayJS(port int) string {
	// Read React instrumentation code from package
	reactCode := getReactInstrumentationCode()
	return fmt.Sprintf(`(function(){
      if (window.__traceletHUD) return;

      // Create container. Shadow DOM keeps the overlay's internals (ids,
      // structure) out of the host page's document.querySelectorAll results —
      // a host app with broad selectors (e.g. '[id]') should never see them.
      var container = document.createElement('div');
      container.style.cssText = 'position:fixed;bottom:16px;right:16px;z-index:2147483647;font-family:ui-sans-serif,system-ui,-apple-system;font-size:13px;pointer-events:auto;';
      var shadowRoot = container.attachShadow({mode: 'open'});

      // Create header (collapsible)
      var header = document.createElement('div');
      header.style.cssText = 'background:#111;color:#fff;padding:10px 14px;border-radius:8px 8px 0 0;cursor:pointer;user-select:none;display:flex;align-items:center;justify-content:space-between;box-shadow:0 2px 8px rgba(0,0,0,0.2);border-bottom:1px solid rgba(255,255,255,0.1);';

      var title = document.createElement('span');
      title.style.cssText = 'font-weight:600;display:flex;align-items:center;gap:8px;';
      title.innerHTML = '<span style="display:inline-block;width:8px;height:8px;border-radius:50%%;background:#3b82f6;animation:pulse 2s infinite;"></span>Tracelet';

      var toggle = document.createElement('span');
      toggle.style.cssText = 'color:#888;font-size:12px;transition:transform 0.2s;';
      toggle.textContent = '−';

      var refreshBtn = document.createElement('span');
      var currentRotation = 0;
      refreshBtn.style.cssText = 'color:#888;font-size:24px;margin-left:8px;cursor:pointer;border-radius:4px;transition:transform 1.2s ease;display:inline-flex;align-items:center;justify-content:center;width:20px;height:20px;';
      refreshBtn.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-refresh-cw" style="display:block;"><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/></svg>';
      refreshBtn.title = 'Refresh metrics';
      refreshBtn.addEventListener('click', function(e){
        e.stopPropagation();
        e.preventDefault();
        // Accumulate rotation on each click (rotate -360deg more each time)
        currentRotation -= 360;
        refreshBtn.style.transform = 'rotate(' + currentRotation + 'deg)';
        if (window.__traceletRefreshMetrics) {
          window.__traceletRefreshMetrics();
        }
      });

      var headerActions = document.createElement('span');
      headerActions.style.cssText = 'display:flex;align-items:center;gap:8px;';
      headerActions.appendChild(refreshBtn);
      headerActions.appendChild(toggle);
      header.appendChild(title);
      header.appendChild(headerActions);

      // Create content panel
      var content = document.createElement('div');
      content.style.cssText = 'background:rgba(17,17,17,0.95);backdrop-filter:blur(10px);color:#fff;padding:12px 14px;border-radius:0 0 8px 8px;box-shadow:0 4px 16px rgba(0,0,0,0.3);max-height:400px;overflow-y:auto;transition:max-height 0.3s ease,opacity 0.2s;';
      content.style.maxHeight = '400px';

      var tabs = document.createElement('div');
      tabs.style.cssText = 'display:flex;gap:4px;margin-bottom:12px;border-bottom:1px solid rgba(255,255,255,0.1);';
      var routesTab = document.createElement('button');
      routesTab.textContent = 'Routes';
      routesTab.style.cssText = 'background:transparent;border:none;color:#fff;padding:6px 12px;cursor:pointer;border-bottom:2px solid #3b82f6;font-size:12px;';
      var metricsTab = document.createElement('button');
      metricsTab.textContent = 'Metrics';
      metricsTab.style.cssText = 'background:transparent;border:none;color:#888;padding:6px 12px;cursor:pointer;border-bottom:2px solid transparent;font-size:12px;';
      var componentsTab = document.createElement('button');
      componentsTab.textContent = 'Components';
      componentsTab.style.cssText = 'background:transparent;border:none;color:#888;padding:6px 12px;cursor:pointer;border-bottom:2px solid transparent;font-size:12px;';
      tabs.appendChild(routesTab);
      tabs.appendChild(metricsTab);
      tabs.appendChild(componentsTab);
      content.appendChild(tabs);

      var routesList = document.createElement('div');
      routesList.style.cssText = 'display:flex;flex-direction:column;gap:8px;';
      content.appendChild(routesList);

      var metricsList = document.createElement('div');
      metricsList.style.cssText = 'display:none;flex-direction:column;gap:8px;';
      content.appendChild(metricsList);

      // Reset control for the Components tab: zero counts, do one interaction,
      // see exactly what it re-rendered.
      var componentsBar = document.createElement('div');
      componentsBar.style.cssText = 'display:none;justify-content:flex-end;margin-bottom:6px;';
      var resetBtn = document.createElement('button');
      resetBtn.textContent = '⟲ Reset counts';
      resetBtn.style.cssText = 'background:rgba(255,255,255,0.06);border:1px solid rgba(255,255,255,0.1);color:#ccc;font-size:10px;padding:4px 8px;border-radius:4px;cursor:pointer;';
      resetBtn.addEventListener('click', function(){
        if (window.__traceletReactInstrumentation) window.__traceletReactInstrumentation.reset();
        if (window.__traceletRefreshMetrics) window.__traceletRefreshMetrics();
        updateComponentsDisplay([]);
      });
      componentsBar.appendChild(resetBtn);
      content.appendChild(componentsBar);

      var componentsList = document.createElement('div');
      componentsList.style.cssText = 'display:none;flex-direction:column;gap:8px;';
      content.appendChild(componentsList);

      var currentTab = 'routes';
      function switchTab(tab){
        currentTab = tab;
        routesList.style.display = tab === 'routes' ? 'flex' : 'none';
        metricsList.style.display = tab === 'metrics' ? 'flex' : 'none';
        componentsList.style.display = tab === 'components' ? 'flex' : 'none';
        componentsBar.style.display = tab === 'components' ? 'flex' : 'none';
        routesTab.style.color = tab === 'routes' ? '#fff' : '#888';
        routesTab.style.borderBottomColor = tab === 'routes' ? '#3b82f6' : 'transparent';
        metricsTab.style.color = tab === 'metrics' ? '#fff' : '#888';
        metricsTab.style.borderBottomColor = tab === 'metrics' ? '#3b82f6' : 'transparent';
        componentsTab.style.color = tab === 'components' ? '#fff' : '#888';
        componentsTab.style.borderBottomColor = tab === 'components' ? '#3b82f6' : 'transparent';
      }
      routesTab.addEventListener('click', function(){ switchTab('routes'); });
      metricsTab.addEventListener('click', function(){ switchTab('metrics'); });
      componentsTab.addEventListener('click', function(){ switchTab('components'); });

      // Add pulse animation
      var style = document.createElement('style');
      style.textContent = '@keyframes pulse{0%%,100%%{opacity:1;}50%%{opacity:0.5;}}';
      shadowRoot.appendChild(style);

      shadowRoot.appendChild(header);
      shadowRoot.appendChild(content);

      var isCollapsed = false;
      header.addEventListener('click', function(){
        isCollapsed = !isCollapsed;
        if (isCollapsed) {
          content.style.maxHeight = '0';
          content.style.opacity = '0';
          content.style.padding = '0 14px';
          toggle.textContent = '+';
          toggle.style.transform = 'rotate(0deg)';
          header.style.borderBottomLeftRadius = '8px';
          header.style.borderBottomRightRadius = '8px';
        } else {
          content.style.maxHeight = '400px';
          content.style.opacity = '1';
          content.style.padding = '12px 14px';
          toggle.textContent = '−';
          toggle.style.transform = 'rotate(180deg)';
          header.style.borderBottomLeftRadius = '0';
          header.style.borderBottomRightRadius = '0';
        }
      });

      function attach(){
        if (document.body && !container.isConnected) {
          document.body.appendChild(container);
        }
      }
      if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', attach);
      } else {
        attach();
      }

      function fmtKB(bytes){ return (bytes/1024).toFixed(1)+'KB'; }
      function fmtMs(ms){ return (ms && ms > 0) ? ms+'ms' : '-'; }
      function fmtCls(cls){
        if (cls === undefined || cls === null) return '-';
        // Round to 2 decimal places: 0.007 -> 0.01, 0.003 -> 0.00
        var rounded = Math.round(cls * 100) / 100;
        return rounded.toFixed(2);
      }

      // React instrumentation injection
      (function(){
        if (window.__traceletReactInjected) return;
        window.__traceletReactInjected = true;
        try {
          var reactCode = %q;
          var script = document.createElement('script');
          script.textContent = reactCode;
          (document.head || document.documentElement).appendChild(script);
          // Don't remove script immediately - let it execute first
          // Scripts execute synchronously when textContent is set, but removing too early can cause issues
          setTimeout(function(){ try { script.remove(); } catch(e){} }, 0);
        } catch(e) {
          console.warn('[tracelet] Failed to inject React instrumentation:', e);
        }
      })();

      // Metrics collection
      (function(){
        if (window.__traceletMetricsCollector) return;
        window.__traceletMetricsCollector = true;
        var routeMetrics = {};
        var currentRoute = location.pathname;
        var metricsData = { ttfb: 0, fcp: 0, lcp: 0, cls: 0, tbt: 0, fsi: 0 };
        var fcpTime = 0;
        var lcpObserver = null;
        var clsObserver = null;
        var longTaskObserver = null;
        var paintObserver = null;

        // Initialize observers for continuous metrics collection
        function initObservers(){
          // TTFB - one-time from navigation timing
          try {
            var nav = performance.getEntriesByType('navigation')[0];
            if (nav) {
              metricsData.ttfb = Math.max(0, Math.round(nav.responseStart - nav.requestStart));
            } else if (performance.timing) {
              var t = performance.timing;
              metricsData.ttfb = Math.max(0, t.responseStart - t.requestStart);
            }
          } catch(e){}

          // FCP observer
          try {
            paintObserver = new PerformanceObserver(function(list){
              for (var i = 0; i < list.getEntries().length; i++) {
                var entry = list.getEntries()[i];
                if (entry.name === 'first-contentful-paint' && !metricsData.fcp) {
                  metricsData.fcp = Math.round(entry.startTime);
                  fcpTime = entry.startTime;
                }
              }
            });
            paintObserver.observe({type: 'paint', buffered: true});
          } catch(e){}

          // LCP observer - keeps updating with latest LCP
          try {
            lcpObserver = new PerformanceObserver(function(list){
              var entries = list.getEntries();
              if (entries.length > 0) {
                var lastEntry = entries[entries.length - 1];
                metricsData.lcp = Math.round(lastEntry.startTime);
              }
            });
            lcpObserver.observe({type: 'largest-contentful-paint', buffered: true});
            // Also get existing entries
            try {
              var lcpEntries = performance.getEntriesByType('largest-contentful-paint');
              if (lcpEntries.length > 0) {
                metricsData.lcp = Math.round(lcpEntries[lcpEntries.length - 1].startTime);
              }
            } catch(e){}
          } catch(e){}

          // CLS observer - accumulates layout shifts
          try {
            clsObserver = new PerformanceObserver(function(list){
              var entries = list.getEntries();
              for (var i = 0; i < entries.length; i++) {
                if (!entries[i].hadRecentInput) {
                  metricsData.cls += entries[i].value;
                }
              }
            });
            clsObserver.observe({type: 'layout-shift', buffered: true});
            // Also get existing entries
            try {
              var clsEntries = performance.getEntriesByType('layout-shift');
              for (var i = 0; i < clsEntries.length; i++) {
                if (!clsEntries[i].hadRecentInput) {
                  metricsData.cls += clsEntries[i].value;
                }
              }
            } catch(e){}
          } catch(e){}

          // Long task observer for TBT and FSI
          try {
            longTaskObserver = new PerformanceObserver(function(list){
              var entries = list.getEntries();
              for (var i = 0; i < entries.length; i++) {
                var entry = entries[i];
                var start = entry.startTime;
                var dur = entry.duration;
                // TBT: long tasks between FCP and FCP+5s
                if (fcpTime > 0 && start >= fcpTime && start <= fcpTime + 5000) {
                  metricsData.tbt += Math.max(0, dur - 50);
                }
                // FSI: first sub-50ms task after FCP
                if (!metricsData.fsi && dur < 50 && start > fcpTime && fcpTime > 0) {
                  metricsData.fsi = Math.round(start);
                }
              }
            });
            longTaskObserver.observe({type: 'longtask', buffered: true});
            // Also get existing entries
            try {
              var longTasks = performance.getEntriesByType('longtask');
              for (var i = 0; i < longTasks.length; i++) {
                var task = longTasks[i];
                var start = task.startTime;
                var dur = task.duration;
                if (fcpTime > 0 && start >= fcpTime && start <= fcpTime + 5000) {
                  metricsData.tbt += Math.max(0, dur - 50);
                }
                if (!metricsData.fsi && dur < 50 && start > fcpTime && fcpTime > 0) {
                  metricsData.fsi = Math.round(start);
                }
              }
            } catch(e){}
          } catch(e){}

          // Take records when page becomes hidden
          document.addEventListener('visibilitychange', function(){
            if (document.visibilityState === 'hidden') {
              try {
                if (lcpObserver) lcpObserver.takeRecords();
                if (clsObserver) clsObserver.takeRecords();
              } catch(e){}
            }
          }, {once: true});
        }

        initObservers();

        function getMetrics(){
          // Round TBT
          var tbtRounded = Math.round(metricsData.tbt);
          // Round CLS to 3 decimal places (don't round to 0)
          var clsRounded = Math.round(metricsData.cls * 1000) / 1000;
          return {
            ttfb: metricsData.ttfb || 0,
            fcp: metricsData.fcp || 0,
            lcp: metricsData.lcp || 0,
            cls: clsRounded,
            tbt: tbtRounded,
            fsi: metricsData.fsi || 0
          };
        }

      var wsConnection = null;

      function sendMetrics(route, metrics){
        if (wsConnection && wsConnection.readyState === WebSocket.OPEN) {
          wsConnection.send(JSON.stringify({
            type: 'metrics',
            route: route,
            metrics: metrics,
            sentAt: new Date().toISOString()
          }));
        }
      }

        function collectAndSend(){
          var route = location.pathname;
          var metrics = getMetrics();
          routeMetrics[route] = metrics;
          sendMetrics(route, metrics);
          updateMetricsDisplay();

          // Also collect and send component data if React instrumentation is available
          if (window.__traceletReactInstrumentation) {
            try {
              var components = window.__traceletReactInstrumentation.getComponents();
              if (window.__TRACELET_DEBUG) {
                console.log('[tracelet] About to send components:', components ? components.length : 'null');
                console.log('[tracelet] Components before stringify:', components);
              }
              // Clean components for JSON serialization (remove any non-serializable properties)
              var cleanComponents = [];
              if (components && Array.isArray(components)) {
                for (var i = 0; i < components.length; i++) {
                  var comp = components[i];
                  cleanComponents.push({
                    name: comp.name || 'Unknown',
                    renderCount: comp.renderCount || 0,
                    instances: comp.instances || 0,
                    maxRenders: comp.maxRenders || 0,
                    depth: comp.depth || 0
                  });
                }
              }
              if (window.__TRACELET_DEBUG) {
                console.log('[tracelet] Cleaned components:', cleanComponents.length);
                console.log('[tracelet] First cleaned component:', cleanComponents[0]);
                var wsState = wsConnection ? (wsConnection.readyState === WebSocket.OPEN ? 'OPEN' : 'state ' + wsConnection.readyState) : 'no connection';
                console.log('[tracelet] WebSocket state:', wsState);
                console.log('[tracelet] wsConnection exists:', !!wsConnection);
                if (wsConnection) {
                  console.log('[tracelet] wsConnection.readyState:', wsConnection.readyState, 'WebSocket.OPEN:', WebSocket.OPEN);
                }
              }
              // Always send, even if empty (for debugging)
              if (wsConnection && wsConnection.readyState === WebSocket.OPEN) {
                var msg = {
                  type: 'components',
                  components: cleanComponents,
                  sentAt: new Date().toISOString()
                };
                var msgStr = JSON.stringify(msg);
                if (window.__TRACELET_DEBUG) {
                  console.log('[tracelet] Sending message (length:', msgStr.length, '):', msgStr.substring(0, 300));
                  console.log('[tracelet] Components array in message:', msg.components ? msg.components.length : 'null');
                }
                try {
                  wsConnection.send(msgStr);
                  if (window.__TRACELET_DEBUG) {
                    console.log('[tracelet] Message sent successfully');
                  }
                } catch(e) {
                  if (window.__TRACELET_DEBUG) {
                    console.error('[tracelet] Error sending message:', e);
                  }
                }
              } else {
                if (window.__TRACELET_DEBUG) {
                  console.warn('[tracelet] WebSocket not open, cannot send components. State:', wsConnection ? wsConnection.readyState : 'no connection');
                }
              }
            } catch(e) {
              if (window.__TRACELET_DEBUG) console.warn('[tracelet] Component collection error:', e);
            }
          }
        }

        // Periodic component collection - use ws directly to avoid closure issues
        var wsForPeriodic = null;
        setInterval(function(){
          // Use wsConnection from outer scope, but also check ws directly
          var activeWS = wsConnection || wsForPeriodic;
          if (!activeWS && window.__traceletWS) {
            activeWS = window.__traceletWS;
          }

          if (window.__traceletReactInstrumentation) {
            try {
              var components = window.__traceletReactInstrumentation.getComponents();
              if (window.__TRACELET_DEBUG) {
                console.log('[tracelet] Periodic collection: found', components ? components.length : 0, 'components');
              }
              // Clean components for JSON serialization
              var cleanComponents = [];
              if (components && Array.isArray(components)) {
                for (var i = 0; i < components.length; i++) {
                  var comp = components[i];
                  cleanComponents.push({
                    name: comp.name || 'Unknown',
                    renderCount: comp.renderCount || 0,
                    instances: comp.instances || 0,
                    maxRenders: comp.maxRenders || 0,
                    depth: comp.depth || 0
                  });
                }
              }
              if (window.__TRACELET_DEBUG) {
                console.log('[tracelet] Periodic collection: cleaned', cleanComponents.length, 'components');
                var wsState = activeWS ? (activeWS.readyState === WebSocket.OPEN ? 'OPEN' : 'state ' + activeWS.readyState) : 'no connection';
                console.log('[tracelet] Periodic collection: WebSocket state:', wsState);
                console.log('[tracelet] Periodic collection: activeWS exists:', !!activeWS);
                console.log('[tracelet] Periodic collection: wsConnection exists:', !!wsConnection);
                if (activeWS) {
                  console.log('[tracelet] Periodic collection: activeWS.readyState:', activeWS.readyState, 'WebSocket.OPEN:', WebSocket.OPEN);
                }
              }
              // Always send component data, even if empty (to update UI state)
              if (activeWS && activeWS.readyState === WebSocket.OPEN) {
                var msg = {
                  type: 'components',
                  components: cleanComponents,
                  sentAt: new Date().toISOString()
                };
                var msgStr = JSON.stringify(msg);
                if (window.__TRACELET_DEBUG) {
                  console.log('[tracelet] Periodic collection: Sending', cleanComponents.length, 'components');
                }
                try {
                  activeWS.send(msgStr);
                  if (window.__TRACELET_DEBUG) {
                    console.log('[tracelet] Periodic collection: Message sent successfully');
                  }
                } catch(e) {
                  if (window.__TRACELET_DEBUG) {
                    console.error('[tracelet] Periodic collection: Error sending:', e);
                  }
                }
              } else {
                if (window.__TRACELET_DEBUG) {
                  console.warn('[tracelet] Periodic collection: WebSocket not open. activeWS:', !!activeWS, 'state:', activeWS ? activeWS.readyState : 'no connection');
                }
              }
            } catch(e) {
              if (window.__TRACELET_DEBUG) console.warn('[tracelet] Component collection error:', e);
            }
          } else {
            if (window.__TRACELET_DEBUG) {
              console.log('[tracelet] Periodic collection: No React instrumentation available');
            }
          }
        }, 3000);

        window.__traceletRefreshMetrics = collectAndSend;
        window.__traceletRouteMetrics = routeMetrics;

        // Queue initial collection to run after WebSocket is open
        var initialCollectionQueued = false;
        function queueInitialCollection() {
          if (wsConnection && wsConnection.readyState === WebSocket.OPEN) {
            // Initial collection - wait longer for LCP/CLS to be ready
            setTimeout(collectAndSend, 2000);
            // Also collect after a longer delay to catch LCP
            setTimeout(collectAndSend, 5000);
            initialCollectionQueued = true;
          } else if (!initialCollectionQueued) {
            // Retry after 100ms if WebSocket isn't ready yet
            setTimeout(queueInitialCollection, 100);
          }
        }
        queueInitialCollection();

        // Route change detection
        var originalPushState = history.pushState;
        var originalReplaceState = history.replaceState;
        history.pushState = function(){
          originalPushState.apply(history, arguments);
          setTimeout(collectAndSend, 500);
        };
        history.replaceState = function(){
          originalReplaceState.apply(history, arguments);
          setTimeout(collectAndSend, 500);
        };
        window.addEventListener('popstate', function(){
          setTimeout(collectAndSend, 500);
        });

        function updateMetricsDisplay(){
          var metricsEl = metricsList;
          if (!metricsEl) return;
          metricsEl.innerHTML = '';
          var routes = Object.keys(routeMetrics).sort();
          if (routes.length === 0) {
            var empty = document.createElement('div');
            empty.style.cssText = 'color:#888;text-align:center;padding:12px;font-size:12px;';
            empty.textContent = 'No metrics collected yet';
            metricsEl.appendChild(empty);
            return;
          }
          routes.forEach(function(route){
            var m = routeMetrics[route];
            var el = document.createElement('div');
            el.style.cssText = 'padding:12px;margin-bottom:8px;border-bottom:1px solid rgba(255,255,255,0.05);background:rgba(255,255,255,0.02);border-radius:4px;';
            var routeLabel = document.createElement('div');
            routeLabel.style.cssText = 'color:#3b82f6;font-family:ui-monospace,monospace;font-size:11px;margin-bottom:10px;font-weight:600;';
            routeLabel.textContent = route || '/';
            el.appendChild(routeLabel);
            var grid = document.createElement('div');
            grid.style.cssText = 'display:grid;grid-template-columns:1fr 1fr;gap:8px 12px;font-size:11px;';
            function getMetricColor(label, value, numValue){
              if (value === '-') return '#888';
              // Thresholds based on Web Vitals recommendations
              switch(label){
                case 'TTFB':
                  if (numValue <= 200) return '#10b981'; // good (green)
                  if (numValue <= 500) return '#f59e0b'; // needs improvement (yellow)
                  return '#ef4444'; // poor (red)
                case 'FCP':
                  if (numValue <= 1800) return '#10b981'; // good
                  if (numValue <= 3000) return '#f59e0b'; // needs improvement
                  return '#ef4444'; // poor
                case 'LCP':
                  if (numValue <= 2500) return '#10b981'; // good
                  if (numValue <= 4000) return '#f59e0b'; // needs improvement
                  return '#ef4444'; // poor
                case 'CLS':
                  if (numValue <= 0.1) return '#10b981'; // good
                  if (numValue <= 0.25) return '#f59e0b'; // needs improvement
                  return '#ef4444'; // poor
                case 'TBT':
                  if (numValue <= 200) return '#10b981'; // good
                  if (numValue <= 600) return '#f59e0b'; // needs improvement
                  return '#ef4444'; // poor
                case 'FSI':
                  if (numValue <= 100) return '#10b981'; // good
                  if (numValue <= 300) return '#f59e0b'; // needs improvement
                  return '#ef4444'; // poor
                default:
                  return '#e5e5e5';
              }
            }
            function addMetric(label, value, numValue){
              var color = getMetricColor(label, value, numValue || 0);
              var item = document.createElement('div');
              item.style.cssText = 'display:flex;justify-content:space-between;align-items:center;padding:4px 0;';
              var lbl = document.createElement('span');
              lbl.style.cssText = 'color:#888;font-size:11px;';
              lbl.textContent = label;
              var val = document.createElement('span');
              val.style.cssText = 'color:' + color + ';font-family:ui-monospace,monospace;font-size:11px;font-weight:500;margin-left:8px;';
              val.textContent = value;
              item.appendChild(lbl);
              item.appendChild(val);
              grid.appendChild(item);
            }
            addMetric('TTFB', fmtMs(m.ttfb), m.ttfb);
            addMetric('FCP', fmtMs(m.fcp), m.fcp);
            addMetric('LCP', fmtMs(m.lcp), m.lcp);
            addMetric('CLS', fmtCls(m.cls), m.cls);
            addMetric('TBT', fmtMs(m.tbt), m.tbt);
            addMetric('FSI', fmtMs(m.fsi), m.fsi);
            el.appendChild(grid);
            metricsEl.appendChild(el);
          });
        }
        window.__traceletUpdateMetricsDisplay = updateMetricsDisplay;
      })();

      function updateStatus(msg){
        // Handle lint messages
        if (msg.type === 'lint' && msg.stats && msg.stats.routes) {
          var routesEl = routesList;
          if (routesEl) {
            routesEl.innerHTML = '';
            var errorCount = 0;
            var warnCount = 0;
            msg.stats.routes.forEach(function(r){
              var routeEl = document.createElement('div');
              routeEl.style.cssText = 'display:flex;align-items:center;justify-content:space-between;padding:6px 0;border-bottom:1px solid rgba(255,255,255,0.05);';
              var status = 'ok';
              if (msg.results) {
                for (var i=0; i<msg.results.length; i++){
                  var it = msg.results[i];
                  if (it.ruleId === 'route-initial-js' && it.route === r.path) {
                    if (it.level === 'error') { status = 'error'; errorCount++; }
                    else if (it.level === 'warn' && status !== 'error') { status = 'warn'; warnCount++; }
                  }
                }
              }
              var pathEl = document.createElement('span');
              pathEl.style.cssText = 'color:#e5e5e5;font-family:ui-monospace,monospace;font-size:12px;flex:1;margin-right:8px;';
              pathEl.textContent = r.path || '/';
              var sizeEl = document.createElement('span');
              sizeEl.style.cssText = 'color:#a0a0a0;font-size:11px;margin-right:8px;';
              sizeEl.textContent = fmtKB(r.jsGzipBytes || 0);
              var iconEl = document.createElement('span');
              iconEl.style.cssText = 'font-size:14px;width:20px;text-align:center;';
              if (status === 'error') {
                iconEl.textContent = '❌';
                iconEl.style.color = '#ef4444';
              } else if (status === 'warn') {
                iconEl.textContent = '⚠️';
                iconEl.style.color = '#f59e0b';
              } else {
                iconEl.textContent = '✓';
                iconEl.style.color = '#10b981';
              }
              routeEl.appendChild(pathEl);
              routeEl.appendChild(sizeEl);
              routeEl.appendChild(iconEl);
              routesEl.appendChild(routeEl);
            });
            if (routesEl.children.length === 0) {
              var empty = document.createElement('div');
              empty.style.cssText = 'color:#888;text-align:center;padding:12px;font-size:12px;';
              empty.textContent = 'No routes found';
              routesEl.appendChild(empty);
            }
            // Update header indicator
            var indicator = header.querySelector('span span');
            if (indicator) {
              if (errorCount > 0) {
                indicator.style.background = '#ef4444';
              } else if (warnCount > 0) {
                indicator.style.background = '#f59e0b';
              } else {
                indicator.style.background = '#10b981';
              }
            }
          }
        }
        // Handle metrics messages
        if (msg.type === 'metrics' && window.__traceletRouteMetrics) {
          if (msg.route && msg.metrics) {
            window.__traceletRouteMetrics[msg.route] = msg.metrics;
          }
          if (window.__traceletUpdateMetricsDisplay) {
            window.__traceletUpdateMetricsDisplay();
          }
        }
        // Handle lint messages that include metrics
        if (msg.type === 'lint' && msg.metrics && window.__traceletRouteMetrics) {
          for (var route in msg.metrics) {
            window.__traceletRouteMetrics[route] = msg.metrics[route];
          }
          if (window.__traceletUpdateMetricsDisplay) {
            window.__traceletUpdateMetricsDisplay();
          }
        }
        // Handle lint messages that include components
        if (msg.type === 'lint' && msg.components) {
          if (window.__TRACELET_DEBUG) console.log('[tracelet] Received components in lint message:', msg.components ? msg.components.length : 0);
          updateComponentsDisplay(msg.components);
        }
        // Handle components messages
        if (msg.type === 'components' && msg.components) {
          if (window.__TRACELET_DEBUG) console.log('[tracelet] Received components message:', msg.components ? msg.components.length : 0);
          updateComponentsDisplay(msg.components);
        }
      }

      function updateComponentsDisplay(components){
        var componentsEl = componentsList;
        if (!componentsEl) {
          if (window.__TRACELET_DEBUG) console.warn('[tracelet] Components element not found');
          return;
        }
        componentsEl.innerHTML = '';

        // Debug logging
        if (window.__TRACELET_DEBUG) {
          console.log('[tracelet] updateComponentsDisplay called with:', components ? components.length : 'null', 'components');
          console.log('[tracelet] Components data:', components);
        }

        if (!components || components.length === 0) {
          var empty = document.createElement('div');
          empty.style.cssText = 'color:#888;text-align:center;padding:12px;font-size:12px;';
          var debugInfo = [];
          if (window.React) debugInfo.push('React found');
          if (window.__REACT_DEVTOOLS_GLOBAL_HOOK__) {
            var hook = window.__REACT_DEVTOOLS_GLOBAL_HOOK__;
            var rendererCount = hook.renderers ? Object.keys(hook.renderers).length : 0;
            debugInfo.push('DevTools hook found (' + rendererCount + ' renderer' + (rendererCount !== 1 ? 's' : '') + ')');
            if (hook.getFiberRoots) {
              var totalRoots = 0;
              try {
                for (var rendererId in hook.renderers || {}) {
                  var roots = hook.getFiberRoots(parseInt(rendererId));
                  if (roots) totalRoots += Array.from(roots).length;
                }
                debugInfo.push(totalRoots + ' root' + (totalRoots !== 1 ? 's' : ''));
              } catch(e) {}
            }
          }
          if (window.__traceletReactInstrumentation) {
            var counts = window.__traceletReactInstrumentation.getRenderCounts ? window.__traceletReactInstrumentation.getRenderCounts() : {};
            var countKeys = Object.keys(counts || {});
            var trackedCount = countKeys.filter(function(k){ return counts[k] > 0; }).length;
            debugInfo.push('Instrumentation loaded (' + trackedCount + ' component' + (trackedCount !== 1 ? 's' : '') + ' tracked)');
          }
          var msg = 'No React re-renders tracked yet.';
          msg += '\n\nRender tracking needs the tracelet hook to load BEFORE React. Add this to <head>, before your app bundle:';
          msg += '\n<script src="http://localhost:%d/hook.js"></script>';
          msg += '\n\nOr, in a bundled app, import "@traceletdev/react" as the first line of your dev entry.';
          if (debugInfo.length > 0) {
            msg += '\n\nStatus: ' + debugInfo.join(', ');
          }
          empty.style.whiteSpace = 'pre-line';
          empty.textContent = msg;
          componentsEl.appendChild(empty);
          return;
        }

        // Rank by the hottest single instance (maxRenders) — a lone instance
        // rendering 200x is the real smell; total is inflated by instance count.
        function maxOf(c){ return c.maxRenders || c.renderCount || 0; }
        var sorted = components.slice().sort(function(a, b){
          return maxOf(b) - maxOf(a);
        });

        sorted.forEach(function(comp){
          var el = document.createElement('div');
          el.style.cssText = 'padding:8px;margin-bottom:4px;border-bottom:1px solid rgba(255,255,255,0.05);background:rgba(255,255,255,0.02);border-radius:4px;';

          var nameRow = document.createElement('div');
          nameRow.style.cssText = 'display:flex;justify-content:space-between;align-items:center;';

          var nameEl = document.createElement('span');
          nameEl.style.cssText = 'color:#3b82f6;font-family:ui-monospace,monospace;font-size:11px;font-weight:600;';
          nameEl.textContent = comp.name || 'Unknown';
          nameRow.appendChild(nameEl);

          // Color by max-per-instance, not total.
          var mx = maxOf(comp);
          var total = comp.renderCount || 0;
          var countEl = document.createElement('span');
          countEl.style.cssText = 'color:' + (mx > 10 ? '#ef4444' : mx > 5 ? '#f59e0b' : '#10b981') + ';font-family:ui-monospace,monospace;font-size:11px;font-weight:600;';
          countEl.textContent = total + ' render' + (total !== 1 ? 's' : '');
          nameRow.appendChild(countEl);
          el.appendChild(nameRow);

          // Per-instance breakdown: "N instances · max M" (only when known).
          var inst = comp.instances || 0;
          if (inst > 0) {
            var sub = document.createElement('div');
            sub.style.cssText = 'display:flex;justify-content:space-between;color:#888;font-size:10px;font-family:ui-monospace,monospace;margin-top:3px;';
            var left = document.createElement('span');
            left.textContent = inst + ' instance' + (inst !== 1 ? 's' : '');
            var right = document.createElement('span');
            right.style.color = (mx > 10 ? '#ef4444' : mx > 5 ? '#f59e0b' : '#888');
            right.textContent = 'max ' + mx + '/instance';
            sub.appendChild(left);
            sub.appendChild(right);
            el.appendChild(sub);
          }

          componentsEl.appendChild(el);
        });
      }

      window.__traceletUpdateComponentsDisplay = updateComponentsDisplay;

      var ws = new WebSocket('ws://'+location.hostname+':%d/ws');
      wsConnection = ws;
      window.__traceletWS = ws; // Also expose globally for debugging
      if (window.__TRACELET_DEBUG) {
        console.log('[tracelet] WebSocket created, initial state:', ws.readyState);
      }
      ws.onopen = function(){
        var indicator = header.querySelector('span span');
        if (indicator) indicator.style.background = '#10b981';
        if (window.__TRACELET_DEBUG) {
          console.log('[tracelet] WebSocket connection opened');
          console.log('[tracelet] ws.readyState after open:', ws.readyState);
          console.log('[tracelet] wsConnection.readyState after open:', wsConnection ? wsConnection.readyState : 'null');
        }
        // Trigger initial collection now that WebSocket is ready
        if (window.__traceletRefreshMetrics && !window.__traceletInitialCollectionQueued) {
          window.__traceletInitialCollectionQueued = true;
          setTimeout(window.__traceletRefreshMetrics, 2000);
          setTimeout(window.__traceletRefreshMetrics, 5000);
        }
      };
      ws.onclose = function(){
        var indicator = header.querySelector('span span');
        if (indicator) indicator.style.background = '#888';
      };
      ws.onerror = function(){
        var indicator = header.querySelector('span span');
        if (indicator) indicator.style.background = '#ef4444';
      };
      ws.onmessage = function(ev){
        try{
          var msg = JSON.parse(ev.data);
          if (window.__TRACELET_DEBUG) {
            console.log('[tracelet] Received WebSocket message type:', msg.type);
            if (msg.type === 'components') {
              console.log('[tracelet] Components in message:', msg.components ? msg.components.length : 'null');
              console.log('[tracelet] First component:', msg.components && msg.components.length > 0 ? msg.components[0] : 'none');
            }
          }
          updateStatus(msg);
        }catch(e){
          if (window.__TRACELET_DEBUG) { console.error('[tracelet]', e); }
        }
      };

      // Initial state
      var empty = document.createElement('div');
      empty.style.cssText = 'color:#888;text-align:center;padding:12px;font-size:12px;';
      empty.textContent = 'Waiting for data…';
      routesList.appendChild(empty);

      window.__traceletHUD = true;
    })();`, reactCode, port, port)
}

// getReactInstrumentationCode returns the React instrumentation JavaScript code
func getReactInstrumentationCode() string {
	reactInst := frameworks.GetInstrumentation("react")
	if reactInst != nil {
		return reactInst.Install()
	}
	return "// React instrumentation not available"
}
