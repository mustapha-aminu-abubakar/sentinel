// Package domain defines the core business types: clients, rate rules, validation, and sentinel errors.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// ClientStatus represents the lifecycle status of an API client.
type ClientStatus string

const (
	// ClientStatusActive indicates the client is active and allowed to make requests.
	ClientStatusActive ClientStatus = "active"
	// ClientStatusInactive indicates the client is inactive and cannot make requests.
	ClientStatusInactive ClientStatus = "inactive"
)

// IsValid returns true if the status is a known value.
func (s ClientStatus) IsValid() bool {
	switch s {
	case ClientStatusActive, ClientStatusInactive:
		return true
	}
	return false
}

// Client represents an API client registered in the system.
type Client struct {
	ID        uuid.UUID    // Unique client identifier.
	Name      string       // Human-readable client name.
	Status    ClientStatus // Current client status.
	CreatedAt time.Time    // Timestamp when the client was created.
	UpdatedAt time.Time    // Timestamp when the client was last updated.
}
