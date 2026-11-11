//go:build windows

package netenforce

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"unsafe"

	"github.com/google/uuid"
	"golang.org/x/sys/windows"
)

type fwpValue0 struct {
	Type  uint32
	Value uintptr
}

type fwpAction0 struct {
	Type       uint32
	FilterType windows.GUID
}

type fwpmFilterCondition0 struct {
	FieldKey       windows.GUID
	MatchType      uint32
	ConditionValue fwpValue0
}

// Layout must match FWPM_FILTER0 on amd64 — adjust if FwpmFilterAdd0 fails at runtime.
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
	ContextRaw          uint64
	Reserved            *windows.GUID
	FilterID            uint64
	EffectiveWeight     fwpValue0
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
		u, err := ipv4ToUint32(ip)
		if err != nil {
			return nil, nil, err
		}
		conds = append(conds, fwpmFilterCondition0{
			FieldKey:  condIPRemoteAddress,
			MatchType: fwpMatchEqual,
			ConditionValue: fwpValue0{
				Type:  fwpUint32,
				Value: uintptr(u),
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
		type v4AddrMask struct {
			Addr uint32
			Mask uint32
		}
		am := &v4AddrMask{Addr: addr, Mask: mask}
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
		keep = append(keep, &pu)
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

func addFilter(
	h engineHandle,
	ruleID, name string,
	block bool,
	weight uint64,
	destType, destValue string,
	port *int,
) error {
	conds, keep, err := buildConditions(destType, destValue, port)
	if err != nil {
		return err
	}
	_ = keep

	var condPtr *fwpmFilterCondition0
	if len(conds) > 0 {
		condPtr = &conds[0]
	}

	w := weight
	weightVal := fwpValue0{
		Type:  fwpUint64,
		Value: uintptr(unsafe.Pointer(&w)),
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

	f := fwpmFilter0{
		FilterKey:           filterKeyForRule(ruleID),
		DisplayData:         fwpmDisplayData0{Name: dispName},
		ProviderKey:         &pk,
		LayerKey:            layerALEAuthConnectV4,
		SubLayerKey:         guidFromUUID(sublayerKey),
		Weight:              weightVal,
		NumFilterConditions: uint32(len(conds)),
		FilterCondition:     condPtr,
		Action:              fwpAction0{Type: actionType},
	}

	r1, _, _ := procFwpmFilterAdd0.Call(
		uintptr(h),
		uintptr(unsafe.Pointer(&f)),
		0,
		0,
	)
	if r1 != 0 {
		return fmt.Errorf("FwpmFilterAdd0 %s: 0x%X", name, r1)
	}
	return nil
}

func applySpecFilters(h engineHandle, spec Spec) error {
	var errs int

	addOne := func(r Rule, block bool, weight uint64) {
		name := r.Label
		if name == "" {
			name = r.ID
		}
		if err := addFilter(h, r.ID, name, block, weight, r.DestType, r.DestValue, r.Port); err != nil {
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
