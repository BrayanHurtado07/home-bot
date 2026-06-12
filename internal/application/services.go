package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/iloremstudio/home-bot/internal/domain"
	"github.com/iloremstudio/home-bot/internal/infrastructure/groq"
)

type AppService struct {
	userRepo     domain.UserRepository
	groupRepo    domain.TenantGroupRepository
	paymentRepo  domain.PaymentRepository
	taskRepo     domain.HouseTaskRepository
	mealRepo     domain.MealScheduleRepository
	habitRepo    domain.PersonalHabitRepository
	habitLogRepo domain.HabitLogRepository
	aiRepo       domain.AIContextRepository
	storeRepo    domain.StoreRepository
	productRepo  domain.ProductRepository
	orderRepo    domain.OrderRepository
	groqClient   *groq.Client
}

func NewAppService(
	userRepo domain.UserRepository,
	groupRepo domain.TenantGroupRepository,
	paymentRepo domain.PaymentRepository,
	taskRepo domain.HouseTaskRepository,
	mealRepo domain.MealScheduleRepository,
	habitRepo domain.PersonalHabitRepository,
	habitLogRepo domain.HabitLogRepository,
	aiRepo domain.AIContextRepository,
	storeRepo domain.StoreRepository,
	productRepo domain.ProductRepository,
	orderRepo domain.OrderRepository,
	groqClient *groq.Client,
) *AppService {
	return &AppService{
		userRepo:     userRepo,
		groupRepo:    groupRepo,
		paymentRepo:  paymentRepo,
		taskRepo:     taskRepo,
		mealRepo:     mealRepo,
		habitRepo:    habitRepo,
		habitLogRepo: habitLogRepo,
		aiRepo:       aiRepo,
		storeRepo:    storeRepo,
		productRepo:  productRepo,
		orderRepo:    orderRepo,
		groqClient:   groqClient,
	}
}

// User & Tenant Management
func (s *AppService) RegisterUser(ctx context.Context, telegramID int64, name string) (*domain.User, error) {
	u, err := s.userRepo.GetByTelegramID(ctx, telegramID)
	if err == nil {
		return u, nil
	}

	role := "roomie"
	u = &domain.User{
		TelegramID: telegramID,
		Role:       role,
		Name:       name,
	}
	err = s.userRepo.Create(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("could not register user: %w", err)
	}
	return u, nil
}

func (s *AppService) JoinOrCreateGroup(ctx context.Context, tgUserID int64, groupName string) (*domain.TenantGroup, error) {
	u, err := s.userRepo.GetByTelegramID(ctx, tgUserID)
	if err != nil {
		return nil, err
	}

	g := &domain.TenantGroup{
		GroupName: groupName,
	}
	err = s.groupRepo.Create(ctx, g)
	if err != nil {
		return nil, err
	}

	u.TenantID = &g.ID
	u.Role = "admin"
	err = s.userRepo.Update(ctx, u)
	if err != nil {
		return nil, err
	}

	return g, nil
}

func (s *AppService) JoinExistingGroup(ctx context.Context, tgUserID int64, groupID string) error {
	u, err := s.userRepo.GetByTelegramID(ctx, tgUserID)
	if err != nil {
		return err
	}

	_, err = s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return fmt.Errorf("el grupo no existe: %w", err)
	}

	u.TenantID = &groupID
	u.Role = "roomie"
	return s.userRepo.Update(ctx, u)
}

func (s *AppService) GetGroupByID(ctx context.Context, groupID string) (*domain.TenantGroup, error) {
	return s.groupRepo.GetByID(ctx, groupID)
}

func (s *AppService) GetUsersInGroup(ctx context.Context, tgUserID int64) ([]*domain.User, error) {
	u, err := s.userRepo.GetByTelegramID(ctx, tgUserID)
	if err != nil {
		return nil, err
	}
	if u.TenantID == nil {
		return nil, fmt.Errorf("no perteneces a ningún grupo")
	}
	return s.userRepo.GetUsersByTenantID(ctx, *u.TenantID)
}

// Payments
func (s *AppService) CreatePayment(ctx context.Context, tgUserID int64, amount float64, billingDate time.Time) (*domain.Payment, error) {
	u, err := s.userRepo.GetByTelegramID(ctx, tgUserID)
	if err != nil {
		return nil, err
	}
	if u.TenantID == nil {
		return nil, fmt.Errorf("no perteneces a ningún grupo")
	}

	p := &domain.Payment{
		TenantID:    *u.TenantID,
		UserID:      u.ID,
		Amount:      amount,
		Status:      "pending",
		BillingDate: billingDate,
	}
	err = s.paymentRepo.Create(ctx, p)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *AppService) UploadPaymentProof(ctx context.Context, tgUserID int64, seqNum int, proofURL string) (*domain.Payment, error) {
	u, err := s.userRepo.GetByTelegramID(ctx, tgUserID)
	if err != nil {
		return nil, err
	}
	if u.TenantID == nil {
		return nil, fmt.Errorf("no perteneces a ningún grupo")
	}

	p, err := s.paymentRepo.GetBySeqNum(ctx, *u.TenantID, seqNum)
	if err != nil {
		return nil, err
	}

	if p.UserID != u.ID {
		return nil, domain.ErrUnauthorized
	}

	p.ProofURL = proofURL
	p.Status = "pending"
	err = s.paymentRepo.Update(ctx, p)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *AppService) ApprovePayment(ctx context.Context, tgAdminID int64, seqNum int) error {
	admin, err := s.userRepo.GetByTelegramID(ctx, tgAdminID)
	if err != nil {
		return err
	}
	if admin.Role != "admin" {
		return domain.ErrUnauthorized
	}
	if admin.TenantID == nil {
		return fmt.Errorf("no perteneces a ningún grupo")
	}

	p, err := s.paymentRepo.GetBySeqNum(ctx, *admin.TenantID, seqNum)
	if err != nil {
		return err
	}

	p.Status = "approved"
	return s.paymentRepo.Update(ctx, p)
}

func (s *AppService) RejectPayment(ctx context.Context, tgAdminID int64, seqNum int) error {
	admin, err := s.userRepo.GetByTelegramID(ctx, tgAdminID)
	if err != nil {
		return err
	}
	if admin.Role != "admin" {
		return domain.ErrUnauthorized
	}
	if admin.TenantID == nil {
		return fmt.Errorf("no perteneces a ningún grupo")
	}

	p, err := s.paymentRepo.GetBySeqNum(ctx, *admin.TenantID, seqNum)
	if err != nil {
		return err
	}

	p.Status = "rejected"
	return s.paymentRepo.Update(ctx, p)
}

func (s *AppService) GetPendingPayments(ctx context.Context, tgUserID int64) ([]*domain.Payment, error) {
	u, err := s.userRepo.GetByTelegramID(ctx, tgUserID)
	if err != nil {
		return nil, err
	}
	if u.TenantID == nil {
		return nil, fmt.Errorf("no perteneces a ningún grupo")
	}
	return s.paymentRepo.GetPendingByTenantID(ctx, *u.TenantID)
}

func (s *AppService) GetGroupPayments(ctx context.Context, tgUserID int64) ([]*domain.Payment, error) {
	u, err := s.userRepo.GetByTelegramID(ctx, tgUserID)
	if err != nil {
		return nil, err
	}
	if u.TenantID == nil {
		return nil, fmt.Errorf("no perteneces a ningún grupo")
	}
	return s.paymentRepo.GetByTenantID(ctx, *u.TenantID)
}

// House Tasks
func (s *AppService) CreateHouseTask(ctx context.Context, tgUserID int64, description string, assignedToTelegramID *int64, dueDate *time.Time) (*domain.HouseTask, error) {
	u, err := s.userRepo.GetByTelegramID(ctx, tgUserID)
	if err != nil {
		return nil, err
	}
	if u.TenantID == nil {
		return nil, fmt.Errorf("no perteneces a ningún grupo")
	}

	var assignedToID *string
	if assignedToTelegramID != nil {
		assignee, err := s.userRepo.GetByTelegramID(ctx, *assignedToTelegramID)
		if err != nil {
			return nil, fmt.Errorf("el usuario asignado no está registrado: %w", err)
		}
		if assignee.TenantID == nil || *assignee.TenantID != *u.TenantID {
			return nil, fmt.Errorf("el usuario asignado no pertenece a tu grupo")
		}
		assignedToID = &assignee.ID
	}

	t := &domain.HouseTask{
		TenantID:    *u.TenantID,
		Description: description,
		AssignedTo:  assignedToID,
		DueDate:     dueDate,
		IsDone:      false,
	}
	err = s.taskRepo.Create(ctx, t)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (s *AppService) CompleteHouseTask(ctx context.Context, tgUserID int64, seqNum int) error {
	u, err := s.userRepo.GetByTelegramID(ctx, tgUserID)
	if err != nil {
		return err
	}
	if u.TenantID == nil {
		return fmt.Errorf("no perteneces a ningún grupo")
	}

	t, err := s.taskRepo.GetBySeqNum(ctx, *u.TenantID, seqNum)
	if err != nil {
		return err
	}

	t.IsDone = true
	return s.taskRepo.Update(ctx, t)
}

func (s *AppService) DeleteHouseTask(ctx context.Context, tgUserID int64, seqNum int) error {
	u, err := s.userRepo.GetByTelegramID(ctx, tgUserID)
	if err != nil {
		return err
	}
	if u.TenantID == nil {
		return fmt.Errorf("no perteneces a ningún grupo")
	}

	t, err := s.taskRepo.GetBySeqNum(ctx, *u.TenantID, seqNum)
	if err != nil {
		return err
	}

	return s.taskRepo.Delete(ctx, t.ID)
}

func (s *AppService) GetPendingTasks(ctx context.Context, tgUserID int64) ([]*domain.HouseTask, error) {
	u, err := s.userRepo.GetByTelegramID(ctx, tgUserID)
	if err != nil {
		return nil, err
	}
	if u.TenantID == nil {
		return nil, fmt.Errorf("no perteneces a ningún grupo")
	}
	return s.taskRepo.GetPendingByTenantID(ctx, *u.TenantID)
}

// Meal Schedule
func (s *AppService) SetMealChef(ctx context.Context, tgUserID int64, day int, mealType string, chefTelegramID int64) error {
	u, err := s.userRepo.GetByTelegramID(ctx, tgUserID)
	if err != nil {
		return err
	}
	if u.TenantID == nil {
		return fmt.Errorf("no perteneces a ningún grupo")
	}

	chef, err := s.userRepo.GetByTelegramID(ctx, chefTelegramID)
	if err != nil {
		return fmt.Errorf("el chef no está registrado: %w", err)
	}

	if chef.TenantID == nil || *chef.TenantID != *u.TenantID {
		return fmt.Errorf("el chef no pertenece a tu grupo")
	}

	m := &domain.MealSchedule{
		TenantID:  *u.TenantID,
		DayOfWeek: day,
		MealType:  mealType,
		ChefID:    &chef.ID,
	}
	return s.mealRepo.CreateOrUpdate(ctx, m)
}

func (s *AppService) GetMealSchedule(ctx context.Context, tgUserID int64) ([]*domain.MealSchedule, error) {
	u, err := s.userRepo.GetByTelegramID(ctx, tgUserID)
	if err != nil {
		return nil, err
	}
	if u.TenantID == nil {
		return nil, fmt.Errorf("no perteneces a ningún grupo")
	}
	return s.mealRepo.GetByTenantID(ctx, *u.TenantID)
}

// Personal Habits
func (s *AppService) AddPersonalHabit(ctx context.Context, tgUserID int64, activityType string, scheduledDays string, reminderTime *string, timezone string) (*domain.PersonalHabit, error) {
	u, err := s.userRepo.GetByTelegramID(ctx, tgUserID)
	if err != nil {
		return nil, err
	}

	if timezone == "" {
		timezone = "America/Bogota"
	}

	h := &domain.PersonalHabit{
		UserID:         u.ID,
		ActivityType:   activityType,
		ScheduledDays:  scheduledDays,
		ProgressStatus: "Iniciado",
		ReminderTime:   reminderTime,
		Timezone:       timezone,
	}
	err = s.habitRepo.Create(ctx, h)
	if err != nil {
		return nil, err
	}
	return h, nil
}

func (s *AppService) LogHabitCheckIn(ctx context.Context, tgUserID int64, seqNum int, status string) error {
	u, err := s.userRepo.GetByTelegramID(ctx, tgUserID)
	if err != nil {
		return err
	}

	h, err := s.habitRepo.GetBySeqNum(ctx, u.ID, seqNum)
	if err != nil {
		return err
	}

	logEntry := &domain.HabitLog{
		HabitID:    h.ID,
		LoggedDate: time.Now().UTC(),
		Status:     status,
	}

	err = s.habitLogRepo.CreateLog(ctx, logEntry)
	if err != nil {
		return err
	}

	h.ProgressStatus = fmt.Sprintf("Completado Hoy (%s)", time.Now().Format("02/01"))
	return s.habitRepo.Update(ctx, h)
}

func (s *AppService) GetPersonalHabitsWithStreak(ctx context.Context, tgUserID int64) ([]*HabitWithStreak, error) {
	u, err := s.userRepo.GetByTelegramID(ctx, tgUserID)
	if err != nil {
		return nil, err
	}

	habits, err := s.habitRepo.GetByUserID(ctx, u.ID)
	if err != nil {
		return nil, err
	}

	var results []*HabitWithStreak
	today := time.Now().UTC()

	for _, h := range habits {
		logs, err := s.habitLogRepo.GetLogsByHabitID(ctx, h.ID)
		if err != nil {
			return nil, err
		}

		streak := CalculateStreak(logs, today)
		results = append(results, &HabitWithStreak{
			Habit:  h,
			Streak: streak,
		})
	}

	return results, nil
}

func (s *AppService) DeletePersonalHabit(ctx context.Context, tgUserID int64, seqNum int) error {
	u, err := s.userRepo.GetByTelegramID(ctx, tgUserID)
	if err != nil {
		return err
	}

	h, err := s.habitRepo.GetBySeqNum(ctx, u.ID, seqNum)
	if err != nil {
		return err
	}

	return s.habitRepo.Delete(ctx, h.ID)
}

// AI Integration with sliding context
func (s *AppService) ProcessAIChat(ctx context.Context, tgUserID int64, message string) (string, error) {
	u, err := s.userRepo.GetByTelegramID(ctx, tgUserID)
	if err != nil {
		return "", err
	}

	var groupCtx strings.Builder
	var userMap = make(map[string]string)

	if u.TenantID != nil {
		roomies, _ := s.userRepo.GetUsersByTenantID(ctx, *u.TenantID)
		for _, r := range roomies {
			userMap[r.ID] = r.Name
		}

		groupCtx.WriteString("\nContexto del Departamento:\n")
		tasks, _ := s.taskRepo.GetPendingByTenantID(ctx, *u.TenantID)
		groupCtx.WriteString("- Tareas pendientes:\n")
		if len(tasks) == 0 {
			groupCtx.WriteString("  Ninguna.\n")
		} else {
			for _, t := range tasks {
				assignee := "Sin asignar"
				if t.AssignedTo != nil {
					if name, ok := userMap[*t.AssignedTo]; ok {
						assignee = name
					}
				}
				groupCtx.WriteString(fmt.Sprintf("  * Tarea Nro %d: %s (Asignado: %s)\n", t.SeqNum, t.Description, assignee))
			}
		}

		payments, _ := s.paymentRepo.GetPendingByTenantID(ctx, *u.TenantID)
		groupCtx.WriteString("- Pagos pendientes:\n")
		if len(payments) == 0 {
			groupCtx.WriteString("  Ninguno.\n")
		} else {
			for _, p := range payments {
				payee := "Desconocido"
				if name, ok := userMap[p.UserID]; ok {
					payee = name
				}
				groupCtx.WriteString(fmt.Sprintf("  * Pago Nro %d de %s: $%.2f (Vence: %s)\n", p.SeqNum, payee, p.Amount, p.BillingDate.Format("2006-01-02")))
			}
		}

		meals, _ := s.mealRepo.GetByTenantID(ctx, *u.TenantID)
		groupCtx.WriteString("- Rol de Cocina:\n")
		if len(meals) == 0 {
			groupCtx.WriteString("  Sin asignar todavía.\n")
		} else {
			days := []string{"", "Lunes", "Martes", "Miércoles", "Jueves", "Viernes", "Sábado", "Domingo"}
			for _, m := range meals {
				chef := "Sin asignar"
				if m.ChefID != nil {
					if name, ok := userMap[*m.ChefID]; ok {
						chef = name
					}
				}
				groupCtx.WriteString(fmt.Sprintf("  * %s (%s): Chef %s\n", days[m.DayOfWeek], m.MealType, chef))
			}
		}
	} else {
		groupCtx.WriteString("\nNo estás registrado en ningún grupo de roomies actualmente.\n")
	}

	var habitsCtx strings.Builder
	habits, _ := s.habitRepo.GetByUserID(ctx, u.ID)
	habitsCtx.WriteString("\nContexto Personal (Hábitos y Trabajo):\n")
	if len(habits) == 0 {
		habitsCtx.WriteString("- Sin hábitos registrados.\n")
	} else {
		for _, h := range habits {
			habitsCtx.WriteString(fmt.Sprintf("- Hábito Nro %d: %s (Días: %s) -> Progreso: %s\n", h.SeqNum, h.ActivityType, h.ScheduledDays, h.ProgressStatus))
		}
	}

	var condensedHistory = ""
	aiCtxRecord, err := s.aiRepo.GetByUserID(ctx, u.ID)
	if err == nil {
		condensedHistory = aiCtxRecord.CondensedHistory
	} else {
		aiCtxRecord = &domain.AIContext{
			UserID:           u.ID,
			CondensedHistory: "Historial de conversación vacío.",
		}
	}

	systemPrompt := fmt.Sprintf(`Eres el "Asistente de Convivencia y Disciplina" del hogar de los roomies. 
Tu rol actual es asistir a %s, quien es %s en el departamento.
Tu tono debe ser estricto pero eficiente, directo y en español. Responde en un máximo de 2 frases.

---
INFORMACIÓN ACTUAL DEL SISTEMA (Usa esto para responder preguntas de estado):
%s
%s
---

Historial reciente de la conversación (Resumen condensado):
%s`, u.Name, u.Role, groupCtx.String(), habitsCtx.String(), condensedHistory)

	response, err := s.groqClient.Chat(ctx, systemPrompt, message)
	if err != nil {
		return "", fmt.Errorf("error querying groq: %w", err)
	}

	newInteraction := fmt.Sprintf("Usuario: %s\nAsistente: %s", message, response)
	newCondensed, err := s.groqClient.CondenseHistory(ctx, condensedHistory, newInteraction)
	if err == nil {
		aiCtxRecord.CondensedHistory = newCondensed
		_ = s.aiRepo.CreateOrUpdate(ctx, aiCtxRecord)
	}

	return response, nil
}

type HabitWithStreak struct {
	Habit  *domain.PersonalHabit
	Streak int
}

func CalculateStreak(logs []*domain.HabitLog, today time.Time) int {
	if len(logs) == 0 {
		return 0
	}

	completedDates := make(map[string]bool)
	for _, l := range logs {
		if l.Status == "completed" {
			completedDates[l.LoggedDate.Format("2006-01-02")] = true
		}
	}

	if len(completedDates) == 0 {
		return 0
	}

	todayStr := today.Format("2006-01-02")
	yesterdayStr := today.AddDate(0, 0, -1).Format("2006-01-02")

	hasToday := completedDates[todayStr]
	hasYesterday := completedDates[yesterdayStr]

	if !hasToday && !hasYesterday {
		return 0
	}

	var currentDay time.Time
	if hasToday {
		currentDay = today
	} else {
		currentDay = today.AddDate(0, 0, -1)
	}

	streak := 0
	for {
		dayStr := currentDay.Format("2006-01-02")
		if completedDates[dayStr] {
			streak++
			currentDay = currentDay.AddDate(0, 0, -1)
		} else {
			break
		}
	}

	return streak
}

// Business Stores, Products, and Orders Management
func (s *AppService) GetOrCreateUserStore(ctx context.Context, tgUserID int64, storeName string) (*domain.Store, error) {
	u, err := s.userRepo.GetByTelegramID(ctx, tgUserID)
	if err != nil {
		return nil, err
	}

	st, err := s.storeRepo.GetByNameAndUser(ctx, storeName, u.ID)
	if err == nil {
		return st, nil
	}

	st = &domain.Store{
		UserID:    u.ID,
		StoreName: storeName,
	}
	err = s.storeRepo.Create(ctx, st)
	if err != nil {
		return nil, err
	}
	return st, nil
}

func (s *AppService) GetUserStores(ctx context.Context, tgUserID int64) ([]*domain.Store, error) {
	u, err := s.userRepo.GetByTelegramID(ctx, tgUserID)
	if err != nil {
		return nil, err
	}
	return s.storeRepo.GetByUserID(ctx, u.ID)
}

func (s *AppService) AddProduct(ctx context.Context, tgUserID int64, storeID string, name string, price float64, stock int) (*domain.Product, error) {
	u, err := s.userRepo.GetByTelegramID(ctx, tgUserID)
	if err != nil {
		return nil, err
	}

	store, err := s.storeRepo.GetByID(ctx, storeID)
	if err != nil {
		return nil, err
	}

	if store.UserID != u.ID {
		return nil, domain.ErrUnauthorized
	}

	p := &domain.Product{
		StoreID: storeID,
		Name:    name,
		Price:   price,
		Stock:   stock,
	}
	err = s.productRepo.Create(ctx, p)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *AppService) GetStoreProducts(ctx context.Context, tgUserID int64, storeID string) ([]*domain.Product, error) {
	u, err := s.userRepo.GetByTelegramID(ctx, tgUserID)
	if err != nil {
		return nil, err
	}

	store, err := s.storeRepo.GetByID(ctx, storeID)
	if err != nil {
		return nil, err
	}

	if store.UserID != u.ID {
		return nil, domain.ErrUnauthorized
	}

	return s.productRepo.GetByStoreID(ctx, storeID)
}

func (s *AppService) DeleteProduct(ctx context.Context, tgUserID int64, storeID string, seqNum int) error {
	u, err := s.userRepo.GetByTelegramID(ctx, tgUserID)
	if err != nil {
		return err
	}

	store, err := s.storeRepo.GetByID(ctx, storeID)
	if err != nil {
		return err
	}

	if store.UserID != u.ID {
		return domain.ErrUnauthorized
	}

	p, err := s.productRepo.GetBySeqNum(ctx, storeID, seqNum)
	if err != nil {
		return err
	}

	return s.productRepo.Delete(ctx, p.ID)
}

func (s *AppService) CreateStoreOrder(ctx context.Context, tgUserID int64, storeID string, clientName string, clientPhone *string, productDetails string, totalCost float64, advancePayment float64, shippingAddress *string, shippingCost float64, status string) (*domain.Order, error) {
	u, err := s.userRepo.GetByTelegramID(ctx, tgUserID)
	if err != nil {
		return nil, err
	}

	store, err := s.storeRepo.GetByID(ctx, storeID)
	if err != nil {
		return nil, err
	}

	if store.UserID != u.ID {
		return nil, domain.ErrUnauthorized
	}

	o := &domain.Order{
		StoreID:         storeID,
		ClientName:      clientName,
		ClientPhone:     clientPhone,
		ProductDetails:  productDetails,
		TotalCost:       totalCost,
		AdvancePayment:  advancePayment,
		ShippingAddress: shippingAddress,
		ShippingCost:    shippingCost,
		Status:          status,
	}
	err = s.orderRepo.Create(ctx, o)
	if err != nil {
		return nil, err
	}
	return o, nil
}

func (s *AppService) GetStoreOrders(ctx context.Context, tgUserID int64, storeID string) ([]*domain.Order, error) {
	u, err := s.userRepo.GetByTelegramID(ctx, tgUserID)
	if err != nil {
		return nil, err
	}

	store, err := s.storeRepo.GetByID(ctx, storeID)
	if err != nil {
		return nil, err
	}

	if store.UserID != u.ID {
		return nil, domain.ErrUnauthorized
	}

	return s.orderRepo.GetByStoreID(ctx, storeID)
}

func (s *AppService) GetPendingStoreOrders(ctx context.Context, tgUserID int64, storeID string) ([]*domain.Order, error) {
	u, err := s.userRepo.GetByTelegramID(ctx, tgUserID)
	if err != nil {
		return nil, err
	}

	store, err := s.storeRepo.GetByID(ctx, storeID)
	if err != nil {
		return nil, err
	}

	if store.UserID != u.ID {
		return nil, domain.ErrUnauthorized
	}

	return s.orderRepo.GetPendingByStoreID(ctx, storeID)
}

func (s *AppService) UpdateOrderStatus(ctx context.Context, tgUserID int64, storeID string, seqNum int, status string) error {
	u, err := s.userRepo.GetByTelegramID(ctx, tgUserID)
	if err != nil {
		return err
	}

	store, err := s.storeRepo.GetByID(ctx, storeID)
	if err != nil {
		return err
	}

	if store.UserID != u.ID {
		return domain.ErrUnauthorized
	}

	o, err := s.orderRepo.GetBySeqNum(ctx, storeID, seqNum)
	if err != nil {
		return err
	}

	o.Status = status
	return s.orderRepo.Update(ctx, o)
}

func (s *AppService) RecordOrderFinalPayment(ctx context.Context, tgUserID int64, storeID string, seqNum int) error {
	u, err := s.userRepo.GetByTelegramID(ctx, tgUserID)
	if err != nil {
		return err
	}

	store, err := s.storeRepo.GetByID(ctx, storeID)
	if err != nil {
		return err
	}

	if store.UserID != u.ID {
		return domain.ErrUnauthorized
	}

	o, err := s.orderRepo.GetBySeqNum(ctx, storeID, seqNum)
	if err != nil {
		return err
	}

	o.AdvancePayment = o.TotalCost
	o.Status = "completed"
	return s.orderRepo.Update(ctx, o)
}

func (s *AppService) DeleteStoreOrder(ctx context.Context, tgUserID int64, storeID string, seqNum int) error {
	u, err := s.userRepo.GetByTelegramID(ctx, tgUserID)
	if err != nil {
		return err
	}

	store, err := s.storeRepo.GetByID(ctx, storeID)
	if err != nil {
		return err
	}

	if store.UserID != u.ID {
		return domain.ErrUnauthorized
	}

	o, err := s.orderRepo.GetBySeqNum(ctx, storeID, seqNum)
	if err != nil {
		return err
	}

	return s.orderRepo.Delete(ctx, o.ID)
}

func (s *AppService) GetProductBySeqNum(ctx context.Context, tgUserID int64, storeID string, seqNum int) (*domain.Product, error) {
	u, err := s.userRepo.GetByTelegramID(ctx, tgUserID)
	if err != nil {
		return nil, err
	}

	store, err := s.storeRepo.GetByID(ctx, storeID)
	if err != nil {
		return nil, err
	}

	if store.UserID != u.ID {
		return nil, domain.ErrUnauthorized
	}

	return s.productRepo.GetBySeqNum(ctx, storeID, seqNum)
}

func (s *AppService) DecreaseProductStock(ctx context.Context, tgUserID int64, storeID string, seqNum int, quantity int) error {
	u, err := s.userRepo.GetByTelegramID(ctx, tgUserID)
	if err != nil {
		return err
	}

	store, err := s.storeRepo.GetByID(ctx, storeID)
	if err != nil {
		return err
	}

	if store.UserID != u.ID {
		return domain.ErrUnauthorized
	}

	p, err := s.productRepo.GetBySeqNum(ctx, storeID, seqNum)
	if err != nil {
		return err
	}

	if p.Stock >= 0 {
		if p.Stock < quantity {
			return fmt.Errorf("stock insuficiente (disponible: %d)", p.Stock)
		}
		p.Stock -= quantity
		return s.productRepo.Update(ctx, p)
	}
	return nil
}
