package flow

import (
	"testing"

	"task142-atcconflict/internal/airspace"
	"task142-atcconflict/internal/model"
)

func TestOccupancy(t *testing.T) {
	air := airspace.New()
	air.AddSector(&model.Sector{ID: "S1", Code: "S1", FloorFL: 100, CeilingFL: 400, CapacityMAP: 2,
		Polygon: []model.LatLon{{Lat: 0, Lon: 0}, {Lat: 0, Lon: 10}, {Lat: 10, Lon: 10}, {Lat: 10, Lon: 0}}})
	air.AddSector(&model.Sector{ID: "S2", Code: "S2", FloorFL: 100, CeilingFL: 400, CapacityMAP: 5,
		Polygon: []model.LatLon{{Lat: 0, Lon: 10}, {Lat: 0, Lon: 20}, {Lat: 10, Lon: 20}, {Lat: 10, Lon: 10}}})
	pos := []model.Position{
		{PlanID: "P1", Lat: 5, Lon: 5, FL: 300},
		{PlanID: "P2", Lat: 6, Lon: 6, FL: 310},
		{PlanID: "P3", Lat: 7, Lon: 7, FL: 320},
		{PlanID: "P4", Lat: 5, Lon: 15, FL: 300},
	}
	got := Occupancy(air, pos)
	m := map[string]model.SectorOccupancy{}
	for _, o := range got {
		m[o.SectorID] = o
	}
	if m["S1"].Count != 3 {
		t.Errorf("S1 count = %d, want 3", m["S1"].Count)
	}
	if !m["S1"].Alert {
		t.Error("S1 should alert (3 > MAP 2)")
	}
	if m["S2"].Count != 1 {
		t.Errorf("S2 count = %d, want 1", m["S2"].Count)
	}
	if m["S2"].Alert {
		t.Error("S2 should not alert (1 <= MAP 5)")
	}
}

func TestMilesInTrailCheck(t *testing.T) {
	// Three flights approaching a fix at 120, 110, 100 nm.
	spacing := []SpacingEntry{
		{PlanID: "A", DistanceToFix: 120},
		{PlanID: "B", DistanceToFix: 110},
		{PlanID: "C", DistanceToFix: 100},
	}
	// Required 15nm: gaps are 10 and 10 -> both violate.
	v := MilesInTrailCheck(spacing, 15)
	if len(v) != 2 {
		t.Fatalf("want 2 violations, got %d", len(v))
	}
	if v[0].Leader != "A" || v[0].Trailer != "B" {
		t.Errorf("first violation = %s/%s, want A/B", v[0].Leader, v[0].Trailer)
	}
	// Required 5nm: gaps of 10 are fine.
	if v2 := MilesInTrailCheck(spacing, 5); len(v2) != 0 {
		t.Errorf("want 0 violations at 5nm, got %d", len(v2))
	}
	// Single flight -> no violations.
	if v2 := MilesInTrailCheck(spacing[:1], 15); len(v2) != 0 {
		t.Errorf("single flight should not violate, got %d", len(v2))
	}
}
