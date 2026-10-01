//go:build linux

package platform

import (
	"github.com/ekkywi/sailguard/agent/internal/host"
	"github.com/ekkywi/sailguard/agent/internal/linux"
)

func NewHost() host.Host { return linux.New() }
