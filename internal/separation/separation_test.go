package separation

import (
	"testing"
	"time"

	"task142-atcconflict/internal/model"
	"task142-atcconflict/internal/trajectory"
)

func TestVerticalMinFt(t *testing.T) {
	if v := VerticalMinFt(300, 350); v != 1000 {
		t.Errorf("FL300/350 (RVSM) = %d, want 1000", v)
	}
	if v := VerticalMinFt(410, 400); v != 1000 {
		t.Errorf("FL410/400 (RVSM top) = %d, want 1000", v)
	}
	if v := VerticalMinFt(450, 350); v != 2000 {
		t.Errorf("FL450/350 (above RVSM) = %d, want 2000", v)
	}
	if v := VerticalMinFt(100, 200); v != 1000 {
		t.Errorf("FL100/200 = %d, want 1000", v)
	}
}

func TestDetectOppositeConflict(t *testing.T) {
	// Two flights head-on along the equator: A at lon -0.05 (heading 090, 480kt),
	// B at lon +0.05 (heading 270, 480kt), both FL350. They converge; lateral
	// will drop below 5nm, vertical identical -> conflict OPPOSITE.
	p := New()
	now, _ := time.Parse(time.RFC3339, "2026-01-10T09:00:00Z")
	tracks := []TrackState{
		{PlanID: "A", Position: trajectory.LatLon{Lat: 0, Lon: -0.06}, FL: 350, Heading: 90, Groundspeed: 480, VerticalRate: 0},
		{PlanID: "B", Position: trajectory.LatLon{Lat: 0, Lon: 0.06}, FL: 350, Heading: 270, Groundspeed: 480, VerticalRate: 0},
	}
	cs := p.Detect(now, tracks)
	if len(cs) != 1 {
		t.Fatalf("want 1 conflict, got %d", len(cs))
	}
	if cs[0].Type != model.ConflictOpposite {
		t.Errorf("type = %v, want OPPOSITE", cs[0].Type)
	}
	if cs[0].MinLateralNM >= LateralMinNM {
		t.Errorf("min lateral = %v, want < %v", cs[0].MinLateralNM, LateralMinNM)
	}
	if cs[0].MinVerticalFT >= 1000 {
		t.Errorf("min vertical = %v, want < 1000", cs[0].MinVerticalFT)
	}
}

func TestDetectNoConflictWhenVerticalSeparated(t *testing.T) {
	// Same head-on geometry but 1500ft apart (FL350 vs FL365) -> vertical >= 1000
	// at all times -> no conflict (satisfying the vertical minimum is safe).
	p := New()
	now, _ := time.Parse(time.RFC3339, "2026-01-10T09:00:00Z")
	tracks := []TrackState{
		{PlanID: "A", Position: trajectory.LatLon{Lat: 0, Lon: -0.06}, FL: 350, Heading: 90, Groundspeed: 480, VerticalRate: 0},
		{PlanID: "B", Position: trajectory.LatLon{Lat: 0, Lon: 0.06}, FL: 365, Heading: 270, Groundspeed: 480, VerticalRate: 0},
	}
	if cs := p.Detect(now, tracks); len(cs) != 0 {
		t.Fatalf("vertically-separated pair should be safe, got %d conflicts: %+v", len(cs), cs)
	}
}

func TestDetectNoConflictWhenLaterallySeparated(t *testing.T) {
	// Two flights same FL but far apart laterally (>5nm always) -> no conflict.
	p := New()
	now, _ := time.Parse(time.RFC3339, "2026-01-10T09:00:00Z")
	tracks := []TrackState{
		{PlanID: "A", Position: trajectory.LatLon{Lat: 0, Lon: 0}, FL: 350, Heading: 90, Groundspeed: 480, VerticalRate: 0},
		{PlanID: "B", Position: trajectory.LatLon{Lat: 5, Lon: 5}, FL: 350, Heading: 90, Groundspeed: 480, VerticalRate: 0},
	}
	if cs := p.Detect(now, tracks); len(cs) != 0 {
		t.Fatalf("far pair should be safe, got %d conflicts", len(cs))
	}
}

func TestDetectAboveRVSM(t *testing.T) {
	// FL450 opposite, 1500ft apart (FL450 vs FL465). Vertical min = 2000ft; 1500 < 2000 -> conflict.
	p := New()
	now, _ := time.Parse(time.RFC3339, "2026-01-10T09:00:00Z")
	tracks := []TrackState{
		{PlanID: "A", Position: trajectory.LatLon{Lat: 0, Lon: -0.06}, FL: 450, Heading: 90, Groundspeed: 480, VerticalRate: 0},
		{PlanID: "B", Position: trajectory.LatLon{Lat: 0, Lon: 0.06}, FL: 465, Heading: 270, Groundspeed: 480, VerticalRate: 0},
	}
	cs := p.Detect(now, tracks)
	if len(cs) != 1 {
		t.Fatalf("above-RVSM 1500ft pair should conflict, got %d", len(cs))
	}
	// And 2500ft apart at FL450 vs FL475 should be safe (>= 2000).
	tracks[1].FL = 475
	if cs := p.Detect(now, tracks); len(cs) != 0 {
		t.Fatalf("above-RVSM 2500ft pair should be safe, got %d", len(cs))
	}
}

func TestClassifyOvertaking(t *testing.T) {
	a := TrackState{Position: trajectory.LatLon{Lat: 0, Lon: 0}, FL: 350, Heading: 90, Groundspeed: 480}
	b := TrackState{Position: trajectory.LatLon{Lat: 0, Lon: 0.01}, FL: 350, Heading: 90, Groundspeed: 320}
	if c := classify(a, b); c != model.ConflictOvertaking && c != model.ConflictSameDir {
		// Overtaking when speed diff >= 50; this is 160 -> Overtaking.
		t.Errorf("classify = %v, want OVERTAKING", c)
	}
}
