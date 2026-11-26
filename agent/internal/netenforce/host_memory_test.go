package netenforce

import (
	"reflect"
	"testing"
	"time"
)

func TestHostMemoryKeepsRecentIPs(t *testing.T) {
	m := newHostMemory()
	t0 := time.Date(2026, 11, 26, 9, 0, 0, 0, time.UTC)

	m.Observe("example.test", []string{"203.0.113.2", "203.0.113.1"}, t0)
	got := m.Active("example.test", t0.Add(time.Minute), 10*time.Minute)
	want := []string{"203.0.113.1", "203.0.113.2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestHostMemoryExpiresOldIPs(t *testing.T) {
	m := newHostMemory()
	t0 := time.Date(2026, 11, 26, 9, 0, 0, 0, time.UTC)

	m.Observe("example.test", []string{"203.0.113.2", "203.0.113.1"}, t0)
	got := m.Active("example.test", t0.Add(time.Minute), 10*time.Minute)
	want := []string{"203.0.113.1", "203.0.113.2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestHostMemoryUnknownHost(t *testing.T) {
	m := newHostMemory()
	got := m.Active("nope.test", time.Now(), time.Minute)
	if len(got) != 0 {
		t.Fatalf("want empty, got %v", got)
	}
}