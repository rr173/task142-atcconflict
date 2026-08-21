package service

import (
	"context"
	"task142-atcconflict/internal/flight"
	"task142-atcconflict/internal/model"
)

func (s *Service) Reconcile(ctx context.Context) (int, error) {
	hs, err := s.st.Handoffs(ctx)
	if err != nil {
		return 0, err
	}
	n := 0
	for i := range hs {
		if flight.CancelUnaccepted(&hs[i]) {
			if err := s.st.PutHandoff(ctx, &hs[i]); err != nil {
				return n, err
			}
			n++
		}
	}
	if err := s.refresh(ctx); err != nil {
		return n, err
	}
	return n, nil
}
func (s *Service) LatestPosition(ctx context.Context, id string) (*model.Position, error) {
	p, err := s.st.Flight(ctx, id)
	if err != nil {
		return nil, err
	}
	r, err := s.st.LatestTrack(ctx, id)
	if err != nil {
		return nil, err
	}
	return &model.Position{PlanID: id, Callsign: p.Callsign, Ts: r.Ts, Lat: r.Lat, Lon: r.Lon, FL: r.FL, Heading: r.Heading, Groundspeed: r.Groundspeed, VerticalRate: r.VerticalRate}, nil
}
