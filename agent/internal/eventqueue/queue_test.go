package eventqueue

import (
	"path/filepath"
	"testing"

	"github.com/ekkywi/sailguard/pkgs/agentcontract"
)

func TestEnqueuePeekRemove(t *testing.T) {
	dir := t.TempDir()
	ev1 := agentcontract.Event{ClientEventID: "a", EventType: agentcontract.EventViolationDetected}
	ev2 := agentcontract.Event{ClientEventID: "b", EventType: agentcontract.EventViolationBlocked}
	if err := Enqueue(dir, []agentcontract.Event{ev1, ev2}, 500); err != nil {
		t.Fatal(err)
	}
	got, err := Peek(dir, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ClientEventID != "a" {
		t.Fatalf("peek want [a], got %#v", got)
	}
	if err := RemoveIDs(dir, []string{"a"}); err != nil {
		t.Fatal(err)
	}
	rest, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(rest) != 1 || rest[0].ClientEventID != "b" {
		t.Fatalf("after remove want [b], got %#v", rest)
	}
	if Path(dir) != filepath.Join(dir, fileName) {
		t.Fatalf("unexpected path %s", Path(dir))
	}
}

func TestEnqueueCap(t *testing.T) {
	dir := t.TempDir()
	var batch []agentcontract.Event
	for i := 0; i < 5; i++ {
		batch = append(batch, agentcontract.Event{
			ClientEventID: string(rune('a' + i)),
			EventType:     agentcontract.EventViolationDetected,
		})
	}
	if err := Enqueue(dir, batch, 3); err != nil {
		t.Fatal(err)
	}
	all, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("want 3 after cap, got %d", len(all))
	}
	if all[0].ClientEventID != "c" || all[2].ClientEventID != "e" {
		t.Fatalf("want oldest dropped, got %#v", all)
	}
}
