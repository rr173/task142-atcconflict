package airspace

import (
	"testing"
	"task142-atcconflict/internal/model"
)

func TestBug02_ReplacingAirwayWithdrawsFormerLegs(t *testing.T) {
	a := New()
	a.AddWaypoint(wp("A", "A", 0, 0)); a.AddWaypoint(wp("B", "B", 0, 1)); a.AddWaypoint(wp("C", "C", 0, 20))
	if err := a.AddAirway(&model.Airway{ID: "J1", Direction: model.AirwayForward, WaypointIDs: []string{"A", "B", "C"}}); err != nil { t.Fatal(err) }
	if err := a.AddAirway(&model.Airway{ID: "J1", Direction: model.AirwayForward, WaypointIDs: []string{"A", "B"}}); err != nil { t.Fatal(err) }
	if err := a.ValidateRoute([]string{"A", "B", "C"}); err == nil { t.Fatal("removed airway leg remained usable after replacement") }
}
