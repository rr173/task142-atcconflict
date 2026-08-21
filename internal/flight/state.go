package flight

import (
	"fmt"
	"task142-atcconflict/internal/model"
)

func CanTransition(from, to model.FlightState) bool {
	switch from {
	case model.FlightFiled:
		return to == model.FlightAccepted
	case model.FlightAccepted:
		return to == model.FlightCleared
	case model.FlightCleared:
		return to == model.FlightActive
	case model.FlightActive:
		return to == model.FlightSuspended || to == model.FlightTerminated
	case model.FlightSuspended:
		return to == model.FlightActive || to == model.FlightTerminated
	}
	return false
}

func Transition(p *model.FlightPlan, to model.FlightState) error {
	if p == nil || !CanTransition(p.State, to) {
		return fmt.Errorf("%w: %s to %s", model.ErrInvalidState, p.State, to)
	}
	p.State = to
	p.Suspended = to == model.FlightSuspended
	return nil
}

func ValidateEnvelope(a *model.AircraftType, fl int, mach float64) error {
	if a == nil || fl < 0 || fl > a.CeilingFL {
		return model.ErrFLRange
	}
	if mach < a.MinMach || mach > a.MaxMach {
		return model.ErrSpeedEnvelope
	}
	return nil
}
