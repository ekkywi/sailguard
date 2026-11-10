//go:build windows

package netenforce

import "github.com/google/uuid"

var (
	providerKey = uuid.MustParse("8f3c2a10-1111-4222-9333-000000004401")
	sublayerKey = uuid.MustParse("8f3c2a10-1111-4222-9333-000000004402")
)

const (
	providerName   = "SailGuard Network"
	providerDesc   = "SailGuard destination allow/deny provider"
	sublayerName   = "SailGuard Network Deny"
	sublayerDesc   = "SailGuard outbond destination filters"
	sublayerWeight = uint16(0x5000)
)
