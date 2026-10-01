//go:build windows

package platform

import (
	"github.com/ekkywi/sailguard/agent/internal/host"
	"github.com/ekkywi/sailguard/agent/internal/windows"
)

func NewHost() host.Host { return windows.New() }
