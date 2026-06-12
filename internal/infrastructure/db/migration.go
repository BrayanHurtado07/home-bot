package db

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

const InitSchemaSQL = `
CREATE TABLE IF NOT EXISTS tenants_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_name VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    telegram_id INT8 UNIQUE NOT NULL,
    tenant_id UUID REFERENCES tenants_groups(id) ON DELETE SET NULL,
    role VARCHAR(20) NOT NULL CHECK (role IN ('admin', 'roomie')),
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants_groups(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount NUMERIC(10, 2) NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'approved', 'rejected')),
    proof_url TEXT,
    billing_date DATE NOT NULL,
    seq_num INT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uniq_tenant_payment_seq UNIQUE(tenant_id, seq_num)
);

CREATE TABLE IF NOT EXISTS house_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants_groups(id) ON DELETE CASCADE,
    description VARCHAR(255) NOT NULL,
    assigned_to UUID REFERENCES users(id) ON DELETE SET NULL,
    due_date DATE,
    is_done BOOLEAN DEFAULT FALSE,
    seq_num INT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uniq_tenant_task_seq UNIQUE(tenant_id, seq_num)
);

CREATE TABLE IF NOT EXISTS meal_schedule (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants_groups(id) ON DELETE CASCADE,
    day_of_week INT NOT NULL CHECK (day_of_week BETWEEN 1 AND 7),
    meal_type VARCHAR(20) NOT NULL CHECK (meal_type IN ('breakfast', 'lunch', 'dinner')),
    chef_id UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uniq_tenant_day_meal UNIQUE(tenant_id, day_of_week, meal_type)
);

CREATE TABLE IF NOT EXISTS personal_habits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    activity_type VARCHAR(50) NOT NULL,
    scheduled_days VARCHAR(100),
    progress_status TEXT,
    reminder_time VARCHAR(5),
    timezone VARCHAR(50) DEFAULT 'America/Bogota',
    last_notified_date DATE,
    seq_num INT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uniq_user_habit_seq UNIQUE(user_id, seq_num)
);

CREATE TABLE IF NOT EXISTS ai_context (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    condensed_history TEXT NOT NULL,
    last_updated TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS habit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    habit_id UUID NOT NULL REFERENCES personal_habits(id) ON DELETE CASCADE,
    logged_date DATE NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('completed', 'skipped')),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uniq_habit_day UNIQUE(habit_id, logged_date)
);

CREATE TABLE IF NOT EXISTS business_stores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    store_name VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS store_products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id UUID NOT NULL REFERENCES business_stores(id) ON DELETE CASCADE,
    name VARCHAR(150) NOT NULL,
    price NUMERIC(10, 2) NOT NULL,
    stock INT NOT NULL DEFAULT -1,
    seq_num INT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uniq_store_product_seq UNIQUE(store_id, seq_num)
);

CREATE TABLE IF NOT EXISTS store_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id UUID NOT NULL REFERENCES business_stores(id) ON DELETE CASCADE,
    client_name VARCHAR(100) NOT NULL,
    client_phone VARCHAR(30),
    product_details TEXT NOT NULL,
    total_cost NUMERIC(10, 2) NOT NULL,
    advance_payment NUMERIC(10, 2) NOT NULL DEFAULT 0.00,
    shipping_address TEXT,
    shipping_cost NUMERIC(10, 2) NOT NULL DEFAULT 0.00,
    status VARCHAR(30) NOT NULL CHECK (status IN ('advance_paid', 'pending_shipment', 'shipped', 'delivered', 'completed')),
    seq_num INT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uniq_store_order_seq UNIQUE(store_id, seq_num)
);
`

func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	log.Println("Ejecutando migraciones de base de datos...")
	_, err := pool.Exec(ctx, InitSchemaSQL)
	if err != nil {
		return fmt.Errorf("error al ejecutar las migraciones: %w", err)
	}
	log.Println("Migraciones completadas exitosamente.")
	return nil
}
