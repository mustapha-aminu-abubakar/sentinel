package engine

import "github.com/prometheus/client_golang/prometheus"

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

func incDegraded(state DegradedState) {
	if state == Normal {
		return
	}
	degradedDecisions.WithLabelValues(state.String()).Inc()
}
