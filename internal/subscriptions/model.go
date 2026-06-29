package subscriptions

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("subscription not found")

type Subscription struct {
	ID          uuid.UUID
	ServiceName string
	Price       int
	UserID      uuid.UUID
	StartMonth  time.Time
	EndMonth    *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CreateInput struct {
	ServiceName string
	Price       int
	UserID      uuid.UUID
	StartMonth  time.Time
	EndMonth    *time.Time
}

type UpdateInput = CreateInput

type ListFilter struct {
	UserID      *uuid.UUID
	ServiceName string
	Limit       int
	Offset      int
}

type TotalFilter struct {
	From        time.Time
	To          time.Time
	UserID      *uuid.UUID
	ServiceName string
}
