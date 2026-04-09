package damon

import (
	"encoding/json"
	"fmt"
	"os"
)

// MonitoringAttrs intervals are in microseconds. Scheme age fields are in multiples of AggrUs.
type MonitoringAttrs struct {
	SampleUs   int `json:"sample_us"`
	AggrUs     int `json:"aggr_us"`
	UpdateUs   int `json:"update_us"`
	MinRegions int `json:"min_regions"`
	MaxRegions int `json:"max_regions"`
}

type StartConfig struct {
	Ops     string          `json:"ops"`
	Attrs   MonitoringAttrs `json:"monitoring_attrs"`
	Schemes []SchemeConfig  `json:"schemes"`
}

type SchemeConfig struct {
	Action        string          `json:"action"`
	MinSzBytes    int             `json:"min_sz_bytes"`
	MaxSzBytes    int             `json:"max_sz_bytes"`
	MinNrAccesses int             `json:"min_nr_accesses"`
	MaxNrAccesses int             `json:"max_nr_accesses"`
	MinAge        int             `json:"min_age"`
	MaxAge        int             `json:"max_age"`
	Quota         QuotaConfig     `json:"quota"`
	Watermarks    WatermarkConfig `json:"watermarks"`
}

type QuotaConfig struct {
	Ms              int `json:"ms"`
	Bytes           int `json:"bytes"`
	ResetIntervalMs int `json:"reset_interval_ms"`
	WeightSz        int `json:"weight_sz"`
	WeightAccesses  int `json:"weight_accesses"`
	WeightAge       int `json:"weight_age"`
}

type WatermarkConfig struct {
	Metric     string `json:"metric"`
	IntervalUs int    `json:"interval_us"`
	High       int    `json:"high"`
	Mid        int    `json:"mid"`
	Low        int    `json:"low"`
}

func DefaultMonitoringAttrs() MonitoringAttrs {
	return MonitoringAttrs{
		SampleUs:   5_000,
		AggrUs:     100_000,
		UpdateUs:   1_000_000,
		MinRegions: 10,
		MaxRegions: 1000,
	}
}

func DefaultSchemeConfig() SchemeConfig {
	return SchemeConfig{
		Action:        "pageout",
		MinSzBytes:    4096,
		MaxSzBytes:    1 << 62,
		MinNrAccesses: 0,
		MaxNrAccesses: 0,
		MinAge:        600,
		MaxAge:        1<<31 - 1,
		Quota: QuotaConfig{
			Ms:              10,
			Bytes:           128 << 20,
			ResetIntervalMs: 1000,
			WeightSz:        0,
			WeightAccesses:  0,
			WeightAge:       1,
		},
		Watermarks: WatermarkConfig{
			Metric:     "free_mem_rate",
			IntervalUs: 5_000_000,
			High:       500,
			Mid:        400,
			Low:        200,
		},
	}
}

func DefaultStartConfig() StartConfig {
	return StartConfig{
		Ops:     "vaddr",
		Attrs:   DefaultMonitoringAttrs(),
		Schemes: []SchemeConfig{DefaultSchemeConfig()},
	}
}

func LoadStartConfig(path string) (StartConfig, error) {
	cfg := DefaultStartConfig()
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read config %s: %w", path, err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config %s: %w", path, err)
	}
	return cfg, nil
}
