package flow

import (
	"testing"

	"task142-atcconflict/internal/airspace"
	"task142-atcconflict/internal/model"
)

// TestOccupancyStableOrder asserts the regression fix: the same airspace snapshot
// yields the sector-load list in a stable, sorted-by-id order across many calls,
// regardless of Go's randomized map iteration. With the old SectorsList() that
// returned the backing map, this order could jump run-to-run.
func TestOccupancyStableOrder(t *testing.T) {
	air := airspace.New()
	// Register several sectors in a deliberately non-sorted id order; AddSector
	// goes into a map, so the only way the output is sorted is if SectorsList
	// imposes an order.
	poly := []model.LatLon{
		{Lat: -1, Lon: -1},
		{Lat: -1, Lon: 1},
		{Lat: 1, Lon: 1},
		{Lat: 1, Lon: -1},
	}
	air.AddSector(&model.Sector{ID: "SEC-D", FloorFL: 100, CeilingFL: 450, CapacityMAP: 5, Polygon: poly})
	air.AddSector(&model.Sector{ID: "SEC-A", FloorFL: 100, CeilingFL: 450, CapacityMAP: 5, Polygon: poly})
	air.AddSector(&model.Sector{ID: "SEC-C", FloorFL: 100, CeilingFL: 450, CapacityMAP: 5, Polygon: poly})
	air.AddSector(&model.Sector{ID: "SEC-B", FloorFL: 100, CeilingFL: 450, CapacityMAP: 5, Polygon: poly})

	positions := []model.Position{{PlanID: "P1", Lat: 0, Lon: 0, FL: 350}}

	want := []string{"SEC-A", "SEC-B", "SEC-C", "SEC-D"}
	// Run many times; each call must produce the identical sorted order.
	prev := []string{}
	for i := 0; i < 200; i++ {
		occ := Occupancy(air, positions)
		if len(occ) != len(want) {
			t.Fatalf("iter %d: got %d sectors, want %d", i, len(occ), len(want))
		}
		got := make([]string, len(occ))
		for j, o := range occ {
			got[j] = o.SectorID
		}
		if len(prev) > 0 && !equalSlices(prev, got) {
			t.Fatalf("iter %d: order changed between calls\nprev=%v\ngot =%v", i, prev, got)
		}
		for j, id := range want {
			if got[j] != id {
				t.Fatalf("iter %d: got order %v, want sorted %v", i, got, want)
			}
		}
		prev = got
	}
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
