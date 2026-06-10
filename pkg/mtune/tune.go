package mtune

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/prometheus/procfs"
	"github.com/sirupsen/logrus"

	"github.com/QQGoblin/damonctl/pkg/utils"
)

func (p *Controller) Run(ctx context.Context) error {
	var (
		err          error
		aggrInterval uint64
		memPSI       procfs.PSIStats
	)

	if p.quotaSz, err = p.ReadUInt64("quota_sz"); err != nil {
		return err
	}

	if memPSI, err = utils.HostMemoryPSIStats(); err != nil {
		return err
	}

	p.lastSomePsiUs = memPSI.Some.Total

	if aggrInterval, err = p.ReadUInt64("aggr_interval"); err != nil {
		return err
	}
	p.aggrSec = aggrInterval / 1000000
	adjustInterval := time.Duration(uint64(p.tuneConfig.Interval)*p.aggrSec) * time.Second

	if adjustInterval < time.Second {
		return fmt.Errorf("interval too small (%d < 1s)", adjustInterval)
	}

	ticker := time.NewTicker(adjustInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err = p.tuneOnce(); err != nil {
				logrus.Errorf("tune cycle error: %v", err)
			}
		}
	}
}

func (p *Controller) tuneOnce() error {
	var (
		err           error
		target        uint64
		available     uint64
		overloadRatio float64
		somePsiDelta  uint64
		memInfo       procfs.Meminfo
	)

	if memInfo, err = utils.HostMemInfo(); err != nil {
		return err
	}
	available = *memInfo.MemAvailableBytes
	overloadRatio = hostOverloadRatio(memInfo)

	if target, err = p.targetMemory(); err != nil {
		return err
	}
	if somePsiDelta, err = p.somePsiDelta(); err != nil {
		return err
	}

	availableQuotaSz := p.nextAvailableQuotaSz(target, available, overloadRatio)
	psiQuotaCap := p.nextPsiQuotaCap(somePsiDelta)
	newQuotaSz := p.nextQuotaSz(availableQuotaSz, psiQuotaCap)

	logFields := logrus.Fields{
		"Memory":           utils.PrettyBytes(available),
		"PSIState":         somePsiDelta,
		"QuotaSZ":          utils.PrettyBytes(newQuotaSz),
		"AvailableQuotaSz": utils.PrettyBytes(availableQuotaSz),
		"PSIQuotaCap":      utils.PrettyBytes(psiQuotaCap),
		"Overload":         fmt.Sprintf("%.4f", overloadRatio),
	}

	if newQuotaSz <= 0 || utils.DiffUInt64(newQuotaSz, p.quotaSz) < 16*1024*1024 {
		logrus.WithFields(logFields).Info("skip quota_sz update")
		return nil
	}

	if err = p.Update("quota_sz", strconv.FormatUint(newQuotaSz, 10)); err != nil {
		return err
	}

	if err = p.CommitInputs(); err != nil {
		return err
	}

	logrus.WithFields(logFields).Info("update quota_sz success")

	p.quotaSz = newQuotaSz
	return nil
}

func (p *Controller) targetMemory() (uint64, error) {
	var (
		err     error
		memInfo procfs.Meminfo
	)

	if memInfo, err = utils.HostMemInfo(); err != nil {
		return 0, err
	}

	ratioTarget := uint64(float64(*memInfo.MemTotalBytes) * p.tuneConfig.AvailableRatio)
	if p.tuneConfig.AvailableBytes < ratioTarget {
		return p.tuneConfig.AvailableBytes, nil
	}

	return ratioTarget, nil
}

func (p *Controller) somePsiDelta() (uint64, error) {
	var (
		memPSI procfs.PSIStats
		err    error
	)

	if memPSI, err = utils.HostMemoryPSIStats(); err != nil {
		return 0, err
	}

	delta := memPSI.Some.Total - p.lastSomePsiUs
	p.lastSomePsiUs = memPSI.Some.Total
	return delta, nil
}

func (p *Controller) nextAvailableQuotaSz(target, current uint64, overloadRatio float64) uint64 {
	if p.minQuotaByOverloadRatio(overloadRatio) {
		return p.tuneConfig.QuotaSzMin
	}

	gap := utils.DiffUInt64(target, current)
	deadBand := uint64(float64(target) * p.tuneConfig.DeadRatio)
	// available 已经超出目标值，减少回收配额
	if current > target && gap >= deadBand {
		return p.tuneConfig.QuotaSzMin
	}

	if gap < deadBand {
		return p.quotaSz
	}

	// available 未达到目标值，基于当前差值调节 quotaSz
	targetReclaimSZ := uint64(float64(gap) / float64(p.tuneConfig.Interval) * p.tuneConfig.Gain) // 理想值
	next := utils.ClampUInt64(targetReclaimSZ, p.tuneConfig.QuotaSzMin, p.tuneConfig.QuotaSzMax) // 计算新值
	return next
}

func (p *Controller) minQuotaByOverloadRatio(overloadRatio float64) bool {
	threshold := p.tuneConfig.OverloadThreshold
	return threshold > 0 && overloadRatio >= threshold
}

func hostOverloadRatio(memInfo procfs.Meminfo) float64 {
	memCap := *memInfo.MemTotalBytes - *memInfo.HugetlbBytes
	memUsed := *memInfo.MemTotalBytes - *memInfo.MemAvailableBytes - *memInfo.HugetlbBytes
	swapUsed := *memInfo.SwapTotalBytes - *memInfo.SwapFreeBytes - *memInfo.SwapCachedBytes
	return float64(swapUsed+memUsed) / float64(memCap)
}

func (p *Controller) nextPsiQuotaCap(somePsiDelta uint64) uint64 {
	target := p.tuneConfig.SomePsiUs
	deadBand := uint64(float64(target) * p.tuneConfig.PsiDeadRatio)

	// PSI 压力变化较小
	if utils.DiffUInt64(somePsiDelta, target) < deadBand {
		return utils.ClampUInt64(p.quotaSz, p.tuneConfig.QuotaSzMin, p.tuneConfig.QuotaSzMax)
	}

	// PSI 压力超过目标 2 倍
	if somePsiDelta >= target*2 {
		return p.tuneConfig.QuotaSzMin
	}

	// 对 PSI 指标进行线行折算
	ratio := float64(somePsiDelta) / float64(target)
	next := uint64(float64(p.quotaSz) * (2 - ratio))
	return utils.ClampUInt64(next, p.tuneConfig.QuotaSzMin, p.tuneConfig.QuotaSzMax)
}

func (p *Controller) nextQuotaSz(availableQuotaSz, psiQuotaCap uint64) uint64 {
	if availableQuotaSz <= 0 {
		availableQuotaSz = p.quotaSz
	}
	requireQuotaSz := utils.MinUInt64(psiQuotaCap, availableQuotaSz)
	return utils.ClampUInt64(requireQuotaSz, p.tuneConfig.QuotaSzMin, p.tuneConfig.QuotaSzMax)
}
