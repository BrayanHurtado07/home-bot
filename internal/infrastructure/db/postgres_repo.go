package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/iloremstudio/home-bot/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepository implementation
type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	query := `INSERT INTO users (telegram_id, tenant_id, role, name) 
	          VALUES ($1, $2, $3, $4) RETURNING id`
	err := r.pool.QueryRow(ctx, query, u.TelegramID, u.TenantID, u.Role, u.Name).Scan(&u.ID)
	if err != nil {
		return fmt.Errorf("error creating user: %w", err)
	}
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `SELECT id, telegram_id, tenant_id, role, name FROM users WHERE id = $1`
	var u domain.User
	err := r.pool.QueryRow(ctx, query, id).Scan(&u.ID, &u.TelegramID, &u.TenantID, &u.Role, &u.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("error getting user by id: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) GetByTelegramID(ctx context.Context, tgID int64) (*domain.User, error) {
	query := `SELECT id, telegram_id, tenant_id, role, name FROM users WHERE telegram_id = $1`
	var u domain.User
	err := r.pool.QueryRow(ctx, query, tgID).Scan(&u.ID, &u.TelegramID, &u.TenantID, &u.Role, &u.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("error getting user by telegram_id: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) Update(ctx context.Context, u *domain.User) error {
	query := `UPDATE users SET tenant_id = $1, role = $2, name = $3 WHERE id = $4`
	_, err := r.pool.Exec(ctx, query, u.TenantID, u.Role, u.Name, u.ID)
	if err != nil {
		return fmt.Errorf("error updating user: %w", err)
	}
	return nil
}

func (r *UserRepository) GetUsersByTenantID(ctx context.Context, tenantID string) ([]*domain.User, error) {
	query := `SELECT id, telegram_id, tenant_id, role, name FROM users WHERE tenant_id = $1`
	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("error querying users by tenant_id: %w", err)
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.TelegramID, &u.TenantID, &u.Role, &u.Name); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, nil
}

// TenantGroupRepository implementation
type TenantGroupRepository struct {
	pool *pgxpool.Pool
}

func NewTenantGroupRepository(pool *pgxpool.Pool) *TenantGroupRepository {
	return &TenantGroupRepository{pool: pool}
}

func (r *TenantGroupRepository) Create(ctx context.Context, g *domain.TenantGroup) error {
	query := `INSERT INTO tenants_groups (group_name) VALUES ($1) RETURNING id`
	err := r.pool.QueryRow(ctx, query, g.GroupName).Scan(&g.ID)
	if err != nil {
		return fmt.Errorf("error creating tenant group: %w", err)
	}
	return nil
}

func (r *TenantGroupRepository) GetByID(ctx context.Context, id string) (*domain.TenantGroup, error) {
	query := `SELECT id, group_name FROM tenants_groups WHERE id = $1`
	var g domain.TenantGroup
	err := r.pool.QueryRow(ctx, query, id).Scan(&g.ID, &g.GroupName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrGroupNotFound
		}
		return nil, fmt.Errorf("error getting tenant group: %w", err)
	}
	return &g, nil
}

// PaymentRepository implementation
type PaymentRepository struct {
	pool *pgxpool.Pool
}

func NewPaymentRepository(pool *pgxpool.Pool) *PaymentRepository {
	return &PaymentRepository{pool: pool}
}

func (r *PaymentRepository) Create(ctx context.Context, p *domain.Payment) error {
	query := `INSERT INTO payments (tenant_id, user_id, amount, status, proof_url, billing_date, seq_num) 
	          VALUES ($1, $2, $3, $4, $5, $6, (SELECT COALESCE(MAX(seq_num), 0) + 1 FROM payments WHERE tenant_id = $1)) 
	          RETURNING id, seq_num, created_at`
	err := r.pool.QueryRow(ctx, query, p.TenantID, p.UserID, p.Amount, p.Status, p.ProofURL, p.BillingDate).Scan(&p.ID, &p.SeqNum, &p.CreatedAt)
	if err != nil {
		return fmt.Errorf("error creating payment: %w", err)
	}
	return nil
}

func (r *PaymentRepository) GetByID(ctx context.Context, id string) (*domain.Payment, error) {
	query := `SELECT id, tenant_id, user_id, amount, status, proof_url, billing_date, seq_num, created_at FROM payments WHERE id = $1`
	var p domain.Payment
	err := r.pool.QueryRow(ctx, query, id).Scan(&p.ID, &p.TenantID, &p.UserID, &p.Amount, &p.Status, &p.ProofURL, &p.BillingDate, &p.SeqNum, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("payment not found")
		}
		return nil, fmt.Errorf("error getting payment: %w", err)
	}
	return &p, nil
}

func (r *PaymentRepository) GetBySeqNum(ctx context.Context, tenantID string, seqNum int) (*domain.Payment, error) {
	query := `SELECT id, tenant_id, user_id, amount, status, proof_url, billing_date, seq_num, created_at 
	          FROM payments WHERE tenant_id = $1 AND seq_num = $2`
	var p domain.Payment
	err := r.pool.QueryRow(ctx, query, tenantID, seqNum).Scan(&p.ID, &p.TenantID, &p.UserID, &p.Amount, &p.Status, &p.ProofURL, &p.BillingDate, &p.SeqNum, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("payment not found by sequence number %d", seqNum)
		}
		return nil, fmt.Errorf("error getting payment by seq_num: %w", err)
	}
	return &p, nil
}

func (r *PaymentRepository) Update(ctx context.Context, p *domain.Payment) error {
	query := `UPDATE payments SET status = $1, proof_url = $2 WHERE id = $3`
	_, err := r.pool.Exec(ctx, query, p.Status, p.ProofURL, p.ID)
	if err != nil {
		return fmt.Errorf("error updating payment: %w", err)
	}
	return nil
}

func (r *PaymentRepository) GetPendingByTenantID(ctx context.Context, tenantID string) ([]*domain.Payment, error) {
	query := `SELECT id, tenant_id, user_id, amount, status, proof_url, billing_date, seq_num, created_at 
	          FROM payments WHERE tenant_id = $1 AND status = 'pending' ORDER BY billing_date ASC`
	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("error querying pending payments: %w", err)
	}
	defer rows.Close()

	var payments []*domain.Payment
	for rows.Next() {
		var p domain.Payment
		if err := rows.Scan(&p.ID, &p.TenantID, &p.UserID, &p.Amount, &p.Status, &p.ProofURL, &p.BillingDate, &p.SeqNum, &p.CreatedAt); err != nil {
			return nil, err
		}
		payments = append(payments, &p)
	}
	return payments, nil
}

func (r *PaymentRepository) GetByTenantID(ctx context.Context, tenantID string) ([]*domain.Payment, error) {
	query := `SELECT id, tenant_id, user_id, amount, status, proof_url, billing_date, seq_num, created_at 
	          FROM payments WHERE tenant_id = $1 ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("error querying payments by tenant_id: %w", err)
	}
	defer rows.Close()

	var payments []*domain.Payment
	for rows.Next() {
		var p domain.Payment
		if err := rows.Scan(&p.ID, &p.TenantID, &p.UserID, &p.Amount, &p.Status, &p.ProofURL, &p.BillingDate, &p.SeqNum, &p.CreatedAt); err != nil {
			return nil, err
		}
		payments = append(payments, &p)
	}
	return payments, nil
}

// HouseTaskRepository implementation
type HouseTaskRepository struct {
	pool *pgxpool.Pool
}

func NewHouseTaskRepository(pool *pgxpool.Pool) *HouseTaskRepository {
	return &HouseTaskRepository{pool: pool}
}

func (r *HouseTaskRepository) Create(ctx context.Context, t *domain.HouseTask) error {
	query := `INSERT INTO house_tasks (tenant_id, description, assigned_to, due_date, is_done, seq_num) 
	          VALUES ($1, $2, $3, $4, $5, (SELECT COALESCE(MAX(seq_num), 0) + 1 FROM house_tasks WHERE tenant_id = $1)) 
	          RETURNING id, seq_num, created_at`
	err := r.pool.QueryRow(ctx, query, t.TenantID, t.Description, t.AssignedTo, t.DueDate, t.IsDone).Scan(&t.ID, &t.SeqNum, &t.CreatedAt)
	if err != nil {
		return fmt.Errorf("error creating task: %w", err)
	}
	return nil
}

func (r *HouseTaskRepository) GetByID(ctx context.Context, id string) (*domain.HouseTask, error) {
	query := `SELECT id, tenant_id, description, assigned_to, due_date, is_done, seq_num, created_at 
	          FROM house_tasks WHERE id = $1`
	var t domain.HouseTask
	err := r.pool.QueryRow(ctx, query, id).Scan(&t.ID, &t.TenantID, &t.Description, &t.AssignedTo, &t.DueDate, &t.IsDone, &t.SeqNum, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("task not found")
		}
		return nil, fmt.Errorf("error getting task by id: %w", err)
	}
	return &t, nil
}

func (r *HouseTaskRepository) GetBySeqNum(ctx context.Context, tenantID string, seqNum int) (*domain.HouseTask, error) {
	query := `SELECT id, tenant_id, description, assigned_to, due_date, is_done, seq_num, created_at 
	          FROM house_tasks WHERE tenant_id = $1 AND seq_num = $2`
	var t domain.HouseTask
	err := r.pool.QueryRow(ctx, query, tenantID, seqNum).Scan(&t.ID, &t.TenantID, &t.Description, &t.AssignedTo, &t.DueDate, &t.IsDone, &t.SeqNum, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("task not found by sequence number %d", seqNum)
		}
		return nil, fmt.Errorf("error getting task by seq_num: %w", err)
	}
	return &t, nil
}

func (r *HouseTaskRepository) Update(ctx context.Context, t *domain.HouseTask) error {
	query := `UPDATE house_tasks SET description = $1, assigned_to = $2, due_date = $3, is_done = $4 WHERE id = $5`
	_, err := r.pool.Exec(ctx, query, t.Description, t.AssignedTo, t.DueDate, t.IsDone, t.ID)
	if err != nil {
		return fmt.Errorf("error updating task: %w", err)
	}
	return nil
}

func (r *HouseTaskRepository) GetPendingByTenantID(ctx context.Context, tenantID string) ([]*domain.HouseTask, error) {
	query := `SELECT id, tenant_id, description, assigned_to, due_date, is_done, seq_num, created_at 
	          FROM house_tasks WHERE tenant_id = $1 AND is_done = FALSE ORDER BY due_date ASC`
	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("error querying pending tasks by tenant: %w", err)
	}
	defer rows.Close()

	var tasks []*domain.HouseTask
	for rows.Next() {
		var t domain.HouseTask
		if err := rows.Scan(&t.ID, &t.TenantID, &t.Description, &t.AssignedTo, &t.DueDate, &t.IsDone, &t.SeqNum, &t.CreatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, &t)
	}
	return tasks, nil
}

func (r *HouseTaskRepository) GetPendingByUserID(ctx context.Context, userID string) ([]*domain.HouseTask, error) {
	query := `SELECT id, tenant_id, description, assigned_to, due_date, is_done, seq_num, created_at 
	          FROM house_tasks WHERE assigned_to = $1 AND is_done = FALSE ORDER BY due_date ASC`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("error querying tasks by user: %w", err)
	}
	defer rows.Close()

	var tasks []*domain.HouseTask
	for rows.Next() {
		var t domain.HouseTask
		if err := rows.Scan(&t.ID, &t.TenantID, &t.Description, &t.AssignedTo, &t.DueDate, &t.IsDone, &t.SeqNum, &t.CreatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, &t)
	}
	return tasks, nil
}

func (r *HouseTaskRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM house_tasks WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("error deleting task: %w", err)
	}
	return nil
}

func (r *HouseTaskRepository) GetStats(ctx context.Context, tenantID string) (total int, completed int, err error) {
	query := `SELECT COUNT(*), COUNT(*) FILTER (WHERE is_done = TRUE) FROM house_tasks WHERE tenant_id = $1`
	err = r.pool.QueryRow(ctx, query, tenantID).Scan(&total, &completed)
	if err != nil {
		return 0, 0, fmt.Errorf("error getting task stats: %w", err)
	}
	return total, completed, nil
}

// MealScheduleRepository implementation
type MealScheduleRepository struct {
	pool *pgxpool.Pool
}

func NewMealScheduleRepository(pool *pgxpool.Pool) *MealScheduleRepository {
	return &MealScheduleRepository{pool: pool}
}

func (r *MealScheduleRepository) CreateOrUpdate(ctx context.Context, m *domain.MealSchedule) error {
	query := `INSERT INTO meal_schedule (tenant_id, day_of_week, meal_type, chef_id) 
	          VALUES ($1, $2, $3, $4)
	          ON CONFLICT (tenant_id, day_of_week, meal_type) 
	          DO UPDATE SET chef_id = EXCLUDED.chef_id RETURNING id, created_at`
	err := r.pool.QueryRow(ctx, query, m.TenantID, m.DayOfWeek, m.MealType, m.ChefID).Scan(&m.ID, &m.CreatedAt)
	if err != nil {
		return fmt.Errorf("error inserting/updating meal schedule: %w", err)
	}
	return nil
}

func (r *MealScheduleRepository) GetByTenantID(ctx context.Context, tenantID string) ([]*domain.MealSchedule, error) {
	query := `SELECT id, tenant_id, day_of_week, meal_type, chef_id, created_at 
	          FROM meal_schedule WHERE tenant_id = $1 ORDER BY day_of_week ASC, meal_type ASC`
	rows, err := r.pool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("error querying meal schedule: %w", err)
	}
	defer rows.Close()

	var schedule []*domain.MealSchedule
	for rows.Next() {
		var m domain.MealSchedule
		if err := rows.Scan(&m.ID, &m.TenantID, &m.DayOfWeek, &m.MealType, &m.ChefID, &m.CreatedAt); err != nil {
			return nil, err
		}
		schedule = append(schedule, &m)
	}
	return schedule, nil
}

func (r *MealScheduleRepository) GetByDayAndType(ctx context.Context, tenantID string, day int, mealType string) (*domain.MealSchedule, error) {
	query := `SELECT id, tenant_id, day_of_week, meal_type, chef_id, created_at 
	          FROM meal_schedule WHERE tenant_id = $1 AND day_of_week = $2 AND meal_type = $3`
	var m domain.MealSchedule
	err := r.pool.QueryRow(ctx, query, tenantID, day, mealType).Scan(&m.ID, &m.TenantID, &m.DayOfWeek, &m.MealType, &m.ChefID, &m.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("meal schedule entry not found")
		}
		return nil, fmt.Errorf("error getting meal schedule entry: %w", err)
	}
	return &m, nil
}

// PersonalHabitRepository implementation
type PersonalHabitRepository struct {
	pool *pgxpool.Pool
}

func NewPersonalHabitRepository(pool *pgxpool.Pool) *PersonalHabitRepository {
	return &PersonalHabitRepository{pool: pool}
}

func (r *PersonalHabitRepository) Create(ctx context.Context, h *domain.PersonalHabit) error {
	query := `INSERT INTO personal_habits (user_id, activity_type, scheduled_days, progress_status, reminder_time, timezone, last_notified_date, seq_num) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7, (SELECT COALESCE(MAX(seq_num), 0) + 1 FROM personal_habits WHERE user_id = $1)) 
	          RETURNING id, seq_num, created_at`
	err := r.pool.QueryRow(ctx, query, h.UserID, h.ActivityType, h.ScheduledDays, h.ProgressStatus, h.ReminderTime, h.Timezone, h.LastNotifiedDate).Scan(&h.ID, &h.SeqNum, &h.CreatedAt)
	if err != nil {
		return fmt.Errorf("error creating personal habit: %w", err)
	}
	return nil
}

func (r *PersonalHabitRepository) GetByID(ctx context.Context, id string) (*domain.PersonalHabit, error) {
	query := `SELECT id, user_id, activity_type, scheduled_days, progress_status, reminder_time, timezone, last_notified_date, seq_num, created_at 
	          FROM personal_habits WHERE id = $1`
	var h domain.PersonalHabit
	err := r.pool.QueryRow(ctx, query, id).Scan(&h.ID, &h.UserID, &h.ActivityType, &h.ScheduledDays, &h.ProgressStatus, &h.ReminderTime, &h.Timezone, &h.LastNotifiedDate, &h.SeqNum, &h.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("personal habit not found")
		}
		return nil, fmt.Errorf("error getting personal habit: %w", err)
	}
	return &h, nil
}

func (r *PersonalHabitRepository) GetBySeqNum(ctx context.Context, userID string, seqNum int) (*domain.PersonalHabit, error) {
	query := `SELECT id, user_id, activity_type, scheduled_days, progress_status, reminder_time, timezone, last_notified_date, seq_num, created_at 
	          FROM personal_habits WHERE user_id = $1 AND seq_num = $2`
	var h domain.PersonalHabit
	err := r.pool.QueryRow(ctx, query, userID, seqNum).Scan(&h.ID, &h.UserID, &h.ActivityType, &h.ScheduledDays, &h.ProgressStatus, &h.ReminderTime, &h.Timezone, &h.LastNotifiedDate, &h.SeqNum, &h.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("personal habit not found by sequence number %d", seqNum)
		}
		return nil, fmt.Errorf("error getting personal habit by seq_num: %w", err)
	}
	return &h, nil
}

func (r *PersonalHabitRepository) Update(ctx context.Context, h *domain.PersonalHabit) error {
	query := `UPDATE personal_habits SET activity_type = $1, scheduled_days = $2, progress_status = $3, reminder_time = $4, timezone = $5, last_notified_date = $6 WHERE id = $7`
	_, err := r.pool.Exec(ctx, query, h.ActivityType, h.ScheduledDays, h.ProgressStatus, h.ReminderTime, h.Timezone, h.LastNotifiedDate, h.ID)
	if err != nil {
		return fmt.Errorf("error updating personal habit: %w", err)
	}
	return nil
}

func (r *PersonalHabitRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.PersonalHabit, error) {
	query := `SELECT id, user_id, activity_type, scheduled_days, progress_status, reminder_time, timezone, last_notified_date, seq_num, created_at 
	          FROM personal_habits WHERE user_id = $1 ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("error querying personal habits: %w", err)
	}
	defer rows.Close()

	var habits []*domain.PersonalHabit
	for rows.Next() {
		var h domain.PersonalHabit
		if err := rows.Scan(&h.ID, &h.UserID, &h.ActivityType, &h.ScheduledDays, &h.ProgressStatus, &h.ReminderTime, &h.Timezone, &h.LastNotifiedDate, &h.SeqNum, &h.CreatedAt); err != nil {
			return nil, err
		}
		habits = append(habits, &h)
	}
	return habits, nil
}

func (r *PersonalHabitRepository) GetHabitsForNotification(ctx context.Context) ([]*domain.PersonalHabit, error) {
	query := `SELECT id, user_id, activity_type, scheduled_days, progress_status, reminder_time, timezone, last_notified_date, seq_num, created_at 
	          FROM personal_habits WHERE reminder_time IS NOT NULL AND reminder_time != ''`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error querying habits for notification: %w", err)
	}
	defer rows.Close()

	var habits []*domain.PersonalHabit
	for rows.Next() {
		var h domain.PersonalHabit
		if err := rows.Scan(&h.ID, &h.UserID, &h.ActivityType, &h.ScheduledDays, &h.ProgressStatus, &h.ReminderTime, &h.Timezone, &h.LastNotifiedDate, &h.SeqNum, &h.CreatedAt); err != nil {
			return nil, err
		}
		habits = append(habits, &h)
	}
	return habits, nil
}

func (r *PersonalHabitRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM personal_habits WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("error deleting personal habit: %w", err)
	}
	return nil
}

// AIContextRepository implementation
type AIContextRepository struct {
	pool *pgxpool.Pool
}

func NewAIContextRepository(pool *pgxpool.Pool) *AIContextRepository {
	return &AIContextRepository{pool: pool}
}

func (r *AIContextRepository) GetByUserID(ctx context.Context, userID string) (*domain.AIContext, error) {
	query := `SELECT id, user_id, condensed_history, last_updated FROM ai_context WHERE user_id = $1`
	var ai domain.AIContext
	err := r.pool.QueryRow(ctx, query, userID).Scan(&ai.ID, &ai.UserID, &ai.CondensedHistory, &ai.LastUpdated)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("ai context not found")
		}
		return nil, fmt.Errorf("error getting ai context: %w", err)
	}
	return &ai, nil
}

func (r *AIContextRepository) CreateOrUpdate(ctx context.Context, ai *domain.AIContext) error {
	query := `INSERT INTO ai_context (user_id, condensed_history, last_updated) 
	          VALUES ($1, $2, CURRENT_TIMESTAMP)
	          ON CONFLICT (user_id) 
	          DO UPDATE SET condensed_history = EXCLUDED.condensed_history, last_updated = CURRENT_TIMESTAMP
	          RETURNING id, last_updated`
	err := r.pool.QueryRow(ctx, query, ai.UserID, ai.CondensedHistory).Scan(&ai.ID, &ai.LastUpdated)
	if err != nil {
		return fmt.Errorf("error inserting/updating ai context: %w", err)
	}
	return nil
}

// HabitLogRepository implementation
type HabitLogRepository struct {
	pool *pgxpool.Pool
}

func NewHabitLogRepository(pool *pgxpool.Pool) *HabitLogRepository {
	return &HabitLogRepository{pool: pool}
}

func (r *HabitLogRepository) CreateLog(ctx context.Context, l *domain.HabitLog) error {
	query := `INSERT INTO habit_logs (habit_id, logged_date, status) 
	          VALUES ($1, $2, $3)
	          ON CONFLICT (habit_id, logged_date) DO UPDATE SET status = EXCLUDED.status
	          RETURNING id, created_at`
	err := r.pool.QueryRow(ctx, query, l.HabitID, l.LoggedDate, l.Status).Scan(&l.ID, &l.CreatedAt)
	if err != nil {
		return fmt.Errorf("error creating/updating habit log: %w", err)
	}
	return nil
}

func (r *HabitLogRepository) GetLogsByHabitID(ctx context.Context, habitID string) ([]*domain.HabitLog, error) {
	query := `SELECT id, habit_id, logged_date, status, created_at 
	          FROM habit_logs WHERE habit_id = $1 ORDER BY logged_date DESC`
	rows, err := r.pool.Query(ctx, query, habitID)
	if err != nil {
		return nil, fmt.Errorf("error querying habit logs: %w", err)
	}
	defer rows.Close()

	var logs []*domain.HabitLog
	for rows.Next() {
		var l domain.HabitLog
		if err := rows.Scan(&l.ID, &l.HabitID, &l.LoggedDate, &l.Status, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, &l)
	}
	return logs, nil
}

// StoreRepository implementation
type StoreRepository struct {
	pool *pgxpool.Pool
}

func NewStoreRepository(pool *pgxpool.Pool) *StoreRepository {
	return &StoreRepository{pool: pool}
}

func (r *StoreRepository) Create(ctx context.Context, s *domain.Store) error {
	query := `INSERT INTO business_stores (user_id, store_name) VALUES ($1, $2) RETURNING id, created_at`
	err := r.pool.QueryRow(ctx, query, s.UserID, s.StoreName).Scan(&s.ID, &s.CreatedAt)
	if err != nil {
		return fmt.Errorf("error creating business store: %w", err)
	}
	return nil
}

func (r *StoreRepository) GetByID(ctx context.Context, id string) (*domain.Store, error) {
	query := `SELECT id, user_id, store_name, created_at FROM business_stores WHERE id = $1`
	var s domain.Store
	err := r.pool.QueryRow(ctx, query, id).Scan(&s.ID, &s.UserID, &s.StoreName, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("tienda no encontrada")
		}
		return nil, fmt.Errorf("error getting store by id: %w", err)
	}
	return &s, nil
}

func (r *StoreRepository) GetByUserID(ctx context.Context, userID string) ([]*domain.Store, error) {
	query := `SELECT id, user_id, store_name, created_at FROM business_stores WHERE user_id = $1 ORDER BY created_at DESC`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("error querying stores by user id: %w", err)
	}
	defer rows.Close()

	var stores []*domain.Store
	for rows.Next() {
		var s domain.Store
		if err := rows.Scan(&s.ID, &s.UserID, &s.StoreName, &s.CreatedAt); err != nil {
			return nil, err
		}
		stores = append(stores, &s)
	}
	return stores, nil
}

func (r *StoreRepository) GetByNameAndUser(ctx context.Context, name string, userID string) (*domain.Store, error) {
	query := `SELECT id, user_id, store_name, created_at FROM business_stores WHERE LOWER(store_name) = LOWER($1) AND user_id = $2`
	var s domain.Store
	err := r.pool.QueryRow(ctx, query, name, userID).Scan(&s.ID, &s.UserID, &s.StoreName, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("tienda no encontrada")
		}
		return nil, fmt.Errorf("error getting store by name and user: %w", err)
	}
	return &s, nil
}

func (r *StoreRepository) AddCollaborator(ctx context.Context, c *domain.StoreCollaborator) error {
	query := `INSERT INTO store_collaborators (store_id, user_id, role) VALUES ($1, $2, $3)
	          ON CONFLICT (store_id, user_id) DO UPDATE SET role = EXCLUDED.role`
	_, err := r.pool.Exec(ctx, query, c.StoreID, c.UserID, c.Role)
	if err != nil {
		return fmt.Errorf("error adding collaborator: %w", err)
	}
	return nil
}

func (r *StoreRepository) GetCollaborators(ctx context.Context, storeID string) ([]*domain.StoreCollaborator, error) {
	query := `SELECT store_id, user_id, role, created_at FROM store_collaborators WHERE store_id = $1`
	rows, err := r.pool.Query(ctx, query, storeID)
	if err != nil {
		return nil, fmt.Errorf("error getting collaborators: %w", err)
	}
	defer rows.Close()

	var list []*domain.StoreCollaborator
	for rows.Next() {
		var c domain.StoreCollaborator
		if err := rows.Scan(&c.StoreID, &c.UserID, &c.Role, &c.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, &c)
	}
	return list, nil
}

func (r *StoreRepository) DeleteCollaborator(ctx context.Context, storeID string, userID string) error {
	query := `DELETE FROM store_collaborators WHERE store_id = $1 AND user_id = $2`
	_, err := r.pool.Exec(ctx, query, storeID, userID)
	if err != nil {
		return fmt.Errorf("error deleting collaborator: %w", err)
	}
	return nil
}

func (r *StoreRepository) IsCollaborator(ctx context.Context, storeID string, userID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM store_collaborators WHERE store_id = $1 AND user_id = $2)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, storeID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("error checking collaborator status: %w", err)
	}
	return exists, nil
}

func (r *StoreRepository) GetSharedByUserID(ctx context.Context, userID string) ([]*domain.Store, error) {
	query := `SELECT s.id, s.user_id, s.store_name, s.created_at 
	          FROM business_stores s
	          JOIN store_collaborators c ON s.id = c.store_id
	          WHERE c.user_id = $1
	          ORDER BY s.store_name ASC`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("error querying shared stores: %w", err)
	}
	defer rows.Close()

	var stores []*domain.Store
	for rows.Next() {
		var s domain.Store
		if err := rows.Scan(&s.ID, &s.UserID, &s.StoreName, &s.CreatedAt); err != nil {
			return nil, err
		}
		stores = append(stores, &s)
	}
	return stores, nil
}


// ProductRepository implementation
type ProductRepository struct {
	pool *pgxpool.Pool
}

func NewProductRepository(pool *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{pool: pool}
}

func (r *ProductRepository) Create(ctx context.Context, p *domain.Product) error {
	query := `INSERT INTO store_products (store_id, name, price, stock, seq_num) 
	          VALUES ($1, $2, $3, $4, (SELECT COALESCE(MAX(seq_num), 0) + 1 FROM store_products WHERE store_id = $1)) 
	          RETURNING id, seq_num, created_at`
	err := r.pool.QueryRow(ctx, query, p.StoreID, p.Name, p.Price, p.Stock).Scan(&p.ID, &p.SeqNum, &p.CreatedAt)
	if err != nil {
		return fmt.Errorf("error creating product: %w", err)
	}
	return nil
}

func (r *ProductRepository) GetByID(ctx context.Context, id string) (*domain.Product, error) {
	query := `SELECT id, store_id, name, price, stock, seq_num, created_at FROM store_products WHERE id = $1`
	var p domain.Product
	err := r.pool.QueryRow(ctx, query, id).Scan(&p.ID, &p.StoreID, &p.Name, &p.Price, &p.Stock, &p.SeqNum, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("producto no encontrado")
		}
		return nil, fmt.Errorf("error getting product by id: %w", err)
	}
	return &p, nil
}

func (r *ProductRepository) GetBySeqNum(ctx context.Context, storeID string, seqNum int) (*domain.Product, error) {
	query := `SELECT id, store_id, name, price, stock, seq_num, created_at 
	          FROM store_products WHERE store_id = $1 AND seq_num = $2`
	var p domain.Product
	err := r.pool.QueryRow(ctx, query, storeID, seqNum).Scan(&p.ID, &p.StoreID, &p.Name, &p.Price, &p.Stock, &p.SeqNum, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("producto no encontrado por seq_num")
		}
		return nil, fmt.Errorf("error getting product by seq_num: %w", err)
	}
	return &p, nil
}

func (r *ProductRepository) GetByStoreID(ctx context.Context, storeID string) ([]*domain.Product, error) {
	query := `SELECT id, store_id, name, price, stock, seq_num, created_at FROM store_products WHERE store_id = $1 ORDER BY seq_num ASC`
	rows, err := r.pool.Query(ctx, query, storeID)
	if err != nil {
		return nil, fmt.Errorf("error querying products by store: %w", err)
	}
	defer rows.Close()

	var products []*domain.Product
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(&p.ID, &p.StoreID, &p.Name, &p.Price, &p.Stock, &p.SeqNum, &p.CreatedAt); err != nil {
			return nil, err
		}
		products = append(products, &p)
	}
	return products, nil
}

func (r *ProductRepository) Update(ctx context.Context, p *domain.Product) error {
	query := `UPDATE store_products SET name = $1, price = $2, stock = $3 WHERE id = $4`
	_, err := r.pool.Exec(ctx, query, p.Name, p.Price, p.Stock, p.ID)
	if err != nil {
		return fmt.Errorf("error updating product: %w", err)
	}
	return nil
}

func (r *ProductRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM store_products WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("error deleting product: %w", err)
	}
	return nil
}

// OrderRepository implementation
type OrderRepository struct {
	pool *pgxpool.Pool
}

func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{pool: pool}
}

func (r *OrderRepository) Create(ctx context.Context, o *domain.Order) error {
	query := `INSERT INTO store_orders (store_id, client_name, client_phone, product_details, total_cost, advance_payment, shipping_address, shipping_cost, status, seq_num) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, (SELECT COALESCE(MAX(seq_num), 0) + 1 FROM store_orders WHERE store_id = $1)) 
	          RETURNING id, seq_num, created_at`
	err := r.pool.QueryRow(ctx, query, o.StoreID, o.ClientName, o.ClientPhone, o.ProductDetails, o.TotalCost, o.AdvancePayment, o.ShippingAddress, o.ShippingCost, o.Status).Scan(&o.ID, &o.SeqNum, &o.CreatedAt)
	if err != nil {
		return fmt.Errorf("error creating store order: %w", err)
	}
	return nil
}

func (r *OrderRepository) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	query := `SELECT id, store_id, client_name, client_phone, product_details, total_cost, advance_payment, shipping_address, shipping_cost, status, seq_num, created_at FROM store_orders WHERE id = $1`
	var o domain.Order
	err := r.pool.QueryRow(ctx, query, id).Scan(&o.ID, &o.StoreID, &o.ClientName, &o.ClientPhone, &o.ProductDetails, &o.TotalCost, &o.AdvancePayment, &o.ShippingAddress, &o.ShippingCost, &o.Status, &o.SeqNum, &o.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("pedido no encontrado")
		}
		return nil, fmt.Errorf("error getting order by id: %w", err)
	}
	return &o, nil
}

func (r *OrderRepository) GetBySeqNum(ctx context.Context, storeID string, seqNum int) (*domain.Order, error) {
	query := `SELECT id, store_id, client_name, client_phone, product_details, total_cost, advance_payment, shipping_address, shipping_cost, status, seq_num, created_at 
	          FROM store_orders WHERE store_id = $1 AND seq_num = $2`
	var o domain.Order
	err := r.pool.QueryRow(ctx, query, storeID, seqNum).Scan(&o.ID, &o.StoreID, &o.ClientName, &o.ClientPhone, &o.ProductDetails, &o.TotalCost, &o.AdvancePayment, &o.ShippingAddress, &o.ShippingCost, &o.Status, &o.SeqNum, &o.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("pedido no encontrado por seq_num")
		}
		return nil, fmt.Errorf("error getting order by seq_num: %w", err)
	}
	return &o, nil
}

func (r *OrderRepository) GetByStoreID(ctx context.Context, storeID string) ([]*domain.Order, error) {
	query := `SELECT id, store_id, client_name, client_phone, product_details, total_cost, advance_payment, shipping_address, shipping_cost, status, seq_num, created_at FROM store_orders WHERE store_id = $1 ORDER BY seq_num DESC`
	rows, err := r.pool.Query(ctx, query, storeID)
	if err != nil {
		return nil, fmt.Errorf("error querying orders by store: %w", err)
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		var o domain.Order
		if err := rows.Scan(&o.ID, &o.StoreID, &o.ClientName, &o.ClientPhone, &o.ProductDetails, &o.TotalCost, &o.AdvancePayment, &o.ShippingAddress, &o.ShippingCost, &o.Status, &o.SeqNum, &o.CreatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, &o)
	}
	return orders, nil
}

func (r *OrderRepository) GetPendingByStoreID(ctx context.Context, storeID string) ([]*domain.Order, error) {
	query := `SELECT id, store_id, client_name, client_phone, product_details, total_cost, advance_payment, shipping_address, shipping_cost, status, seq_num, created_at FROM store_orders WHERE store_id = $1 AND status != 'completed' ORDER BY seq_num DESC`
	rows, err := r.pool.Query(ctx, query, storeID)
	if err != nil {
		return nil, fmt.Errorf("error querying pending orders: %w", err)
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		var o domain.Order
		if err := rows.Scan(&o.ID, &o.StoreID, &o.ClientName, &o.ClientPhone, &o.ProductDetails, &o.TotalCost, &o.AdvancePayment, &o.ShippingAddress, &o.ShippingCost, &o.Status, &o.SeqNum, &o.CreatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, &o)
	}
	return orders, nil
}

func (r *OrderRepository) Update(ctx context.Context, o *domain.Order) error {
	query := `UPDATE store_orders SET client_name = $1, client_phone = $2, product_details = $3, total_cost = $4, advance_payment = $5, shipping_address = $6, shipping_cost = $7, status = $8 WHERE id = $9`
	_, err := r.pool.Exec(ctx, query, o.ClientName, o.ClientPhone, o.ProductDetails, o.TotalCost, o.AdvancePayment, o.ShippingAddress, o.ShippingCost, o.Status, o.ID)
	if err != nil {
		return fmt.Errorf("error updating order: %w", err)
	}
	return nil
}

func (r *OrderRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM store_orders WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("error deleting order: %w", err)
	}
	return nil
}
