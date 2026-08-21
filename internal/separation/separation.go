// Package separation implements the loss-of-separation (conflict) detection
// used by the ATC engine.
//
// Conflict model. Each ACTIVE flight has a current track report giving position,
// flight level, true heading, groundspeed and vertical rate. Over a short
// lookahead window (default 15 minutes) each flight is projected forward under
// the constant-velocity assumption (see trajectory.ProjectFromTrack). At each
// sample step (default 12 seconds) the pairwise lateral distance and vertical
// separation are computed. A conflict exists iff, at some sampled instant,
// BOTH minima are breached simultaneously:
//
//	lateral < LateralMinNM (5nm)  AND  vertical < VerticalMinFt(...)
//
// (Satisfying either minimum is safe; only losing both at the same instant is a
// conflict.) The vertical minimum is RVSM-aware: 1000ft when both aircraft are
// at or below FL410, else 2000ft (non-RVSM above the band). The earliest
// breaching instant and the closest lateral/vertical approach within the
// breach are recorded, and the conflict is classified by relative geometry
// (same-direction, overtaking, opposite, crossing, vertical-only).
package separation

import (
	"time"

	"task142-atcconflict/internal/model"
	"task142-atcconflict/internal/trajectory"
)

// LateralMinNM is the radar lateral separation minimum in nautical miles.
const LateralMinNM = 5.0

// DefaultLookaheadSec is the default conflict lookahead horizon.
const DefaultLookaheadSec = 900.0 // 15 minutes

// DefaultStepSec is the default sampling step.
const DefaultStepSec = 12.0

// VerticalMinFt returns the vertical separation minimum for two flight levels,
// RVSM-aware: 1000ft when both are at or below FL410, otherwise 2000ft.
func VerticalMinFt(flA, flB int) int {
	if flA <= 410 && flB <= 410 {
		return 1000
	}
	return 2000
}

// flToFeet converts a flight level to feet (1 FL = 100 ft).
func flToFeet(fl int) float64 { return float64(fl) * 100.0 }

// Prober is the conflict detector. It is stateless; all inputs are passed per
// call so it is trivially unit-testable and deterministic.
type Prober struct {
	LookaheadSec float64
	StepSec      float64
}

// New returns a Prober with the defaults.
func New() *Prober { return &Prober{LookaheadSec: DefaultLookaheadSec, StepSec: DefaultStepSec} }

// TrackState is the projection of one flight's current track report used by the
// probe. The service layer fills this from the latest track report.
type TrackState struct {
	PlanID       string
	Position     trajectory.LatLon
	FL           int
	Heading      float64
	Groundspeed  int
	VerticalRate int
}

// Detect runs the pairwise conflict probe over the given active flights and
// returns the conflicts found, earliest-first. It is O(N² × steps).
func (p *Prober) Detect(now time.Time, tracks []TrackState) []model.Conflict {
	var out []model.Conflict
	for i := 0; i < len(tracks); i++ {
		for j := i + 1; j < len(tracks); j++ {
			c := p.detectPair(now, tracks[i], tracks[j])
			if c != nil {
				out = append(out, *c)
			}
		}
	}
	return out
}

// detectPair probes one pair over the lookahead window.
func (p *Prober) detectPair(now time.Time, a, b TrackState) *model.Conflict {
	vmin := VerticalMinFt(a.FL, b.FL)
	step := p.StepSec
	if step <= 0 {
		step = DefaultStepSec
	}
	lookahead := p.LookaheadSec
	if lookahead <= 0 {
		lookahead = DefaultLookaheadSec
	}
	var (
		firstBreach float64 = -1
		minLat      = -1.0
		minVert     = -1.0
		breachTS    time.Time
	)
	for t := 0.0; t <= lookahead; t += step {
		pa, fla := trajectory.ProjectFromTrack(a.Position, a.Heading, a.Groundspeed, a.FL, a.VerticalRate, t)
		pb, flb := trajectory.ProjectFromTrack(b.Position, b.Heading, b.Groundspeed, b.FL, b.VerticalRate, t)
		lat := trajectory.HaversineNM(pa, pb)
		vert := absFL(fla, flb)
		if lat < LateralMinNM && vert < float64(vmin) {
			if firstBreach < 0 {
				firstBreach = t
				breachTS = now.Add(time.Duration(t*float64(time.Second)))
			}
			if minLat < 0 || lat < minLat {
				minLat = lat
			}
			if minVert < 0 || vert < minVert {
				minVert = vert
			}
		}
	}
	if firstBreach < 0 {
		return nil
	}
	// If we never recorded a min (e.g. only one sample), use the breach values.
	if minLat < 0 {
		minLat = LateralMinNM
	}
	if minVert < 0 {
		minVert = float64(vmin)
	}
	return &model.Conflict{
		PlanA:         a.PlanID,
		PlanB:         b.PlanID,
		Type:          classify(a, b),
		CPATs:         breachTS.UTC().Format(time.RFC3339),
		MinLateralNM: minLat,
		MinVerticalFT: minVert,
		DetectedAt:    now.UTC().Format(time.RFC3339),
	}
}

// absFL returns the absolute vertical separation in FEET between two flight
// levels (1 FL = 100 ft). The conflict minimum (VerticalMinFt) is in feet, so
// the separation must be in feet too.
func absFL(a, b int) float64 {
	d := a - b
	if d < 0 {
		d = -d
	}
	return float64(d) * 100.0
}

// classify determines the ConflictType from the two flights' headings and the
// rate of lateral closure. This is an approximation suitable for labeling.
func classify(a, b TrackState) model.ConflictType {
	// Head-on (each flying toward the other) is the dominant geometry; check it
	// first regardless of the heading-difference buckets below, because two
	// aircraft on reciprocal headings (diff ~180) must read as OPPOSITE.
	if isOpposite(a, b) {
		return model.ConflictOpposite
	}
	// Track difference in [0,180].
	diff := absHeadingDiff(a.Heading, b.Heading)
	switch {
	case diff < 15 || diff > 345:
		// Same direction: overtaking if speeds differ enough, else same-dir.
		if absInt(a.Groundspeed-b.Groundspeed) >= 50 {
			return model.ConflictOvertaking
		}
		// If vertical rates bring them together, vertical.
		if closesVertically(a, b) {
			return model.ConflictVertical
		}
		return model.ConflictSameDir
	default:
		// Acute or perpendicular crossing.
		return model.ConflictCrossing
	}
}

// absHeadingDiff returns the smaller angle between two headings in [0,180].
func absHeadingDiff(a, b float64) float64 {
	d := a - b
	if d < 0 {
		d = -d
	}
	d = mod(d, 360)
	if d > 180 {
		d = 360 - d
	}
	return d
}

// isOpposite returns true if the two flights are flying roughly toward each
// other (headings pointing at each other within ~20deg of reciprocal).
func isOpposite(a, b TrackState) bool {
	// The bearing from a to b; if a's heading is toward b and b's heading is
	// toward a, they are opposite.
	brgAB := trajectory.InitialBearing(a.Position, b.Position)
	brgBA := trajectory.InitialBearing(b.Position, a.Position)
	return absHeadingDiff(a.Heading, brgAB) < 25 && absHeadingDiff(b.Heading, brgBA) < 25
}

// closesVertically reports whether the two flights' vertical rates bring them
// closer in altitude.
func closesVertically(a, b TrackState) bool {
	// If a is below b and climbing, or a is above b and descending, they close.
	if a.FL < b.FL && a.VerticalRate > 0 {
		return true
	}
	if a.FL > b.FL && a.VerticalRate < 0 {
		return true
	}
	if b.FL < a.FL && b.VerticalRate > 0 {
		return true
	}
	if b.FL > a.FL && b.VerticalRate < 0 {
		return true
	}
	return false
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func mod(a, b float64) float64 {
	m := a
	for m >= b {
		m -= b
	}
	for m < 0 {
		m += b
	}
	return m
}
