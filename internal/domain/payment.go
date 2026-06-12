package domain

import (
	"context"
	"time"
)

type Payment struct {
	ID          string
	TenantID    string
	UserID      string
	Amount      float64
	Status      string // "pending", "approved", "rejected"
	ProofURL    string
	BillingDate time.Time
	SeqNum      int
	CreatedAt   time.Time
}

type PaymentRepository interface {
	Create(ctx context.Context, p *Payment) error
	GetByID(ctx context.Context, id string) (*Payment, error)
	GetBySeqNum(ctx context.Context, tenantID string, seqNum int) (*Payment, error)
	Update(ctx context.Context, p *Payment) error
	GetPendingByTenantID(ctx context.Context, tenantID string) ([]*Payment, error)
	GetByTenantID(ctx context.Context, tenantID string) ([]*Payment, error)
}
