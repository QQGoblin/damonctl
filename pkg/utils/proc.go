package utils

import "github.com/prometheus/procfs"

func DefaultProcFS() (procfs.FS, error) {
	return procfs.NewDefaultFS()
}

func HostMemInfo() (procfs.Meminfo, error) {
	var (
		fs  procfs.FS
		err error
	)

	if fs, err = DefaultProcFS(); err != nil {
		return procfs.Meminfo{}, err
	}

	return fs.Meminfo()
}

func HostMemoryPSIStats() (procfs.PSIStats, error) {
	var (
		fs  procfs.FS
		err error
	)

	if fs, err = DefaultProcFS(); err != nil {
		return procfs.PSIStats{}, err
	}
	return fs.PSIStatsForResource("memory")
}
