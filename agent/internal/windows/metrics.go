//go:build windows

package windows

import (
	"time"
	"unsafe"

	"github.com/ekkywi/sailguard/agent/internal/host"
	"golang.org/x/sys/windows"
)

var (
	procGetTickCount64 = modkernel32.NewProc("GetTickCount64")
	procGetSystemTimes = modkernel32.NewProc("GetSystemTimes")
)

func (h *Host) Metrics() (host.ResourceSample, error) {
	sample := host.ResourceSample{
		UptimeSec:   readUptimeSec(),
		CPUPercent:  sampleCPUPercent(200 * time.Millisecond),
		DiskPercent: readSystemDiskPercent(),
	}
	sample.RAMUsedMB, sample.RAMPercent = readRAMUsage()
	return sample, nil
}

func readUptimeSec() int64 {
	r1, _, _ := procGetTickCount64.Call()
	ms := uint64(r1)
	return int64(ms / 1000)
}

func readRAMUsage() (usedMB int64, percent float64) {
	var mem memoryStatusEx
	mem.Length = uint32(unsafe.Sizeof(mem))
	r1, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&mem)))
	if r1 == 0 || mem.TotalPhys == 0 {
		return 0, 0
	}
	used := mem.TotalPhys - mem.AvailPhys
	usedMB = int64(used / (1024 * 1024))
	percent = float64(used) * 100 / float64(mem.TotalPhys)
	return usedMB, percent
}

func readSystemDiskPercent() float64 {
	path, err := windows.UTF16PtrFromString(`C:\`)
	if err != nil {
		return 0
	}
	var freeBytesAvailable, totalBytes, totalFreeBytes uint64
	err = windows.GetDiskFreeSpaceEx(path, &freeBytesAvailable, &totalBytes, &totalFreeBytes)
	if err != nil || totalBytes == 0 {
		return 0
	}
	used := totalBytes - totalFreeBytes
	return float64(used) * 100 / float64(totalBytes)
}

type filetime struct {
	LowDateTime  uint32
	HighDateTime uint32
}

func filetimeToUint64(ft filetime) uint64 {
	return uint64(ft.HighDateTime)<<32 | uint64(ft.LowDateTime)
}

func readSystemTimes() (idle, kernel, user uint64, ok bool) {
	var i, k, u filetime
	r1, _, _ := procGetSystemTimes.Call(
		uintptr(unsafe.Pointer(&i)),
		uintptr(unsafe.Pointer(&k)),
		uintptr(unsafe.Pointer(&u)),
	)
	if r1 == 0 {
		return 0, 0, 0, false
	}
	return filetimeToUint64(i), filetimeToUint64(k), filetimeToUint64(u), true
}

func sampleCPUPercent(wait time.Duration) float64 {
	idle1, kernel1, user1, ok := readSystemTimes()
	if !ok {
		return 0
	}
	time.Sleep(wait)
	idle2, kernel2, user2, ok := readSystemTimes()
	if !ok {
		return 0
	}
	// Kernel includes idle time on Windows.
	idle := idle2 - idle1
	kernel := kernel2 - kernel1
	user := user2 - user1
	total := kernel + user
	if total == 0 {
		return 0
	}
	busy := float64(total-idle) * 100 / float64(total)
	if busy < 0 {
		return 0
	}
	if busy > 100 {
		return 100
	}
	return busy
}
