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
	mu      sync.Mutex
	conns   map[net.Conn]struct{}
	lastMsg []byte
	metrics map[string]RouteMetrics // route path -> metrics
}

func NewHub() *Hub {
	return &Hub{
		conns:   make(map[net.Conn]struct{}),
		metrics: make(map[string]RouteMetrics),
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

func Start(port int, cfgPath string) error {
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
					return
				}
				// Try to parse as metrics message
				var msg map[string]interface{}
				if err := json.Unmarshal(data, &msg); err == nil {
					if msgType, ok := msg["type"].(string); ok && msgType == "metrics" {
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
		}()
	})

	http.HandleFunc("/overlay.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		fmt.Fprint(w, overlayJS(port))
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
				hub.mu.Unlock()
				evt := map[string]interface{}{
					"type":    "lint",
					"results": results,
					"stats":   st,
					"metrics": metricsCopy,
					"sentAt":  time.Now().UTC().Format(time.RFC3339),
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
	return fmt.Sprintf(`(function(){
      if (window.__traceletHUD) return;

      // Create container
      var container = document.createElement('div');
      container.id = '__tracelet_hud';
      container.style.cssText = 'position:fixed;bottom:16px;right:16px;z-index:2147483647;font-family:ui-sans-serif,system-ui,-apple-system;font-size:13px;pointer-events:auto;';

      // Create header (collapsible)
      var header = document.createElement('div');
      header.style.cssText = 'background:#111;color:#fff;padding:10px 14px;border-radius:8px 8px 0 0;cursor:pointer;user-select:none;display:flex;align-items:center;justify-content:space-between;box-shadow:0 2px 8px rgba(0,0,0,0.2);border-bottom:1px solid rgba(255,255,255,0.1);';

      var title = document.createElement('span');
      title.style.cssText = 'font-weight:600;display:flex;align-items:center;gap:8px;';
      title.innerHTML = '<span style="display:inline-block;width:8px;height:8px;border-radius:50%%;background:#3b82f6;animation:pulse 2s infinite;"></span>Tracelet';

      var toggle = document.createElement('span');
      toggle.style.cssText = 'color:#888;font-size:12px;transition:transform 0.2s;';
      toggle.textContent = '−';
      toggle.id = '__tracelet_toggle';

      var refreshBtn = document.createElement('span');
      refreshBtn.id = '__tracelet_refresh_btn';
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
      content.id = '__tracelet_content';
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
      tabs.appendChild(routesTab);
      tabs.appendChild(metricsTab);
      content.appendChild(tabs);

      var routesList = document.createElement('div');
      routesList.id = '__tracelet_routes';
      routesList.style.cssText = 'display:flex;flex-direction:column;gap:8px;';
      content.appendChild(routesList);

      var metricsList = document.createElement('div');
      metricsList.id = '__tracelet_metrics';
      metricsList.style.cssText = 'display:none;flex-direction:column;gap:8px;';
      content.appendChild(metricsList);

      var currentTab = 'routes';
      routesTab.addEventListener('click', function(){
        currentTab = 'routes';
        routesList.style.display = 'flex';
        metricsList.style.display = 'none';
        routesTab.style.color = '#fff';
        routesTab.style.borderBottomColor = '#3b82f6';
        metricsTab.style.color = '#888';
        metricsTab.style.borderBottomColor = 'transparent';
      });
      metricsTab.addEventListener('click', function(){
        currentTab = 'metrics';
        routesList.style.display = 'none';
        metricsList.style.display = 'flex';
        routesTab.style.color = '#888';
        routesTab.style.borderBottomColor = 'transparent';
        metricsTab.style.color = '#fff';
        metricsTab.style.borderBottomColor = '#3b82f6';
      });

      // Add pulse animation
      var style = document.createElement('style');
      style.textContent = '@keyframes pulse{0%%,100%%{opacity:1;}50%%{opacity:0.5;}}';
      document.head.appendChild(style);

      container.appendChild(header);
      container.appendChild(content);

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
        }

        window.__traceletRefreshMetrics = collectAndSend;
        window.__traceletRouteMetrics = routeMetrics;

        // Initial collection - wait longer for LCP/CLS to be ready
        setTimeout(collectAndSend, 2000);

        // Also collect after a longer delay to catch LCP
        setTimeout(collectAndSend, 5000);

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
          var metricsEl = document.getElementById('__tracelet_metrics');
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
          var routesEl = document.getElementById('__tracelet_routes');
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
      }

      var ws = new WebSocket('ws://'+location.hostname+':%d/ws');
      wsConnection = ws;
      ws.onopen = function(){
        var indicator = header.querySelector('span span');
        if (indicator) indicator.style.background = '#10b981';
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
          if (window.__TRACELET_DEBUG) { console.debug('[tracelet]', msg); }
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
    })();`, port)
}
