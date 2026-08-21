package httpapi

import (
	"net/http"
	"task142-atcconflict/internal/metrics"
	"task142-atcconflict/internal/model"
)

func (a *API) health(w http.ResponseWriter, r *http.Request) {
	if err := a.svc.Store().Ping(r.Context()); err != nil {
		fail(w, err)
		return
	}
	write(w, http.StatusOK, map[string]string{"status": "ok"})
}
func (a *API) metrics(w http.ResponseWriter, r *http.Request) {
	p, e := a.svc.Flights(r.Context())
	if e != nil {
		fail(w, e)
		return
	}
	c, e := a.svc.Conflicts(r.Context())
	if e != nil {
		fail(w, e)
		return
	}
	write(w, http.StatusOK, metrics.From(p, c))
}
func (a *API) aircraft(w http.ResponseWriter, r *http.Request) {
	var v model.AircraftType
	if !decode(w, r, &v) {
		return
	}
	out, err := a.svc.RegisterAircraft(r.Context(), &v)
	if err != nil {
		fail(w, err)
		return
	}
	write(w, http.StatusCreated, out)
}
func (a *API) file(w http.ResponseWriter, r *http.Request) {
	var v model.FlightPlan
	if !decode(w, r, &v) {
		return
	}
	out, err := a.svc.File(r.Context(), &v)
	if err != nil {
		fail(w, err)
		return
	}
	write(w, http.StatusCreated, out)
}
func (a *API) flights(w http.ResponseWriter, r *http.Request) {
	out, err := a.svc.Flights(r.Context())
	if err != nil {
		fail(w, err)
		return
	}
	write(w, http.StatusOK, map[string]any{"flights": out})
}
func (a *API) flight(w http.ResponseWriter, r *http.Request) {
	out, err := a.svc.Flight(r.Context(), r.PathValue("id"))
	if err != nil {
		fail(w, err)
		return
	}
	write(w, http.StatusOK, out)
}
func (a *API) transition(w http.ResponseWriter, r *http.Request) {
	var v struct {
		To model.FlightState `json:"to"`
	}
	if !decode(w, r, &v) {
		return
	}
	out, err := a.svc.Move(r.Context(), r.PathValue("id"), v.To)
	if err != nil {
		fail(w, err)
		return
	}
	write(w, http.StatusOK, out)
}
func (a *API) track(w http.ResponseWriter, r *http.Request) {
	var v model.TrackReport
	if !decode(w, r, &v) {
		return
	}
	out, err := a.svc.Report(r.Context(), &v)
	if err != nil {
		fail(w, err)
		return
	}
	write(w, http.StatusCreated, out)
}
func (a *API) conflicts(w http.ResponseWriter, r *http.Request) {
	out, err := a.svc.Conflicts(r.Context())
	if err != nil {
		fail(w, err)
		return
	}
	write(w, http.StatusOK, map[string]any{"conflicts": out})
}
func (a *API) handoff(w http.ResponseWriter, r *http.Request) {
	var v struct {
		PlanID string `json:"plan_id"`
		From   string `json:"from_sector"`
		To     string `json:"to_sector"`
	}
	if !decode(w, r, &v) {
		return
	}
	out, err := a.svc.InitiateHandoff(r.Context(), v.PlanID, v.From, v.To)
	if err != nil {
		fail(w, err)
		return
	}
	write(w, http.StatusCreated, out)
}
func (a *API) acceptHandoff(w http.ResponseWriter, r *http.Request) {
	out, err := a.svc.AcceptHandoff(r.Context(), r.PathValue("id"))
	if err != nil {
		fail(w, err)
		return
	}
	write(w, http.StatusOK, out)
}
func (a *API) reconcile(w http.ResponseWriter, r *http.Request) {
	n, err := a.svc.Reconcile(r.Context())
	if err != nil {
		fail(w, err)
		return
	}
	write(w, http.StatusOK, map[string]int{"cancelled_handoffs": n})
}
