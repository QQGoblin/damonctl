package damon

import (
	"path/filepath"
	"strconv"
)

const defaultDamonRoot = "/sys/kernel/mm/damon/admin"

// PathBuilder centralizes all DAMON sysfs path construction.
// It mirrors the kernel sysfs hierarchy:
//
//	root/kdamonds/
//	├── nr_kdamonds
//	└── {slotID}/
//	    ├── state
//	    ├── pid
//	    └── contexts/
//	        ├── nr_contexts
//	        └── 0/
//	            ├── operations
//	            ├── monitoring_attrs/intervals/{sample_us,aggr_us,update_us}
//	            ├── monitoring_attrs/nr_regions/{min,max}
//	            ├── targets/nr_targets
//	            ├── targets/{targetID}/pid_target
//	            └── schemes/
//	                ├── nr_schemes
//	                └── {schemeID}/
//	                    ├── action
//	                    ├── access_pattern/{sz,nr_accesses,age}/{min,max}
//	                    ├── quotas/{ms,bytes,reset_interval_ms}
//	                    ├── quotas/weights/{sz_permil,nr_accesses_permil,age_permil}
//	                    └── watermarks/{metric,interval_us,high,mid,low}
type PathBuilder struct {
	root string
}

func newSysfsPath(root string) PathBuilder {
	return PathBuilder{root: root}
}

func (p PathBuilder) kdamonds() string {
	return filepath.Join(p.root, "kdamonds")
}

func (p PathBuilder) kdamond(slotID int) string {
	return filepath.Join(p.kdamonds(), strconv.Itoa(slotID))
}

func (p PathBuilder) contexts(slotID int) string {
	return filepath.Join(p.kdamond(slotID), "contexts")
}

func (p PathBuilder) context(slotID int) string {
	return filepath.Join(p.contexts(slotID), "0")
}

func (p PathBuilder) monitoringAttrs(slotID int) string {
	return filepath.Join(p.context(slotID), "monitoring_attrs")
}

func (p PathBuilder) targets(slotID int) string {
	return filepath.Join(p.context(slotID), "targets")
}

func (p PathBuilder) target(slotID, targetID int) string {
	return filepath.Join(p.targets(slotID), strconv.Itoa(targetID))
}

func (p PathBuilder) schemes(slotID int) string {
	return filepath.Join(p.context(slotID), "schemes")
}

func (p PathBuilder) scheme(slotID, schemeID int) string {
	return filepath.Join(p.schemes(slotID), strconv.Itoa(schemeID))
}

func (p PathBuilder) NrKdamonds() string {
	return filepath.Join(p.kdamonds(), "nr_kdamonds")
}

func (p PathBuilder) KdamondState(slotID int) string {
	return filepath.Join(p.kdamond(slotID), "state")
}

func (p PathBuilder) KdamondPid(slotID int) string {
	return filepath.Join(p.kdamond(slotID), "pid")
}

func (p PathBuilder) NrContexts(slotID int) string {
	return filepath.Join(p.contexts(slotID), "nr_contexts")
}

func (p PathBuilder) Operations(slotID int) string {
	return filepath.Join(p.context(slotID), "operations")
}

func (p PathBuilder) SampleUs(slotID int) string {
	return filepath.Join(p.monitoringAttrs(slotID), "intervals", "sample_us")
}

func (p PathBuilder) AggrUs(slotID int) string {
	return filepath.Join(p.monitoringAttrs(slotID), "intervals", "aggr_us")
}

func (p PathBuilder) UpdateUs(slotID int) string {
	return filepath.Join(p.monitoringAttrs(slotID), "intervals", "update_us")
}

func (p PathBuilder) MinRegions(slotID int) string {
	return filepath.Join(p.monitoringAttrs(slotID), "nr_regions", "min")
}

func (p PathBuilder) MaxRegions(slotID int) string {
	return filepath.Join(p.monitoringAttrs(slotID), "nr_regions", "max")
}

func (p PathBuilder) NrTargets(slotID int) string {
	return filepath.Join(p.targets(slotID), "nr_targets")
}

func (p PathBuilder) PidTarget(slotID, targetID int) string {
	return filepath.Join(p.target(slotID, targetID), "pid_target")
}

func (p PathBuilder) NrSchemes(slotID int) string {
	return filepath.Join(p.schemes(slotID), "nr_schemes")
}

func (p PathBuilder) Action(slotID, schemeID int) string {
	return filepath.Join(p.scheme(slotID, schemeID), "action")
}

func (p PathBuilder) MinSz(slotID, schemeID int) string {
	return filepath.Join(p.scheme(slotID, schemeID), "access_pattern", "sz", "min")
}

func (p PathBuilder) MaxSz(slotID, schemeID int) string {
	return filepath.Join(p.scheme(slotID, schemeID), "access_pattern", "sz", "max")
}

func (p PathBuilder) MinNrAccesses(slotID, schemeID int) string {
	return filepath.Join(p.scheme(slotID, schemeID), "access_pattern", "nr_accesses", "min")
}

func (p PathBuilder) MaxNrAccesses(slotID, schemeID int) string {
	return filepath.Join(p.scheme(slotID, schemeID), "access_pattern", "nr_accesses", "max")
}

func (p PathBuilder) MinAge(slotID, schemeID int) string {
	return filepath.Join(p.scheme(slotID, schemeID), "access_pattern", "age", "min")
}

func (p PathBuilder) MaxAge(slotID, schemeID int) string {
	return filepath.Join(p.scheme(slotID, schemeID), "access_pattern", "age", "max")
}

func (p PathBuilder) QuotaMs(slotID, schemeID int) string {
	return filepath.Join(p.scheme(slotID, schemeID), "quotas", "ms")
}

func (p PathBuilder) QuotaBytes(slotID, schemeID int) string {
	return filepath.Join(p.scheme(slotID, schemeID), "quotas", "bytes")
}

func (p PathBuilder) QuotaResetIntervalMs(slotID, schemeID int) string {
	return filepath.Join(p.scheme(slotID, schemeID), "quotas", "reset_interval_ms")
}

func (p PathBuilder) WeightSz(slotID, schemeID int) string {
	return filepath.Join(p.scheme(slotID, schemeID), "quotas", "weights", "sz_permil")
}

func (p PathBuilder) WeightNrAccesses(slotID, schemeID int) string {
	return filepath.Join(p.scheme(slotID, schemeID), "quotas", "weights", "nr_accesses_permil")
}

func (p PathBuilder) WeightAge(slotID, schemeID int) string {
	return filepath.Join(p.scheme(slotID, schemeID), "quotas", "weights", "age_permil")
}

func (p PathBuilder) WatermarkMetric(slotID, schemeID int) string {
	return filepath.Join(p.scheme(slotID, schemeID), "watermarks", "metric")
}

func (p PathBuilder) WatermarkIntervalUs(slotID, schemeID int) string {
	return filepath.Join(p.scheme(slotID, schemeID), "watermarks", "interval_us")
}

func (p PathBuilder) WatermarkHigh(slotID, schemeID int) string {
	return filepath.Join(p.scheme(slotID, schemeID), "watermarks", "high")
}

func (p PathBuilder) WatermarkMid(slotID, schemeID int) string {
	return filepath.Join(p.scheme(slotID, schemeID), "watermarks", "mid")
}

func (p PathBuilder) WatermarkLow(slotID, schemeID int) string {
	return filepath.Join(p.scheme(slotID, schemeID), "watermarks", "low")
}
