package application

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/iloremstudio/home-bot/internal/domain"
)

type NotificationSender interface {
	SendNotification(tgUserID int64, text string)
}

type Scheduler struct {
	habitRepo domain.PersonalHabitRepository
	userRepo  domain.UserRepository
	sender    NotificationSender
}

func NewScheduler(habitRepo domain.PersonalHabitRepository, userRepo domain.UserRepository, sender NotificationSender) *Scheduler {
	return &Scheduler{
		habitRepo: habitRepo,
		userRepo:  userRepo,
		sender:    sender,
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	log.Println("Iniciando planificador de alertas en segundo plano (Scheduler)...")

	// Run check immediately on start, then every 1 minute
	s.checkAndNotify(ctx)

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Deteniendo el planificador de alertas...")
			return
		case <-ticker.C:
			s.checkAndNotify(ctx)
		}
	}
}

func (s *Scheduler) checkAndNotify(ctx context.Context) {
	habits, err := s.habitRepo.GetHabitsForNotification(ctx)
	if err != nil {
		log.Printf("[Scheduler Error] No se pudieron obtener los hábitos: %v", err)
		return
	}

	weekdaySp := map[time.Weekday]string{
		time.Monday:    "lunes",
		time.Tuesday:   "martes",
		time.Wednesday: "miercoles",
		time.Thursday:  "jueves",
		time.Friday:    "viernes",
		time.Saturday:  "sabado",
		time.Sunday:    "domingo",
	}

	for _, h := range habits {
		if h.ReminderTime == nil || *h.ReminderTime == "" {
			continue
		}

		// Load user to get Telegram ID
		user, err := s.userRepo.GetByID(ctx, h.UserID)
		if err != nil {
			log.Printf("[Scheduler Warning] No se pudo cargar usuario ID %s para el hábito ID %s: %v", h.UserID, h.ID, err)
			continue
		}

		// Parse timezone
		tz := h.Timezone
		if tz == "" {
			tz = "America/Bogota"
		}
		loc, err := time.LoadLocation(tz)
		if err != nil {
			// Fallback to UTC if timezone is invalid
			loc = time.UTC
		}

		nowInTZ := time.Now().In(loc)
		todayStr := nowInTZ.Format("2006-01-02")

		// 1. Check if already notified today
		if h.LastNotifiedDate != nil && h.LastNotifiedDate.Format("2006-01-02") == todayStr {
			continue
		}

		// 2. Check if current time (HH:MM) matches reminder_time
		currentTimeStr := nowInTZ.Format("15:04")
		if currentTimeStr != *h.ReminderTime {
			continue
		}

		// 3. Check if current day of week matches scheduled_days
		currentDaySp := weekdaySp[nowInTZ.Weekday()]
		scheduledDaysNormalized := normalizeDays(h.ScheduledDays)

		matchesDay := false
		for _, d := range scheduledDaysNormalized {
			if d == currentDaySp || d == "todos" || d == "diario" {
				matchesDay = true
				break
			}
		}

		if !matchesDay {
			continue
		}

		// Send notification
		log.Printf("[Scheduler] Enviando alerta de hábito '%s' para usuario %s (TelegramID: %d)", h.ActivityType, user.Name, user.TelegramID)
		alertText := "⏰ *Recordatorio de Disciplina:*\n"
		alertText += "Es hora de realizar tu hábito: *" + h.ActivityType + "*\n"
		alertText += "¡Mantén tu racha activa! Escribe /habitos para registrar tu avance."

		s.sender.SendNotification(user.TelegramID, alertText)

		// Update last_notified_date
		notifiedTime := nowInTZ
		h.LastNotifiedDate = &notifiedTime
		_ = s.habitRepo.Update(ctx, h)
	}
}

func normalizeDays(scheduledDays string) []string {
	parts := strings.Split(scheduledDays, ",")
	normalized := make([]string, 0, len(parts))
	for _, p := range parts {
		normalized = append(normalized, normalizeString(p))
	}
	return normalized
}

func normalizeString(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "á", "a")
	s = strings.ReplaceAll(s, "é", "e")
	s = strings.ReplaceAll(s, "í", "i")
	s = strings.ReplaceAll(s, "ó", "o")
	s = strings.ReplaceAll(s, "ú", "u")
	return strings.TrimSpace(s)
}
