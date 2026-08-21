package flight

import (
	"fmt"
	"task142-atcconflict/internal/model"
)

func Initiate(planID, from, to, controller, now string) (*model.Handoff, error) {
	if planID == "" || from == "" || to == "" || from == to {
		return nil, model.ErrHandoffNotAdjacent
	}
	// Only the flight's current controlling sector may transfer it. Without
	// this check a non-responsible sector could offer the flight, handing the
	// receiving sector an untrustworthy control request from a sector that
	// never held control. An empty controller means nobody is responsible, so
	// nobody may initiate either.
	if controller == "" || from != controller {
		return nil, model.ErrHandoffNotController
	}
	return &model.Handoff{PlanID: planID, FromSector: from, ToSector: to, State: model.HandoffInitiated, InitiatedAt: now}, nil
}

func Accept(h *model.Handoff, now string) error {
	if h == nil || h.State != model.HandoffInitiated {
		return fmt.Errorf("%w: handoff is not pending", model.ErrInvalidState)
	}
	h.State, h.AcceptedAt = model.HandoffCompleted, now
	return nil
}

func CancelUnaccepted(h *model.Handoff) bool {
	if h != nil && h.State == model.HandoffInitiated {
		h.State = model.HandoffCancelled
		return true
	}
	return false
}
