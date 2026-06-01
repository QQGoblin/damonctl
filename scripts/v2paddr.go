//go:build ignore

// paddr.go - 给定 PID 和一段虚拟地址范围，找出映射到的最高/最低物理地址
//
// 编译: go build -o bin/paddr scripts/paddr.go
// 用法: ./paddr --pid <PID> --start <VADDR_START> --end <VADDR_END>
//   地址支持十六进制(0x前缀)或十进制
//
// 示例:
//   sudo ./paddr --pid 1234 --start 0x7f8a00000000 --end 0x7f8a00001000

package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	pageShift  = 12
	pageSize   = 1 << pageShift
	pageMask   = pageSize - 1
	pfnMask    = (1 << 55) - 1
	presentBit = 1 << 63
)

func parseAddr(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		return strconv.ParseUint(s[2:], 16, 64)
	}
	return strconv.ParseUint(s, 10, 64)
}

func main() {
	var (
		pid   = flag.Int("pid", 0, "target process PID (required)")
		start = flag.String("start", "", "virtual start address, hex or decimal (required)")
		end   = flag.String("end", "", "virtual end address, hex or decimal (required)")
	)
	flag.Parse()

	if *pid <= 0 || *start == "" || *end == "" {
		flag.Usage()
		os.Exit(1)
	}

	vstart, err := parseAddr(*start)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid start address %q: %v\n", *start, err)
		os.Exit(1)
	}
	vend, err := parseAddr(*end)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid end address %q: %v\n", *end, err)
		os.Exit(1)
	}
	if vend <= vstart {
		fmt.Fprintf(os.Stderr, "invalid range: end (0x%x) <= start (0x%x)\n", vend, vstart)
		os.Exit(1)
	}

	pagemapPath := fmt.Sprintf("/proc/%d/pagemap", *pid)
	f, err := os.Open(pagemapPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open %s: %v\n", pagemapPath, err)
		os.Exit(1)
	}
	defer f.Close()

	var minPaddr, maxPaddr uint64
	var hasPage bool
	var pages, present uint64

	startPage := vstart >> pageShift
	endPage := (vend + pageMask) >> pageShift

	buf := make([]byte, 8)
	for page := startPage; page < endPage; page++ {
		offset := int64(page) * 8
		if _, err := f.ReadAt(buf, offset); err != nil {
			fmt.Fprintf(os.Stderr, "read pagemap at offset %d: %v\n", offset, err)
			os.Exit(1)
		}

		entry := binary.LittleEndian.Uint64(buf)
		pages++

		if entry&presentBit == 0 {
			continue
		}
		present++

		pfn := entry & pfnMask
		vaddrBase := page << pageShift
		paddrBase := pfn << pageShift

		pageStart := vaddrBase
		pageEnd := vaddrBase + pageSize
		if pageStart < vstart {
			pageStart = vstart
		}
		if pageEnd > vend {
			pageEnd = vend
		}

		pStart := paddrBase + (pageStart - vaddrBase)
		pEnd := paddrBase + (pageEnd - vaddrBase) - 1

		if !hasPage {
			minPaddr = pStart
			maxPaddr = pEnd
			hasPage = true
		} else {
			if pStart < minPaddr {
				minPaddr = pStart
			}
			if pEnd > maxPaddr {
				maxPaddr = pEnd
			}
		}
	}

	if !hasPage {
		fmt.Fprintf(os.Stderr, "no present pages found in range 0x%x - 0x%x\n", vstart, vend)
		os.Exit(1)
	}

	fmt.Printf("PID:        %d\n", *pid)
	fmt.Printf("VAddrRange: 0x%016x - 0x%016x\n", vstart, vend)
	fmt.Printf("MinPaddr:   0x%016x (%d)\n", minPaddr, minPaddr)
	fmt.Printf("MaxPaddr:   0x%016x (%d)\n", maxPaddr, maxPaddr)
	fmt.Printf("Scanned:    %d pages, present: %d\n", pages, present)
}
