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

var (
	layerALEAuthConnectV4 = mustWinGUID("c38d57d1-05a7-4c33-904f-7fbceee60e82")
	condIPRemoteAddress   = mustWinGUID("b235ae06-9a79-4a80-bee7-c6b1f012b274")
	condIPRemotePort      = mustWinGUID("c35a604d-d22b-4e55-a924-290c3284f87c")
)

const (
	fwpActionFlagTerminating = 0x00001000
	fwpActionBlock           = 0x00000001 | fwpActionFlagTerminating
	fwpActionPermit          = 0x00000002 | fwpActionFlagTerminating

	fwpMatchEqual = 0

	fwpUint16        = 2
	fwpUint32        = 3
	fwpUint64        = 4
	fwpV4AddrAndMask = 0x100

	weightPermit = uint64(0x8000)
	weightDeny   = uint64(0x4000)
)

func mustWinGUID(s string) windows.GUID {
	return guidFromUUID(uuid.MustParse(s))
}
