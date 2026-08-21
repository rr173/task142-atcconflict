package metrics

import (
	"task142-atcconflict/internal/model"
	"task142-atcconflict/internal/report"
)

type Snapshot struct {
	Plans     int `json:"plans"`
	Active    int `json:"active"`
	Conflicts int `json:"conflicts"`
}

func From(plans []model.FlightPlan, conflicts []model.Conflict) Snapshot {
	s := report.Build(plans, conflicts)
	return Snapshot{Plans: len(plans), Active: s.Active, Conflicts: s.Conflicts}
}
