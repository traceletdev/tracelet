package ci

import (
	"encoding/json"
	"os"

	"tracelet/internal/lint"
	"tracelet/internal/stats"
)

type Baseline struct {
	Results lint.Results `json:"results"`
	Stats   stats.Stats  `json:"stats"`
}

func LoadBaseline(path string) (Baseline, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Baseline{}, err
	}
	var bl Baseline
	if err := json.Unmarshal(b, &bl); err != nil {
		return Baseline{}, err
	}
	return bl, nil
}

func SaveBaseline(path string, bl Baseline) error {
	if err := os.MkdirAll(".tracelet", 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(bl)
}

type Diff struct {
	ErrorsBefore int `json:"errorsBefore"`
	ErrorsAfter  int `json:"errorsAfter"`
	WarnsBefore  int `json:"warnsBefore"`
	WarnsAfter   int `json:"warnsAfter"`
}

func ComputeDiff(prev, curr lint.Results) Diff {
	count := func(rs lint.Results, level string) int {
		n := 0
		for _, r := range rs {
			if r.Level == level {
				n++
			}
		}
		return n
	}
	return Diff{
		ErrorsBefore: count(prev, "error"),
		ErrorsAfter:  count(curr, "error"),
		WarnsBefore:  count(prev, "warn"),
		WarnsAfter:   count(curr, "warn"),
	}
}
