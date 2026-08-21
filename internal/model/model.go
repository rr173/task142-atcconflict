// Package model holds the domain types, enums and error sentinels for the ATC
// flight conflict detection and sector handoff engine. It is a leaf package:
// it imports only the standard library and is imported by every other package
// so that airspace geometry, trajectory math, separation, flow and flight state
// share one vocabulary.
package model

import "errors"

// FlightState is the lifecycle state of a flight plan.
type FlightState string

const (
	// FlightFiled has been submitted but not accepted by a controller.
	FlightFiled FlightState = "FILED"
	// FlightAccepted has been acknowledged and is awaiting departure clearance.
	FlightAccepted FlightState = "ACCEPTED"
	// FlightCleared has a departure/route clearance and is ready to go active.
	FlightCleared FlightState = "CLEARED"
	// FlightActive is airborne; it has at least one track report and a current
	// 4D position. Conflicts and sector occupancy are only evaluated for active
	// flights.
	FlightActive FlightState = "ACTIVE"
	// FlightSuspended is held (e.g. a separation breach is being resolved); it
	// keeps its last position but is excluded from conflict detection.
	FlightSuspended FlightState = "SUSPENDED"
	// FlightTerminated has landed or left controlled airspace.
	FlightTerminated FlightState = "TERMINATED"
)

// IsOperational reports whether the state is one where the flight still has a
// controlling sector (i.e. it has not terminated).
func (s FlightState) IsOperational() bool {
	return s == FlightAccepted || s == FlightCleared || s == FlightActive || s == FlightSuspended
}

// IsAirborne reports whether the flight is actively being tracked (has a live
// position and participates in conflict detection). Only ACTIVE counts.
func (s FlightState) IsAirborne() bool { return s == FlightActive }

// HandoffState is the state of a sector-to-sector handoff.
type HandoffState string

const (
	// HandoffInitiated: the transferring sector has offered the flight; the
	// receiving sector has not yet accepted. Separation responsibility still
	// belongs to the transferring sector.
	HandoffInitiated HandoffState = "INITIATED"
	// HandoffCompleted: the receiving sector accepted; control transferred.
	HandoffCompleted HandoffState = "COMPLETED"
	// HandoffRejected: the receiving sector refused; responsibility stays with
	// the transferring sector.
	HandoffRejected HandoffState = "REJECTED"
	// HandoffCancelled: an un-accepted handoff at process restart is rolled
	// back to this state (the transfer never became durable).
	HandoffCancelled HandoffState = "CANCELLED"
)

// ConflictType classifies a loss-of-separation by the relative geometry of the
// two aircraft.
type ConflictType string

const (
	ConflictSameDir   ConflictType = "SAME_DIR"   // same track, same direction, converging altitude
	ConflictOvertaking ConflictType = "OVERTAKING"  // faster behind, catching up
	ConflictOpposite  ConflictType = "OPPOSITE"   // head-on, opposite tracks
	ConflictCrossing  ConflictType = "CROSSING"    // tracks cross at an angle
	ConflictVertical  ConflictType = "VERTICAL"    // near-identical horizontal, closing vertically
)

// ClearanceKind is the kind of ATC clearance issued to a flight.
type ClearanceKind string

const (
	ClearanceDirect    ClearanceKind = "DIRECT"     // cleared direct to a waypoint
	ClearanceLevel     ClearanceKind = "LEVEL"      // cleared to a new flight level
	ClearanceSpeed     ClearanceKind = "SPEED"       // cleared to a new Mach
	ClearanceRouteRele ClearanceKind = "ROUTE_RELIEF" // cleared via a relief route segment
)

// WaypointKind classifies a navigation fix.
type WaypointKind string

const (
	WaypointFix     WaypointKind = "FIX"
	WaypointVOR     WaypointKind = "VOR"
	WaypointNDB     WaypointKind = "NDB"
	WaypointAerodrome WaypointKind = "AERODROME"
)

// AirwayDirection constrains travel on an airway.
type AirwayDirection string

const (
	AirwayBoth    AirwayDirection = "BOTH"
	AirwayForward AirwayDirection = "FORWARD"
	AirwayReverse AirwayDirection = "REVERSE"
)

// --- Domain structs ---

// Sector is a 3D airspace volume owned by a control position. A flight is
// "inside" the sector when its lat/lon falls inside the polygon AND its flight
// level is within [FloorFL, CeilingFL].
type Sector struct {
	ID         string  `json:"id"`
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	FloorFL    int     `json:"floor_fl"`
	CeilingFL  int     `json:"ceiling_fl"`
	CapacityMAP int    `json:"capacity_map"` // Monitor Alert Parameter: max active flights
	Owner      string  `json:"owner"`
	// Polygon is the sector boundary as a list of lat/lon vertices in order.
	// At least 3 vertices. The boundary is closed implicitly (last→first).
	Polygon   []LatLon `json:"polygon"`
	CreatedAt string  `json:"created_at"`
}

// LatLon is a geographic point in decimal degrees.
type LatLon struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// Waypoint is a navigation fix.
type Waypoint struct {
	ID        string       `json:"id"`
	Code      string       `json:"code"`
	Name      string       `json:"name"`
	Position  LatLon       `json:"position"`
	Kind      WaypointKind `json:"kind"`
	CreatedAt string       `json:"created_at"`
}

// Airway is a sequence of waypoint-to-waypoint segments with a direction and an
// altitude band.
type Airway struct {
	ID        string         `json:"id"`
	Code      string         `json:"code"`
	Name      string         `json:"name"`
	Direction AirwayDirection `json:"direction"`
	MinFL     int            `json:"min_fl"`
	MaxFL     int            `json:"max_fl"`
	// Waypoints is the ordered list of waypoint IDs along the airway.
	WaypointIDs []string  `json:"waypoint_ids"`
	CreatedAt   string    `json:"created_at"`
}

// AircraftType is the performance envelope of an aircraft model.
type AircraftType struct {
	ID        string  `json:"id"`
	Code      string  `json:"code"`
	Name      string  `json:"name"`
	MinMach   float64 `json:"min_mach"`
	MaxMach   float64 `json:"max_mach"`
	CeilingFL int    `json:"ceiling_fl"`
	CreatedAt string  `json:"created_at"`
}

// RouteSegment is one leg of a flight plan route: from the previous waypoint
// (or departure) to WaypointID, cruising at FL and Mach.
type RouteSegment struct {
	Seq        int     `json:"seq"`
	WaypointID string  `json:"waypoint_id"`
	FL         int     `json:"fl"`
	Mach       float64 `json:"mach"`
}

// FlightPlan is a filed flight plan with its current state and controlling
// sector.
type FlightPlan struct {
	ID                string          `json:"id"`
	Callsign          string          `json:"callsign"`
	AircraftTypeID    string          `json:"aircraft_type_id"`
	DepAerodrome      string          `json:"dep_aerodrome"`
	Destination       string          `json:"destination"`
	EOBT              string          `json:"eobt"` // estimated off-block time, RFC3339
	Rules             string          `json:"rules"` // IFR/VFR
	State             FlightState     `json:"state"`
	CruiseFL          int             `json:"cruise_fl"`
	CruiseMach        float64         `json:"cruise_mach"`
	ControllingSector string          `json:"controlling_sector"`
	Suspended         bool            `json:"suspended"`
	CreatedAt         string          `json:"created_at"`
	UpdatedAt         string          `json:"updated_at"`
}

// Clearance is an ATC instruction that modifies a flight plan (route, level or
// speed). The latest clearance per flight is the effective plan.
type Clearance struct {
	ID            string        `json:"id"`
	PlanID        string        `json:"plan_id"`
	Seq           int           `json:"seq"`
	Kind          ClearanceKind `json:"kind"`
	TargetWaypoint string       `json:"target_waypoint"` // for DIRECT
	TargetFL      int           `json:"target_fl"`       // for LEVEL
	TargetMach    float64       `json:"target_mach"`      // for SPEED
	IssuedAt      string        `json:"issued_at"`
}

// TrackReport is a timestamped surveillance position for an active flight. The
// append-only track_reports table is the source of truth for restart recovery.
type TrackReport struct {
	ID            int64   `json:"id"`
	PlanID        string  `json:"plan_id"`
	Ts            string  `json:"ts"` // RFC3339
	Lat           float64 `json:"lat"`
	Lon           float64 `json:"lon"`
	FL            int     `json:"fl"`
	Heading       float64 `json:"heading"`        // degrees, true
	Groundspeed   int     `json:"groundspeed"`    // knots
	VerticalRate  int     `json:"vertical_rate"`  // ft/min, positive climb
}

// Handoff is a sector-to-sector transfer of an active flight.
type Handoff struct {
	ID           string       `json:"id"`
	PlanID       string       `json:"plan_id"`
	FromSector   string       `json:"from_sector"`
	ToSector     string       `json:"to_sector"`
	State        HandoffState `json:"state"`
	InitiatedAt  string       `json:"initiated_at"`
	AcceptedAt   string       `json:"accepted_at"`
}

// Conflict is a detected loss-of-separation between two active flights within
// the lookahead window. MinLateralNM and MinVerticalFT are the closest approach
// values; CPATs is the time of closest approach.
type Conflict struct {
	ID            string       `json:"id"`
	PlanA         string       `json:"plan_a"`
	PlanB         string       `json:"plan_b"`
	Type          ConflictType `json:"type"`
	CPATs         string       `json:"cpa_ts"`
	MinLateralNM float64      `json:"min_lateral_nm"`
	MinVerticalFT float64      `json:"min_vertical_ft"`
	DetectedAt    string       `json:"detected_at"`
}

// FlowRestriction is a miles-in-trail metering restriction on a fix/airway.
type FlowRestriction struct {
	ID           string `json:"id"`
	FixWaypoint  string `json:"fix_waypoint"`
	Airway       string `json:"airway"`
	MilesInTrail int    `json:"miles_in_trail"`
	Active       bool   `json:"active"`
	IssuedAt     string `json:"issued_at"`
}

// Position is a 4D reconstructed position of an active flight (from its latest
// track report).
type Position struct {
	PlanID       string  `json:"plan_id"`
	Callsign     string  `json:"callsign"`
	Ts           string  `json:"ts"`
	Lat          float64 `json:"lat"`
	Lon          float64 `json:"lon"`
	FL           int     `json:"fl"`
	Heading      float64 `json:"heading"`
	Groundspeed  int     `json:"groundspeed"`
	VerticalRate int     `json:"vertical_rate"`
}

// TrajectoryPoint is one sampled point of a predicted 4D trajectory.
type TrajectoryPoint struct {
	Ts      string  `json:"ts"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	FL      int     `json:"fl"`
	Heading float64 `json:"heading"`
}

// SectorOccupancy is the current load of a sector.
type SectorOccupancy struct {
	SectorID string   `json:"sector_id"`
	Count    int      `json:"count"`
	Capacity int      `json:"capacity"`
	Alert    bool     `json:"alert"`
	Plans    []string `json:"plans"`
}

// --- Error sentinels ---

var (
	// ErrNotFound is returned by store getters when a row does not exist.
	ErrNotFound = errors.New("atc: not found")

	// State-machine errors.
	ErrInvalidState      = errors.New("invalid flight state transition")
	ErrCallsignTaken     = errors.New("callsign already in use by an active flight")
	ErrRouteInvalid      = errors.New("route is not connected by declared airways or within direct-leg distance")
	ErrTrackOutOfOrder   = errors.New("track report timestamp is not strictly increasing")
	ErrHandoffPending     = errors.New("a handoff is already pending for this flight")
	ErrHandoffNotAdjacent = errors.New("receiving sector is not adjacent or flight is not entering it")
	ErrSpeedEnvelope      = errors.New("mach is outside the aircraft type envelope")
	ErrFLRange            = errors.New("flight level is out of range or above the aircraft ceiling")
	ErrNotAirborne        = errors.New("flight is not airborne")
	ErrAlreadyAirborne    = errors.New("flight is already airborne")
	ErrSuspended          = errors.New("flight is suspended")
	ErrConflictExists     = errors.New("flight cannot be cleared into a known conflict")
)

// IsBusinessError reports whether err is one of the domain business errors that
// should map to a 4xx status (as opposed to a generic 500).
func IsBusinessError(err error) bool {
	switch {
	case errors.Is(err, ErrNotFound),
		errors.Is(err, ErrInvalidState),
		errors.Is(err, ErrCallsignTaken),
		errors.Is(err, ErrRouteInvalid),
		errors.Is(err, ErrTrackOutOfOrder),
		errors.Is(err, ErrHandoffPending),
		errors.Is(err, ErrHandoffNotAdjacent),
		errors.Is(err, ErrSpeedEnvelope),
		errors.Is(err, ErrFLRange),
		errors.Is(err, ErrNotAirborne),
		errors.Is(err, ErrAlreadyAirborne),
		errors.Is(err, ErrSuspended),
		errors.Is(err, ErrConflictExists):
		return true
	}
	return false
}
