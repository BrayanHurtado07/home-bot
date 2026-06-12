package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/iloremstudio/home-bot/internal/application"
	"github.com/iloremstudio/home-bot/internal/config"
	"github.com/iloremstudio/home-bot/internal/infrastructure/db"
	"github.com/iloremstudio/home-bot/internal/infrastructure/groq"
	"github.com/iloremstudio/home-bot/internal/infrastructure/telegram"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	log.Println("Iniciando backend del Bot de Gestión del Hogar y Disciplina...")

	// 1. Load Configurations
	cfg := config.Load()
	if cfg.TelegramBotToken == "" {
		log.Fatal("ERROR: La variable de entorno TELEGRAM_BOT_TOKEN no está configurada.")
	}
	if cfg.GroqAPIKey == "" {
		log.Fatal("ERROR: La variable de entorno GROQ_API_KEY no está configurada.")
	}

	// 2. Connect to CockroachDB / PostgreSQL
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("Conectando a base de datos CockroachDB...")
	dbConfig, err := pgxpool.ParseConfig(cfg.CockroachDBURL)
	if err != nil {
		log.Fatalf("Error al parsear COCKROACH_DB_URL: %v", err)
	}

	// Set connection pool settings
	dbConfig.MaxConns = 10
	dbConfig.MinConns = 2
	dbConfig.MaxConnLifetime = 30 * time.Minute

	dbPool, err := pgxpool.NewWithConfig(ctx, dbConfig)
	if err != nil {
		log.Fatalf("Error al crear el pool de conexiones: %v", err)
	}
	defer dbPool.Close()

	// Ping database to verify connection
	if err := dbPool.Ping(ctx); err != nil {
		log.Fatalf("No se pudo conectar a CockroachDB: %v", err)
	}
	log.Println("Conectado a CockroachDB exitosamente.")

	// 3. Run Auto-migrations
	if err := db.RunMigrations(ctx, dbPool); err != nil {
		log.Fatalf("Error en las migraciones de DB: %v", err)
	}

	// 4. Initialize Repositories
	userRepo := db.NewUserRepository(dbPool)
	groupRepo := db.NewTenantGroupRepository(dbPool)
	paymentRepo := db.NewPaymentRepository(dbPool)
	taskRepo := db.NewHouseTaskRepository(dbPool)
	mealRepo := db.NewMealScheduleRepository(dbPool)
	habitRepo := db.NewPersonalHabitRepository(dbPool)
	habitLogRepo := db.NewHabitLogRepository(dbPool)
	aiRepo := db.NewAIContextRepository(dbPool)

	// 5. Initialize Groq API Client
	groqClient := groq.NewClient(cfg.GroqAPIKey)

	// 6. Initialize Application Services
	appService := application.NewAppService(
		userRepo,
		groupRepo,
		paymentRepo,
		taskRepo,
		mealRepo,
		habitRepo,
		habitLogRepo,
		aiRepo,
		groqClient,
	)

	// 7. Initialize Telegram Bot
	bot, err := telegram.NewBot(cfg.TelegramBotToken, appService)
	if err != nil {
		log.Fatalf("Error al inicializar el bot de Telegram: %v", err)
	}

	// 8. Initialize and start Notification Scheduler in background
	sched := application.NewScheduler(habitRepo, userRepo, bot)
	go sched.Start(ctx)

	// 9. Start Telegram Bot in a goroutine
	go bot.Start(ctx)

	log.Println("El bot está corriendo. Presiona Ctrl+C para salir.")

	// Wait for shutdown signal
	<-ctx.Done()
	log.Println("Apagando el servidor con gracia (Graceful Shutdown)...")
	
	// Give database and connections time to close cleanly
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	dbPool.Close()
	<-shutdownCtx.Done()
	
	log.Println("Servidor apagado con éxito.")
}
