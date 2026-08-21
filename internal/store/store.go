package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	_ "modernc.org/sqlite"
	"task142-atcconflict/internal/model"
)

type Store struct {
	db *sql.DB
	mu sync.Mutex
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS flight_plans(id TEXT PRIMARY KEY,callsign TEXT UNIQUE,payload TEXT NOT NULL);
	CREATE TABLE IF NOT EXISTS aircraft_types(id TEXT PRIMARY KEY,payload TEXT NOT NULL);
	CREATE TABLE IF NOT EXISTS track_reports(id INTEGER PRIMARY KEY AUTOINCREMENT,plan_id TEXT NOT NULL,ts TEXT NOT NULL,payload TEXT NOT NULL,UNIQUE(plan_id,ts));
	CREATE TABLE IF NOT EXISTS conflicts(id TEXT PRIMARY KEY,payload TEXT NOT NULL);
	CREATE TABLE IF NOT EXISTS handoffs(id TEXT PRIMARY KEY,payload TEXT NOT NULL);
	CREATE TABLE IF NOT EXISTS events(id INTEGER PRIMARY KEY AUTOINCREMENT,ts TEXT NOT NULL,kind TEXT NOT NULL,entity_id TEXT NOT NULL,payload TEXT NOT NULL);`)
	return err
}

func put(ctx context.Context, db *sql.DB, table, id string, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, "INSERT INTO "+table+"(id,payload) VALUES(?,?) ON CONFLICT(id) DO UPDATE SET payload=excluded.payload", id, string(b))
	return err
}
func get(ctx context.Context, db *sql.DB, table, id string, value any) error {
	var p string
	err := db.QueryRowContext(ctx, "SELECT payload FROM "+table+" WHERE id=?", id).Scan(&p)
	if errors.Is(err, sql.ErrNoRows) {
		return model.ErrNotFound
	}
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(p), value)
}
func all(ctx context.Context, db *sql.DB, table string, decode func([]byte) error) error {
	rows, err := db.QueryContext(ctx, "SELECT payload FROM "+table+" ORDER BY id")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return err
		}
		if err := decode([]byte(p)); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (s *Store) PutAircraft(ctx context.Context, a *model.AircraftType) error {
	return put(ctx, s.db, "aircraft_types", a.ID, a)
}
func (s *Store) Aircraft(ctx context.Context, id string) (*model.AircraftType, error) {
	var a model.AircraftType
	err := get(ctx, s.db, "aircraft_types", id, &a)
	return &a, err
}
func (s *Store) PutFlight(ctx context.Context, p *model.FlightPlan) error {
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, "INSERT INTO flight_plans(id,callsign,payload) VALUES(?,?,?) ON CONFLICT(id) DO UPDATE SET callsign=excluded.callsign,payload=excluded.payload", p.ID, p.Callsign, string(b))
	return err
}
func (s *Store) Flight(ctx context.Context, id string) (*model.FlightPlan, error) {
	var p model.FlightPlan
	err := get(ctx, s.db, "flight_plans", id, &p)
	return &p, err
}
func (s *Store) Flights(ctx context.Context) ([]model.FlightPlan, error) {
	var out []model.FlightPlan
	err := all(ctx, s.db, "flight_plans", func(b []byte) error {
		var p model.FlightPlan
		if err := json.Unmarshal(b, &p); err != nil {
			return err
		}
		out = append(out, p)
		return nil
	})
	return out, err
}
func (s *Store) CallsignExists(ctx context.Context, c string) bool {
	var n int
	_ = s.db.QueryRowContext(ctx, "SELECT count(*) FROM flight_plans WHERE callsign=?", c).Scan(&n)
	return n > 0
}
func (s *Store) AddTrack(ctx context.Context, r *model.TrackReport) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, "INSERT INTO track_reports(plan_id,ts,payload) VALUES(?,?,?)", r.PlanID, r.Ts, string(b))
	return err
}
func (s *Store) LatestTrack(ctx context.Context, id string) (*model.TrackReport, error) {
	var p string
	err := s.db.QueryRowContext(ctx, "SELECT payload FROM track_reports WHERE plan_id=? ORDER BY ts DESC LIMIT 1", id).Scan(&p)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var r model.TrackReport
	err = json.Unmarshal([]byte(p), &r)
	return &r, err
}
func (s *Store) PutHandoff(ctx context.Context, h *model.Handoff) error {
	return put(ctx, s.db, "handoffs", h.ID, h)
}
func (s *Store) Handoff(ctx context.Context, id string) (*model.Handoff, error) {
	var h model.Handoff
	err := get(ctx, s.db, "handoffs", id, &h)
	return &h, err
}
func (s *Store) Handoffs(ctx context.Context) ([]model.Handoff, error) {
	var out []model.Handoff
	err := all(ctx, s.db, "handoffs", func(b []byte) error {
		var h model.Handoff
		if err := json.Unmarshal(b, &h); err != nil {
			return err
		}
		out = append(out, h)
		return nil
	})
	return out, err
}
func (s *Store) ReplaceConflicts(ctx context.Context, cs []model.Conflict) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM conflicts"); err == nil {
		for _, c := range cs {
			b, _ := json.Marshal(c)
			if _, err = tx.ExecContext(ctx, "INSERT INTO conflicts(id,payload) VALUES(?,?)", c.ID, string(b)); err != nil {
				break
			}
		}
	}
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
func (s *Store) Conflicts(ctx context.Context) ([]model.Conflict, error) {
	var out []model.Conflict
	err := all(ctx, s.db, "conflicts", func(b []byte) error {
		var c model.Conflict
		if err := json.Unmarshal(b, &c); err != nil {
			return err
		}
		out = append(out, c)
		return nil
	})
	return out, err
}
func (s *Store) Event(ctx context.Context, ts, kind, id string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, "INSERT INTO events(ts,kind,entity_id,payload) VALUES(?,?,?,?)", ts, kind, id, string(b))
	return err
}
func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }
func (s *Store) String() string                 { return fmt.Sprintf("atc-store:%p", s) }
