//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	productName   = "SailGuard Agent"
	publisher     = "SailGuard"
	installSubdir = `SailGuard\Agent`
	dataSubdir    = `SailGuard\agent`
	uninstallKey  = `Software\Microsoft\Windows\CurrentVersion\Uninstall\SailGuardAgent`
)

type fileConfig struct {
	ServerURL       string `json:"server_url"`
	EnrollmentToken string `json:"enrollment_token"`
	DataDir         string `json:"data_dir,omitempty"`
	AgentVersion    string `json:"agent_version,omitempty"`
}

func runInstall(silent, runAfter bool) error {
	srcDir, err := sourceDir()
	if err != nil {
		return err
	}

	elev, err := isElevated()
	if err != nil {
		return fmt.Errorf("check elevation: %w", err)
	}
	if !elev {
		if err := relaunchElevated(srcDir); err != nil {
			return fmt.Errorf("request administrator approval: %w", err)
		}
		return nil
	}

	agentName, err := findAgentBinary(srcDir)
	if err != nil {
		return err
	}
	cfgPath := filepath.Join(srcDir, "config.json")
	cfg, err := loadConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("read config.json (place it next to the setup exe): %w", err)
	}
	if strings.TrimSpace(cfg.ServerURL) == "" {
		return fmt.Errorf("config.json: server_url is required")
	}
	if strings.TrimSpace(cfg.EnrollmentToken) == "" {
		return fmt.Errorf("config.json: enrollment_token is required (first install)")
	}

	programFiles := os.Getenv("ProgramFiles")
	if programFiles == "" {
		programFiles = `C:\Program Files`
	}
	programData := os.Getenv("ProgramData")
	if programData == "" {
		programData = `C:\ProgramData`
	}

	installDir := filepath.Join(programFiles, installSubdir)
	dataDir := filepath.Join(programData, dataSubdir)
	cfg.DataDir = dataDir

	if !silent {
		fmt.Println(productName + " Setup")
		fmt.Println("======================")
		fmt.Println("Source:     ", srcDir)
		fmt.Println("Install to: ", installDir)
		fmt.Println("Data dir:   ", dataDir)
		fmt.Println("Server:     ", cfg.ServerURL)
		fmt.Println()
	}

	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return fmt.Errorf("create install dir: %w", err)
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	dstAgent := filepath.Join(installDir, agentName)
	if err := copyFile(filepath.Join(srcDir, agentName), dstAgent); err != nil {
		return fmt.Errorf("copy agent: %w", err)
	}

	cfgBytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	cfgBytes = append(cfgBytes, '\n')
	// Config lives next to the binary (agent discovers it) and under ProgramData.
	if err := os.WriteFile(filepath.Join(installDir, "config.json"), cfgBytes, 0o644); err != nil {
		return fmt.Errorf("write install config: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "config.json"), cfgBytes, 0o644); err != nil {
		return fmt.Errorf("write data config: %w", err)
	}

	if err := writeStartScripts(installDir, agentName); err != nil {
		return err
	}
	if err := writeUninstallScript(installDir, dataDir, agentName); err != nil {
		return err
	}
	if err := writeStartMenuShortcuts(installDir); err != nil {
		return fmt.Errorf("start menu: %w", err)
	}
	if err := writeUninstallRegistry(installDir, agentName); err != nil {
		return fmt.Errorf("registry uninstall entry: %w", err)
	}

	if !silent {
		fmt.Println("Installed successfully.")
		fmt.Println("Start Menu → SailGuard → Start Agent (Administrator)")
		fmt.Println("Credentials after enroll stay in:", dataDir)
		fmt.Println()
	}

	if runAfter {
		if err := startAgentElevated(installDir, agentName); err != nil {
			return fmt.Errorf("start agent: %w", err)
		}
	}

	if !silent {
		fmt.Println("Press Enter to close…")
		_, _ = fmt.Scanln()
	}
	return nil
}

func sourceDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return filepath.Dir(exe), nil
	}
	return filepath.Dir(exe), nil
}

func findAgentBinary(dir string) (string, error) {
	candidates := []string{
		"sailguard-agent-win7.exe",
		"sailguard-agent-windows.exe",
		"sailguard-agent.exe",
	}
	for _, name := range candidates {
		path := filepath.Join(dir, name)
		st, err := os.Stat(path)
		if err == nil && !st.IsDir() {
			return name, nil
		}
	}
	return "", fmt.Errorf(
		"agent exe not found next to setup (expected one of %s)",
		strings.Join(candidates, ", "),
	)
}

func loadConfig(path string) (fileConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return fileConfig{}, err
	}
	var cfg fileConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return fileConfig{}, err
	}
	return cfg, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	_ = os.Remove(dst)
	return os.Rename(tmp, dst)
}

func writeStartScripts(installDir, agentName string) error {
	startBat := "" +
		"@echo off\r\n" +
		"cd /d \"%~dp0\"\r\n" +
		"echo SailGuard agent — close this window to stop the agent.\r\n" +
		agentName + "\r\n" +
		"echo.\r\n" +
		"pause\r\n"
	startAdmin := "" +
		"@echo off\r\n" +
		"cd /d \"%~dp0\"\r\n" +
		"powershell -NoProfile -Command \"Start-Process -FilePath '.\\" + agentName + "' -WorkingDirectory '%~dp0' -Verb RunAs\"\r\n" +
		"if errorlevel 1 pause\r\n"
	if err := os.WriteFile(filepath.Join(installDir, "start.bat"), []byte(startBat), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(installDir, "start-admin.bat"), []byte(startAdmin), 0o755)
}

func writeUninstallScript(installDir, dataDir, agentName string) error {
	body := fmt.Sprintf(""+
		"@echo off\r\n"+
		"echo Uninstalling %s...\r\n"+
		"taskkill /F /IM %s >nul 2>&1\r\n"+
		"rmdir /S /Q \"%s\" 2>nul\r\n"+
		"echo.\r\n"+
		"echo Agent files removed from Program Files.\r\n"+
		"echo Data folder left in place (device.json):\r\n"+
		"echo   %s\r\n"+
		"echo Delete that folder manually if you want a clean re-enroll.\r\n"+
		"reg delete \"HKLM\\%s\" /f >nul 2>&1\r\n"+
		"rmdir /S /Q \"%%ProgramData%%\\Microsoft\\Windows\\Start Menu\\Programs\\SailGuard\" 2>nul\r\n"+
		"echo.\r\n"+
		"pause\r\n",
		productName, agentName, installDir, dataDir, uninstallKey)
	return os.WriteFile(filepath.Join(installDir, "uninstall.bat"), []byte(body), 0o755)
}

func writeStartMenuShortcuts(installDir string) error {
	base := os.Getenv("ProgramData")
	if base == "" {
		base = `C:\ProgramData`
	}
	menu := filepath.Join(base, `Microsoft\Windows\Start Menu\Programs\SailGuard`)
	if err := os.MkdirAll(menu, 0o755); err != nil {
		return err
	}
	// .bat launchers in Start Menu (no COM/.lnk dependency — Win7 friendly).
	entries := map[string]string{
		"Start Agent.bat": "" +
			"@echo off\r\n" +
			"call \"" + filepath.Join(installDir, "start.bat") + "\"\r\n",
		"Start Agent (Administrator).bat": "" +
			"@echo off\r\n" +
			"call \"" + filepath.Join(installDir, "start-admin.bat") + "\"\r\n",
		"Uninstall SailGuard Agent.bat": "" +
			"@echo off\r\n" +
			"call \"" + filepath.Join(installDir, "uninstall.bat") + "\"\r\n",
	}
	for name, body := range entries {
		if err := os.WriteFile(filepath.Join(menu, name), []byte(body), 0o755); err != nil {
			return err
		}
	}
	return nil
}

func writeUninstallRegistry(installDir, agentName string) error {
	key, _, err := registry.CreateKey(
		registry.LOCAL_MACHINE,
		uninstallKey,
		registry.SET_VALUE,
	)
	if err != nil {
		return err
	}
	defer key.Close()

	set := func(name, value string) error {
		return key.SetStringValue(name, value)
	}
	if err := set("DisplayName", productName); err != nil {
		return err
	}
	if err := set("Publisher", publisher); err != nil {
		return err
	}
	if err := set("InstallLocation", installDir); err != nil {
		return err
	}
	if err := set("DisplayIcon", filepath.Join(installDir, agentName)); err != nil {
		return err
	}
	if err := set("UninstallString", filepath.Join(installDir, "uninstall.bat")); err != nil {
		return err
	}
	return key.SetDWordValue("NoModify", 1)
}

func startAgentElevated(installDir, agentName string) error {
	ps := fmt.Sprintf(
		"Start-Process -FilePath '%s' -WorkingDirectory '%s' -Verb RunAs",
		filepath.Join(installDir, agentName),
		installDir,
	)
	cmd := exec.Command("powershell", "-NoProfile", "-Command", ps)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Start()
}

func isElevated() (bool, error) {
	var token windows.Token
	err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token)
	if err != nil {
		return false, err
	}
	defer token.Close()
	return token.IsElevated(), nil
}

func relaunchElevated(workDir string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	verb, err := windows.UTF16PtrFromString("runas")
	if err != nil {
		return err
	}
	file, err := windows.UTF16PtrFromString(exe)
	if err != nil {
		return err
	}
	dir, err := windows.UTF16PtrFromString(workDir)
	if err != nil {
		return err
	}
	var argStr string
	if len(os.Args) > 1 {
		argStr = strings.Join(os.Args[1:], " ")
	}
	var params *uint16
	if argStr != "" {
		params, err = windows.UTF16PtrFromString(argStr)
		if err != nil {
			return err
		}
	}
	return windows.ShellExecute(0, verb, file, params, dir, windows.SW_SHOWNORMAL)
}
