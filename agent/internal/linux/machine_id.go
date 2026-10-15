package linux

import (
	"fmt"
	"os"
	"strings"
)

func readMachineID() (string, error) {
	candidates := []string{
		"/etc/machine-id",
		"/var/lib/dbus/machine-id",
	}
	for _, path := range candidates {
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		id := strings.TrimSpace(string(b))
		if id != "" {
			return id, nil
		}
	}
	return "", fmt.Errorf("machine-id not found")
}