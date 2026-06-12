package domain

import (
	"context"
	"time"
)

type MealSchedule struct {
	ID        string
	TenantID  string
	DayOfWeek int
	MealType  string
	ChefID    *string // User ID (UUID)
	CreatedAt time.Time
}

type MealScheduleRepository interface {
	CreateOrUpdate(ctx context.Context, m *MealSchedule) error
	GetByTenantID(ctx context.Context, tenantID string) ([]*MealSchedule, error)
	GetByDayAndType(ctx context.Context, tenantID string, day int, mealType string) (*MealSchedule, error)
}
