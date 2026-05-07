package code_core

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Store is the SQLite-backed code.core adapter. It owns the entity table and
// the adjacency table that backs `calls` / `references` traversal (SPEC §6.14
// — adjacency in SQLite + planner BFS, no property-graph DB).
type Store struct {
	db *sql.DB
}

// NewStore wraps an opened *sql.DB and ensures the schema exists.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.initSchema(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) initSchema() error {
	const schema = `
CREATE TABLE IF NOT EXISTS code_entities (
    id             TEXT PRIMARY KEY,
    kind           TEXT NOT NULL,
    language_id    TEXT NOT NULL DEFAULT '',
    qualified_name TEXT,
    receiver       TEXT,
    path           TEXT,
    body_hash      TEXT,
    created_seq    INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_code_entities_qn ON code_entities(qualified_name);
CREATE INDEX IF NOT EXISTS idx_code_entities_kind ON code_entities(kind);

CREATE TABLE IF NOT EXISTS code_relations (
    relation TEXT NOT NULL,         -- "calls" | "references"
    from_id  TEXT NOT NULL,
    to_id    TEXT NOT NULL,
    PRIMARY KEY (relation, from_id, to_id)
);
CREATE INDEX IF NOT EXISTS idx_relations_from ON code_relations(relation, from_id);
CREATE INDEX IF NOT EXISTS idx_relations_to   ON code_relations(relation, to_id);
`
	_, err := s.db.Exec(schema)
	return err
}

// PutEntity inserts or replaces an entity. Idempotent at the same content
// (same id) — entity identity is content-addressable.
func (s *Store) PutEntity(ctx context.Context, e Entity, createdSeq uint64) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO code_entities (id, kind, language_id, qualified_name, receiver, path, body_hash, created_seq)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
		    kind = excluded.kind,
		    language_id = excluded.language_id,
		    qualified_name = excluded.qualified_name,
		    receiver = excluded.receiver,
		    path = excluded.path,
		    body_hash = excluded.body_hash
	`, e.ID, string(e.Kind), e.LanguageID, e.QualifiedName, e.Receiver, e.Path, e.BodyHash, createdSeq)
	return err
}

// AddRelation inserts a typed edge (idempotent on dup).
func (s *Store) AddRelation(ctx context.Context, relation, fromID, toID string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO code_relations (relation, from_id, to_id) VALUES (?, ?, ?)
	`, relation, fromID, toID)
	return err
}

// LookupByQualifiedNameSuffix returns the first entity whose qualified_name
// ends with `.<suffix>` or equals `<suffix>`. Used by change.process when
// resolving touched function names to fully-qualified code.core entities.
func (s *Store) LookupByQualifiedNameSuffix(ctx context.Context, suffix string) (*Entity, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, kind, language_id, qualified_name, receiver, path, body_hash
		 FROM code_entities
		 WHERE qualified_name = ? OR qualified_name LIKE '%.' || ?
		 LIMIT 1`, suffix, suffix)
	var e Entity
	var kind string
	var receiver, path, bodyHash sql.NullString
	if err := row.Scan(&e.ID, &kind, &e.LanguageID, &e.QualifiedName, &receiver, &path, &bodyHash); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	e.Kind = EntityKind(kind)
	e.Receiver = receiver.String
	e.Path = path.String
	e.BodyHash = bodyHash.String
	return &e, nil
}

// LookupByQualifiedName returns the first entity matching qn (case-sensitive).
// Used by selector resolution.
func (s *Store) LookupByQualifiedName(ctx context.Context, qn string) (*Entity, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, kind, language_id, qualified_name, receiver, path, body_hash
		 FROM code_entities WHERE qualified_name = ? LIMIT 1`, qn)
	var e Entity
	var kind string
	var receiver, path, bodyHash sql.NullString
	if err := row.Scan(&e.ID, &kind, &e.LanguageID, &e.QualifiedName, &receiver, &path, &bodyHash); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	e.Kind = EntityKind(kind)
	e.Receiver = receiver.String
	e.Path = path.String
	e.BodyHash = bodyHash.String
	return &e, nil
}

// CountByKind returns how many entities of a given kind are stored.
func (s *Store) CountByKind(ctx context.Context, kind EntityKind) (int, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM code_entities WHERE kind = ?`, string(kind))
	var n int
	if err := row.Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// BFS performs a bounded breadth-first traversal of the named relation,
// starting at startID, up to maxDepth hops. Used by change.process stage 5
// to compute the impacted set (P0.T31). Naive in P0; Mangle-driven in P3.
func (s *Store) BFS(ctx context.Context, relation, startID string, maxDepth int) ([]string, error) {
	visited := map[string]struct{}{startID: {}}
	frontier := []string{startID}
	for d := 0; d < maxDepth && len(frontier) > 0; d++ {
		args := make([]any, 0, len(frontier)+1)
		args = append(args, relation)
		placeholders := make([]string, len(frontier))
		for i, id := range frontier {
			placeholders[i] = "?"
			args = append(args, id)
		}
		query := fmt.Sprintf(
			`SELECT to_id FROM code_relations WHERE relation = ? AND from_id IN (%s)`,
			joinComma(placeholders))
		// #nosec G201,G202 -- placeholders are constant "?", IDs are bound
		// via args. The relation parameter is also bound. No string concat
		// of user data into SQL.
		next, err := s.bfsHopOnce(ctx, query, args)
		if err != nil {
			return nil, err
		}
		fresh := next[:0]
		for _, to := range next {
			if _, seen := visited[to]; !seen {
				visited[to] = struct{}{}
				fresh = append(fresh, to)
			}
		}
		frontier = fresh
	}
	delete(visited, startID)
	out := make([]string, 0, len(visited))
	for id := range visited {
		out = append(out, id)
	}
	return out, nil
}

func joinComma(s []string) string {
	return strings.Join(s, ",")
}

func (s *Store) bfsHopOnce(ctx context.Context, query string, args []any) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var to string
		if err := rows.Scan(&to); err != nil {
			return nil, err
		}
		out = append(out, to)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
