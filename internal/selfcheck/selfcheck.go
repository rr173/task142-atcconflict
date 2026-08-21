package selfcheck

import (
	"fmt"
	"time"

	"task142-atcconflict/internal/airspace"
	"task142-atcconflict/internal/flow"
	"task142-atcconflict/internal/model"
	"task142-atcconflict/internal/separation"
	"task142-atcconflict/internal/trajectory"
)

func Run() error {
	air := airspace.New()
	air.AddSector(&model.Sector{ID: "E", Code: "E", FloorFL: 100, CeilingFL: 450, CapacityMAP: 1, Polygon: []model.LatLon{{Lat: -1, Lon: -1}, {Lat: -1, Lon: 1}, {Lat: 1, Lon: 1}, {Lat: 1, Lon: -1}}})
	occ := flow.Occupancy(air, []model.Position{{PlanID: "A", Lat: 0, Lon: 0, FL: 350}, {PlanID: "B", Lat: 0.2, Lon: 0.2, FL: 350}})
	if len(occ) != 1 || !occ[0].Alert {
		return fmt.Errorf("capacity monitor did not alert")
	}
	conflicts := separation.New().Detect(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), []separation.TrackState{
		{PlanID: "A", Position: trajectory.LatLon{Lat: 0, Lon: -0.05}, FL: 350, Heading: 90, Groundspeed: 480},
		{PlanID: "B", Position: trajectory.LatLon{Lat: 0, Lon: 0.05}, FL: 350, Heading: 270, Groundspeed: 480},
	})
	if len(conflicts) != 1 {
		return fmt.Errorf("conflict detector returned %d conflicts", len(conflicts))
	}
	return nil
}
