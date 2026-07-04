package probe

import (
	"context"
	"encoding/json"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"os"
)

type profileSettings struct {
	cpuSlowdown    float64
	downloadKbps   int64
	uploadKbps     int64
	latencyMs      int64
	viewportWidth  int64
	viewportHeight int64
}

func getProfileSettings(p Profile) profileSettings {
	if p == ProfileMobile {
		return profileSettings{
			cpuSlowdown:    4,
			downloadKbps:   1500,
			uploadKbps:     750,
			latencyMs:      150,
			viewportWidth:  375,
			viewportHeight: 667,
		}
	}
	return profileSettings{
		cpuSlowdown:    2,
		downloadKbps:   10000,
		uploadKbps:     5000,
		latencyMs:      40,
		viewportWidth:  1366,
		viewportHeight: 768,
	}
}

// runOnce executes a single probe run and returns collected metrics.
func runOnce(ctx context.Context, url string, ps profileSettings) (Metrics, error) {
	// Throttling + viewport
	if err := chromedp.Run(ctx,
		emulation.SetCPUThrottlingRate(ps.cpuSlowdown),
		emulation.SetDeviceMetricsOverride(ps.viewportWidth, ps.viewportHeight, 1.0, false),
		network.Enable(),
		network.SetCacheDisabled(true),
		network.EmulateNetworkConditions(false, float64(ps.latencyMs), float64(ps.downloadKbps*1024/8), float64(ps.uploadKbps*1024/8)),
	); err != nil {
		return Metrics{}, err
	}

	// Inject observers before any document loads
	inject := `
      (function(){
        if (window.__traceletMetrics) return;
        const data = { ttfb: 0, fcp: 0, lcp: 0, cls: 0, tbtLite: 0, fsi: 0 };
        // TTFB
        try {
          const nav = performance.getEntriesByType('navigation')[0];
          if (nav) {
            data.ttfb = Math.max(0, Math.round(nav.responseStart - nav.requestStart));
          } else if (performance.timing) {
            const t = performance.timing;
            data.ttfb = Math.max(0, t.responseStart - t.requestStart);
          }
        } catch(e){}
        // FCP
        try {
          const po = new PerformanceObserver((list)=>{
            for (const e of list.getEntries()) {
              if (e.name === 'first-contentful-paint' && !data.fcp) {
                data.fcp = Math.round(e.startTime);
              }
            }
          });
          po.observe({type:'paint', buffered:true});
        } catch(e){}
        // LCP
        try {
          let lastLcp = 0;
          const po = new PerformanceObserver((list)=>{
            for (const e of list.getEntries()) {
              lastLcp = e.startTime;
            }
            data.lcp = Math.round(lastLcp);
          });
          po.observe({type:'largest-contentful-paint', buffered:true});
          addEventListener('visibilitychange', ()=>{ if (document.visibilityState==='hidden'){ try{ po.takeRecords(); }catch(_){} } }, {once:true});
        } catch(e){}
        // CLS
        try {
          let cls = 0;
          const po = new PerformanceObserver((list)=>{
            for (const e of list.getEntries()) {
              if (!e.hadRecentInput) cls += e.value;
            }
            data.cls = Math.round(cls*1000)/1000;
          });
          po.observe({type:'layout-shift', buffered:true});
        } catch(e){}
        // TBT-Lite and FSI via long tasks
        try {
          let fcpTime = 0; let fsi = 0; let tbt = 0;
          const paintObs = new PerformanceObserver((list)=>{
            for (const e of list.getEntries()) {
              if (e.name==='first-contentful-paint') fcpTime = e.startTime;
            }
          });
          paintObs.observe({type:'paint', buffered:true});
          const longObs = new PerformanceObserver((list)=>{
            for (const e of list.getEntries()) {
              const start = e.startTime; const dur = e.duration;
              if (fcpTime>0 && start >= fcpTime && start <= fcpTime + 5000) {
                const over = Math.max(0, dur - 50);
                tbt += over;
              }
              if (!fsi && start > 0 && dur < 50 && start > (fcpTime||0)) {
                // naive FSI surrogate: first sub-50ms long idle proxy
                fsi = Math.round(start);
              }
            }
            data.tbtLite = Math.round(tbt);
            if (fsi) data.fsi = fsi;
          });
          longObs.observe({type:'longtask', buffered:true});
        } catch(e){}
        Object.defineProperty(window, '__traceletMetrics', { value: data, writable: false });
      })();
    `

	if err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		_, err := page.AddScriptToEvaluateOnNewDocument(inject).Do(ctx)
		return err
	})); err != nil {
		return Metrics{}, err
	}

	// Navigate and wait a short, deterministic time for observers to flush
	if err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(2*time.Second),
	); err != nil {
		return Metrics{}, err
	}

	var jsonStr string
	if err := chromedp.Run(ctx, chromedp.EvaluateAsDevTools("JSON.stringify(window.__traceletMetrics||{})", &jsonStr)); err != nil {
		return Metrics{}, err
	}
	var m Metrics
	_ = json.Unmarshal([]byte(jsonStr), &m)
	return m, nil
}

func runChromedp(url string, prof Profile, runs int, verbose bool) (Response, error) {
	if runs <= 0 {
		runs = 1
	}
	ps := getProfileSettings(prof)

	// Allocator with minimal flags for determinism
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Flag("disable-background-timer-throttling", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("headless", true),
		chromedp.Flag("hide-scrollbars", true),
		chromedp.Flag("mute-audio", true),
	)
	if exec := os.Getenv("CHROME_PATH"); exec != "" {
		opts = append(opts, chromedp.ExecPath(exec))
	}
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	ctx, cancelCtx := chromedp.NewContext(allocCtx)
	defer cancelCtx()

	// Run N times and average metrics
	var acc Metrics
	var samples []Metrics
	for i := 0; i < runs; i++ {
		m, err := runOnce(ctx, url, ps)
		if err != nil {
			return Response{}, err
		}
		acc.TTFBms += m.TTFBms
		acc.FCPms += m.FCPms
		acc.LCPms += m.LCPms
		acc.TBTms += m.TBTms
		acc.FSIms += m.FSIms
		acc.CLS += m.CLS
		if verbose {
			samples = append(samples, m)
		}
	}
	// average
	avg := Metrics{
		TTFBms: acc.TTFBms / runs,
		FCPms:  acc.FCPms / runs,
		LCPms:  acc.LCPms / runs,
		TBTms:  acc.TBTms / runs,
		FSIms:  acc.FSIms / runs,
		CLS:    acc.CLS / float64(runs),
	}
	return Response{
		URL:     url,
		Profile: string(prof),
		RunAt:   time.Now().UTC().Format(time.RFC3339),
		Metrics: avg,
		Runs:    runs,
		Samples: samples,
	}, nil
}
