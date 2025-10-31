package probe

import (
    "encoding/json"
    "io"
    "time"
)

type Profile string

const (
    ProfileDesktop Profile = "desktop"
    ProfileMobile  Profile = "mobile"
)

type Request struct {
    URL     string
    Profile Profile
    Runs    int
    Verbose bool
}

type Metrics struct {
    TTFBms int `json:"ttfb"`
    FCPms  int `json:"fcp"`
    LCPms  int `json:"lcp"`
    CLS    float64 `json:"cls"`
    TBTms  int `json:"tbtLite"`
    FSIms  int `json:"fsi"`
}

type Response struct {
    URL     string  `json:"url"`
    Profile string  `json:"profile"`
    RunAt   string  `json:"runAt"`
    Metrics Metrics `json:"metrics"`
    Note    string  `json:"note,omitempty"`
    Runs    int      `json:"runs,omitempty"`
    Samples []Metrics `json:"samples,omitempty"`
}

// Run executes the probe using a headless Chrome session via chromedp.
func Run(req Request) Response {
    resp, err := runChromedp(req.URL, req.Profile, req.Runs, req.Verbose)
    if err != nil {
        // Fall back to deterministic stub if Chrome is unavailable
        return Response{
            URL:     req.URL,
            Profile: string(req.Profile),
            RunAt:   time.Now().UTC().Format(time.RFC3339),
            Metrics: Metrics{TTFBms: 100, FCPms: 800, LCPms: 1200, CLS: 0.02, TBTms: 20, FSIms: 1500},
            Note:    "fallback stub metrics (chromedp error): " + err.Error(),
            Runs:    req.Runs,
        }
    }
    return resp
}

func WriteJSON(w io.Writer, r Response) error {
    enc := json.NewEncoder(w)
    enc.SetIndent("", "  ")
    return enc.Encode(r)
}


