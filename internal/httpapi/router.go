package httpapi

import (
	"io/fs"
	"net/http"
	"task142-atcconflict/internal/service"
)

type API struct{ svc *service.Service }

func New(s *service.Service, web fs.FS) http.Handler {
	a := &API{svc: s}
	m := http.NewServeMux()
	m.HandleFunc("GET /health", a.health)
	m.HandleFunc("GET /metrics", a.metrics)
	m.HandleFunc("POST /aircraft-types", a.aircraft)
	m.HandleFunc("POST /flight-plans", a.file)
	m.HandleFunc("GET /flight-plans", a.flights)
	m.HandleFunc("GET /flight-plans/{id}", a.flight)
	m.HandleFunc("POST /flight-plans/{id}/transition", a.transition)
	m.HandleFunc("POST /track-reports", a.track)
	m.HandleFunc("GET /conflicts", a.conflicts)
	m.HandleFunc("POST /handoffs", a.handoff)
	m.HandleFunc("POST /handoffs/{id}/accept", a.acceptHandoff)
	m.HandleFunc("POST /admin/rebuild", a.reconcile)
	if web != nil {
		m.Handle("GET /", http.FileServer(http.FS(web)))
	}
	return m
}
