package facts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/shivamstaq/graph-harness/internal/kernel"
)

// CreateSnapshot writes a frozen view of the event log up to seq into
// kernel_snapshots. The strategy here is event_replay: the snapshot payload
// is the events themselves; restore replays them. Per-layer materialized
// snapshots are layered on top by individual Facts adapters.
func (e *EventLog) CreateSnapshot(ctx context.Context, layer string, seq uint64) (kernel.SnapshotHandle, error) {
	if seq == 0 {
		seq = e.LastSeq()
	}
	events, err := e.ReadAsOf(ctx, seq)
	if err != nil {
		return kernel.SnapshotHandle{}, err
	}
	// Filter to the named layer when caller specified one. Empty layer = all.
	if layer != "" {
		filtered := make([]kernel.Event, 0, len(events))
		for _, ev := range events {
			if ev.Layer == layer {
				filtered = append(filtered, ev)
			}
		}
		events = filtered
	}
	payload, err := json.Marshal(events)
	if err != nil {
		return kernel.SnapshotHandle{}, err
	}
	id := snapshotID(layer, seq, payload)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := e.db.ExecContext(ctx,
		`INSERT INTO kernel_snapshots (id, seq, layer, payload, created_at) VALUES (?, ?, ?, ?, ?)`,
		id, seq, layer, payload, now); err != nil {
		return kernel.SnapshotHandle{}, err
	}
	return kernel.SnapshotHandle{ID: id, Seq: seq, Layer: layer}, nil
}

// LoadSnapshot returns the events stored in a snapshot. Callers replay them
// to reconstruct layer state.
func (e *EventLog) LoadSnapshot(ctx context.Context, h kernel.SnapshotHandle) ([]kernel.Event, error) {
	row := e.db.QueryRowContext(ctx,
		`SELECT payload FROM kernel_snapshots WHERE id = ?`, h.ID)
	var payload []byte
	if err := row.Scan(&payload); err != nil {
		return nil, fmt.Errorf("snapshot %s: %w", h.ID, err)
	}
	var events []kernel.Event
	if err := json.Unmarshal(payload, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func snapshotID(layer string, seq uint64, payload []byte) string {
	h := sha256.New()
	_, _ = h.Write([]byte(layer))
	_, _ = h.Write([]byte{0})
	var seqBuf [8]byte
	for i := range 8 {
		seqBuf[i] = byte((seq >> (8 * i)) & 0xff)
	}
	_, _ = h.Write(seqBuf[:])
	_, _ = h.Write([]byte{0})
	_, _ = h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))[:16]
}
