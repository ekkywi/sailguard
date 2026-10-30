//go:build linux

package linux

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ekkywi/sailguard/agent/internal/host"
)

func (h *Host) Metrics() (host.ResourceSample, error) {
	sample := host.ResourceSample{
		UptimeSec:   readUptimeSec(),
		CPUPercent:  sampleCPUPercent(200 * time.Millisecond),
		DiskPercent: readRootDiskPercent(),
	}
	sample.RAMUsedMB, sample.RAMPercent = readRAMUsage()
	return sample, nil
}

func readUptimeSec() int64 {
	b, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(b))
	if len(fields) < 1 {
		return 0
	}
	sec, err := strconv.ParseFloat(fields[0], 64)
	if err != nil || sec < 0 {
		return 0
	}
	return int64(sec)
}

func readRAMUsage() (usedMB int64, percent float64) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	var totalKB, availKB, freeKB, buffersKB, cachedKB int64
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "MemTotal:"):
			totalKB = parseMeminfoKB(line)
		case strings.HasPrefix(line, "MemAvailable:"):
			availKB = parseMeminfoKB(line)
		case strings.HasPrefix(line, "MemFree:"):
			freeKB = parseMeminfoKB(line)
		case strings.HasPrefix(line, "Buffers:"):
			buffersKB = parseMeminfoKB(line)
		case strings.HasPrefix(line, "Cached:"):
			cachedKB = parseMeminfoKB(line)
		}
	}
	if totalKB <= 0 {
		return 0, 0
	}
	if availKB <= 0 {
		availKB = freeKB + buffersKB + cachedKB
	}
	usedKB := totalKB - availKB
	if usedKB < 0 {
		usedKB = 0
	}
	usedMB = usedKB / 1024
	percent = float64(usedKB) * 100 / float64(totalKB)
	return usedMB, percent
}

func parseMeminfoKB(line string) int64 {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0
	}
	v, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return 0
	}
	return v
}

func readRootDiskPercent() float64 {
	var st syscall.Statfs_t
	if err := syscall.Statfs("/", &st); err != nil {
		return 0
	}
	if st.Blocks == 0 {
		return 0
	}
	used := st.Blocks - st.Bfree
	return float64(used) * 100 / float64(st.Blocks)
}

type cpuTimes struct {
	idle  uint64
	total uint64
}

func readCPUTimes() (cpuTimes, error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return cpuTimes{}, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	if !sc.Scan() {
		return cpuTimes{}, fmt.Errorf("empty /proc/stat")
	}
	fields := strings.Fields(sc.Text())
	if len(fields) < 5 || fields[0] != "cpu" {
		return cpuTimes{}, fmt.Errorf("unexpected /proc/stat line")
	}
	var total, idle uint64
	for i := 1; i < len(fields); i++ {
		v, err := strconv.ParseUint(fields[i], 10, 64)
		if err != nil {
			continue
		}
		total += v
		if i == 4 { // idle
			idle = v
		}
	}
	return cpuTimes{idle: idle, total: total}, nil
}

func sampleCPUPercent(wait time.Duration) float64 {
	a, err := readCPUTimes()
	if err != nil {
		return 0
	}
	time.Sleep(wait)
	b, err := readCPUTimes()
	if err != nil {
		return 0
	}
	dt := b.total - a.total
	di := b.idle - a.idle
	if dt == 0 {
		return 0
	}
	busy := float64(dt-di) * 100 / float64(dt)
	if busy < 0 {
		return 0
	}
	if busy > 100 {
		return 100
	}
	return busy
}
