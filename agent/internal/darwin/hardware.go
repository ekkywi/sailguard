//go:build darwin

package darwin

import (
	"runtime"

	"github.com/ekkywi/sailguard/agent/internal/host"
)

func (h *Host) Hardware() (host.HardwareInfo, error) {
	return host.HardwareInfo{
		CPUCores: runtime.NumCPU(),
	}, nil
}
