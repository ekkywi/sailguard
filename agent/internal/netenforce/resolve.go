package netenforce

import (
	"fmt"
	"log"
	"net"
)

func expandHosts(spec Spec) Spec {
	out := spec
	out.SafetyAllow = expandRuleList(spec.SafetyAllow, "safety")
	out.Rules = expandRuleList(spec.Rules, "rules")
	return out
}

func expandRuleList(in []Rule, kind string) []Rule {
	if len(in) == 0 {
		return []Rule{}
	}
	out := make([]Rule, 0, len(in))
	for _, r := range in {
		if r.DestType != "host" {
			out = append(out, r)
			continue
		}
		ips, err := resolveHostIPv4(r.DestValue)
		if err != nil {
			log.Printf("network enforce: resolve %s %q failed: %v — skip", kind, r.DestValue, err)
			continue
		}
		if len(ips) == 0 {
			log.Printf("network enforce: resolve %s %q: no IPv4 — skip", kind, r.DestValue)
			continue
		}
		for _, ip := range ips {
			rr := r
			rr.ID = fmt.Sprintf("%s/%s", r.ID, ip)
			rr.DestType = "ip"
			rr.DestValue = ip
			if rr.Label == "" {
				rr.Label = r.DestValue
			} else {
				rr.Label = fmt.Sprintf("%s (%s)", r.Label, ip)
			}
			out = append(out, rr)
		}
		log.Printf(
			"network enforce: host %q → %d IPv4 (%v)",
			r.DestValue, len(ips), ips,
		)
	}
	return out
}

func resolveHostIPv4(host string) ([]string, error) {
	addrs, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(addrs))
	seen := map[string]struct{}{}
	for _, a := range addrs {
		v4 := a.To4()
		if v4 == nil {
			continue
		}
		s := v4.String()
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out, nil
}