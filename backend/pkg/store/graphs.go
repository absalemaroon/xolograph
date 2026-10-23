package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// maxSnapshotBytes stays below the API's 1 MiB request cap.
const maxSnapshotBytes = 900 * 1024

// GraphSnapshot is a saved graph view. Data is opaque JSON owned by the UI: node positions, flags and the
// aggregates that were loaded at save time. It is a point-in-time record; restoring it does not re-query the chain.
type GraphSnapshot struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Seeds     []string        `json:"seeds"`
	NodeCount int             `json:"nodeCount"`
	EdgeCount int             `json:"edgeCount"`
	CreatedAt time.Time       `json:"createdAt"`
	Data      json.RawMessage `json:"data,omitempty"`
}

func (s *Store) SaveGraphSnapshot(ctx context.Context, name string, seeds []string, nodes, edges int, data json.RawMessage, actor string) (*GraphSnapshot, error) {
	name = strings.TrimSpace(name)
	switch {
	case name == "" || len(name) > 120:
		return nil, fmt.Errorf("%w: name must be 1-120 characters", ErrInvalid)
	case len(seeds) == 0 || len(seeds) > 20:
		return nil, fmt.Errorf("%w: a snapshot needs 1-20 seed addresses", ErrInvalid)
	case len(data) == 0 || len(data) > maxSnapshotBytes:
		return nil, fmt.Errorf("%w: snapshot data must be under %d KB", ErrInvalid, maxSnapshotBytes/1024)
	case !json.Valid(data):
		return nil, fmt.Errorf("%w: snapshot data is not valid JSON", ErrInvalid)
	case nodes < 0 || edges < 0:
		return nil, fmt.Errorf("%w: invalid counts", ErrInvalid)
	}
	sj, _ := json.Marshal(seeds)
	g := &GraphSnapshot{ID: newID("gs"), Name: name, Seeds: seeds, NodeCount: nodes, EdgeCount: edges, CreatedAt: now()}
	err := s.inTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO graph_snapshots(id,name,seeds,node_count,edge_count,data,created_at) VALUES(?,?,?,?,?,?,?)`,
			g.ID, g.Name, string(sj), nodes, edges, string(data), ts(g.CreatedAt)); err != nil {
			return err
		}
		return s.appendAudit(ctx, tx, actor, "GRAPH_SNAPSHOT_SAVED", "graph", g.ID, fmt.Sprintf("%s (%d nodes, %d edges)", name, nodes, edges))
	})
	if err != nil {
		return nil, err
	}
	return g, nil
}

func (s *Store) ListGraphSnapshots(ctx context.Context) ([]GraphSnapshot, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,name,seeds,node_count,edge_count,created_at FROM graph_snapshots ORDER BY created_at DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []GraphSnapshot{}
	for rows.Next() {
		var g GraphSnapshot
		var seeds, c string
		if err := rows.Scan(&g.ID, &g.Name, &seeds, &g.NodeCount, &g.EdgeCount, &c); err != nil {
			return nil, err
		}
		if json.Unmarshal([]byte(seeds), &g.Seeds) != nil {
			g.Seeds = []string{}
		}
		g.CreatedAt = parseTS(c)
		out = append(out, g)
	}
	return out, rows.Err()
}

func (s *Store) GetGraphSnapshot(ctx context.Context, id string) (*GraphSnapshot, error) {
	var g GraphSnapshot
	var seeds, c, data string
	err := s.DB.QueryRowContext(ctx, `SELECT id,name,seeds,node_count,edge_count,data,created_at FROM graph_snapshots WHERE id=?`, id).
		Scan(&g.ID, &g.Name, &seeds, &g.NodeCount, &g.EdgeCount, &data, &c)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if json.Unmarshal([]byte(seeds), &g.Seeds) != nil {
		g.Seeds = []string{}
	}
	g.CreatedAt, g.Data = parseTS(c), json.RawMessage(data)
	return &g, nil
}

func (s *Store) DeleteGraphSnapshot(ctx context.Context, id, actor string) error {
	return s.inTx(ctx, func(tx *sql.Tx) error {
		var name string
		if err := tx.QueryRowContext(ctx, `SELECT name FROM graph_snapshots WHERE id=?`, id).Scan(&name); err == sql.ErrNoRows {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM graph_snapshots WHERE id=?`, id); err != nil {
			return err
		}
		return s.appendAudit(ctx, tx, actor, "GRAPH_SNAPSHOT_DELETED", "graph", id, name)
	})
}
