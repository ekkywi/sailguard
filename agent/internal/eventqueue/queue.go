package eventqueue

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ekkywi/sailguard/pkgs/agentcontract"
)

const fileName = "events-queue.json"

func Path(dir string) string {
	return filepath.Join(dir, fileName)
}

func Load(dir string) ([]agentcontract.Event, error) {
	b, err := os.ReadFile(Path(dir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read event queue: %w", err)
	}
	var events []agentcontract.Event
	if err := json.Unmarshal(b, &events); err != nil {
		return nil, fmt.Errorf("parse event queue: %w", err)
	}
	if events == nil {
		events = []agentcontract.Event{}
	}
	return events, nil
}

func Save(dir string, events []agentcontract.Event) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("mkdir data dir: %w", err)
	}
	if events == nil {
		events = []agentcontract.Event{}
	}
	b, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')

	path := Path(dir)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return fmt.Errorf("write temp event queue: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename event queue: %w", err)
	}
	return nil
}

// Enqueue appends events and drops the oldest entries when len exceeds max.
func Enqueue(dir string, events []agentcontract.Event, max int) error {
	if len(events) == 0 {
		return nil
	}
	if max <= 0 {
		max = 500
	}
	cur, err := Load(dir)
	if err != nil {
		return err
	}
	cur = append(cur, events...)
	if len(cur) > max {
		cur = cur[len(cur)-max:]
	}
	return Save(dir, cur)
}

// Peek returns up to limit events from the front of the queue (does not remove).
func Peek(dir string, limit int) ([]agentcontract.Event, error) {
	cur, err := Load(dir)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit >= len(cur) {
		out := make([]agentcontract.Event, len(cur))
		copy(out, cur)
		return out, nil
	}
	out := make([]agentcontract.Event, limit)
	copy(out, cur[:limit])
	return out, nil
}

// RemoveIDs drops queued events whose client_event_id is in ids.
func RemoveIDs(dir string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	drop := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id != "" {
			drop[id] = struct{}{}
		}
	}
	cur, err := Load(dir)
	if err != nil {
		return err
	}
	kept := make([]agentcontract.Event, 0, len(cur))
	for _, ev := range cur {
		if _, ok := drop[ev.ClientEventID]; ok {
			continue
		}
		kept = append(kept, ev)
	}
	return Save(dir, kept)
}
