package airspace

import (
	"testing"
	"task142-atcconflict/internal/model"
	"task142-atcconflict/internal/trajectory"
)

func TestBug07_OverlapSelectionIsDeterministic(t *testing.T) {
	a := New(); poly := []model.LatLon{{},{Lon:1},{Lat:1,Lon:1},{Lat:1}}
	a.AddSector(&model.Sector{ID:"B", FloorFL:100, CeilingFL:400, Polygon:poly}); a.AddSector(&model.Sector{ID:"A", FloorFL:100, CeilingFL:400, Polygon:poly})
	for i:=0;i<100;i++ { if got:=a.SectorForPosition(trajectory.LatLon{Lat:.5,Lon:.5},300); got!="A" { t.Fatalf("overlap owner = %q, want deterministic A",got) } }
}
