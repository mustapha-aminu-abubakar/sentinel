package engine

import "github.com/prometheus/client_golang/prometheus"

// degradedDecisions tracks rate-limit decisions made outside the Normal state.
var degradedDecisions = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "sentinel_degraded_decisions_total",
		Help: "Total number of rate limit decisions made in a degraded state.",
	},
	[]string{"state"},
)

func init() {
	prometheus.MustRegister(degradedDecisions)
}

// incDegraded increments the degraded-decisions counter for non-Normal states.
func incDegraded(state DegradedState) {
	if state == Normal {
		return
	}
	degradedDecisions.WithLabelValues(state.String()).Inc()
}
