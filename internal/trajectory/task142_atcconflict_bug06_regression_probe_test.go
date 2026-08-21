package trajectory

import "testing"

func TestBug06_PreDepartureRouteTimeStaysAtFirstFix(t *testing.T) {
	p, _, leg := PositionAtTime(PlanRoute{CruiseMach:.8, Legs:[]PlanLeg{{Position:LatLon{Lat:0,Lon:0}, FL:300, Mach:.8},{Position:LatLon{Lat:0,Lon:10}, FL:300, Mach:.8}}}, -30)
	if p.Lat != 0 || p.Lon != 0 || leg != 0 { t.Fatalf("pre-departure position = %+v leg %d, want first fix", p, leg) }
}
