package domain

import (
	"time"

	"github.com/google/uuid"
)

type ClientStatus string

const (
	ClientStatusActive   ClientStatus = "active"
	ClientStatusInactive ClientStatus = "inactive"
)

func (s ClientStatus) IsValid() bool {
	switch s {
	case ClientStatusActive, ClientStatusInactive:
		return true
	}
	return false
}

type Client struct {
	ID        uuid.UUID
	Name      string
	Status    ClientStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}
