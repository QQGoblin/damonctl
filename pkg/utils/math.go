package utils

import "fmt"

func ClampUInt64(v, lo, hi uint64) uint64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func DiffUInt64(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}

func MinUInt64(u, v uint64) uint64 {
	if u < v {
		return u
	}
	return v
}

func PrettyBytes(b uint64) string {
	const unit = 1024
	if b < 0 {
		return "-" + PrettyBytes(-b)
	}
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	value := float64(b)
	suffixes := []string{"KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	i := 0
	for value >= unit && i < len(suffixes) {
		value /= unit
		i++
	}
	return fmt.Sprintf("%.2f%s", value, suffixes[i-1])
}
