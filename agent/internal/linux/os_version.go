package linux

import (
	"bufio"
	"os"
	"strings"
)

// readOSVersion returns a human-readable Linux OS string from os-release files.
func readOSVersion() string {
	for _, path := range []string{"/etc/os-release", "/usr/lib/os-release"} {
		if v := parseOSRelease(path); v != "" {
			return v
		}
	}
	if b, err := os.ReadFile("/proc/version"); err == nil {
		line := strings.TrimSpace(string(b))
		if line != "" {
			// Keep it short: first clause of /proc/version.
			if i := strings.Index(line, " ("); i > 0 {
				return line[:i]
			}
			return line
		}
	}
	return "Linux"
}

func parseOSRelease(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	vals := make(map[string]string)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		vals[key] = unquoteOSRelease(val)
	}

	if pretty := strings.TrimSpace(vals["PRETTY_NAME"]); pretty != "" {
		return pretty
	}
	name := strings.TrimSpace(vals["NAME"])
	version := strings.TrimSpace(vals["VERSION"])
	if version == "" {
		version = strings.TrimSpace(vals["VERSION_ID"])
	}
	switch {
	case name != "" && version != "":
		return name + " " + version
	case name != "":
		return name
	default:
		return ""
	}
}

func unquoteOSRelease(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 2 {
		if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
			return v[1 : len(v)-1]
		}
	}
	return v
}
