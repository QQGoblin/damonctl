package damon

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/QQGoblin/damonctl/pkg/utils"
)

func (k *Kdamon) SlotID() int {
	return k.slotID
}

func (k *Kdamon) Start(pid int, cfg StartConfig) error {
	if err := k.setupContext(cfg.Ops); err != nil {
		return fmt.Errorf("setup context: %w", err)
	}
	if err := k.SetMonitoringAttrs(cfg.Attrs); err != nil {
		return fmt.Errorf("set monitoring attrs: %w", err)
	}
	if err := k.setupTarget(pid, cfg.Ops); err != nil {
		return fmt.Errorf("setup target: %w", err)
	}
	if err := k.SetSchemes(cfg.Schemes); err != nil {
		return fmt.Errorf("set schemes: %w", err)
	}
	if err := k.turnOn(); err != nil {
		return fmt.Errorf("turn on: %w", err)
	}
	return nil
}

func (k *Kdamon) Stop() error {
	state, err := k.ReadState()
	if err != nil {
		return err
	}
	if state != "on" {
		return fmt.Errorf("kdamond %d is not running", k.slotID)
	}
	return k.turnOff()
}

func (k *Kdamon) SetMonitoringAttrs(attrs MonitoringAttrs) error {
	p, id := k.paths, k.slotID
	writes := []struct {
		path string
		val  int
	}{
		{p.SampleUs(id), attrs.SampleUs},
		{p.AggrUs(id), attrs.AggrUs},
		{p.UpdateUs(id), attrs.UpdateUs},
		{p.MinRegions(id), attrs.MinRegions},
		{p.MaxRegions(id), attrs.MaxRegions},
	}
	for _, w := range writes {
		if err := utils.WriteInt(w.path, w.val); err != nil {
			return err
		}
	}
	return nil
}

func (k *Kdamon) SetSchemes(schemes []SchemeConfig) error {
	p, id := k.paths, k.slotID
	if err := utils.WriteInt(p.NrSchemes(id), len(schemes)); err != nil {
		return fmt.Errorf("set nr_schemes: %w", err)
	}
	for i, s := range schemes {
		if err := k.writeScheme(i, s); err != nil {
			return fmt.Errorf("scheme %d: %w", i, err)
		}
	}
	return nil
}

func (k *Kdamon) IsRunning() (bool, error) {
	state, err := k.ReadState()
	if err != nil {
		return false, err
	}
	return state == "on", nil
}

func (k *Kdamon) ReadState() (string, error) {
	return utils.ReadString(k.paths.KdamondState(k.slotID))
}

func (k *Kdamon) ReadPid() (int, error) {
	return utils.ReadInt(k.paths.KdamondPid(k.slotID))
}

func (k *Kdamon) writeScheme(schemeID int, cfg SchemeConfig) error {
	p, id := k.paths, k.slotID

	if err := utils.WriteString(p.Action(id, schemeID), cfg.Action); err != nil {
		return err
	}

	apWrites := []struct {
		path string
		val  int
	}{
		{p.MinSz(id, schemeID), cfg.MinSzBytes},
		{p.MaxSz(id, schemeID), cfg.MaxSzBytes},
		{p.MinNrAccesses(id, schemeID), cfg.MinNrAccesses},
		{p.MaxNrAccesses(id, schemeID), cfg.MaxNrAccesses},
		{p.MinAge(id, schemeID), cfg.MinAge},
		{p.MaxAge(id, schemeID), cfg.MaxAge},
	}
	for _, w := range apWrites {
		if err := utils.WriteInt(w.path, w.val); err != nil {
			return err
		}
	}

	if q := cfg.Quota; q != nil {
		qWrites := []struct {
			path string
			val  int
		}{
			{p.QuotaMs(id, schemeID), q.Ms},
			{p.QuotaBytes(id, schemeID), q.Bytes},
			{p.QuotaResetIntervalMs(id, schemeID), q.ResetIntervalMs},
			{p.WeightSz(id, schemeID), q.WeightSz},
			{p.WeightNrAccesses(id, schemeID), q.WeightAccesses},
			{p.WeightAge(id, schemeID), q.WeightAge},
		}
		for _, w := range qWrites {
			if err := utils.WriteInt(w.path, w.val); err != nil {
				return err
			}
		}
	}

	if wm := cfg.Watermarks; wm != nil {
		if err := utils.WriteString(p.WatermarkMetric(id, schemeID), wm.Metric); err != nil {
			return err
		}
		wmWrites := []struct {
			path string
			val  int
		}{
			{p.WatermarkIntervalUs(id, schemeID), wm.IntervalUs},
			{p.WatermarkHigh(id, schemeID), wm.High},
			{p.WatermarkMid(id, schemeID), wm.Mid},
			{p.WatermarkLow(id, schemeID), wm.Low},
		}
		for _, w := range wmWrites {
			if err := utils.WriteInt(w.path, w.val); err != nil {
				return err
			}
		}
	}

	return nil
}

func (k *Kdamon) setupContext(ops string) error {
	p, id := k.paths, k.slotID
	if err := utils.WriteInt(p.NrContexts(id), 1); err != nil {
		return fmt.Errorf("set nr_contexts: %w", err)
	}
	return utils.WriteString(p.Operations(id), ops)
}

func (k *Kdamon) setupTarget(pid int, ops string) error {
	if ops == "paddr" {
		return k.setupPaddrTarget()
	}
	return k.setupVaddrTarget(pid)
}

func (k *Kdamon) setupVaddrTarget(pid int) error {
	p, id := k.paths, k.slotID
	if err := utils.WriteInt(p.NrTargets(id), 1); err != nil {
		return fmt.Errorf("set nr_targets: %w", err)
	}
	return utils.WriteInt(p.PidTarget(id, 0), pid)
}

func (k *Kdamon) setupPaddrTarget() error {
	p, id := k.paths, k.slotID
	if err := utils.WriteInt(p.NrTargets(id), 1); err != nil {
		return fmt.Errorf("set nr_targets: %w", err)
	}

	if err := utils.WriteInt(p.PidTarget(id, 0), 0); err != nil {
		return fmt.Errorf("set pid_target: %w", err)
	}

	if err := utils.WriteInt(p.TargetNrRegions(id, 0), 1); err != nil {
		return fmt.Errorf("set nr_regions: %w", err)
	}

	hostTotalMemory, err := utils.HostMemTotal()
	if err != nil {
		return fmt.Errorf("get host_total_memory: %w", err)
	}

	if err := utils.WriteInt(p.RegionStart(id, 0, 0), 0); err != nil {
		return fmt.Errorf("set nr_regions: %w", err)
	}

	return utils.WriteInt64(p.RegionEnd(id, 0, 0), hostTotalMemory)
}

func (k *Kdamon) turnOn() error {
	return utils.WriteString(k.paths.KdamondState(k.slotID), "on")
}

func (k *Kdamon) turnOff() error {
	return utils.WriteString(k.paths.KdamondState(k.slotID), "off")
}

func (k *Kdamon) UpdateSchemesTried() error {
	return utils.WriteString(k.paths.KdamondState(k.slotID), "update_schemes_tried_regions")
}

func (k *Kdamon) ReadTriedRegions() ([]SchemeTriedRegions, error) {
	p, id := k.paths, k.slotID

	nrSchemes, err := utils.ReadInt(p.NrSchemes(id))
	if err != nil {
		return nil, fmt.Errorf("read nr_schemes: %w", err)
	}

	results := make([]SchemeTriedRegions, 0, nrSchemes)
	for si := 0; si < nrSchemes; si++ {
		regions, err := k.schemeTriedRegions(si)
		if err != nil {
			return nil, fmt.Errorf("scheme %d tried_regions: %w", si, err)
		}
		results = append(results, SchemeTriedRegions{
			SchemeID: si,
			Regions:  regions,
		})
	}
	return results, nil
}

func (k *Kdamon) schemeTriedRegions(schemeID int) ([]TriedRegionInfo, error) {
	p, id := k.paths, k.slotID
	trDir := filepath.Join(p.ctx(id), "schemes", strconv.Itoa(schemeID), "tried_regions")

	entries, err := os.ReadDir(trDir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", trDir, err)
	}

	var regions []TriedRegionInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		regionID, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}

		startAddr, err := utils.ReadUint64(p.TriedRegionStart(id, schemeID, regionID))
		if err != nil {
			return nil, fmt.Errorf("region %d start: %w", regionID, err)
		}
		endAddr, err := utils.ReadUint64(p.TriedRegionEnd(id, schemeID, regionID))
		if err != nil {
			return nil, fmt.Errorf("region %d end: %w", regionID, err)
		}
		nrAccesses, err := utils.ReadInt(p.TriedRegionNrAccesses(id, schemeID, regionID))
		if err != nil {
			return nil, fmt.Errorf("region %d nr_accesses: %w", regionID, err)
		}
		age, err := utils.ReadInt(p.TriedRegionAge(id, schemeID, regionID))
		if err != nil {
			return nil, fmt.Errorf("region %d age: %w", regionID, err)
		}

		regions = append(regions, TriedRegionInfo{
			Start:      startAddr,
			End:        endAddr,
			NrAccesses: nrAccesses,
			Age:        age,
		})
	}
	return regions, nil
}
