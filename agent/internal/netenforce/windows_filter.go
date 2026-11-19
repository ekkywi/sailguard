//go:build windows

package netenforce

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"runtime"
	"unsafe"

	"github.com/google/uuid"
	"golang.org/x/sys/windows"
)

// FWP_VALUE0 / FWP_CONDITION_VALUE0 — type (UINT32) + 4-byte pad + 8-byte union (amd64).
type fwpValue0 struct {
	Type  uint32
	_     uint32
	Value uintptr
}

// FWPM_ACTION0 is 20 bytes: UINT32 type then GUID at offset 4 (MSVC, no pad).
// Do not insert padding before FilterType — that breaks FWPM_FILTER0 offsets.
type fwpAction0 struct {
	Type       uint32
	FilterType windows.GUID
}

type fwpmFilterCondition0 struct {
	FieldKey       windows.GUID
	MatchType      uint32
	ConditionValue fwpValue0 // Go pads 4 bytes so this starts at offset 24
}

// FWPM_FILTER0 amd64 layout (200 bytes). Matches WireGuard / Windows SDK:
// action @128 (20), pad 4, providerContextKey GUID @152, reserved @168, filterId @176.
type fwpmFilter0 struct {
	FilterKey           windows.GUID
	DisplayData         fwpmDisplayData0
	Flags               uint32
	ProviderKey         *windows.GUID
	ProviderData        fwpByteBlob
	LayerKey            windows.GUID
	SubLayerKey         windows.GUID
	Weight              fwpValue0
	NumFilterConditions uint32
	FilterCondition     *fwpmFilterCondition0
	Action              fwpAction0
	_                   [4]byte // align providerContextKey to offset 152
	ProviderContextKey  windows.GUID
	Reserved            *windows.GUID
	FilterID            uint64
	EffectiveWeight     fwpValue0
}

// Host-order IPv4 addr+mask (FWP_V4_ADDR_AND_MASK).
type wfpV4AddrMask struct {
	Addr uint32
	Mask uint32
}

func filterKeyForRule(id string) windows.GUID {
	if u, err := uuid.Parse(id); err == nil {
		return guidFromUUID(u)
	}
	return guidFromUUID(uuid.NewSHA1(uuid.NameSpaceOID, []byte("sailguard-net/"+id)))
}

func ipv4ToUint32(ip net.IP) (uint32, error) {
	v4 := ip.To4()
	if v4 == nil {
		return 0, fmt.Errorf("not IPv4: %v", ip)
	}
	// Numeric host-order form (first octet in high byte), same as WireGuard.
	return binary.BigEndian.Uint32(v4), nil
}

func buildConditions(destType, destValue string, port *int) ([]fwpmFilterCondition0, []any, error) {
	keep := make([]any, 0, 4)
	conds := make([]fwpmFilterCondition0, 0, 2)

	switch destType {
	case "ip":
		ip := net.ParseIP(destValue)
		if ip == nil {
			return nil, nil, fmt.Errorf("invalid ip %q", destValue)
		}
		addr, err := ipv4ToUint32(ip)
		if err != nil {
			return nil, nil, err
		}
		// Exact IPv4: FWP_UINT32 inline (WireGuard pattern).
		conds = append(conds, fwpmFilterCondition0{
			FieldKey:  condIPRemoteAddress,
			MatchType: fwpMatchEqual,
			ConditionValue: fwpValue0{
				Type:  fwpUint32,
				Value: uintptr(addr),
			},
		})
	case "cidr":
		_, network, err := net.ParseCIDR(destValue)
		if err != nil {
			return nil, nil, err
		}
		ones, bits := network.Mask.Size()
		if bits != 32 {
			return nil, nil, fmt.Errorf("only IPv4 cidr supported")
		}
		addr, err := ipv4ToUint32(network.IP)
		if err != nil {
			return nil, nil, err
		}
		mask := uint32(0)
		if ones != 0 {
			mask = ^uint32(0) << uint(32-ones)
		}
		am := &wfpV4AddrMask{Addr: addr, Mask: mask}
		keep = append(keep, am)
		conds = append(conds, fwpmFilterCondition0{
			FieldKey:  condIPRemoteAddress,
			MatchType: fwpMatchEqual,
			ConditionValue: fwpValue0{
				Type:  fwpV4AddrAndMask,
				Value: uintptr(unsafe.Pointer(am)),
			},
		})
	default:
		return nil, nil, fmt.Errorf("unsupported dest_type %q", destType)
	}

	if port != nil {
		pu := uint16(*port)
		conds = append(conds, fwpmFilterCondition0{
			FieldKey:  condIPRemotePort,
			MatchType: fwpMatchEqual,
			ConditionValue: fwpValue0{
				Type:  fwpUint16,
				Value: uintptr(pu),
			},
		})
	}

	return conds, keep, nil
}

func (e *windowsEnforcer) addFilterLocked(
	ruleID, name string,
	block bool,
	weight uint8,
	destType, destValue string,
	port *int,
) error {
	conds, keep, err := buildConditions(destType, destValue, port)
	if err != nil {
		return err
	}

	var condPtr *fwpmFilterCondition0
	if len(conds) > 0 {
		condPtr = &conds[0]
	}

	weightVal := fwpValue0{
		Type:  fwpUint8,
		Value: uintptr(weight),
	}

	pk := guidFromUUID(providerKey)
	dispName, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return err
	}
	actionType := uint32(fwpActionPermit)
	if block {
		actionType = fwpActionBlock
	}

	key := filterKeyForRule(ruleID)
	f := fwpmFilter0{
		FilterKey:           key,
		DisplayData:         fwpmDisplayData0{Name: dispName},
		ProviderKey:         &pk,
		LayerKey:            layerALEAuthConnectV4,
		SubLayerKey:         guidFromUUID(sublayerKey),
		Weight:              weightVal,
		NumFilterConditions: uint32(len(conds)),
		FilterCondition:     condPtr,
		Action:              fwpAction0{Type: actionType},
	}

	var filterID uint64
	r1, _, _ := procFwpmFilterAdd0.Call(
		uintptr(e.eng),
		uintptr(unsafe.Pointer(&f)),
		0,
		uintptr(unsafe.Pointer(&filterID)),
	)

	runtime.KeepAlive(keep)
	runtime.KeepAlive(conds)
	runtime.KeepAlive(pk)
	runtime.KeepAlive(dispName)
	runtime.KeepAlive(f)

	if r1 != 0 {
		return fmt.Errorf("FwpmFilterAdd0 %s: 0x%X", name, r1)
	}

	rec := installedFilters{
		Key:       key,
		FilterID:  filterID,
		RuleID:    ruleID,
		DestType:  destType,
		DestValue: destValue,
		Port:      port,
		Block:     block,
	}
	e.installed = append(e.installed, rec)
	if e.byFilterID == nil {
		e.byFilterID = make(map[uint64]installedFilters)
	}
	e.byFilterID[filterID] = rec

	log.Printf(
		"network enforce: filter id=%d rule=%s block=%v dest=%s/%s",
		filterID, ruleID, block, destType, destValue,
	)
	return nil
}

func (e *windowsEnforcer) applySpecFiltersLocked(spec Spec) error {
	var errs int

	addOne := func(r Rule, block bool, weight uint8) {
		name := r.Label
		if name == "" {
			name = r.ID
		}
		if err := e.addFilterLocked(r.ID, name, block, weight, r.DestType, r.DestValue, r.Port); err != nil {
			log.Printf("network enforce: skip rule %s: %v", r.ID, err)
			errs++
		}
	}

	for _, r := range spec.SafetyAllow {
		addOne(r, false, weightPermit)
	}
	for _, r := range spec.Rules {
		switch r.Action {
		case "allow":
			addOne(r, false, weightPermit)
		case "deny":
			addOne(r, true, weightDeny)
		default:
			log.Printf("network enforce: skip rule %s: unknown action %q", r.ID, r.Action)
		}
	}

	if errs > 0 {
		return fmt.Errorf("network enforce: %d rule(s) failed to add", errs)
	}
	return nil
}
