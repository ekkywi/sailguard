package networkpolicy

import (
	"net"
	"strings"
	"unicode"
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
	case "ip", "cidr", "host":
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
	case "host":
		host, err := normalizeHost(destValue)
		if err != nil {
			return "", "", "", "", nil, err
		}
		destValue = host
	}

	if port != nil {
		p := *port
		if p < 1 || p > 65535 {
			return "", "", "", "", nil, ErrInvalid
		}
	}

	return action, destType, destValue, label, port, nil
}

func normalizeHost(v string) (string, error) {
	v = strings.TrimSpace(strings.ToLower(v))
	v = strings.TrimSuffix(v, ".")
	if v == "" {
		return "", ErrInvalid
	}
	if strings.ContainsAny(v, " /\\:") {
		return "", ErrInvalid
	}
	if net.ParseIP(v) != nil {
		return "", ErrInvalid
	}
	if len(v) > 253 {
		return "", ErrInvalid
	}

	labels := strings.Split(v, ".")
	if len(labels) == 0 {
		return "", ErrInvalid
	}
	for _, label := range labels {
		if err := validateHostLabel(label); err != nil {
			return "", err
		}
	}
	return v, nil
}

func validateHostLabel(label string) error {
	n := len(label)
	if n < 1 || n > 63 {
		return ErrInvalid
	}
	if label[0] == '-' || label[n-1] == '-' {
		return ErrInvalid
	}
	for _, r := range label {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			continue
		}
		return ErrInvalid
	}
	return nil
}
