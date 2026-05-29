//go:build ignore

// hotmem.go - 申请一块内存并反复读写, 使其成为内存热点
//
// 构建 (在任意平台均可):
//   Linux:        go build -o hotmem        scripts/hotmem.go
//   Windows .exe: GOOS=windows GOARCH=amd64 go build -o hotmem.exe scripts/hotmem.go
//
// 用法: ./hotmem <size>
//   size 支持单位 K/M/G (大小写均可), 例如 64M, 1G, 512K, 1048576

package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

func parseSize(s string) (uint64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty size")
	}
	upper := strings.ToUpper(s)
	upper = strings.TrimSuffix(upper, "B")
	mult := uint64(1)
	switch upper[len(upper)-1] {
	case 'K':
		mult = 1024
		upper = upper[:len(upper)-1]
	case 'M':
		mult = 1024 * 1024
		upper = upper[:len(upper)-1]
	case 'G':
		mult = 1024 * 1024 * 1024
		upper = upper[:len(upper)-1]
	}
	v, err := strconv.ParseUint(upper, 10, 64)
	if err != nil {
		return 0, err
	}
	if v == 0 {
		return 0, fmt.Errorf("size must be > 0")
	}
	if v > (^uint64(0))/mult {
		return 0, fmt.Errorf("size overflow")
	}
	return v * mult, nil
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr,
			"用法: %s <size>\n  size 支持单位 K/M/G, 例如: 64M, 1G, 512K, 1048576\n",
			os.Args[0])
		os.Exit(1)
	}

	size, err := parseSize(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "无效的 size 参数 %q: %v\n", os.Args[1], err)
		os.Exit(1)
	}

	// 对齐到 8 字节, 方便按 uint64 访问
	bytes := size &^ 7
	if bytes < 8 {
		fmt.Fprintln(os.Stderr, "size 太小, 至少 8 字节")
		os.Exit(1)
	}

	// 分配并触达所有页, 确保物理页真正落实
	buf := make([]byte, bytes)
	for i := range buf {
		buf[i] = 0x5A
	}

	// 用 unsafe 把 []byte 视为 []uint64, 避免循环里反复解码
	n := bytes / 8
	words := unsafe.Slice((*uint64)(unsafe.Pointer(&buf[0])), n)

	var stop atomic.Bool
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		stop.Store(true)
	}()

	fmt.Printf("[hotmem] pid=%d size=%d bytes (%.2f MiB), elements=%d\n", os.Getpid(), bytes, float64(bytes)/1048576.0, n)
	fmt.Println("[hotmem] 开始反复读写, Ctrl-C 退出...")

	start := time.Now()
	var iter uint64
	var sink uint64

	for !stop.Load() {
		// 顺序读写一整轮: 累加并写回, 形成读+写的内存流量
		for i := uint64(0); i < n; i++ {
			v := words[i] + iter + i
			words[i] = v
			sink ^= v
		}
		iter++

		if iter&0xF == 0 {
			sec := time.Since(start).Seconds()
			gib := float64(bytes) * float64(iter) * 2.0 / (1024 * 1024 * 1024)
			tput := 0.0
			if sec > 0 {
				tput = gib / sec
			}
			fmt.Printf("[hotmem] iter=%d elapsed=%.2fs throughput=%.2f GiB/s\n",
				iter, sec, tput)
		}
	}

	sec := time.Since(start).Seconds()
	gib := float64(bytes) * float64(iter) * 2.0 / (1024 * 1024 * 1024)
	avg := 0.0
	if sec > 0 {
		avg = gib / sec
	}
	fmt.Printf("\n[hotmem] 退出: iter=%d elapsed=%.2fs avg=%.2f GiB/s sink=0x%x\n",
		iter, sec, avg, sink)
}
