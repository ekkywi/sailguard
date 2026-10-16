//go:build windows

package winsvc

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const (
	Name        = "SailGuardAgent"
	DisplayName = "SailGuard Agent"
	Description = "SailGuard endpoint application control agent"
	TaskName    = `SailGuard\Agent`
)

func IsService() (bool, error) {
	return svc.IsWindowsService()
}

func runSC(args ...string) ([]byte, error) {
	cmd := exec.Command("sc.exe", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.CombinedOutput()
}

func runSchtasks(args ...string) ([]byte, error) {
	cmd := exec.Command("schtasks.exe", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.CombinedOutput()
}

func toServiceBinaryPath(exePath string) (string, error) {
	exePath = strings.TrimSpace(exePath)
	if exePath == "" {
		return "", fmt.Errorf("empty service binary path")
	}
	if short, err := getShortPath(exePath); err == nil && short != "" {
		return short, nil
	}
	return exePath, nil
}

func getShortPath(path string) (string, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return "", err
	}
	n, err := windows.GetShortPathName(p, nil, 0)
	if err != nil || n == 0 {
		return "", err
	}
	buf := make([]uint16, n)
	n, err = windows.GetShortPathName(p, &buf[0], n)
	if err != nil {
		return "", err
	}
	return windows.UTF16ToString(buf[:n]), nil
}

func quote(path string) string {
	if strings.HasPrefix(path, `"`) {
		return path
	}
	return `"` + path + `"`
}

// Install registers (or recreates) the Windows service via sc.exe.
func Install(exePath string) error {
	bin, err := toServiceBinaryPath(exePath)
	if err != nil {
		return err
	}

	_, _ = runSC("stop", Name)
	time.Sleep(500 * time.Millisecond)
	_, _ = runSC("delete", Name)

	// sc.exe: space after binPath= is significant; quote paths that contain spaces.
	binArg := bin
	if strings.Contains(bin, " ") {
		binArg = quote(bin)
	}

	var last []byte
	deadline := time.Now().Add(60 * time.Second)
	for {
		out, err := runSC(
			"create", Name,
			"binPath=", binArg,
			"start=", "auto",
			"DisplayName=", DisplayName,
			"obj=", "LocalSystem",
		)
		last = out
		if err == nil {
			_, _ = runSC("description", Name, Description)
			_, _ = runSC("sdset", Name,
				"D:(A;;CCLCSWRPWPDTLOCRRC;;;SY)(A;;CCDCLCSWRPWPDTLOCRSDRCWDWO;;;BA)(A;;CCLCSWLOCRRC;;;IU)(A;;CCLCSWLOCRRC;;;SU)")
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("sc create %s: %w (%s)", Name, err, strings.TrimSpace(string(last)))
		}
		_, _ = runSC("delete", Name)
		time.Sleep(700 * time.Millisecond)
	}
}

func Uninstall() error {
	_, _ = runSC("stop", Name)
	time.Sleep(400 * time.Millisecond)
	_, _ = runSC("delete", Name)
	_, _ = runSchtasks("/Delete", "/TN", TaskName, "/F")
	return nil
}

func Start() error {
	if out, err := runSC("start", Name); err != nil {
		return fmt.Errorf("%w (%s)", err, strings.TrimSpace(string(out)))
	}
	return waitRunning(45 * time.Second)
}

func Stop() error {
	_, _ = runSC("stop", Name)
	return nil
}

func waitRunning(timeout time.Duration) error {
	m, err := mgr.Connect()
	if err != nil {
		return nil
	}
	defer m.Disconnect()
	s, err := m.OpenService(Name)
	if err != nil {
		return nil
	}
	defer s.Close()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		st, err := s.Query()
		if err != nil {
			return err
		}
		if st.State == svc.Running {
			return nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("service did not reach RUNNING")
}

func QueryConfigLine() string {
	out, _ := runSC("qc", Name)
	line := strings.TrimSpace(string(out))
	if line == "" {
		return "not installed"
	}
	// Keep it short for setup console.
	for _, row := range strings.Split(line, "\n") {
		row = strings.TrimSpace(row)
		if strings.Contains(strings.ToUpper(row), "BINARY_PATH_NAME") ||
			strings.Contains(strings.ToUpper(row), "BIN_PATH") ||
			strings.HasPrefix(strings.ToUpper(row), "BINARY") {
			return row
		}
	}
	if len(line) > 180 {
		return line[:180] + "…"
	}
	return line
}

func IsRunning() bool {
	out, _ := runSC("query", Name)
	return strings.Contains(string(out), "RUNNING")
}

func EnsureTaskAutoStart(exePath string) error {
	exePath = strings.TrimSpace(exePath)
	if exePath == "" {
		return fmt.Errorf("empty exe path")
	}
	_, _ = runSchtasks("/Delete", "/TN", TaskName, "/F")
	out, err := runSchtasks(
		"/Create",
		"/TN", TaskName,
		"/SC", "ONSTART",
		"/DELAY", "0000:20",
		"/RL", "HIGHEST",
		"/RU", "SYSTEM",
		"/NP",
		"/F",
		"/TR", quote(exePath),
	)
	if err != nil {
		out2, err2 := runSchtasks(
			"/Create",
			"/TN", TaskName,
			"/SC", "ONSTART",
			"/RL", "HIGHEST",
			"/RU", "SYSTEM",
			"/NP",
			"/F",
			"/TR", quote(exePath),
		)
		if err2 != nil {
			return fmt.Errorf("schtasks: %v (%s); retry: %v (%s)",
				err, strings.TrimSpace(string(out)), err2, strings.TrimSpace(string(out2)))
		}
	}
	return nil
}

func RunTaskNow() error {
	out, err := runSchtasks("/Run", "/TN", TaskName)
	if err != nil {
		return fmt.Errorf("schtasks run: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func StartDetached(exePath string) error {
	exePath = strings.TrimSpace(exePath)
	if exePath == "" {
		return fmt.Errorf("empty exe path")
	}
	cmd := exec.Command(exePath)
	cmd.Dir = filepath.Dir(exePath)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.DETACHED_PROCESS,
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	_ = cmd.Process.Release()
	return nil
}

// EnsureRunning starts the agent via Windows service when possible; otherwise
// installs an ONSTART scheduled task and starts the process detached.
// mode is "service", "task", or "detached".
func EnsureRunning(exePath string) (mode string, note string, err error) {
	if IsRunning() {
		_, _ = runSchtasks("/Delete", "/TN", TaskName, "/F")
		return "service", "", nil
	}
	svcErr := Start()
	if svcErr == nil && IsRunning() {
		_, _ = runSchtasks("/Delete", "/TN", TaskName, "/F")
		return "service", "", nil
	}

	// Avoid a later successful SCM auto-start racing the task at boot.
	_, _ = runSC("config", Name, "start=", "disabled")

	if tErr := EnsureTaskAutoStart(exePath); tErr != nil {
		if dErr := StartDetached(exePath); dErr != nil {
			return "", "", fmt.Errorf("service: %v; task: %v; detached: %v", svcErr, tErr, dErr)
		}
		return "detached", fmt.Sprintf(
			"service start blocked (%v); agent started in background (no reboot persistence — task failed: %v)",
			svcErr, tErr,
		), nil
	}
	_ = RunTaskNow()
	_ = StartDetached(exePath)
	note = fmt.Sprintf(
		"service start blocked (%v); using scheduled task %s for boot auto-start and started agent now",
		svcErr, TaskName,
	)
	return "task", note, nil
}
