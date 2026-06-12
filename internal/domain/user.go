package domain

import (
	"context"
	"errors"
)

var (
	ErrUserNotFound   = errors.New("usuario no encontrado")
	ErrGroupNotFound  = errors.New("grupo no encontrado")
	ErrUnauthorized   = errors.New("no autorizado para esta acción")
)

type User struct {
	ID         string
	TelegramID int64
	TenantID   *string // Can be nil
	Role       string  // "admin" or "roomie"
	Name       string
}

type TenantGroup struct {
	ID        string
	GroupName string
}

type UserRepository interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByTelegramID(ctx context.Context, tgID int64) (*User, error)
	Update(ctx context.Context, u *User) error
	GetUsersByTenantID(ctx context.Context, tenantID string) ([]*User, error)
}

type TenantGroupRepository interface {
	Create(ctx context.Context, g *TenantGroup) error
	GetByID(ctx context.Context, id string) (*TenantGroup, error)
}
