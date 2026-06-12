package domain

import (
	"context"
	"time"
)

type PersonalHabit struct {
	ID               string
	UserID           string
	ActivityType     string // "gym", "run", "diet_plan", "job_search"
	ScheduledDays    string // Comma separated, e.g. "Monday,Wednesday,Friday"
	ProgressStatus   string
	ReminderTime     *string    // "HH:MM" format, optional
	Timezone         string     // e.g. "America/Bogota"
	LastNotifiedDate *time.Time // can be nil
	SeqNum           int
	CreatedAt        time.Time
}

type PersonalHabitRepository interface {
	Create(ctx context.Context, h *PersonalHabit) error
	GetByID(ctx context.Context, id string) (*PersonalHabit, error)
	GetBySeqNum(ctx context.Context, userID string, seqNum int) (*PersonalHabit, error)
	Update(ctx context.Context, h *PersonalHabit) error
	GetByUserID(ctx context.Context, userID string) ([]*PersonalHabit, error)
	GetHabitsForNotification(ctx context.Context) ([]*PersonalHabit, error)
	Delete(ctx context.Context, id string) error
}

type HabitLog struct {
	ID         string
	HabitID    string
	LoggedDate time.Time
	Status     string // "completed", "skipped"
	CreatedAt  time.Time
}

type HabitLogRepository interface {
	CreateLog(ctx context.Context, log *HabitLog) error
	GetLogsByHabitID(ctx context.Context, habitID string) ([]*HabitLog, error)
}
