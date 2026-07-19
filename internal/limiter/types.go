// Package limiter provides a Redis-backed sliding-window rate limiter.
package limiter

// Rule defines the rate-limit parameters for a single API per client.
type Rule struct {
	ClientID        string // Client identifier this rule applies to.
	API             string // API endpoint identifier this rule applies to.
	RequestsAllowed int    // Maximum number of requests allowed in the window.
	WindowSeconds   int    // Sliding window duration in seconds.
}

// Decision represents the outcome of a rate-limit check.
type Decision struct {
	Allowed    bool // Whether the request is allowed.
	Remaining  int  // Remaining requests in the current window (-1 when unknown).
	RetryAfter int  // Seconds until the client can retry (0 when allowed).
}
