//go:build darwin

package darwin

import "github.com/ekkywi/sailguard/agent/internal/host"

func (h *Host) Metrics() (host.ResourceSample, error) {
	return host.ResourceSample{}, nil
}
