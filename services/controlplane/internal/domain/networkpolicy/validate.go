package networkpolicy

import (
	"net"
	"strings"
)

func normalizeRuleFields(action, destType, destValue, label string, port *int) (string, string, string, string, *int, error) {
	action = strings.TrimSpace(strings.ToLower(action))
	destType = strings.TrimSpace(strings.ToLower(destType))
	destValue = strings.TrimSpace(destValue)
	label = strings.TrimSpace(label)

	switch action {
	case "allow", "deny":
	default:
		return "", "", "", "", nil, ErrInvalid
	}
	switch destType {
	case "ip", "cidr":
	default:
		return "", "", "", "", nil, ErrInvalid
	}
	if destValue == "" {
		return "", "", "", "", nil, ErrInvalid
	}

	switch destType {
	case "ip":
		if net.ParseIP(destValue) == nil {
			return "", "", "", "", nil, ErrInvalid
		}
	case "cidr":
		if _, _, err := net.ParseCIDR(destValue); err != nil {
			return "", "", "", "", nil, ErrInvalid
		}
	}

	if port != nil {
		p := *port
		if p < 1 || p > 65535 {
			return "", "", "", "", nil, ErrInvalid
		}
	}

	return action, destType, destValue, label, port, nil
}
