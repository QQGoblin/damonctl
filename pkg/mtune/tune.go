package mtune

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/QQGoblin/damonctl/pkg/utils"
)

func (p *Controller) Run(ctx context.Context) error {

	var (
		err          error
		aggrInterval int64
	)

	if p.quotaSz, err = p.ReadInt64("quota_sz"); err != nil {
		return err
	}

	if aggrInterval, err = p.ReadInt64("aggr_interval"); err != nil {
		return err
	}
	p.aggrSec = aggrInterval / 1000000
	adjustInterval := time.Duration(p.tuneConfig.Interval*p.aggrSec) * time.Second

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
		err       error
		target    int64
		available int64
	)

	if target, err = p.target(); err != nil {
		return err
	}

	if available, err = utils.HostMemAvailable(); err != nil {
		return err
	}

	newQuotaSz := p.nextQuotaSz(target, available)
	if newQuotaSz < 0 || math.Abs(float64(newQuotaSz-p.quotaSz)) < 16*1024*1024 {
		logrus.WithFields(logrus.Fields{
			"current": utils.PrettyBytes(newQuotaSz),
			"target":  utils.PrettyBytes(target - available),
		}).Info("the changes are too minor, skip.")
		return nil
	}

	if err = p.Update("quota_sz", strconv.FormatInt(newQuotaSz, 10)); err != nil {
		return err
	}

	if err = p.CommitInputs(); err != nil {
		return err
	}

	logrus.WithFields(logrus.Fields{
		"old":     utils.PrettyBytes(p.quotaSz),
		"current": utils.PrettyBytes(newQuotaSz),
		"target":  utils.PrettyBytes(target - available),
	}).Info("update quota_sz")

	p.quotaSz = newQuotaSz
	return nil
}

func (p *Controller) target() (int64, error) {

	var (
		totalMemory int64
		err         error
	)

	if totalMemory, err = utils.HostMemTotal(); err != nil {
		return 0, err
	}
	ratioTarget := int64(float64(totalMemory) * p.tuneConfig.AvailableRatio)
	if p.tuneConfig.AvailableBytes < ratioTarget {
		return p.tuneConfig.AvailableBytes, nil
	}

	return ratioTarget, nil
}

func (p *Controller) nextQuotaSz(target, current int64) int64 {

	gap := target - current
	deadBand := int64(float64(target) * p.tuneConfig.DeadRatio)

	if math.Abs(float64(gap)) < float64(deadBand) {
		return -1
	}

	// available 内存已经达到目标值，设置 quotaSz 为 QuotaSzMin
	if gap < 0 {
		return p.tuneConfig.QuotaSzMin
	}

	// available 未达到目标值，基于当前差值调节 quotaSz
	targetReclaimSZ := int64(float64(gap) / float64(p.tuneConfig.Interval) * p.tuneConfig.Gain) // 理想值
	next := clampInt64(targetReclaimSZ, p.tuneConfig.QuotaSzMin, p.tuneConfig.QuotaSzMax)       // 计算新值
	return next
}

func clampInt64(v, lo, hi int64) int64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
