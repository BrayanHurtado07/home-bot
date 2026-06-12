package domain

import (
	"context"
	"time"
)

type HouseTask struct {
	ID          string
	TenantID    string
	Description string
	AssignedTo  *string // User ID (UUID)
	DueDate     *time.Time
	IsDone      bool
	SeqNum      int
	CreatedAt   time.Time
}

type HouseTaskRepository interface {
	Create(ctx context.Context, t *HouseTask) error
	GetByID(ctx context.Context, id string) (*HouseTask, error)
	GetBySeqNum(ctx context.Context, tenantID string, seqNum int) (*HouseTask, error)
	Update(ctx context.Context, t *HouseTask) error
	GetPendingByTenantID(ctx context.Context, tenantID string) ([]*HouseTask, error)
	GetPendingByUserID(ctx context.Context, userID string) ([]*HouseTask, error)
	Delete(ctx context.Context, id string) error
	GetStats(ctx context.Context, tenantID string) (total int, completed int, err error)
}
