//go:build windows

package platform

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/ekkywi/sailguard/agent/internal/winsvc"
	"golang.org/x/sys/windows/svc"
)

// RunAgent is the process entry for Windows: register with SCM immediately when
// launched as a service (before enroll / network), otherwise run in the console.
func RunAgent(run func(ctx context.Context) error) error {
	isSvc, err := winsvc.IsService()
	if err != nil {
		log.Printf("windows service detect: %v — console mode", err)
		return runConsole(run)
	}
	if !isSvc {
		return runConsole(run)
	}

	if closer, err := redirectServiceLog(); err != nil {
		// Continue anyway — Event Log may still show SCM errors.
		fmt.Fprintf(os.Stderr, "service log file: %v\n", err)
	} else if closer != nil {
		defer closer()
	}

	log.Printf("sailguard-agent starting as Windows service %s", winsvc.Name)
	if err := svc.Run(winsvc.Name, &agentSVC{run: run}); err != nil {
		return fmt.Errorf("service run: %w", err)
	}
	return nil
}

func runConsole(run func(ctx context.Context) error) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	return run(ctx)
}

type agentSVC struct {
	run func(context.Context) error
}

func (m *agentSVC) Execute(
	args []string,
	r <-chan svc.ChangeRequest,
	changes chan<- svc.Status,
) (ssec bool, errno uint32) {
	_ = args
	const accepts = svc.AcceptStop | svc.AcceptShutdown

	changes <- svc.Status{State: svc.StartPending}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- m.run(ctx)
	}()

	changes <- svc.Status{State: svc.Running, Accepts: accepts}
	log.Printf("service %s status=running", winsvc.Name)

	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				log.Printf("service %s stop requested", winsvc.Name)
				changes <- svc.Status{State: svc.StopPending}
				cancel()
				select {
				case err := <-errCh:
					if err != nil {
						log.Printf("service worker exit: %v", err)
					}
				case <-time.After(35 * time.Second):
					log.Printf("service worker stop timeout")
				}
				return false, 0
			default:
			}
		case err := <-errCh:
			if err != nil && ctx.Err() == nil {
				log.Printf("service worker exited: %v", err)
				return true, 1
			}
			return false, 0
		}
	}
}

func redirectServiceLog() (func(), error) {
	dir := serviceDataDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "agent-service.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	return func() { _ = f.Close() }, nil
}

func serviceDataDir() string {
	if v := os.Getenv("SG_DATA_DIR"); v != "" {
		return v
	}
	base := os.Getenv("ProgramData")
	if base == "" {
		base = `C:\ProgramData`
	}
	return filepath.Join(base, "SailGuard", "agent")
}
