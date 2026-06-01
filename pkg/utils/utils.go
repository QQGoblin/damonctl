package utils

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func ReadString(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return strings.TrimSpace(string(data)), nil
}

func ReadInt(path string) (int, error) {
	s, err := ReadString(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(s)
}

func WriteString(path, value string) error {
	return os.WriteFile(path, []byte(value), 0o600)
}

func WriteInt(path string, value int) error {
	return WriteString(path, strconv.Itoa(value))
}

func WriteInt64(path string, value int64) error {
	return WriteString(path, strconv.FormatInt(value, 10))
}

func WriteUint64(path string, value uint64) error {
	return WriteString(path, strconv.FormatUint(value, 10))
}

func ReadUint64(path string) (uint64, error) {
	s, err := ReadString(path)
	if err != nil {
		return 0, err
	}
	s = strings.TrimSpace(s)
	return strconv.ParseUint(s, 10, 64)
}

func HostMemTotal() (int64, error) {
	return meminfoBytes("MemTotal")
}

func HostMemAvailable() (int64, error) {
	return meminfoBytes("MemAvailable")
}

func meminfoBytes(key string) (int64, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, fmt.Errorf("read /proc/meminfo: %w", err)
	}
	prefix := key + ":"
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0, fmt.Errorf("unexpected %s line: %s", key, line)
		}
		kb, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parse %s value %q: %w", key, fields[1], err)
		}
		return kb * 1024, nil
	}
	return 0, fmt.Errorf("%s not found in /proc/meminfo", key)
}

func PrettyBytes(b int64) string {
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
