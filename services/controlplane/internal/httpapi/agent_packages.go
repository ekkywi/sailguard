package httpapi

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"time"
)

type windowsAgentPackageRequest struct {
	Channel         string `json:"channel"` // windows-legacy | windows-current | linux
	PackageKind     string `json:"package_kind"` // install (default) | update
	ServerURL       string `json:"server_url"`
	EnrollmentToken string `json:"enrollment_token"`
	DataDir         string `json:"data_dir"`
}

type agentFileConfig struct {
	ServerURL       string `json:"server_url"`
	EnrollmentToken string `json:"enrollment_token,omitempty"`
	DataDir         string `json:"data_dir,omitempty"`
}

type agentChannel struct {
	ID           string
	Label        string
	Description  string
	OSFamily     string
	Toolchain    string
	BinaryName   string
	BinaryPath   string
	BuildHint    string
	Downloadable bool
	ComingSoon   bool
}

const (
	defaultWindowsDataDir = `C:\ProgramData\SailGuard\agent`
	defaultLinuxDataDir   = `/var/lib/sailguard/agent`
	setupBinaryName       = "sailguard-agent-setup.exe"
	setupBuildHint        = "Run ./scripts/build-agent-setup.sh (requires Go 1.20)."
	linuxInstallBinDir    = "/usr/local/bin"
	linuxUnitName         = "sailguard-agent.service"
)

func (a *API) agentChannels() []agentChannel {
	return []agentChannel{
		{
			ID:           "windows-legacy",
			Label:        "Windows 7 SP1+",
			Description:  "Legacy channel for Windows 7 SP1, 8.1, and mixed fleets. Built with Go 1.20.",
			OSFamily:     "windows",
			Toolchain:    "go1.20",
			BinaryName:   "sailguard-agent-win7.exe",
			BinaryPath:   a.cfg.AgentWin7Path,
			BuildHint:    "Run ./scripts/build-agent-win7.sh (requires Go 1.20).",
			Downloadable: true,
		},
		{
			ID:           "windows-current",
			Label:        "Windows 10 / 11",
			Description:  "Current channel for modern Windows. Built with the repo Go toolchain (uptodate).",
			OSFamily:     "windows",
			Toolchain:    "go-current",
			BinaryName:   "sailguard-agent-windows.exe",
			BinaryPath:   a.cfg.AgentWindowsPath,
			BuildHint:    "Run ./scripts/build-agent-windows.sh (current Go on PATH).",
			Downloadable: true,
		},
		{
			ID:           "linux",
			Label:        "Linux",
			Description:  "Linux amd64 agent with install.sh and systemd unit.",
			OSFamily:     "linux",
			Toolchain:    "go-current",
			BinaryName:   "sailguard-agent",
			BinaryPath:   a.cfg.AgentLinuxPath,
			BuildHint:    "Run ./scripts/build-agent-linux.sh",
			Downloadable: true,
			ComingSoon:   false,
		},
	}
}

func (a *API) channelByID(id string) (agentChannel, bool) {
	for _, ch := range a.agentChannels() {
		if ch.ID == id {
			return ch, true
		}
	}
	return agentChannel{}, false
}

func binaryStat(path string) (available bool, size int64, modTime string) {
	if path == "" {
		return false, 0, ""
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false, 0, ""
	}
	return true, info.Size(), info.ModTime().UTC().Format(time.RFC3339)
}

func (a *API) defaultAgentServerURL(r *http.Request) string {
	defaultURL := strings.TrimSpace(a.cfg.PublicAgentURL)
	if defaultURL != "" {
		return defaultURL
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if fwd := r.Header.Get("X-Forwarded-Proto"); fwd != "" {
		scheme = fwd
	}
	if host := r.Host; host != "" {
		return scheme + "://" + host
	}
	return ""
}

func (a *API) handleAgentPackageMeta(w http.ResponseWriter, r *http.Request) {
	channels := make([]map[string]any, 0)
	for _, ch := range a.agentChannels() {
		available, size, modTime := binaryStat(ch.BinaryPath)
		channels = append(channels, map[string]any{
			"id":                 ch.ID,
			"label":              ch.Label,
			"description":        ch.Description,
			"os_family":          ch.OSFamily,
			"toolchain":          ch.Toolchain,
			"binary_name":        ch.BinaryName,
			"binary_path":        ch.BinaryPath,
			"binary_available":   available,
			"binary_size_bytes":  size,
			"binary_modified_at": modTime,
			"build_hint":         ch.BuildHint,
			"downloadable":       ch.Downloadable,
			"coming_soon":        ch.ComingSoon,
		})
	}

	setupOK, setupSize, setupMod := binaryStat(a.cfg.AgentSetupPath)
	writeJSON(w, http.StatusOK, envelope{
		OK: true,
		Data: map[string]any{
			"default_server_url":         a.defaultAgentServerURL(r),
			"default_data_dir":           defaultWindowsDataDir,
			"default_data_dir_windows":   defaultWindowsDataDir,
			"default_data_dir_linux":     defaultLinuxDataDir,
			"setup_binary_name":          setupBinaryName,
			"setup_binary_path":          a.cfg.AgentSetupPath,
			"setup_binary_available":     setupOK,
			"setup_binary_size_bytes":    setupSize,
			"setup_binary_modified_at":   setupMod,
			"setup_build_hint":           setupBuildHint,
			"channels":                   channels,
		},
	})
}

func (a *API) handleDownloadWindowsAgentPackage(w http.ResponseWriter, r *http.Request) {
	var req windowsAgentPackageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid JSON body",
			},
		})
		return
	}

	channelID := strings.TrimSpace(req.Channel)
	if channelID == "" {
		channelID = "windows-legacy"
	}
	ch, ok := a.channelByID(channelID)
	if !ok || !ch.Downloadable {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "unsupported or unavailable channel (use windows-legacy, windows-current, or linux)",
			},
		})
		return
	}

	packageKind := strings.ToLower(strings.TrimSpace(req.PackageKind))
	if packageKind == "" {
		packageKind = "install"
	}
	switch packageKind {
	case "install", "update":
	default:
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "package_kind must be install or update",
			},
		})
		return
	}

	serverURL := strings.TrimSpace(req.ServerURL)
	token := strings.TrimSpace(req.EnrollmentToken)
	dataDir := strings.TrimSpace(req.DataDir)
	if dataDir == "" {
		if ch.OSFamily == "linux" {
			dataDir = defaultLinuxDataDir
		} else {
			dataDir = defaultWindowsDataDir
		}
	}
	if serverURL == "" {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "server_url is required",
			},
		})
		return
	}
	if packageKind == "install" && token == "" {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "enrollment_token is required for install packages",
			},
		})
		return
	}
	if packageKind == "update" {
		// Update packages must not bake a token — upgrades reuse device.json.
		token = ""
	}
	if !strings.HasPrefix(serverURL, "http://") && !strings.HasPrefix(serverURL, "https://") {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "server_url must start with http:// or https://",
			},
		})
		return
	}

	exeFile, err := os.Open(ch.BinaryPath)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "binary_missing",
				"message": fmt.Sprintf("%s binary not found at %s — %s", ch.Label, ch.BinaryPath, ch.BuildHint),
			},
		})
		return
	}
	defer exeFile.Close()

	cfgBytes, err := json.MarshalIndent(agentFileConfig{
		ServerURL:       serverURL,
		EnrollmentToken: token,
		DataDir:         dataDir,
	}, "", "  ")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to encode config.json",
			},
		})
		return
	}
	cfgBytes = append(cfgBytes, '\n')

	if ch.OSFamily == "linux" {
		a.writeLinuxAgentZip(w, r, ch, channelID, packageKind, serverURL, dataDir, cfgBytes, exeFile)
		return
	}

	setupPath := a.cfg.AgentSetupPath
	setupFile, err := os.Open(setupPath)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "setup_missing",
				"message": fmt.Sprintf("setup binary not found at %s — %s", setupPath, setupBuildHint),
			},
		})
		return
	}
	defer setupFile.Close()

	a.writeWindowsAgentZip(w, r, ch, channelID, packageKind, serverURL, dataDir, cfgBytes, exeFile, setupFile)
}

func (a *API) writeWindowsAgentZip(
	w http.ResponseWriter,
	r *http.Request,
	ch agentChannel,
	channelID, packageKind, serverURL, dataDir string,
	cfgBytes []byte,
	exeFile, setupFile *os.File,
) {
	startBat := []byte("" +
		"@echo off\r\n" +
		"cd /d \"%~dp0\"\r\n" +
		"echo SailGuard agent — close this window to stop the agent.\r\n" +
		ch.BinaryName + "\r\n" +
		"echo.\r\n" +
		"pause\r\n")

	startAdminBat := []byte("" +
		"@echo off\r\n" +
		"cd /d \"%~dp0\"\r\n" +
		"powershell -NoProfile -Command \"Start-Process -FilePath '.\\" + ch.BinaryName + "' -WorkingDirectory '%~dp0' -Verb RunAs\"\r\n" +
		"if errorlevel 1 pause\r\n")

	var readme []byte
	if packageKind == "update" {
		readme = []byte(fmt.Sprintf(""+
			"SailGuard agent UPDATE package (%s)\n"+
			"==================================\n\n"+
			"Channel: %s (%s)\n"+
			"Toolchain: %s\n"+
			"Kind: update (no enrollment token)\n\n"+
			"Use this when the device is already enrolled.\n"+
			"Keep ProgramData so device.json is reused — do not delete:\n"+
			"  %s\\device.json\n\n"+
			"Steps:\n"+
			"1. Unzip this folder.\n"+
			"2. Right-click %s → Run as administrator.\n"+
			"3. Setup stops the SailGuardAgent service, replaces files, then starts it again.\n"+
			"4. Keep %s\\device.json — do not delete ProgramData.\n\n"+
			"Enrollment token is NOT required and is not included.\n"+
			"Server URL in config.json: %s\n"+
			"Service name: SailGuardAgent (Automatic)\n",
			ch.Label, ch.ID, ch.Description, ch.Toolchain,
			dataDir, setupBinaryName, dataDir, serverURL))
	} else {
		readme = []byte(fmt.Sprintf(""+
			"SailGuard agent installer package (%s)\n"+
			"=====================================\n\n"+
			"Channel: %s (%s)\n"+
			"Toolchain: %s\n"+
			"Kind: first install (enrollment token baked into config.json)\n\n"+
			"Recommended:\n"+
			"1. Unzip this folder anywhere (e.g. Desktop).\n"+
			"2. Right-click %s → Run as administrator.\n"+
			"3. Setup installs to:\n"+
			"     C:\\Program Files\\SailGuard\\Agent\\\n"+
			"   data under:\n"+
			"     %s\n"+
			"4. Registers Windows service SailGuardAgent (Automatic) and starts it.\n"+
			"5. Optional Start Menu: Start/Stop SailGuard Service; Console mode for debug.\n\n"+
			"Server: %s\n",
			ch.Label, ch.ID, ch.Description, ch.Toolchain,
			setupBinaryName, dataDir, serverURL))
	}

	zipName := fmt.Sprintf("sailguard-agent-%s-%s.zip", channelID, packageKind)
	a.recordAudit(r, "agent_package.download", "agent_package", channelID,
		"Downloaded Windows agent package",
		map[string]any{
			"channel":      channelID,
			"package_kind": packageKind,
			"server_url":   serverURL,
		},
	)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, zipName))
	w.WriteHeader(http.StatusOK)

	zw := zip.NewWriter(w)
	defer zw.Close()

	if err := writeZipFile(zw, "config.json", cfgBytes); err != nil {
		return
	}
	if err := writeZipFile(zw, "start.bat", startBat); err != nil {
		return
	}
	if err := writeZipFile(zw, "start-admin.bat", startAdminBat); err != nil {
		return
	}
	if err := writeZipFile(zw, "README.txt", readme); err != nil {
		return
	}

	setupEntry, err := zw.Create(setupBinaryName)
	if err != nil {
		return
	}
	if _, err := io.Copy(setupEntry, setupFile); err != nil {
		return
	}

	exeEntry, err := zw.Create(ch.BinaryName)
	if err != nil {
		return
	}
	if _, err := io.Copy(exeEntry, exeFile); err != nil {
		return
	}
}

func (a *API) writeLinuxAgentZip(
	w http.ResponseWriter,
	r *http.Request,
	ch agentChannel,
	channelID, packageKind, serverURL, dataDir string,
	cfgBytes []byte,
	exeFile *os.File,
) {
	unit := linuxSystemdUnit(dataDir)
	installSh := linuxInstallScript(packageKind, dataDir)
	readme := linuxReadme(ch, packageKind, serverURL, dataDir)

	zipName := fmt.Sprintf("sailguard-agent-%s-%s.zip", channelID, packageKind)
	a.recordAudit(r, "agent_package.download", "agent_package", channelID,
		"Downloaded Linux agent package",
		map[string]any{
			"channel":      channelID,
			"package_kind": packageKind,
			"server_url":   serverURL,
		},
	)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, zipName))
	w.WriteHeader(http.StatusOK)

	zw := zip.NewWriter(w)
	defer zw.Close()

	if err := writeZipFile(zw, "config.json", cfgBytes); err != nil {
		return
	}
	if err := writeZipFile(zw, linuxUnitName, []byte(unit)); err != nil {
		return
	}
	if err := writeZipFileMode(zw, "install.sh", []byte(installSh), 0o755); err != nil {
		return
	}
	if err := writeZipFile(zw, "README.txt", []byte(readme)); err != nil {
		return
	}

	exeEntry, err := writeZipCreateMode(zw, ch.BinaryName, 0o755)
	if err != nil {
		return
	}
	if _, err := io.Copy(exeEntry, exeFile); err != nil {
		return
	}
}

func linuxSystemdUnit(dataDir string) string {
	return fmt.Sprintf(`[Unit]
Description=SailGuard endpoint agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=%s/sailguard-agent
WorkingDirectory=%s
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
`, linuxInstallBinDir, dataDir)
}

func linuxInstallScript(packageKind, dataDir string) string {
	return fmt.Sprintf(`#!/usr/bin/env bash
# SailGuard Linux agent installer (package_kind=%s)
set -euo pipefail

if [[ "${EUID:-$(id -u)}" -ne 0 ]]; then
  echo "Run as root: sudo ./install.sh" >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_SRC="${SCRIPT_DIR}/sailguard-agent"
CFG_SRC="${SCRIPT_DIR}/config.json"
UNIT_SRC="${SCRIPT_DIR}/%s"
BIN_DST="%s/sailguard-agent"
CFG_DST="%s/config.json"
UNIT_DST="/etc/systemd/system/%s"
DATA_DIR="%s"

if [[ ! -f "$BIN_SRC" ]]; then
  echo "missing sailguard-agent next to install.sh" >&2
  exit 1
fi
if [[ ! -f "$CFG_SRC" ]]; then
  echo "missing config.json next to install.sh" >&2
  exit 1
fi
if [[ ! -f "$UNIT_SRC" ]]; then
  echo "missing %s next to install.sh" >&2
  exit 1
fi

install -d -m 755 "%s"
install -d -m 700 "$DATA_DIR"

systemctl stop sailguard-agent.service 2>/dev/null || true

install -m 755 "$BIN_SRC" "$BIN_DST"
# Agent loads config.json from the executable directory.
install -m 600 "$CFG_SRC" "$CFG_DST"
install -m 644 "$UNIT_SRC" "$UNIT_DST"

systemctl daemon-reload
systemctl enable sailguard-agent.service
systemctl restart sailguard-agent.service

echo "Installed SailGuard agent."
echo "  binary:  $BIN_DST"
echo "  config:  $CFG_DST"
echo "  data:    $DATA_DIR (keep device.json on updates)"
echo "  service: sailguard-agent.service"
systemctl --no-pager --full status sailguard-agent.service || true
`, packageKind, linuxUnitName, linuxInstallBinDir, linuxInstallBinDir, linuxUnitName, dataDir,
		linuxUnitName, linuxInstallBinDir)
}

func linuxReadme(ch agentChannel, packageKind, serverURL, dataDir string) string {
	if packageKind == "update" {
		return fmt.Sprintf(""+
			"SailGuard agent UPDATE package (%s)\n"+
			"==================================\n\n"+
			"Channel: %s (%s)\n"+
			"Toolchain: %s\n"+
			"Kind: update (no enrollment token)\n\n"+
			"Use this when the device is already enrolled.\n"+
			"Keep the data directory so device.json is reused — do not delete:\n"+
			"  %s/device.json\n\n"+
			"Steps:\n"+
			"1. Unzip this folder.\n"+
			"2. chmod +x install.sh sailguard-agent   # if needed\n"+
			"3. sudo ./install.sh\n"+
			"4. Confirm: systemctl status sailguard-agent\n\n"+
			"Enrollment token is NOT required and is not included.\n"+
			"Server URL in config.json: %s\n"+
			"Service: sailguard-agent.service\n",
			ch.Label, ch.ID, ch.Description, ch.Toolchain, dataDir, serverURL)
	}
	return fmt.Sprintf(""+
		"SailGuard agent installer package (%s)\n"+
		"=====================================\n\n"+
		"Channel: %s (%s)\n"+
		"Toolchain: %s\n"+
		"Kind: first install (enrollment token baked into config.json)\n\n"+
		"Recommended:\n"+
		"1. Unzip this folder.\n"+
		"2. chmod +x install.sh sailguard-agent   # if needed\n"+
		"3. sudo ./install.sh\n"+
		"4. install.sh copies the binary to %s/sailguard-agent,\n"+
		"   writes config next to the binary, creates data dir:\n"+
		"     %s\n"+
		"   and enables systemd unit sailguard-agent.service.\n\n"+
		"Server: %s\n",
		ch.Label, ch.ID, ch.Description, ch.Toolchain,
		linuxInstallBinDir, dataDir, serverURL)
}

func writeZipFile(zw *zip.Writer, name string, body []byte) error {
	return writeZipFileMode(zw, name, body, 0o644)
}

func writeZipFileMode(zw *zip.Writer, name string, body []byte, mode fs.FileMode) error {
	w, err := writeZipCreateMode(zw, name, mode)
	if err != nil {
		return err
	}
	_, err = w.Write(body)
	return err
}

func writeZipCreateMode(zw *zip.Writer, name string, mode fs.FileMode) (io.Writer, error) {
	h := &zip.FileHeader{
		Name:     name,
		Method:   zip.Deflate,
		Modified: time.Now(),
	}
	h.SetMode(mode)
	return zw.CreateHeader(h)
}
