package flow

import (
	"testing"
	"task142-atcconflict/internal/airspace"
	"task142-atcconflict/internal/model"
)

func TestBug03_OccupancyOrderIsStable(t *testing.T) {
	a := airspace.New()
	a.AddSector(&model.Sector{ID:"Z", Polygon:[]model.LatLon{{},{Lon:1},{Lat:1,Lon:1}}, CapacityMAP:1})
	a.AddSector(&model.Sector{ID:"A", Polygon:[]model.LatLon{{},{Lon:1},{Lat:1,Lon:1}}, CapacityMAP:1})
	for i := 0; i < 100; i++ { got := Occupancy(a, nil); if got[0].SectorID != "A" || got[1].SectorID != "Z" { t.Fatalf("occupancy order = %q,%q, want A,Z", got[0].SectorID, got[1].SectorID) } }
}
