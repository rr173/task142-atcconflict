package flight

import (
	"fmt"
	"task142-atcconflict/internal/model"
)

func Initiate(planID, from, to, now string) (*model.Handoff, error) {
	if planID == "" || from == "" || to == "" || from == to {
		return nil, model.ErrHandoffNotAdjacent
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
