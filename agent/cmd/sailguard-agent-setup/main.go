// SailGuard Windows agent setup: installs binary + config to standard paths.
// Build for Windows (prefer Go 1.20 so the setup itself runs on Windows 7 SP1+):
//
//	GOOS=windows GOARCH=amd64 go build -o bin/sailguard-agent-setup.exe ./cmd/sailguard-agent-setup
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	silent := flag.Bool("silent", false, "no prompts; do not pause at end")
	runAfter := flag.Bool("run", false, "legacy flag; service always starts after install")
	flag.Parse()

	if err := runInstall(*silent, *runAfter); err != nil {
		fmt.Fprintf(os.Stderr, "SailGuard setup failed: %v\n", err)
		if !*silent {
			fmt.Fprintln(os.Stderr, "Press Enter to close…")
			_, _ = fmt.Scanln()
		}
		os.Exit(1)
	}
}
