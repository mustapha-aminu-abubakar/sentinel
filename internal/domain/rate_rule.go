package domain

import (
	"time"

	"github.com/google/uuid"
)

// RateRule defines the rate-limit configuration for a client on a specific API.
type RateRule struct {
	ID              uuid.UUID // Unique rule identifier.
	ClientID        uuid.UUID // Client this rule belongs to.
	API             string    // API identifier this rule applies to.
	RequestsAllowed int       // Maximum requests permitted in the window.
	WindowSeconds   int       // Duration of the rate-limit window in seconds.
	CreatedAt       time.Time // Timestamp when the rule was created.
	UpdatedAt       time.Time // Timestamp when the rule was last updated.
}
