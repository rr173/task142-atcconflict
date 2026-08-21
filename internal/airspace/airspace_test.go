package airspace

import (
	"testing"

	"task142-atcconflict/internal/model"
	"task142-atcconflict/internal/trajectory"
)

func wp(id, code string, lat, lon float64) *model.Waypoint {
	return &model.Waypoint{ID: id, Code: code, Position: model.LatLon{Lat: lat, Lon: lon}}
}

func sec(id string, floor, ceil, cap int, poly []model.LatLon) *model.Sector {
	return &model.Sector{ID: id, Code: id, FloorFL: floor, CeilingFL: ceil, CapacityMAP: cap, Polygon: poly}
}

func TestPointInPolygon(t *testing.T) {
	// A 1x1 degree square from (0,0) to (1,1).
	square := []model.LatLon{{Lat: 0, Lon: 0}, {Lat: 0, Lon: 1}, {Lat: 1, Lon: 1}, {Lat: 1, Lon: 0}}
	cases := []struct {
		p    trajectory.LatLon
		want bool
	}{
		{trajectory.LatLon{Lat: 0.5, Lon: 0.5}, true},
		{trajectory.LatLon{Lat: 0.9, Lon: 0.9}, true},
		{trajectory.LatLon{Lat: 1.5, Lon: 0.5}, false},
		{trajectory.LatLon{Lat: 0.5, Lon: -0.5}, false},
		{trajectory.LatLon{Lat: 0, Lon: 0}, true}, // corner vertex: ray-cast treats as inside (boundary inclusive)
	}
	for _, c := range cases {
		if got := PointInPolygon(square, c.p); got != c.want {
			t.Errorf("PointInPolygon(%+v) = %v, want %v", c.p, got, c.want)
		}
	}
	// Fewer than 3 vertices -> never inside.
	if PointInPolygon([]model.LatLon{{Lat: 0, Lon: 0}, {Lat: 1, Lon: 1}}, trajectory.LatLon{Lat: 0.5, Lon: 0.5}) {
		t.Error("2-vertex polygon should not contain anything")
	}
}

func TestInsideSector(t *testing.T) {
	s := sec("S1", 200, 400, 5, []model.LatLon{{Lat: 0, Lon: 0}, {Lat: 0, Lon: 10}, {Lat: 10, Lon: 10}, {Lat: 10, Lon: 0}})
	if !InsideSector(s, trajectory.LatLon{Lat: 5, Lon: 5}, 300) {
		t.Error("inside volume should be true")
	}
	if InsideSector(s, trajectory.LatLon{Lat: 5, Lon: 5}, 100) {
		t.Error("FL below floor should be false")
	}
	if InsideSector(s, trajectory.LatLon{Lat: 5, Lon: 5}, 500) {
		t.Error("FL above ceiling should be false")
	}
	if InsideSector(s, trajectory.LatLon{Lat: 50, Lon: 5}, 300) {
		t.Error("outside polygon should be false")
	}
}

func TestSectorForPosition(t *testing.T) {
	a := New()
	a.AddSector(sec("S1", 100, 400, 5, []model.LatLon{{Lat: 0, Lon: 0}, {Lat: 0, Lon: 10}, {Lat: 10, Lon: 10}, {Lat: 10, Lon: 0}}))
	a.AddSector(sec("S2", 100, 400, 5, []model.LatLon{{Lat: 0, Lon: 10}, {Lat: 0, Lon: 20}, {Lat: 10, Lon: 20}, {Lat: 10, Lon: 10}}))
	if id := a.SectorForPosition(trajectory.LatLon{Lat: 5, Lon: 5}, 300); id != "S1" {
		t.Errorf("S1 expected, got %q", id)
	}
	if id := a.SectorForPosition(trajectory.LatLon{Lat: 5, Lon: 15}, 300); id != "S2" {
		t.Errorf("S2 expected, got %q", id)
	}
	if id := a.SectorForPosition(trajectory.LatLon{Lat: 5, Lon: 50}, 300); id != "" {
		t.Errorf("no sector expected, got %q", id)
	}
}

// TestSectorForPositionOverlappingStable covers the responsibility-sector jitter
// bug: when adjacent sectors' boundary configurations briefly overlap, a point
// can fall inside more than one sector's volume. The assignment must be stable
// and consistent across recomputations — the same position must always resolve
// to the same sector regardless of Go's randomized map iteration order, and the
// pick must be deterministic (lexically smallest sector id wins).
func TestSectorForPositionOverlappingStable(t *testing.T) {
	// Two sectors whose polygons overlap in the [5,15]x[5,15] region. A point in
	// the overlap (10,10) is inside both volumes.
	a := New()
	a.AddSector(sec("B", 100, 400, 5, []model.LatLon{{Lat: 0, Lon: 0}, {Lat: 0, Lon: 15}, {Lat: 15, Lon: 15}, {Lat: 15, Lon: 0}}))
	a.AddSector(sec("A", 100, 400, 5, []model.LatLon{{Lat: 5, Lon: 5}, {Lat: 5, Lon: 20}, {Lat: 20, Lon: 20}, {Lat: 20, Lon: 5}}))
	p := trajectory.LatLon{Lat: 10, Lon: 10}
	const want = "A" // lexically smallest of the overlapping sectors
	first := a.SectorForPosition(p, 300)
	if first != want {
		t.Fatalf("overlap resolved to %q, want %q", first, want)
	}
	// The same point must resolve identically on every recomputation; map
	// iteration order is randomized, so this would jitter before the fix.
	for i := 0; i < 200; i++ {
		if got := a.SectorForPosition(p, 300); got != first {
			t.Fatalf("overlap assignment not stable: iteration %d gave %q, want %q", i, got, first)
		}
	}
}

func TestValidateRoute(t *testing.T) {
	a := New()
	a.AddWaypoint(wp("A", "AAA", 0, 0))
	a.AddWaypoint(wp("B", "BBB", 0, 1))
	a.AddWaypoint(wp("C", "CCC", 0, 2))
	a.AddWaypoint(wp("D", "DDD", 0, 50))
	// Airway A->B->C.
	aw := &model.Airway{ID: "AW1", Code: "AW1", Direction: model.AirwayBoth, WaypointIDs: []string{"A", "B", "C"}}
	if err := a.AddAirway(aw); err != nil {
		t.Fatal(err)
	}
	// Route A->B->C: covered by airway -> valid.
	if err := a.ValidateRoute([]string{"A", "B", "C"}); err != nil {
		t.Errorf("valid airway route rejected: %v", err)
	}
	// Route A->B->D: B->D is a direct leg of ~60nm*48 = 2880nm > MaxDirect -> invalid.
	if err := a.ValidateRoute([]string{"A", "B", "D"}); err == nil {
		t.Error("over-long direct leg should be rejected")
	}
	// Route A->C->B: C->B is reverse of airway, BOTH direction covers it -> valid.
	if err := a.ValidateRoute([]string{"A", "C", "B"}); err != nil {
		t.Errorf("reverse airway leg rejected under BOTH: %v", err)
	}
	// Forward-only airway: reverse leg should fail.
	a2 := New()
	a2.AddWaypoint(wp("A", "AAA", 0, 0))
	a2.AddWaypoint(wp("B", "BBB", 0, 1))
	awF := &model.Airway{ID: "AW2", Code: "AW2", Direction: model.AirwayForward, WaypointIDs: []string{"A", "B"}}
	a2.AddAirway(awF)
	// A->B valid, B->A is reverse of forward-only airway; distance 60nm < MaxDirect so still valid as direct!
	if err := a2.ValidateRoute([]string{"B", "A"}); err != nil {
		t.Errorf("B->A short direct should be valid, got %v", err)
	}
}

func TestAdjacent(t *testing.T) {
	a := New()
	a.AddSector(sec("S1", 100, 400, 5, []model.LatLon{{Lat: 0, Lon: 0}, {Lat: 0, Lon: 10}, {Lat: 10, Lon: 10}, {Lat: 10, Lon: 0}}))
	a.AddSector(sec("S2", 100, 400, 5, []model.LatLon{{Lat: 0, Lon: 10}, {Lat: 0, Lon: 20}, {Lat: 10, Lon: 20}, {Lat: 10, Lon: 10}}))
	a.AddSector(sec("S3", 100, 400, 5, []model.LatLon{{Lat: 0, Lon: 30}, {Lat: 0, Lon: 40}, {Lat: 10, Lon: 40}, {Lat: 10, Lon: 30}}))
	if !a.Adjacent("S1", "S2") {
		t.Error("S1 and S2 share edge -> adjacent")
	}
	if a.Adjacent("S1", "S3") {
		t.Error("S1 and S3 do not touch -> not adjacent")
	}
	if a.Adjacent("S1", "S1") {
		t.Error("sector is not adjacent to itself")
	}
}
