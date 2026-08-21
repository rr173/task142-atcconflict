package trajectory

import (
	"math"
	"testing"
)

func approxEq(a, b, eps float64) bool {
	return math.Abs(a-b) < eps
}

func TestHaversineNM(t *testing.T) {
	// Same point -> 0.
	if d := HaversineNM(LatLon{0, 0}, LatLon{0, 0}); d > 1e-6 {
		t.Errorf("same point distance = %v, want ~0", d)
	}
	// Equator 1 degree of longitude ~ 60.04 nm.
	d := HaversineNM(LatLon{0, 0}, LatLon{0, 1})
	if !approxEq(d, 60.04, 0.5) {
		t.Errorf("1deg equator = %v, want ~60.04", d)
	}
	// LHR (51.47,-0.45) to JFK (40.64,-73.78) ~ 3009 nm (great circle).
	d = HaversineNM(LatLon{51.47, -0.45}, LatLon{40.64, -73.78})
	if !approxEq(d, 3009, 30) {
		t.Errorf("LHR-JFK = %v, want ~3009", d)
	}
}

func TestInitialBearing(t *testing.T) {
	// Due east along the equator -> 90.
	b := InitialBearing(LatLon{0, 0}, LatLon{0, 1})
	if !approxEq(b, 90, 1) {
		t.Errorf("east bearing = %v, want 90", b)
	}
	// Due north -> 0.
	b = InitialBearing(LatLon{0, 0}, LatLon{1, 0})
	if !approxEq(b, 0, 1) {
		t.Errorf("north bearing = %v, want 0", b)
	}
}

func TestInterpolate(t *testing.T) {
	// Halfway along the equator from lon 0 to lon 2 -> lon ~1.
	p := Interpolate(LatLon{0, 0}, LatLon{0, 2}, 0.5)
	if !approxEq(p.Lat, 0, 1e-6) || !approxEq(p.Lon, 1, 1e-3) {
		t.Errorf("midpoint = %+v, want lat~0 lon~1", p)
	}
	// f=0 returns start.
	p = Interpolate(LatLon{5, 10}, LatLon{6, 20}, 0)
	if !approxEq(p.Lat, 5, 1e-9) || !approxEq(p.Lon, 10, 1e-9) {
		t.Errorf("f=0 = %+v, want start", p)
	}
}

func TestProjectFromTrack(t *testing.T) {
	// Heading 090 (east), 360 kt, 1 hour -> ~360 nm east along equator.
	p, fl := ProjectFromTrack(LatLon{0, 0}, 90, 360, 350, 0, 3600)
	if fl != 350 {
		t.Errorf("FL changed = %d, want 350 (no vertical rate)", fl)
	}
	// ~360nm / 60.04 nm per degree ~ 5.99 deg east.
	if !approxEq(p.Lon, 6.0, 0.2) {
		t.Errorf("east projection lon = %v, want ~6.0", p.Lon)
	}
	if !approxEq(p.Lat, 0, 1e-6) {
		t.Errorf("lat = %v, want 0", p.Lat)
	}
	// Climbing at 1000 ft/min for 1 min -> +10 FL.
	_, fl2 := ProjectFromTrack(LatLon{0, 0}, 0, 0, 100, 1000, 60)
	if fl2 != 110 {
		t.Errorf("FL after climb = %d, want 110", fl2)
	}
}

func TestPositionAtTime(t *testing.T) {
	r := PlanRoute{
		CruiseMach: 0.74,
		Legs: []PlanLeg{
			{Position: LatLon{0, 0}, FL: 350, Mach: 0.74},
			{Position: LatLon{0, 1}, FL: 350, Mach: 0.74},
		},
	}
	// 1 deg ≈ 60.04nm; at M0.74*643≈475.8kt, time = 60.04/475.8*3600 ≈ 454.2s.
	// At half that time, lon ≈ 0.5.
	p, fl, leg := PositionAtTime(r, 227.0)
	if !approxEq(p.Lon, 0.5, 0.05) {
		t.Errorf("midpoint lon = %v, want ~0.5", p.Lon)
	}
	if fl != 350 {
		t.Errorf("FL = %d, want 350", fl)
	}
	if leg != 0 {
		t.Errorf("leg = %d, want 0", leg)
	}
	// After full leg time, clamp to endpoint.
	p, _, _ = PositionAtTime(r, 10000)
	if !approxEq(p.Lon, 1, 1e-6) {
		t.Errorf("clamp lon = %v, want 1", p.Lon)
	}
	// Before departure (negative elapsed): the flight has not taken off yet,
	// so it must stay parked at the first waypoint rather than being projected
	// backwards off the departure point.
	p, fl, leg = PositionAtTime(r, -600)
	if !approxEq(p.Lat, 0, 1e-9) || !approxEq(p.Lon, 0, 1e-9) {
		t.Errorf("pre-departure position = %+v, want origin (first waypoint)", p)
	}
	if fl != 350 {
		t.Errorf("pre-departure FL = %d, want 350", fl)
	}
	if leg != 0 {
		t.Errorf("pre-departure leg = %d, want 0", leg)
	}
	// Elapsed of exactly 0 also stays at the first waypoint.
	p, _, _ = PositionAtTime(r, 0)
	if !approxEq(p.Lon, 0, 1e-9) {
		t.Errorf("t=0 position = %+v, want origin", p)
	}
}

func TestDirectLegDistanceNM(t *testing.T) {
	if d := DirectLegDistanceNM(LatLon{0, 0}, LatLon{0, 1}); !approxEq(d, 60.04, 0.5) {
		t.Errorf("direct distance = %v, want ~60", d)
	}
}

func TestPredictSectorEntry(t *testing.T) {
	r := PlanRoute{CruiseMach: 0.74, Legs: []PlanLeg{
		{Position: LatLon{0, 0}, FL: 350, Mach: 0.74},
		{Position: LatLon{0, 10}, FL: 350, Mach: 0.74},
	}}
	// Sector "inside" = lon > 5. Entry happens after ~5/10 of the leg.
	entry, _ := PredictSectorEntry(r, 10, 10000, func(p LatLon, fl int) bool {
		return p.Lon > 5.0
	})
	if entry < 0 {
		t.Fatal("should enter")
	}
	// Sanity: entry at roughly half the leg time (~2271s).
	if !approxEq(entry, 2271, 30) {
		t.Errorf("entry time = %v, want ~2271", entry)
	}
	// Never enters a far sector.
	entry, _ = PredictSectorEntry(r, 10, 100, func(p LatLon, fl int) bool { return p.Lon > 9.5 })
	if entry >= 0 {
		t.Errorf("should not enter, got entry=%v", entry)
	}
}
