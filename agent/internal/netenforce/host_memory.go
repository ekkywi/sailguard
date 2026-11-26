package netenforce

import (
	"sort"
	"sync"
	"time"
)

type hostMemory struct {
	mu   sync.Mutex
	seen map[string]map[string]time.Time
}

func newHostMemory() *hostMemory {
	return &hostMemory{
		seen: make(map[string]map[string]time.Time),
	}
}

func (m *hostMemory) Observe(host string, ips []string, now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	byIP, ok := m.seen[host]
	if !ok {
		byIP = make(map[string]time.Time)
		m.seen[host] = byIP
	}
	for _, ip := range ips {
		byIP[ip] = now
	}
}

func (m *hostMemory) Active(host string, now time.Time, ttl time.Duration) []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	byIP, ok := m.seen[host]
	if !ok {
		return nil
	}

	out := make([]string, 0, len(byIP))
	for ip, last := range byIP {
		if now.Sub(last) <= ttl {
			out = append(out, ip)
		} else {
			delete(byIP, ip)
		}
	}
	if len(byIP) == 0 {
		delete(m.seen, host)
	}

	sort.Strings(out)
	return out
}
