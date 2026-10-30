//go:build windows

package windows

import (
	"runtime"
	"strings"
	"unsafe"

	"github.com/ekkywi/sailguard/agent/internal/host"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	modkernel32              = windows.NewLazySystemDLL("kernel32.dll")
	procGlobalMemoryStatusEx = modkernel32.NewProc("GlobalMemoryStatusEx")
)

type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

func (h *Host) Hardware() (host.HardwareInfo, error) {
	info := host.HardwareInfo{
		CPUCores: runtime.NumCPU(),
		CPUModel: readCPUModel(),
		RAMMB:    readRAMMB(),
		DiskGB:   readSystemDiskGB(),
	}
	info.HWManufacturer, info.HWModel = readSystemProduct()
	return info, nil
}

func readCPUModel() string {
	k, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`HARDWARE\DESCRIPTION\System\CentralProcessor\0`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return ""
	}
	defer k.Close()
	name, _, err := k.GetStringValue("ProcessorNameString")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(name)
}

func readRAMMB() int64 {
	var mem memoryStatusEx
	mem.Length = uint32(unsafe.Sizeof(mem))
	r1, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&mem)))
	if r1 == 0 || mem.TotalPhys == 0 {
		return 0
	}
	return int64(mem.TotalPhys / (1024 * 1024))
}

func readSystemDiskGB() int64 {
	path, err := windows.UTF16PtrFromString(`C:\`)
	if err != nil {
		return 0
	}
	var freeBytesAvailable, totalBytes, totalFreeBytes uint64
	err = windows.GetDiskFreeSpaceEx(path, &freeBytesAvailable, &totalBytes, &totalFreeBytes)
	if err != nil || totalBytes == 0 {
		return 0
	}
	return int64(totalBytes / (1024 * 1024 * 1024))
}

func readSystemProduct() (manufacturer, model string) {
	k, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`HARDWARE\DESCRIPTION\System\BIOS`,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return "", ""
	}
	defer k.Close()
	if v, _, err := k.GetStringValue("SystemManufacturer"); err == nil {
		manufacturer = strings.TrimSpace(v)
	}
	if v, _, err := k.GetStringValue("SystemProductName"); err == nil {
		model = strings.TrimSpace(v)
	}
	return manufacturer, model
}
