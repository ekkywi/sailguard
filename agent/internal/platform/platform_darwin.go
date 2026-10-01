//go:build darwin

package platform

import (
	"github.com/ekkywi/sailguard/agent/internal/host"
	"github.com/ekkywi/sailguard/agent/internal/darwin"
)

func NewHost() host.Host { return darwin.New() }
