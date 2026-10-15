//go:build !windows

package main

import "fmt"

func runInstall(silent, runAfter bool) error {
	_ = silent
	_ = runAfter
	return fmt.Errorf("sailguard-agent-setup only runs on Windows")
}
