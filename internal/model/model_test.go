package model

import "testing"

func TestFlightStateIsOperational(t *testing.T) {
	cases := []struct {
		state FlightState
		want  bool
	}{
		{FlightFiled, false},
		{FlightAccepted, true},
		{FlightCleared, true},
		{FlightActive, true},
		{FlightSuspended, true},
		{FlightTerminated, false},
	}
	for _, c := range cases {
		if got := c.state.IsOperational(); got != c.want {
			t.Errorf("%s.IsOperational() = %v, want %v", c.state, got, c.want)
		}
	}
}

func TestFlightStateIsAirborne(t *testing.T) {
	if !FlightActive.IsAirborne() {
		t.Error("ACTIVE should be airborne")
	}
	if FlightSuspended.IsAirborne() {
		t.Error("SUSPENDED should not be airborne")
	}
}

func TestIsBusinessError(t *testing.T) {
	if !IsBusinessError(ErrRouteInvalid) {
		t.Error("ErrRouteInvalid should be a business error")
	}
	if !IsBusinessError(ErrNotFound) {
		t.Error("ErrNotFound should be a business error")
	}
	if IsBusinessError(nil) {
		t.Error("nil should not be a business error")
	}
	if IsBusinessError(errCustom) {
		t.Error("a non-sentinel error should not be a business error")
	}
}

var errCustom = newSentinel("custom")

func newSentinel(msg string) error { return &simpleError{msg} }

type simpleError struct{ msg string }

func (e *simpleError) Error() string { return e.msg }
