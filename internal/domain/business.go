package domain

import (
	"context"
	"time"
)

type Store struct {
	ID        string
	UserID    string
	StoreName string
	CreatedAt time.Time
}

type Product struct {
	ID        string
	StoreID   string
	Name      string
	Price     float64
	Stock     int // -1 for infinite / custom made
	SeqNum    int
	CreatedAt time.Time
}

type Order struct {
	ID              string
	StoreID         string
	ClientName      string
	ClientPhone     *string
	ProductDetails  string
	TotalCost       float64
	AdvancePayment  float64
	ShippingAddress *string
	ShippingCost    float64
	Status          string // "advance_paid", "pending_shipment", "shipped", "delivered", "completed"
	SeqNum          int
	CreatedAt       time.Time
}

type StoreRepository interface {
	Create(ctx context.Context, s *Store) error
	GetByID(ctx context.Context, id string) (*Store, error)
	GetByUserID(ctx context.Context, userID string) ([]*Store, error)
	GetByNameAndUser(ctx context.Context, name string, userID string) (*Store, error)
}

type ProductRepository interface {
	Create(ctx context.Context, p *Product) error
	GetByID(ctx context.Context, id string) (*Product, error)
	GetBySeqNum(ctx context.Context, storeID string, seqNum int) (*Product, error)
	GetByStoreID(ctx context.Context, storeID string) ([]*Product, error)
	Update(ctx context.Context, p *Product) error
	Delete(ctx context.Context, id string) error
}

type OrderRepository interface {
	Create(ctx context.Context, o *Order) error
	GetByID(ctx context.Context, id string) (*Order, error)
	GetBySeqNum(ctx context.Context, storeID string, seqNum int) (*Order, error)
	GetByStoreID(ctx context.Context, storeID string) ([]*Order, error)
	GetPendingByStoreID(ctx context.Context, storeID string) ([]*Order, error)
	Update(ctx context.Context, o *Order) error
	Delete(ctx context.Context, id string) error
}
