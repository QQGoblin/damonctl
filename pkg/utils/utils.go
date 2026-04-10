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
	if err := os.WriteFile(path, []byte(value), 0o600); err != nil {
		return fmt.Errorf("write %q to %s: %w", value, path, err)
	}
	return nil
}

func WriteInt(path string, value int) error {
	return WriteString(path, strconv.Itoa(value))
}

func ReadUint64(path string) (uint64, error) {
	s, err := ReadString(path)
	if err != nil {
		return 0, err
	}
	s = strings.TrimSpace(s)
	return strconv.ParseUint(s, 10, 64)
}
