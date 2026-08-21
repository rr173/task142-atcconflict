// Package flow monitors sector capacity and meters traffic. It computes, for a
// set of active flights' current positions, which flights are inside each
// sector's volume (delegating the geometry to airspace) and whether the count
// exceeds the sector's Monitor Alert Parameter (MAP). It also evaluates
// miles-in-trail (MIT) spacing on a fix/airway.
//
// The package is stateless: the service layer passes the current snapshot of
// active positions + the airspace, and flow returns occupancy + alerts. This
// keeps the capacity logic pure and unit-testable.
package flow

import (
	"sort"

	"task142-atcconflict/internal/airspace"
	"task142-atcconflict/internal/model"
	"task142-atcconflict/internal/trajectory"
)

// Occupancy computes the per-sector load from a set of active flight positions.
// It returns one SectorOccupancy per declared sector, ordered by sector id.
// A flight is counted in a sector when its position is inside the sector volume.
func Occupancy(air *airspace.Airspace, positions []model.Position) []model.SectorOccupancy {
	out := make([]model.SectorOccupancy, 0, len(air.SectorsList()))
	// Collect per-sector plan ids.
	type acc struct {
		plans []string
	}
	bySector := map[string]*acc{}
	for _, p := range positions {
		ll := trajectory.LatLon{Lat: p.Lat, Lon: p.Lon}
		sid := air.SectorForPosition(ll, p.FL)
		if sid == "" {
			continue
		}
		if bySector[sid] == nil {
			bySector[sid] = &acc{}
		}
		bySector[sid].plans = append(bySector[sid].plans, p.PlanID)
	}
	for sid, s := range air.SectorsList() {
		a := bySector[sid]
		count := 0
		var plans []string
		if a != nil {
			count = len(a.plans)
			plans = a.plans
		}
		if plans == nil {
			plans = []string{}
		}
		out = append(out, model.SectorOccupancy{
			SectorID: sid,
			Count:    count,
			Capacity: s.CapacityMAP,
			Alert:    count > s.CapacityMAP,
			Plans:    plans,
		})
	}
	return out
}

// MilesInTrailCheck evaluates whether a sequence of flights on the same fix
// respects the miles-in-trail spacing. It returns the violating pairs (lead,
// trail) where the gap is below required nm. The caller provides each flight's
// distance to the fix (via airspace.LegDistance or HaversineNM).
//
// Radar snapshots are occasionally delivered out of order, so the entries are
// sorted here by distance-to-fix descending (farthest first = leader) before the
// adjacent pairs are formed. This guarantees the real leader/trailer pairing
// regardless of the order the snapshot arrived in, and avoids mismatching a
// leader with the wrong trailer (which would either swallow a true spacing
// breach or emit a false one). The caller's slice is not mutated.
func MilesInTrailCheck(spacing []SpacingEntry, requiredNM int) []Violation {
	if len(spacing) < 2 || requiredNM <= 0 {
		return nil
	}
	// Sort by distance-to-fix descending so adjacent entries are the true
	// leader/trailer pairs. Work on a copy so the caller's slice is untouched.
	ordered := make([]SpacingEntry, len(spacing))
	copy(ordered, spacing)
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].DistanceToFix > ordered[j].DistanceToFix
	})
	var out []Violation
	for i := 0; i < len(ordered)-1; i++ {
		lead := ordered[i]
		trail := ordered[i+1]
		// lead is farther; trail is closer. Gap = lead.Dist - trail.Dist.
		gap := lead.DistanceToFix - trail.DistanceToFix
		if gap < float64(requiredNM) {
			out = append(out, Violation{
				Leader: lead.PlanID, Trailer: trail.PlanID,
				GapNM: gap, RequiredNM: requiredNM,
			})
		}
	}
	return out
}

// SpacingEntry is one flight's distance to a common fix, used for MIT metering.
type SpacingEntry struct {
	PlanID         string
	DistanceToFix  float64 // nautical miles
}

// Violation is a miles-in-trail spacing breach.
type Violation struct {
	Leader     string  `json:"leader"`
	Trailer    string  `json:"trailer"`
	GapNM      float64 `json:"gap_nm"`
	RequiredNM int     `json:"required_nm"`
}
