//go:build windows

package windows

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ekkywi/sailguard/agent/internal/host"
	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

// watchWMI blocks until ctx is cancelled, calling onCreate for each new Win32_Process.
// Uses WMI __InstanceCreationEvent (WITHIN 0.5 ≈ 0.5s). On setup failure, Watch falls back to PID-diff.
func (h *Host) watchWMI(ctx context.Context, onCreate func(host.Process)) error {
	if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
		return fmt.Errorf("CoInitializeEx: %w", err)
	}
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("WbemScripting.SWbemLocator")
	if err != nil {
		return fmt.Errorf("SWbemLocator: %w", err)
	}
	defer unknown.Release()

	locator, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return fmt.Errorf("locator QI: %w", err)
	}
	defer locator.Release()

	serviceVar, err := oleutil.CallMethod(locator, "ConnectServer", nil, `root\cimv2`)
	if err != nil {
		return fmt.Errorf("ConnectServer: %w", err)
	}
	service := serviceVar.ToIDispatch()
	defer serviceVar.Clear()
	if service != nil {
		defer service.Release()
	}
	if service == nil {
		return fmt.Errorf("ConnectServer: nil service")
	}

	const query = `SELECT * FROM __InstanceCreationEvent WITHIN 0.2 WHERE TargetInstance ISA 'Win32_Process'`
	eventsVar, err := oleutil.CallMethod(service, "ExecNotificationQuery", query)
	if err != nil {
		return fmt.Errorf("ExecNotificationQuery: %w", err)
	}
	events := eventsVar.ToIDispatch()
	defer eventsVar.Clear()
	if events != nil {
		defer events.Release()
	}
	if events == nil {
		return fmt.Errorf("ExecNotificationQuery: nil events")
	}

	log.Printf("windows watch: WMI process-creation subscribed")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		evVar, err := oleutil.CallMethod(events, "NextEvent", 1000)
		if err != nil {
			// Timeout / no event — keep waiting until ctx cancel.
			continue
		}

		func() {
			defer evVar.Clear()
			ev := evVar.ToIDispatch()
			if ev == nil {
				return
			}
			defer ev.Release()

			targetVar, err := oleutil.GetProperty(ev, "TargetInstance")
			if err != nil {
				return
			}
			defer targetVar.Clear()
			target := targetVar.ToIDispatch()
			if target == nil {
				return
			}
			defer target.Release()

			pidVar, err := oleutil.GetProperty(target, "ProcessId")
			if err != nil {
				return
			}
			defer pidVar.Clear()
			nameVar, err := oleutil.GetProperty(target, "Name")
			if err != nil {
				return
			}
			defer nameVar.Clear()

			pid, ok := variantUint32(pidVar)
			if !ok || pid == 0 {
				return
			}
			name := variantString(nameVar)
			if name == "" {
				return
			}
			onCreate(host.Process{
				PID:      pid,
				Name:     name,
				UserName: lookupProcessUser(pid),
			})
		}()

		time.Sleep(10 * time.Millisecond)
	}
}

func variantUint32(v *ole.VARIANT) (uint32, bool) {
	if v == nil {
		return 0, false
	}
	switch x := v.Value().(type) {
	case int:
		if x < 0 {
			return 0, false
		}
		return uint32(x), true
	case int8:
		if x < 0 {
			return 0, false
		}
		return uint32(x), true
	case int16:
		if x < 0 {
			return 0, false
		}
		return uint32(x), true
	case int32:
		if x < 0 {
			return 0, false
		}
		return uint32(x), true
	case int64:
		if x < 0 || x > int64(^uint32(0)) {
			return 0, false
		}
		return uint32(x), true
	case uint:
		return uint32(x), true
	case uint8:
		return uint32(x), true
	case uint16:
		return uint32(x), true
	case uint32:
		return x, true
	case uint64:
		if x > uint64(^uint32(0)) {
			return 0, false
		}
		return uint32(x), true
	default:
		return 0, false
	}
}

func variantString(v *ole.VARIANT) string {
	if v == nil {
		return ""
	}
	if s, ok := v.Value().(string); ok {
		return s
	}
	return fmt.Sprint(v.Value())
}
