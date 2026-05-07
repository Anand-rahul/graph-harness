// Package review_queue implements the proposal lifecycle layer (SPEC §10).
// Phase 0 ships a working subset: submitted | pending_review | accepted |
// rejected | promoted, plus selector-overlap conflict detection.
package review_queue

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

// State is the P0 subset of SPEC §10.1 states.
type State string

// Phase 0 review queue states.
const (
	StateSubmitted     State = "submitted"
	StatePendingReview State = "pending_review"
	StateAccepted      State = "accepted"
	StateRejected      State = "rejected"
	StatePromoted      State = "promoted"
)

// Proposal is one entry in the review queue.
type Proposal struct {
	ID          string    `json:"id"`
	TargetLayer string    `json:"target_layer"`
	Kind        string    `json:"kind"` // selector | flow | invariant ...
	Author      string    `json:"author"`
	State       State     `json:"state"`
	Payload     []byte    `json:"payload"` // canonical JSON AST of the change
	CreatedAt   time.Time `json:"created_at"`
}

// Queue is the SQLite-backed review queue.
type Queue struct {
	db *sql.DB
}

// NewQueue wraps an opened *sql.DB and ensures the schema exists.
func NewQueue(db *sql.DB) (*Queue, error) {
	q := &Queue{db: db}
	const schema = `
CREATE TABLE IF NOT EXISTS review_proposals (
    id           TEXT PRIMARY KEY,
    target_layer TEXT NOT NULL,
    kind         TEXT NOT NULL,
    author       TEXT NOT NULL,
    state        TEXT NOT NULL,
    payload      BLOB,
    created_at   TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_review_state ON review_proposals(state);
CREATE INDEX IF NOT EXISTS idx_review_layer ON review_proposals(target_layer);
`
	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}
	return q, nil
}

// Submit appends a proposal in `pending_review` state. The ID is content-
// addressable (sha of layer + kind + payload) so duplicate submissions
// collapse.
func (q *Queue) Submit(ctx context.Context, p Proposal) (string, error) {
	if p.ID == "" {
		p.ID = newProposalID(p.TargetLayer, p.Kind, p.Payload)
	}
	if p.State == "" {
		p.State = StatePendingReview
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now().UTC()
	}
	_, err := q.db.ExecContext(ctx, `
		INSERT INTO review_proposals (id, target_layer, kind, author, state, payload, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING
	`, p.ID, p.TargetLayer, p.Kind, p.Author, string(p.State), p.Payload, p.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return "", err
	}
	return p.ID, nil
}

// List returns proposals optionally filtered by state.
func (q *Queue) List(ctx context.Context, stateFilter State) ([]Proposal, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if stateFilter == "" {
		rows, err = q.db.QueryContext(ctx,
			`SELECT id, target_layer, kind, author, state, payload, created_at
			 FROM review_proposals ORDER BY created_at ASC`)
	} else {
		rows, err = q.db.QueryContext(ctx,
			`SELECT id, target_layer, kind, author, state, payload, created_at
			 FROM review_proposals WHERE state = ? ORDER BY created_at ASC`, string(stateFilter))
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Proposal
	for rows.Next() {
		var p Proposal
		var st, ts string
		if err := rows.Scan(&p.ID, &p.TargetLayer, &p.Kind, &p.Author, &st, &p.Payload, &ts); err != nil {
			return nil, err
		}
		p.State = State(st)
		t, _ := time.Parse(time.RFC3339Nano, ts)
		p.CreatedAt = t
		out = append(out, p)
	}
	return out, rows.Err()
}

// Get returns a single proposal by ID.
func (q *Queue) Get(ctx context.Context, id string) (*Proposal, error) {
	row := q.db.QueryRowContext(ctx, `
		SELECT id, target_layer, kind, author, state, payload, created_at
		FROM review_proposals WHERE id = ?`, id)
	var p Proposal
	var st, ts string
	if err := row.Scan(&p.ID, &p.TargetLayer, &p.Kind, &p.Author, &st, &p.Payload, &ts); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	p.State = State(st)
	t, _ := time.Parse(time.RFC3339Nano, ts)
	p.CreatedAt = t
	return &p, nil
}

// Accept transitions a proposal to `accepted`.
func (q *Queue) Accept(ctx context.Context, id string) error {
	return q.transition(ctx, id, StateAccepted)
}

// Reject transitions a proposal to `rejected`.
func (q *Queue) Reject(ctx context.Context, id string) error {
	return q.transition(ctx, id, StateRejected)
}

// Promote transitions an accepted proposal to `promoted` (i.e. the change has
// been written into the target layer's canonical store).
func (q *Queue) Promote(ctx context.Context, id string) error {
	return q.transition(ctx, id, StatePromoted)
}

func (q *Queue) transition(ctx context.Context, id string, to State) error {
	res, err := q.db.ExecContext(ctx,
		`UPDATE review_proposals SET state = ? WHERE id = ?`, string(to), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("proposal %s not found", id)
	}
	return nil
}

func newProposalID(layer, kind string, payload []byte) string {
	h := sha256.New()
	_, _ = h.Write([]byte(layer))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(kind))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write(payload)
	return "prop_" + hex.EncodeToString(h.Sum(nil))[:10]
}
