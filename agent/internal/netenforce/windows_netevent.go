//go:build windows

package netenforce

import (
	"fmt"
	"log"
	"net"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	procFwpmNetEventSubscribe0   = modFwpuclnt.NewProc("FwpmNetEventSubscribe0")
	procFwpmNetEventSubscribe1   = modFwpuclnt.NewProc("FwpmNetEventSubscribe1")
	procFwpmNetEventUnsubscribe0 = modFwpuclnt.NewProc("FwpmNetEventUnsubscribe0")
)

const (
	fwpmNetEventClassifyDrop = 3 // FWPM_NET_EVENT_TYPE_CLASSIFY_DROP
	fwpIPVersionV4           = 0
)

// FWPM_NET_EVENT_HEADER2 (Win8+) — used with FwpmNetEventSubscribe1 → FWPM_NET_EVENT2.
// HEADER1 (Subscribe0) has a large trailing reserved union; our old short HEADER1
// mis-read Type as 0. Prefer Subscribe1 on modern Windows.
type fwpmNetEventHeader2 struct {
	TimeStamp     windows.Filetime
	Flags         uint32
	IPVersion     uint32
	IPProtocol    uint8
	_             [3]byte
	LocalAddr     [16]byte
	RemoteAddr    [16]byte
	LocalPort     uint16
	RemotePort    uint16
	ScopeID       uint32
	AppID         fwpByteBlob
	UserID        uintptr
	AddressFamily uint32
	PackageSID    uintptr // Go pads 4 bytes before this on amd64
}

type fwpmNetEvent2 struct {
	Header fwpmNetEventHeader2
	Type   uint32
	_      uint32 // pad for pointer
	Data   uintptr
}

// filterId is the first field in CLASSIFY_DROP0/1/2 — enough for lookup.
type fwpmClassifyDrop struct {
	FilterID uint64
	LayerID  uint16
}

// FWPM_NET_EVENT_SUBSCRIPTION0 — must be non-null (NULL → FWP_E_NULL_POINTER).
type fwpmNetEventSubscription0 struct {
	EnumTemplate uintptr
	Flags        uint32
	_            uint32
	SessionKey   windows.GUID
}

type netEventContext struct {
	enf *windowsEnforcer
}

// Caps noisy callback logs so a flood of system drops does not fill the service log.
var netEventDiagLeft atomic.Int32

func netEventCallback(ctx uintptr, event uintptr) uintptr {
	if ctx == 0 || event == 0 {
		return 0
	}
	c := (*netEventContext)(unsafe.Pointer(ctx))
	if c == nil || c.enf == nil {
		return 0
	}

	ev := (*fwpmNetEvent2)(unsafe.Pointer(event))
	if left := netEventDiagLeft.Add(-1); left >= 0 {
		log.Printf("network enforce: net-event diag type=%d data=0x%X", ev.Type, ev.Data)
	}

	if ev.Type != fwpmNetEventClassifyDrop || ev.Data == 0 {
		return 0
	}
	drop := (*fwpmClassifyDrop)(unsafe.Pointer(ev.Data))
	filterID := drop.FilterID

	e := c.enf
	if !e.mu.TryLock() {
		return 0
	}
	rec, ok := e.byFilterID[filterID]
	fn := e.onDrop
	e.mu.Unlock()

	remote := remoteIPv4(&ev.Header)
	if !ok || !rec.Block {
		log.Printf(
			"network enforce: drop filterID=%d (unmatched block=%v ok=%v) remote=%s:%d",
			filterID, rec.Block, ok, remote, ev.Header.RemotePort,
		)
		return 0
	}

	info := DropInfo{
		RuleID:    rec.RuleID,
		DestType:  rec.DestType,
		DestValue: rec.DestValue,
		DestIP:    remote,
		DestPort:  int(ev.Header.RemotePort),
		Protocol:  ev.Header.IPProtocol,
	}
	log.Printf(
		"network enforce: drop filterID=%d rule=%s dest=%s/%s remote=%s:%d proto=%d",
		filterID, info.RuleID, info.DestType, info.DestValue,
		info.DestIP, info.DestPort, info.Protocol,
	)
	if fn != nil {
		fn(info) // non-blocking: app uses buffered channel + default drop
	}
	return 0
}

var netEventCallbackPtr = windows.NewCallback(netEventCallback)

func (e *windowsEnforcer) subscribeNetEventsLocked() error {
	if e.netEvents != 0 {
		return nil
	}
	ctx := &netEventContext{enf: e}
	e.netEventCtx = ctx

	sub := fwpmNetEventSubscription0{
		EnumTemplate: 0,
		Flags:        0,
		SessionKey:   e.sessionKey,
	}

	var h windows.Handle
	api := "FwpmNetEventSubscribe1"
	var r1 uintptr
	// Prefer Subscribe1 (EVENT2 / HEADER2) on Win8+; Subscribe0 alone mis-parses on modern OS
	// because HEADER1 has a large trailing reserved union we previously omitted.
	if err := procFwpmNetEventSubscribe1.Find(); err == nil {
		r1, _, _ = procFwpmNetEventSubscribe1.Call(
			uintptr(e.eng),
			uintptr(unsafe.Pointer(&sub)),
			netEventCallbackPtr,
			uintptr(unsafe.Pointer(ctx)),
			uintptr(unsafe.Pointer(&h)),
		)
	} else {
		r1 = 1 // force Subscribe0 path (e.g. Win7)
	}
	if r1 != 0 {
		api = "FwpmNetEventSubscribe0"
		r1, _, _ = procFwpmNetEventSubscribe0.Call(
			uintptr(e.eng),
			uintptr(unsafe.Pointer(&sub)),
			netEventCallbackPtr,
			uintptr(unsafe.Pointer(ctx)),
			uintptr(unsafe.Pointer(&h)),
		)
	}
	if r1 != 0 {
		e.netEventCtx = nil
		return fmt.Errorf("%s: 0x%X", api, r1)
	}
	e.netEvents = h
	netEventDiagLeft.Store(20)
	log.Printf("network enforce: net-events subscribed via %s", api)
	return nil
}

func (e *windowsEnforcer) unsubscribeNetEventsLocked() {
	if e.netEvents == 0 {
		e.netEventCtx = nil
		return
	}
	r1, _, _ := procFwpmNetEventUnsubscribe0.Call(
		uintptr(e.eng),
		uintptr(e.netEvents),
	)
	if r1 != 0 {
		log.Printf("network enforce: FwpmNetEventUnsubscribe0: 0x%X", r1)
	} else {
		log.Printf("network enforce: net-events unsubscribed")
	}
	e.netEvents = 0
	e.netEventCtx = nil
}

func remoteIPv4(h *fwpmNetEventHeader2) string {
	if h.IPVersion != fwpIPVersionV4 {
		return ""
	}
	// WFP stores IPv4 in the union as host-order UINT32 (first 4 bytes).
	addr := uint32(h.RemoteAddr[0]) | uint32(h.RemoteAddr[1])<<8 |
		uint32(h.RemoteAddr[2])<<16 | uint32(h.RemoteAddr[3])<<24
	return net.IPv4(byte(addr>>24), byte(addr>>16), byte(addr>>8), byte(addr)).String()
}
