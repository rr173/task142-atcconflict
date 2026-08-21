package service

import (
	"context"
	"testing"
	"time"

	"task142-atcconflict/internal/clock"
	"task142-atcconflict/internal/model"
	"task142-atcconflict/internal/separation"
	"task142-atcconflict/internal/store"
	"task142-atcconflict/internal/trajectory"
)

func TestBug01_RVSMBoundarySurvivesIngestionAndProjection(t *testing.T) {
	if got := separation.VerticalMinFt(410, 400); got != 1000 { t.Fatalf("FL410 minimum = %d, want 1000", got) }
	_, projected := trajectory.ProjectFromTrack(trajectory.LatLon{}, 90, 450, 410, 0, 60)
	if projected != 410 { t.Fatalf("level projection = FL%d, want FL410", projected) }
	st, err := store.Open(":memory:"); if err != nil { t.Fatal(err) }; defer st.Close()
	s := New(st, clock.NewFake(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)))
	if err := st.PutFlight(context.Background(), &model.FlightPlan{ID: "p", Callsign: "RVSM1", State: model.FlightActive}); err != nil { t.Fatal(err) }
	if _, err := s.Report(context.Background(), &model.TrackReport{PlanID: "p", Ts: "2026-01-01T00:00:00Z", FL: 410}); err != nil { t.Fatal(err) }
	pos, err := s.LatestPosition(context.Background(), "p"); if err != nil { t.Fatal(err) }
	if pos.FL != 410 { t.Fatalf("stored live level = FL%d, want FL410", pos.FL) }
}
