//go:build windows

package netenforce

import (
	"github.com/google/uuid"
	"golang.org/x/sys/windows"
)

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

// GUIDs from fwpmu.h (same values WireGuard uses).
var (
	layerALEAuthConnectV4 = mustWinGUID("c38d57d1-05a7-4c33-904f-7fbceee60e82")
	condIPRemoteAddress   = mustWinGUID("b235ae9a-1d64-49b8-a44c-5ff3d9095045")
	condIPRemotePort      = mustWinGUID("c35a604d-d22b-4e1a-91b4-68f674ee674b")
)

const (
	fwpActionFlagTerminating = 0x00001000
	fwpActionBlock           = 0x00000001 | fwpActionFlagTerminating
	fwpActionPermit          = 0x00000002 | fwpActionFlagTerminating

	fwpMatchEqual = 0

	fwpUint8         = 1
	fwpUint16        = 2
	fwpUint32        = 3
	fwpUint64        = 4
	fwpV4AddrAndMask = 0x100

	// FWP_UINT8 weight range index 0–15 (permit must be higher than deny).
	weightPermit = uint8(10)
	weightDeny   = uint8(5)
)

func mustWinGUID(s string) windows.GUID {
	return guidFromUUID(uuid.MustParse(s))
}
