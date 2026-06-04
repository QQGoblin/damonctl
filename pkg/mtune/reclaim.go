package mtune

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/QQGoblin/damonctl/pkg/utils"
)

const defaultReclaimParameters = "/sys/module/damon_reclaim/parameters"

var ErrModuleNotAvailable = errors.New("damon_reclaim module is not available")

type Controller struct {
	reclaimConfig *ReclaimConfig
	tuneConfig    *TuneConfig
	quotaSz       uint64
	aggrSec       uint64
	lastSomePsiUs uint64
}

func NewController(cfg Config) (*Controller, error) {
	ctl := &Controller{
		reclaimConfig: &cfg.Reclaim,
		tuneConfig:    &cfg.Tune,
	}

	if err := ctl.CheckAvailable(); err != nil {
		return nil, err
	}

	return ctl, nil
}

func (p *Controller) Update(name string, val string) error {
	return utils.WriteString(filepath.Join(defaultReclaimParameters, name), val)
}

func (p *Controller) Read(name string) (string, error) {
	return utils.ReadString(filepath.Join(defaultReclaimParameters, name))
}

func (p *Controller) ReadUInt64(name string) (uint64, error) {
	data, err := utils.ReadString(filepath.Join(defaultReclaimParameters, name))
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(data, 10, 64)
}

func (p *Controller) Enabled() error {
	return utils.WriteString(filepath.Join(defaultReclaimParameters, "enabled"), "Y")
}

func (p *Controller) CommitInputs() error {
	return utils.WriteString(filepath.Join(defaultReclaimParameters, "commit_inputs"), "Y")
}

func (p *Controller) CheckAvailable() error {
	info, err := os.Stat(defaultReclaimParameters)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrModuleNotAvailable
		}
		return fmt.Errorf("stat %s: %w", defaultReclaimParameters, err)
	}
	if !info.IsDir() {
		return ErrModuleNotAvailable
	}
	return nil
}

func (p *Controller) Initialize(cfg ReclaimConfig) error {
	monitorStart := cfg.MonitorRegionStart
	if monitorStart == "" {
		monitorStart = "0"
	}
	monitorEnd := cfg.MonitorRegionEnd
	if monitorEnd == "" {
		memInfo, err := utils.HostMemInfo()
		if err != nil {
			return err
		}
		monitorEnd = strconv.FormatUint(*memInfo.MemTotalBytes, 10)
	}

	parameters := map[string]string{
		"min_age":                 cfg.MinAge,
		"min_nr_regions":          cfg.MinNrRegions,
		"max_nr_regions":          cfg.MaxNrRegions,
		"sample_interval":         cfg.SampleInterval,
		"aggr_interval":           cfg.AggrInterval,
		"quota_ms":                cfg.QuotaMs,
		"quota_sz":                cfg.QuotaSz,
		"quota_reset_interval_ms": cfg.QuotaResetIntervalMs,
		"wmarks_high":             cfg.WmarksHigh,
		"wmarks_mid":              cfg.WmarksMid,
		"wmarks_low":              cfg.WmarksLow,
		"monitor_region_start":    monitorStart,
		"monitor_region_end":      monitorEnd,
	}

	for key, value := range parameters {

		if err := p.Update(key, value); err != nil {
			return fmt.Errorf("update %s: %w", key, err)
		}
	}

	if err := p.Enabled(); err != nil {
		return fmt.Errorf("enable reclaim module: %w", err)
	}

	if err := p.CommitInputs(); err != nil {
		return fmt.Errorf("enable reclaim module: %w", err)
	}
	return nil
}
