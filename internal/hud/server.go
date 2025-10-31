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

type Hub struct {
	mu      sync.Mutex
	conns   map[net.Conn]struct{}
	lastMsg []byte
}

func NewHub() *Hub {
	return &Hub{conns: make(map[net.Conn]struct{})}
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
			// read pump to keep connection alive; discard
			for {
				if _, _, err := wsutil.ReadClientData(conn); err != nil {
					return
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
				evt := map[string]interface{}{
					"type":    "lint",
					"results": results,
					"stats":   st,
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

      header.appendChild(title);
      header.appendChild(toggle);

      // Create content panel
      var content = document.createElement('div');
      content.id = '__tracelet_content';
      content.style.cssText = 'background:rgba(17,17,17,0.95);backdrop-filter:blur(10px);color:#fff;padding:12px 14px;border-radius:0 0 8px 8px;box-shadow:0 4px 16px rgba(0,0,0,0.3);max-height:400px;overflow-y:auto;transition:max-height 0.3s ease,opacity 0.2s;';
      content.style.maxHeight = '400px';

      var routesList = document.createElement('div');
      routesList.id = '__tracelet_routes';
      routesList.style.cssText = 'display:flex;flex-direction:column;gap:8px;';
      content.appendChild(routesList);

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

      function updateStatus(msg){
        if (msg.type !== 'lint' || !msg.stats || !msg.stats.routes) return;

        var routesEl = document.getElementById('__tracelet_routes');
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
          routesList.appendChild(routeEl);
        });

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

        if (routesEl.children.length === 0) {
          var empty = document.createElement('div');
          empty.style.cssText = 'color:#888;text-align:center;padding:12px;font-size:12px;';
          empty.textContent = 'No routes found';
          routesEl.appendChild(empty);
        }
      }

      var ws = new WebSocket('ws://'+location.hostname+':%d/ws');
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
