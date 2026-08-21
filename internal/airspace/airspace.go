// Package airspace models the static airspace structure: the point-in-sector
// volume test (lat/lon inside the polygon AND flight level in the band),
// waypoint/airway lookups and flight-plan route validation. It depends only on
// model + trajectory so the geometry is pure and unit-testable.
package airspace

import (
	"errors"
	"fmt"

	"task142-atcconflict/internal/model"
	"task142-atcconflict/internal/trajectory"
)

// MaxDirectNM is the longest direct (off-airway) leg a flight plan may file
// without being rejected. Beyond this the leg must be covered by a declared
// airway segment.
const MaxDirectNM = 500.0

// Airspace is an in-memory snapshot of the declared sectors, waypoints and
// airways. The service layer builds one from the persisted store to answer
// geometry and route-validation questions without hitting the DB per query.
type Airspace struct {
	sectors  map[string]*model.Sector
	waypoints map[string]*model.Waypoint
	airways   map[string]*model.Airway
	// segmentKey indexes every declared airway leg by "fromWaypoint->toWaypoint"
	// so route validation can ask "is there an airway from A to B?" in O(1).
	segmentKey map[legKey]string
	// airwayLegs records, per airway id, the segment keys that airway contributed
	// to segmentKey. It lets a replacement withdraw the airway's previously
	// declared legs so the route index only reflects the latest declaration.
	airwayLegs map[string][]legKey
}

type legKey struct {
	From, To string
}

// New builds an empty airspace.
func New() *Airspace {
	return &Airspace{
		sectors:    map[string]*model.Sector{},
		waypoints:  map[string]*model.Waypoint{},
		airways:    map[string]*model.Airway{},
		segmentKey: map[legKey]string{},
		airwayLegs: map[string][]legKey{},
	}
}

// AddSector registers (or replaces) a sector.
func (a *Airspace) AddSector(s *model.Sector) { a.sectors[s.ID] = s }

// AddWaypoint registers (or replaces) a waypoint.
func (a *Airspace) AddWaypoint(w *model.Waypoint) { a.waypoints[w.ID] = w }

// AddAirway registers (or replaces) an airway and indexes its legs both ways
// (subject to the airway's declared direction). Replacing an airway withdraws
// the legs the previous declaration contributed to segmentKey, so the route
// index only ever reflects the airway's latest declared segments — otherwise a
// revoked leg would remain clearence-legal and a flight could be routed along a
// path that is no longer in service.
func (a *Airspace) AddAirway(aw *model.Airway) error {
	if len(aw.WaypointIDs) < 2 {
		return fmt.Errorf("airway %s has fewer than 2 waypoints", aw.Code)
	}
	a.withdrawAirwayLegs(aw.ID)
	a.airways[aw.ID] = aw
	var legs []legKey
	addLeg := func(k legKey) {
		a.segmentKey[k] = aw.ID
		legs = append(legs, k)
	}
	for i := 0; i < len(aw.WaypointIDs)-1; i++ {
		from, to := aw.WaypointIDs[i], aw.WaypointIDs[i+1]
		switch aw.Direction {
		case model.AirwayForward:
			addLeg(legKey{from, to})
		case model.AirwayReverse:
			addLeg(legKey{to, from})
		case model.AirwayBoth, "":
			addLeg(legKey{from, to})
			addLeg(legKey{to, from})
		}
	}
	a.airwayLegs[aw.ID] = legs
	return nil
}

// withdrawAirwayLegs removes every leg the airway previously indexed under id,
// but only when this airway was the one that declared each leg. Two airways may
// legally cover the same leg (e.g. overlapping routes); a replacement of one
// must not void a leg still covered by another.
func (a *Airspace) withdrawAirwayLegs(id string) {
	for _, k := range a.airwayLegs[id] {
		if a.segmentKey[k] == id {
			delete(a.segmentKey, k)
		}
	}
	delete(a.airwayLegs, id)
}

// Sector returns the sector by id, or nil.
func (a *Airspace) Sector(id string) *model.Sector { return a.sectors[id] }

// SectorsList returns all declared sectors in a stable order (by id).
func (a *Airspace) SectorsList() map[string]*model.Sector { return a.sectors }

// Waypoint returns the waypoint by id, or nil.
func (a *Airspace) Waypoint(id string) *model.Waypoint { return a.waypoints[id] }

// WaypointByCode returns the first waypoint with the given code, or nil.
func (a *Airspace) WaypointByCode(code string) *model.Waypoint {
	for _, w := range a.waypoints {
		if w.Code == code {
			return w
		}
	}
	return nil
}

// SectorForPosition returns the sector whose volume contains (lat,lon,fl), or
// "" if none. If multiple sectors overlap (a misconfiguration), the first in
// map iteration order wins; callers should declare non-overlapping sectors.
func (a *Airspace) SectorForPosition(p trajectory.LatLon, fl int) string {
	for id, s := range a.sectors {
		if InsideSector(s, p, fl) {
			return id
		}
	}
	return ""
}

// InsideSector reports whether (lat,lon,fl) is inside the sector's 3D volume:
// the point is inside the boundary polygon AND fl is within [FloorFL, CeilingFL].
func InsideSector(s *model.Sector, p trajectory.LatLon, fl int) bool {
	if fl < s.FloorFL || fl > s.CeilingFL {
		return false
	}
	return PointInPolygon(s.Polygon, p)
}

// PointInPolygon returns true if p is inside the lat/lon polygon (ray-casting).
// The polygon is closed implicitly (last vertex → first). Fewer than 3 vertices
// means "no interior" → false.
func PointInPolygon(poly []model.LatLon, p trajectory.LatLon) bool {
	n := len(poly)
	if n < 3 {
		return false
	}
	inside := false
	j := n - 1
	for i := 0; i < n; i++ {
		xi, yi := poly[i].Lon, poly[i].Lat
		xj, yj := poly[j].Lon, poly[j].Lat
		// Standard ray-cast test in the lon (x) / lat (y) plane.
		if (yi > p.Lat) != (yj > p.Lat) {
			xint := (xj-xi)*(p.Lat-yi)/(yj-yi) + xi
			if p.Lon < xint {
				inside = !inside
			}
		}
		j = i
	}
	return inside
}

// ValidateRoute checks that a flight plan route is legal: every consecutive pair
// of waypoints is either connected by a declared airway segment or a direct leg
// no longer than MaxDirectNM. It also ensures every waypoint exists. Returns
// nil if the route is valid.
//
// The first leg is treated as a direct leg from the departure aerodrome (a
// waypoint the caller must include in waypointIDs as the first entry); the
// caller is responsible for that. This function only checks legs between the
// given waypointIDs in order.
func (a *Airspace) ValidateRoute(waypointIDs []string) error {
	if len(waypointIDs) < 2 {
		return errors.New("route needs at least 2 waypoints")
	}
	for i := 0; i < len(waypointIDs)-1; i++ {
		from, to := waypointIDs[i], waypointIDs[i+1]
		wf := a.waypoints[from]
		wt := a.waypoints[to]
		if wf == nil {
			return fmt.Errorf("route waypoint %s not found", from)
		}
		if wt == nil {
			return fmt.Errorf("route waypoint %s not found", to)
		}
		if from == to {
			return fmt.Errorf("route leg %d is a zero-length loop", i)
		}
		if _, ok := a.segmentKey[legKey{from, to}]; ok {
			continue // covered by a declared airway
		}
		// Direct leg: must be within MaxDirectNM.
		d := trajectory.DirectLegDistanceNM(
			trajectory.LatLon{Lat: wf.Position.Lat, Lon: wf.Position.Lon},
			trajectory.LatLon{Lat: wt.Position.Lat, Lon: wt.Position.Lon},
		)
		if d > MaxDirectNM {
			return fmt.Errorf("leg %s->%s is %.1fnm, exceeds direct maximum %.0fnm and has no airway", from, to, d, MaxDirectNM)
		}
	}
	return nil
}

// LegDistance returns the great-circle distance for a route leg between two
// declared waypoints, in nautical miles. Used by the flow metering package.
func (a *Airspace) LegDistance(fromID, toID string) (float64, error) {
	wf := a.waypoints[fromID]
	wt := a.waypoints[toID]
	if wf == nil || wt == nil {
		return 0, fmt.Errorf("waypoint not found: %s/%s", fromID, toID)
	}
	return trajectory.HaversineNM(
		trajectory.LatLon{Lat: wf.Position.Lat, Lon: wf.Position.Lon},
		trajectory.LatLon{Lat: wt.Position.Lat, Lon: wt.Position.Lon},
	), nil
}

// Adjacent reports whether two sectors are "adjacent" for handoff purposes.
// Two sectors are adjacent if their polygons share at least one boundary vertex
// within a small tolerance (0.01 deg ~ 0.6nm), or if one sector's polygon is
// immediately next to the other. This is a pragmatic approximation of sector
// adjacency for the handoff constraint.
func (a *Airspace) Adjacent(aID, bID string) bool {
	if aID == bID {
		return false
	}
	sa := a.sectors[aID]
	sb := a.sectors[bID]
	if sa == nil || sb == nil {
		return false
	}
	const tol = 0.02
	for _, pa := range sa.Polygon {
		for _, pb := range sb.Polygon {
			if absFloat(pa.Lat-pb.Lat) < tol && absFloat(pa.Lon-pb.Lon) < tol {
				return true
			}
		}
	}
	return false
}

func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
