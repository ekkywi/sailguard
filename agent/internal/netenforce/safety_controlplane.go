package netenforce

import (
	"net"
	"net/url"
	"strconv"
	"strings"
)

const safetyControlPlaneID = "safety-controlplane"

// EnsureControlPlaneSafety prepends a permit for the agent control-plane URL.
// Inject at apply time only - do not rewrite the synced policy hash on disk.

func EnsureControlPlaneSafety(spec Spec, serverURL string) Spec {
	r, ok := controlPlaneSafetyRule(serverURL)
	if !ok {
		return spec
	}
	out := spec
	out.SafetyAllow = append([]Rule{}, spec.SafetyAllow...)
	for _, existing := range out.SafetyAllow {
		if existing.ID == safetyControlPlaneID {
			return out
		}
	}
	out.SafetyAllow = append([]Rule{r}, out.SafetyAllow...)
	return out
}

func controlPlaneSafetyRule(serverURL string) (Rule, bool) {
	raw := strings.TrimSpace(serverURL)
	if raw == "" {
		return Rule{}, false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return Rule{}, false
	}

	host := u.Hostname()
	portStr := u.Port()
	if portStr == "" {
		if strings.EqualFold(u.Scheme, "https") {
			portStr = "443"
		} else {
			portStr = "80"
		}
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		return Rule{}, false
	}
	p := port

	r := Rule{
		ID: safetyControlPlaneID,
		Action: "allow",
		Port: &p,
		Label: "controlplane",
		Source: "safety",
	}
	if ip := net.ParseIP(host); ip != nil && ip.To4() != nil {
		r.DestType = "ip"
		r.DestValue = ip.To4().String()
	} else {
		r.DestType = "host"
		r.DestValue = host
	}
	return r, true
}