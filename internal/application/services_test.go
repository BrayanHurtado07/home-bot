package application

import (
	"testing"
	"time"

	"github.com/iloremstudio/home-bot/internal/domain"
)

func parseTime(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func TestCalculateStreak(t *testing.T) {
	today := parseTime("2026-06-08")

	tests := []struct {
		name     string
		logs     []*domain.HabitLog
		expected int
	}{
		{
			name:     "empty logs",
			logs:     []*domain.HabitLog{},
			expected: 0,
		},
		{
			name: "completed today only",
			logs: []*domain.HabitLog{
				{LoggedDate: parseTime("2026-06-08"), Status: "completed"},
			},
			expected: 1,
		},
		{
			name: "completed yesterday only",
			logs: []*domain.HabitLog{
				{LoggedDate: parseTime("2026-06-07"), Status: "completed"},
			},
			expected: 1,
		},
		{
			name: "completed today and yesterday",
			logs: []*domain.HabitLog{
				{LoggedDate: parseTime("2026-06-08"), Status: "completed"},
				{LoggedDate: parseTime("2026-06-07"), Status: "completed"},
			},
			expected: 2,
		},
		{
			name: "completed yesterday and day before",
			logs: []*domain.HabitLog{
				{LoggedDate: parseTime("2026-06-07"), Status: "completed"},
				{LoggedDate: parseTime("2026-06-06"), Status: "completed"},
			},
			expected: 2,
		},
		{
			name: "gap today and yesterday",
			logs: []*domain.HabitLog{
				{LoggedDate: parseTime("2026-06-05"), Status: "completed"},
				{LoggedDate: parseTime("2026-06-04"), Status: "completed"},
			},
			expected: 0,
		},
		{
			name: "gap in the middle",
			logs: []*domain.HabitLog{
				{LoggedDate: parseTime("2026-06-08"), Status: "completed"},
				{LoggedDate: parseTime("2026-06-07"), Status: "completed"},
				{LoggedDate: parseTime("2026-06-05"), Status: "completed"}, // missing 2026-06-06
			},
			expected: 2,
		},
		{
			name: "skipped log ignored in streak",
			logs: []*domain.HabitLog{
				{LoggedDate: parseTime("2026-06-08"), Status: "completed"},
				{LoggedDate: parseTime("2026-06-07"), Status: "skipped"},
				{LoggedDate: parseTime("2026-06-06"), Status: "completed"},
			},
			expected: 1, // broken due to skipped status
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateStreak(tt.logs, today)
			if got != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, got)
			}
		})
	}
}

func TestNormalizeDays(t *testing.T) {
	input := " Lunes, Miércoles , Sábado, todos "
	expected := []string{"lunes", "miercoles", "sabado", "todos"}

	got := normalizeDays(input)
	if len(got) != len(expected) {
		t.Fatalf("expected len %d, got %d", len(expected), len(got))
	}

	for i, v := range got {
		if v != expected[i] {
			t.Errorf("at index %d: expected %s, got %s", i, expected[i], v)
		}
	}
}
