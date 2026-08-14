package main

import (
	"net/http"
	"sync/atomic"
	"time"
)

type Router struct {
	lbs  []LBServer
	next uint64
}

func NewRouter(lbs []LBServer) *Router {
	return &Router{lbs: lbs}
}

func isHealthy(healthURL string) bool {
	client := http.Client{
		Timeout: 2 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get(healthURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 400
}

func (r *Router) selectHealthyLB() (LBServer, bool) {
	n := len(r.lbs)
	if n == 0 {
		return LBServer{}, false
	}

	start := atomic.AddUint64(&r.next, 1) - 1

	for i := 0; i < n; i++ {
		lb := r.lbs[(int(start)+i)%n]

		// Disabled LBs are never eligible, even if their health check passes.
		if !lb.Enable {
			continue
		}

		if isHealthy(lb.Health) {
			return lb, true
		}
	}

	return LBServer{}, false
}

func (r *Router) RouteHandler(w http.ResponseWriter, req *http.Request) {
	lb, ok := r.selectHealthyLB()
	if !ok {
		http.Error(w, "No enabled healthy Main LB available", http.StatusServiceUnavailable)
		return
	}

	// 307 keeps the original HTTP method.
	http.Redirect(w, req, lb.BaseURL, http.StatusTemporaryRedirect)
}

func (r *Router) HealthHandler(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("router ok"))
}
