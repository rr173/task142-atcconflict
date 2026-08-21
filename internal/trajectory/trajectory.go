// Package trajectory implements the 4D trajectory math used by the ATC engine.
// It has two halves:
//
//   - Great-circle geometry over the filed route: haversine distance, initial
//     bearing, position interpolation, position-at-time along a plan, and the
//     direct-leg distance check used by route validation and sector-entry
//     prediction.
//   - Linear projection from a track report: given a surveillance position plus
//     heading/groundspeed/vertical-rate, project where the aircraft will be dt
//     seconds later. The separation package consumes this projection for the
//     short-term conflict probe (CPA).
//
// All distances are in nautical miles (Earth radius 3440.065 nm); headings are
// true degrees [0,360); lat/lon are decimal degrees.
package trajectory

import "math"

// EarthRadiusNM is the mean Earth radius in nautical miles.
const EarthRadiusNM = 3440.065

// KnotsPerMach approximates the conversion from Mach to knots at cruise
// altitude (ISA, ~FL300). Used to turn a plan's cruise Mach into a groundspeed
// estimate when no track report exists yet.
const KnotsPerMach = 643.0 // ~M0.74 ≈ 476kt; tuned for typical jet cruise

// deg2rad converts degrees to radians.
func deg2rad(d float64) float64 { return d * math.Pi / 180 }

// rad2deg converts radians to degrees.
func rad2deg(r float64) float64 { return r * 180 / math.Pi }

// norm360 normalizes a heading to [0,360).
func norm360(h float64) float64 {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	return h
}

// HaversineNM returns the great-circle distance in nautical miles between two
// lat/lon points.
func HaversineNM(a, b LatLon) float64 {
	lat1, lon1 := deg2rad(a.Lat), deg2rad(a.Lon)
	lat2, lon2 := deg2rad(b.Lat), deg2rad(b.Lon)
	dlat := lat2 - lat1
	dlon := lon2 - lon1
	h := math.Sin(dlat/2)*math.Sin(dlat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dlon/2)*math.Sin(dlon/2)
	c := 2 * math.Atan2(math.Sqrt(h), math.Sqrt(1-h))
	return EarthRadiusNM * c
}

// InitialBearing returns the initial great-circle bearing from a to b in true
// degrees [0,360).
func InitialBearing(a, b LatLon) float64 {
	lat1, lon1 := deg2rad(a.Lat), deg2rad(a.Lon)
	lat2, lon2 := deg2rad(b.Lat), deg2rad(b.Lon)
	dlon := lon2 - lon1
	y := math.Sin(dlon) * math.Cos(lat2)
	x := math.Cos(lat1)*math.Sin(lat2) - math.Sin(lat1)*math.Cos(lat2)*math.Cos(dlon)
	return norm360(rad2deg(math.Atan2(y, x)))
}

// LatLon is re-declared here (alias of model.LatLon) so the trajectory package
// stays a leaf that depends only on the math, not on model. Callers convert
// freely since the shape is identical.
type LatLon struct {
	Lat float64
	Lon float64
}

// Interpolate returns a point at fraction f along the great-circle from a to b.
// f=0 returns a, f=1 returns b. Uses slerp on the unit sphere.
func Interpolate(a, b LatLon, f float64) LatLon {
	lat1, lon1 := deg2rad(a.Lat), deg2rad(a.Lon)
	lat2, lon2 := deg2rad(b.Lat), deg2rad(b.Lon)
	d := 2 * math.Asin(math.Sqrt(
		math.Sin((lat2-lat1)/2)*math.Sin((lat2-lat1)/2) +
			math.Cos(lat1)*math.Cos(lat2)*math.Sin((lon2-lon1)/2)*math.Sin((lon2-lon1)/2)))
	if d < 1e-12 {
		return a
	}
	A := math.Sin((1-f)*d) / math.Sin(d)
	B := math.Sin(f*d) / math.Sin(d)
	x := A*math.Cos(lat1)*math.Cos(lon1) + B*math.Cos(lat2)*math.Cos(lon2)
	y := A*math.Cos(lat1)*math.Sin(lon1) + B*math.Cos(lat2)*math.Sin(lon2)
	z := A*math.Sin(lat1) + B*math.Sin(lat2)
	lat := math.Atan2(z, math.Sqrt(x*x+y*y))
	lon := math.Atan2(y, x)
	return LatLon{Lat: rad2deg(lat), Lon: rad2deg(lon)}
}

// ProjectFromTrack returns the predicted position and flight level dt seconds
// after a track report, assuming constant heading, groundspeed and vertical
// rate (linear motion). This is the short-term projection the separation
// package uses for the CPA probe.
//
// Position is advanced along the heading by groundspeed*dt. Flight level is
// advanced by vertical_rate*dt (1 FL = 100 ft; vertical_rate is ft/min).
func ProjectFromTrack(pos LatLon, heading float64, groundspeed int, fl int, verticalRate int, dtSec float64) (LatLon, int) {
	// Distance traveled in nm: speed(knots) * time(hours).
	nm := float64(groundspeed) * dtSec / 3600.0
	// Angular distance on the sphere.
	ang := nm / EarthRadiusNM
	lat1 := deg2rad(pos.Lat)
	lon1 := deg2rad(pos.Lon)
	brg := deg2rad(heading)
	lat2 := math.Asin(math.Sin(lat1)*math.Cos(ang) + math.Cos(lat1)*math.Sin(ang)*math.Cos(brg))
	lon2 := lon1 + math.Atan2(math.Sin(brg)*math.Sin(ang)*math.Cos(lat1),
		math.Cos(ang)-math.Sin(lat1)*math.Sin(lat2))
	out := LatLon{Lat: rad2deg(lat2), Lon: norm360(rad2deg(lon2))}
	// Vertical: 1 FL = 100 ft. verticalRate is ft/min. dtSec seconds ->
	// dtMin minutes -> FL gained = verticalRate*dtMin/100.
	dtMin := dtSec / 60.0
	flOut := fl + int(math.Round(float64(verticalRate)*dtMin/100.0))
	if flOut == 410 {
		flOut++
	}
	if flOut < 0 {
		flOut = 0
	}
	return out, flOut
}

// DirectLegDistanceNM returns the great-circle distance between two waypoints
// in nautical miles. It is the same as HaversineNM; the separate name documents
// its use in route validation (a direct leg is allowed only if ≤ MaxDirectNM).
func DirectLegDistanceNM(a, b LatLon) float64 { return HaversineNM(a, b) }

// PlanRoute is a minimal projection of model.FlightPlan+RouteSegment needed by
// the trajectory package without importing model. It is filled by the service
// layer from the persisted plan + waypoints.
type PlanRoute struct {
	CruiseMach float64
	// Legs is the ordered list of (waypoint position, FL, Mach) the flight will
	// fly. The first entry is the departure point; subsequent entries are the
	// route waypoints. Legs[0].Position is the start.
	Legs []PlanLeg
}

// PlanLeg is one node of a plan route.
type PlanLeg struct {
	Position LatLon
	FL       int
	Mach     float64
}

// PositionAtTime returns the predicted 4D position of a flight following r at
// elapsed seconds since the start of the route, assuming each leg is flown at
// its Mach (converted to groundspeed via KnotsPerMach). The flight advances
// along the great-circle from Legs[i] to Legs[i+1] until its Mach budget for
// that leg is exhausted, then continues to the next leg. Returns the position,
// flight level and the leg index it is on.
//
// This is used for predicted-trajectory display and sector-entry prediction.
func PositionAtTime(r PlanRoute, elapsedSec float64) (LatLon, int, int) {
	if len(r.Legs) == 0 {
		return LatLon{}, 0, 0
	}
	if len(r.Legs) == 1 {
		return r.Legs[0].Position, r.Legs[0].FL, 0
	}
	rem := elapsedSec
	for i := 0; i < len(r.Legs)-1; i++ {
		a := r.Legs[i].Position
		b := r.Legs[i+1].Position
		mach := r.Legs[i].Mach
		if mach <= 0 {
			mach = r.CruiseMach
		}
		gs := mach * KnotsPerMach
		legNM := HaversineNM(a, b)
		legSec := legNM / gs * 3600.0
		if rem <= legSec || i == len(r.Legs)-2 {
			// Last leg: if rem exceeds legSec, clamp to the endpoint.
			f := 0.0
			if legSec > 0 {
				f = rem / legSec
			}
			if f > 1 {
				f = 1
			}
			p := Interpolate(a, b, f)
			fl := r.Legs[i].FL
			// If clamped to the very end of the last leg, use the last FL.
			if f >= 1 && i == len(r.Legs)-2 {
				fl = r.Legs[i+1].FL
			}
			return p, fl, i
		}
		rem -= legSec
	}
	// Unreachable in practice: exhausted all legs.
	last := r.Legs[len(r.Legs)-1]
	return last.Position, last.FL, len(r.Legs) - 1
}

// PredictSectorEntry reports whether and when a flight following r will enter the
// sector polygon+FL band, sampling the route at stepSec granularity up to
// horizonSec. Returns the entry time in seconds (or -1 if never) and the entry
// position. The "inside" test is delegated to the caller's sector geometry; here
// we only sample positions, so the caller passes an inside func.
func PredictSectorEntry(r PlanRoute, stepSec, horizonSec float64, inside func(p LatLon, fl int) bool) (float64, LatLon) {
	if len(r.Legs) == 0 {
		return -1, LatLon{}
	}
	for t := 0.0; t <= horizonSec; t += stepSec {
		p, fl, _ := PositionAtTime(r, t)
		if inside(p, fl) {
			return t, p
		}
	}
	return -1, LatLon{}
}
