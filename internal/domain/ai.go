package domain

import (
	"context"
	"time"
)

type AIContext struct {
	ID               string
	UserID           string
	CondensedHistory string
	LastUpdated      time.Time
}

type AIContextRepository interface {
	GetByUserID(ctx context.Context, userID string) (*AIContext, error)
	CreateOrUpdate(ctx context.Context, ai *AIContext) error
}
