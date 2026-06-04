package mtune

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Reclaim ReclaimConfig `json:"reclaim"`
	Tune    TuneConfig    `json:"tune"`
}

type ReclaimConfig struct {
	MinAge               string `json:"min_age"`
	MinNrRegions         string `json:"min_nr_regions"`
	MaxNrRegions         string `json:"max_nr_regions"`
	SampleInterval       string `json:"sample_interval"`
	AggrInterval         string `json:"aggr_interval"`
	QuotaMs              string `json:"quota_ms"`
	QuotaSz              string `json:"quota_sz"`
	QuotaResetIntervalMs string `json:"quota_reset_interval_ms"`
	WmarksHigh           string `json:"wmarks_high"`
	WmarksMid            string `json:"wmarks_mid"`
	WmarksLow            string `json:"wmarks_low"`
	MonitorRegionStart   string `json:"monitor_region_start"`
	MonitorRegionEnd     string `json:"monitor_region_end"`
}

type TuneConfig struct {
	Interval       int64   `json:"interval"`
	AvailableBytes uint64  `json:"available_bytes"`
	AvailableRatio float64 `json:"available_ratio"`
	DeadRatio      float64 `json:"dead_ratio"`
	QuotaSzMin     uint64  `json:"quota_sz_min"`
	QuotaSzMax     uint64  `json:"quota_sz_max"`
	Gain           float64 `json:"gain"`
	SomePsiUs      uint64  `json:"some_psi_us"`
	PsiDeadRatio   float64 `json:"psi_dead_ratio"`
}

func DefaultTuneConfig() TuneConfig {
	return TuneConfig{
		Interval:       120,
		AvailableBytes: 20 * 1024 * 1024 * 1024,
		AvailableRatio: 0.10,
		DeadRatio:      0.05,
		QuotaSzMin:     64 * 1024 * 1024,
		QuotaSzMax:     1 * 1024 * 1024 * 1024,
		Gain:           10,
		SomePsiUs:      1 * 1000 * 1000,
		PsiDeadRatio:   0.05,
	}
}

func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}

	cfg := Config{Tune: DefaultTuneConfig()}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}

	if cfg.Reclaim == (ReclaimConfig{}) {
		var flat ReclaimConfig
		if err := json.Unmarshal(data, &flat); err == nil {
			cfg.Reclaim = flat
		}
	}

	return cfg, nil
}
