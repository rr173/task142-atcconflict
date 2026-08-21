package flight

import (
	"errors"
	"testing"

	"task142-atcconflict/internal/model"
)

func TestInitiateOnlyByControllingSector(t *testing.T) {
	const (
		plan       = "flt-1"
		controller = "SEC_A"
		next       = "SEC_B"
		now        = "2026-08-21T10:00:00Z"
	)
	// The controlling sector initiates a handoff to the next sector.
	h, err := Initiate(plan, controller, next, controller, now)
	if err != nil {
		t.Fatalf("controlling sector should initiate, got %v", err)
	}
	if h.FromSector != controller || h.ToSector != next || h.State != model.HandoffInitiated {
		t.Errorf("unexpected handoff: %+v", h)
	}

	// A different (non-responsible) sector cannot initiate the handoff: the
	// receiving sector would otherwise be handed an untrustworthy request from
	// a sector that never held control.
	_, err = Initiate(plan, "SEC_X", next, controller, now)
	if !errors.Is(err, model.ErrHandoffNotController) {
		t.Fatalf("non-controlling sector should be rejected with ErrHandoffNotController, got %v", err)
	}

	// Nobody may initiate when no sector is responsible for the flight yet.
	_, err = Initiate(plan, controller, next, "", now)
	if !errors.Is(err, model.ErrHandoffNotController) {
		t.Fatalf("empty controller should be rejected with ErrHandoffNotController, got %v", err)
	}

	// Structural checks still apply (empty/identical sectors).
	_, err = Initiate(plan, controller, controller, controller, now)
	if !errors.Is(err, model.ErrHandoffNotAdjacent) {
		t.Fatalf("from==to should be rejected with ErrHandoffNotAdjacent, got %v", err)
	}
}
