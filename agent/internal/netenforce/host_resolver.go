package netenforce

import (
	"log"
	"time"
)

type hostResolver struct {
	mem	*hostMemory
	ttl	time.Duration
	now func() time.Time
}

var defaultHostResolver = &hostResolver{
	mem: newHostMemory(),
	ttl: 10 * time.Minute,
	now: time.Now,
}

func (r *hostResolver) ips(host string) ([]string, error) {
	now := r.now()
	fresh, err := resolveHostIPv4(host)
	if err == nil && len(fresh) > 0 {
		r.mem.Observe(host, fresh, now)
	}

	active := r.mem.Active(host, now, r.ttl)
	if len(active) == 0 {
		return nil, err
	}
	if err != nil {
		log.Printf("network enforce: resolve %q failed, using remembered IPs %v: %v", host, active, err)
	}
	return active, nil
}