//go:build linux

package linux

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/ekkywi/sailguard/agent/internal/host"
)

func (h *Host) Hardware() (host.HardwareInfo, error) {
	info := host.HardwareInfo{
		CPUCores: runtime.NumCPU(),
		CPUModel: readCPUModel(),
		RAMMB:    readRAMMB(),
		DiskGB:   readRootDiskGB(),
	}
	info.HWManufacturer = readDMI("sys_vendor")
	info.HWModel = readDMI("product_name")
	return info, nil
}

func readCPUModel() string {
	f, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return ""
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "model name") {
			_, rest, ok := strings.Cut(line, ":")
			if ok {
				return strings.TrimSpace(rest)
			}
		}
	}
	return ""
}

func readRAMMB() int64 {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "MemTotal:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0
		}
		kb, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil || kb <= 0 {
			return 0
		}
		return kb / 1024
	}
	return 0
}

func readRootDiskGB() int64 {
	var st syscall.Statfs_t
	if err := syscall.Statfs("/", &st); err != nil {
		return 0
	}
	total := int64(st.Blocks) * int64(st.Bsize)
	if total <= 0 {
		return 0
	}
	return total / (1024 * 1024 * 1024)
}

func readDMI(name string) string {
	b, err := os.ReadFile(fmt.Sprintf("/sys/class/dmi/id/%s", name))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}
