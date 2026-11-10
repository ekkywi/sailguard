//go:build windows

package netenforce

import (
	"encoding/binary"
	"fmt"
	"unsafe"

	"github.com/google/uuid"
	"golang.org/x/sys/windows"
)

var (
	modFwpuclnt = windows.NewLazySystemDLL("fwpuclnt.dll")

	procFwpmEngineOpen0    = modFwpuclnt.NewProc("FwpmEngineOpen0")
	procFwpmEngineClose0   = modFwpuclnt.NewProc("FwpmEngineClose0")
	procFwpmProviderAdd0   = modFwpuclnt.NewProc("FwpmProviderAdd0")
	procFwpmSubLayerAdd0   = modFwpuclnt.NewProc("FwpmSubLayerAdd0")
)

const (
	rpcCAuthnWinNT          = 10
	fwpmSessionFlagDynamic  = 0x00000001
	fwpEAlreadyExists       = 0x80320009
)

type engineHandle windows.Handle

type fwpmDisplayData0 struct {
	Name        *uint16
	Description *uint16
}

type fwpmSession0 struct {
	SessionKey          windows.GUID
	DisplayData         fwpmDisplayData0
	Flags               uint32
	TxnWaitTimeoutInMSec uint32
	ProcessID           uint32
	SID                 uintptr
	Username            *uint16
	KernelMode          int32
}

type fwpmProvider0 struct {
	ProviderKey  windows.GUID
	DisplayData  fwpmDisplayData0
	Flags        uint32
	ProviderData fwpByteBlob
	ServiceName  *uint16
}

type fwpByteBlob struct {
	Size uint32
	Data *byte
}

type fwpmSubLayer0 struct {
	SubLayerKey  windows.GUID
	DisplayData  fwpmDisplayData0
	Flags        uint32
	ProviderKey  *windows.GUID
	ProviderData fwpByteBlob
	Weight       uint16
}

func guidFromUUID(id uuid.UUID) windows.GUID {
	var g windows.GUID
	b := id[:]
	g.Data1 = binary.BigEndian.Uint32(b[0:4])
	g.Data2 = binary.BigEndian.Uint16(b[4:6])
	g.Data3 = binary.BigEndian.Uint16(b[6:8])
	copy(g.Data4[:], b[8:16])
	return g
}

func openEngine() (engineHandle, error) {
	name, err := windows.UTF16PtrFromString(providerName + " session")
	if err != nil {
		return 0, err
	}
	desc, err := windows.UTF16PtrFromString(providerDesc)
	if err != nil {
		return 0, err
	}

	session := fwpmSession0{
		DisplayData: fwpmDisplayData0{Name: name, Description: desc},
		Flags: fwpmSessionFlagDynamic,
	}

	var h windows.Handle
	r1, _, _ := procFwpmEngineOpen0.Call(
		0,
		uintptr(rpcCAuthnWinNT),
		0,
		uintptr(unsafe.Pointer(&session)),
		uintptr(unsafe.Pointer(&h)),
	)
	if r1 != 0 {
		return 0, fmt.Errorf("FwpmEngineOpen0: 0x%X (need admin/service?)", r1)
	}
	return engineHandle(h), nil
}

func closeEngine(h engineHandle) {
	if h != 0 {
		procFwpmEngineClose0.Call(uintptr(h))
	}
}

func ensureProvider(h engineHandle) error {
	name, err := windows.UTF16PtrFromString(providerName)
	if err != nil {
		return err
	}
	desc, err := windows.UTF16PtrFromString(providerDesc)
	if err != nil {
		return err
	}
	p := fwpmProvider0{
		ProviderKey: guidFromUUID(providerKey),
		DisplayData: fwpmDisplayData0{Name: name, Description: desc},
	}
	r1, _, _ := procFwpmProviderAdd0.Call(
		uintptr(h),
		uintptr(unsafe.Pointer(&p)),
		0,
	)
	if r1 != 0 && r1 != fwpEAlreadyExists {
		return fmt.Errorf("FwpmProviderAdd0: 0x%X", r1)
	}
	return nil
}

func ensureSublayer(h engineHandle) error {
	name, err := windows.UTF16PtrFromString(sublayerName)
	if err != nil {
		return err
	}
	desc, err := windows.UTF16PtrFromString(sublayerDesc)
	if err != nil {
		return err
	}
	pk := guidFromUUID(providerKey)
	s := fwpmSubLayer0{
		SubLayerKey: guidFromUUID(sublayerKey),
		DisplayData: fwpmDisplayData0{Name: name, Description: desc},
		ProviderKey: &pk,
		Weight:      sublayerWeight,
	}
	r1, _, _ := procFwpmSubLayerAdd0.Call(
		uintptr(h),
		uintptr(unsafe.Pointer(&s)),
		0,
	)
	if r1 != 0 && r1 != fwpEAlreadyExists {
		return fmt.Errorf("FwpmSubLayerAdd0: 0x%X", r1)
	}
	return nil
}