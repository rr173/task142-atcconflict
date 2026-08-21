package report

import (
	"task142-atcconflict/internal/model"
)

type Summary struct {
	Filed, Active, Suspended, Terminated, Conflicts int `json:"-"`
}

func Build(plans []model.FlightPlan, conflicts []model.Conflict) Summary {
	var s Summary
	for _, p := range plans {
		switch p.State {
		case model.FlightFiled:
			s.Filed++
		case model.FlightActive:
			s.Active++
		case model.FlightSuspended:
			s.Suspended++
		case model.FlightTerminated:
			s.Terminated++
		}
	}
	s.Conflicts = len(conflicts)
	return s
}
