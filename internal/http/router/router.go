package router

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	analyticshttp "sentinel/internal/analytics/http"
	"sentinel/internal/engine"
	"sentinel/internal/http/handlers"
	"sentinel/internal/repository"
)

func NewRouter(clientRepo repository.ClientRepository, ruleRepo repository.RateRuleRepository, decisionEngine *engine.DecisionEngine, pool *pgxpool.Pool) http.Handler {
	if decisionEngine == nil {
		panic("router: decisionEngine is required for POST /v1/check")
	}

	mux := http.NewServeMux()

	clients := handlers.NewClientsHandler(clientRepo)
	rules := handlers.NewRulesHandler(ruleRepo)

	mux.HandleFunc("GET /health", healthHandler)

	mux.HandleFunc("GET /clients", clients.List)
	mux.HandleFunc("POST /clients", clients.Create)
	mux.HandleFunc("GET /clients/{id}", clients.Get)
	mux.HandleFunc("PATCH /clients/{id}", clients.Update)

	mux.HandleFunc("GET /rules", rules.List)
	mux.HandleFunc("POST /rules", rules.Create)
	mux.HandleFunc("GET /rules/{id}", rules.Get)
	mux.HandleFunc("PATCH /rules/{id}", rules.Update)

	check := handlers.NewCheckHandler(decisionEngine)
	mux.HandleFunc("POST /v1/check", check.Check)

	analytics := analyticshttp.NewAnalyticsHandler(pool)
	mux.HandleFunc("GET /analytics/usage", analytics.GetUsage)
	mux.HandleFunc("GET /analytics/latency", analytics.GetLatency)

	return corsMiddleware(mux)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
