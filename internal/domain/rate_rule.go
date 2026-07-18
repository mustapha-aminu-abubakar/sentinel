package domain

import (
	"time"

	"github.com/google/uuid"
)

type RateRule struct {
	ID              uuid.UUID
	ClientID        uuid.UUID
	API             string
	RequestsAllowed int
	WindowSeconds   int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
