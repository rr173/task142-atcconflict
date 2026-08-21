package service

import (
	"context"
	"fmt"
	"task142-atcconflict/internal/clock"
	"task142-atcconflict/internal/flight"
	"task142-atcconflict/internal/idlib"
	"task142-atcconflict/internal/model"
	"task142-atcconflict/internal/separation"
	"task142-atcconflict/internal/store"
	"task142-atcconflict/internal/trajectory"
	"time"
)

type Service struct {
	st    *store.Store
	clk   clock.Clock
	probe *separation.Prober
}

func New(st *store.Store, c clock.Clock) *Service {
	if c == nil {
		c = clock.Real{}
	}
	return &Service{st: st, clk: c, probe: separation.New()}
}
func (s *Service) now() string         { return s.clk.Now().UTC().Format(time.RFC3339) }
func (s *Service) Store() *store.Store { return s.st }
func (s *Service) RegisterAircraft(ctx context.Context, a *model.AircraftType) (*model.AircraftType, error) {
	if a == nil || a.Code == "" || a.MinMach <= 0 || a.MaxMach < a.MinMach {
		return nil, fmt.Errorf("invalid aircraft type")
	}
	a.ID = idlib.Prefixed("ac", a.ID)
	a.CreatedAt = s.now()
	return a, s.st.PutAircraft(ctx, a)
}
func (s *Service) File(ctx context.Context, p *model.FlightPlan) (*model.FlightPlan, error) {
	if p == nil || p.Callsign == "" {
		return nil, fmt.Errorf("invalid flight plan")
	}
	if s.st.CallsignExists(ctx, p.Callsign) {
		return nil, model.ErrCallsignTaken
	}
	a, err := s.st.Aircraft(ctx, p.AircraftTypeID)
	if err != nil {
		return nil, err
	}
	if err = flight.ValidateEnvelope(a, p.CruiseFL, p.CruiseMach); err != nil {
		return nil, err
	}
	p.ID = idlib.Prefixed("flt", p.ID)
	p.State = model.FlightFiled
	p.CreatedAt = s.now()
	p.UpdatedAt = p.CreatedAt
	return p, s.st.PutFlight(ctx, p)
}
func (s *Service) Flight(ctx context.Context, id string) (*model.FlightPlan, error) {
	return s.st.Flight(ctx, id)
}
func (s *Service) Flights(ctx context.Context) ([]model.FlightPlan, error) { return s.st.Flights(ctx) }
func (s *Service) Move(ctx context.Context, id string, to model.FlightState) (*model.FlightPlan, error) {
	p, err := s.st.Flight(ctx, id)
	if err != nil {
		return nil, err
	}
	if err = flight.Transition(p, to); err != nil {
		return nil, err
	}
	p.UpdatedAt = s.now()
	if err = s.st.PutFlight(ctx, p); err != nil {
		return nil, err
	}
	_ = s.st.Event(ctx, p.UpdatedAt, "flight."+string(to), p.ID, p)
	// A state change can take a flight out of the active set (e.g. terminated or
	// suspended); recompute the conflict picture so stale conflicts involving a
	// now-inactive flight are dropped immediately rather than lingering on the
	// controller display until the next track report.
	if err = s.refresh(ctx); err != nil {
		return nil, err
	}
	return p, nil
}
func (s *Service) Report(ctx context.Context, r *model.TrackReport) (*model.Position, error) {
	if r == nil || r.PlanID == "" || r.Ts == "" {
		return nil, fmt.Errorf("invalid track report")
	}
	p, err := s.st.Flight(ctx, r.PlanID)
	if err != nil {
		return nil, err
	}
	if p.State != model.FlightActive {
		return nil, model.ErrNotAirborne
	}
	if old, err := s.st.LatestTrack(ctx, r.PlanID); err == nil && old.Ts >= r.Ts {
		return nil, model.ErrTrackOutOfOrder
	}
	if err = s.st.AddTrack(ctx, r); err != nil {
		return nil, err
	}
	pos := &model.Position{PlanID: p.ID, Callsign: p.Callsign, Ts: r.Ts, Lat: r.Lat, Lon: r.Lon, FL: r.FL, Heading: r.Heading, Groundspeed: r.Groundspeed, VerticalRate: r.VerticalRate}
	if err = s.refresh(ctx); err != nil {
		return nil, err
	}
	return pos, nil
}
func (s *Service) refresh(ctx context.Context) error {
	plans, err := s.st.Flights(ctx)
	if err != nil {
		return err
	}
	var tracks []separation.TrackState
	for _, p := range plans {
		if p.State != model.FlightActive {
			continue
		}
		r, e := s.st.LatestTrack(ctx, p.ID)
		if e != nil {
			continue
		}
		tracks = append(tracks, separation.TrackState{PlanID: p.ID, Position: trajectory.LatLon{Lat: r.Lat, Lon: r.Lon}, FL: r.FL, Heading: r.Heading, Groundspeed: r.Groundspeed, VerticalRate: r.VerticalRate})
	}
	cs := s.probe.Detect(s.clk.Now(), tracks)
	for i := range cs {
		cs[i].ID = idlib.New("cfl")
	}
	return s.st.ReplaceConflicts(ctx, cs)
}
func (s *Service) Conflicts(ctx context.Context) ([]model.Conflict, error) {
	return s.st.Conflicts(ctx)
}
func (s *Service) InitiateHandoff(ctx context.Context, plan, from, to string) (*model.Handoff, error) {
	p, err := s.st.Flight(ctx, plan)
	if err != nil {
		return nil, err
	}
	if p.State != model.FlightActive {
		return nil, model.ErrNotAirborne
	}
	hs, _ := s.st.Handoffs(ctx)
	for _, h := range hs {
		if h.PlanID == plan && h.State == model.HandoffInitiated {
			return nil, model.ErrHandoffPending
		}
	}
	h, err := flight.Initiate(plan, from, to, s.now())
	if err != nil {
		return nil, err
	}
	h.ID = idlib.New("hof")
	return h, s.st.PutHandoff(ctx, h)
}
func (s *Service) AcceptHandoff(ctx context.Context, id string) (*model.Handoff, error) {
	h, err := s.st.Handoff(ctx, id)
	if err != nil {
		return nil, err
	}
	if err = flight.Accept(h, s.now()); err != nil {
		return nil, err
	}
	p, err := s.st.Flight(ctx, h.PlanID)
	if err != nil {
		return nil, err
	}
	p.ControllingSector = h.ToSector
	p.UpdatedAt = s.now()
	if err = s.st.PutFlight(ctx, p); err != nil {
		return nil, err
	}
	return h, s.st.PutHandoff(ctx, h)
}
