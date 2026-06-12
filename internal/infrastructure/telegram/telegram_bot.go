package telegram

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/iloremstudio/home-bot/internal/application"
	"github.com/iloremstudio/home-bot/internal/domain"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// States for our Finite State Machine (FSM)
const (
	StateNone = iota
	StateAwaitingGroupName
	StateAwaitingJoinGroupID
	StateAwaitingPaymentAmount
	StateAwaitingPaymentProof
	StateAwaitingTaskDesc
	StateAwaitingTaskAssignee
	StateAwaitingTaskDueDate
	StateAwaitingHabitName
	StateAwaitingHabitDays
	StateAwaitingHabitTime
)

type UserState struct {
	State int
	Data  map[string]interface{}
}

type Bot struct {
	api        *tgbotapi.BotAPI
	appService *application.AppService
	states     map[int64]*UserState
	statesMu   sync.RWMutex
}

func NewBot(token string, appService *application.AppService) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("error creating telegram bot: %w", err)
	}

	return &Bot{
		api:        api,
		appService: appService,
		states:     make(map[int64]*UserState),
	}, nil
}

func (b *Bot) Start(ctx context.Context) {
	log.Printf("Bot autorizado como %s", b.api.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			log.Println("Deteniendo el bot de Telegram...")
			return
		case update := <-updates:
			go b.handleUpdate(ctx, update)
		}
	}
}

func (b *Bot) handleUpdate(ctx context.Context, update tgbotapi.Update) {
	if update.Message != nil {
		b.handleMessage(ctx, update.Message)
	} else if update.CallbackQuery != nil {
		b.handleCallbackQuery(ctx, update.CallbackQuery)
	}
}

func (b *Bot) handleMessage(ctx context.Context, msg *tgbotapi.Message) {
	userID := msg.From.ID
	userName := msg.From.FirstName
	if msg.From.LastName != "" {
		userName += " " + msg.From.LastName
	}

	user, err := b.appService.RegisterUser(ctx, userID, userName)
	if err != nil {
		log.Printf("Error registrando usuario %d: %v", userID, err)
		b.sendText(userID, "⚠️ Error al conectarse con el sistema. Intenta de nuevo más tarde.")
		return
	}

	b.statesMu.RLock()
	state, hasState := b.states[userID]
	b.statesMu.RUnlock()

	if msg.Text == "/cancelar" || strings.ToLower(msg.Text) == "cancelar" {
		b.clearState(userID)
		b.sendText(userID, "❌ Proceso cancelado.")
		b.sendMainMenu(userID, user)
		return
	}

	if hasState && state.State != StateNone {
		b.handleStateFlow(ctx, msg, user, state)
		return
	}

	if msg.IsCommand() {
		switch msg.Command() {
		case "start":
			b.sendWelcome(userID, user)
		case "creargrupo":
			b.setState(userID, StateAwaitingGroupName, nil)
			b.sendText(userID, "🏠 Ingresa el nombre para tu nuevo grupo de convivencia/departamento:")
		case "unirse":
			b.setState(userID, StateAwaitingJoinGroupID, nil)
			b.sendText(userID, "🔑 Ingresa el ID único (UUID) del grupo al que deseas unirte:")
		case "pagar":
			if user.TenantID == nil {
				b.sendText(userID, "⚠️ Primero debes crear o unirte a un grupo con /creargrupo o /unirse.")
				return
			}
			b.setState(userID, StateAwaitingPaymentAmount, make(map[string]interface{}))
			b.sendText(userID, "💵 Ingresa el monto a pagar (ej: 450.50):")
		case "pagospendientes":
			b.listPayments(ctx, userID, user)
		case "tareas":
			b.listTasks(ctx, userID, user)
		case "creartarea":
			if user.TenantID == nil {
				b.sendText(userID, "⚠️ Primero debes crear o unirte a un grupo con /creargrupo o /unirse.")
				return
			}
			b.setState(userID, StateAwaitingTaskDesc, nil)
			b.sendText(userID, "🧹 Ingresa la descripción de la tarea de limpieza:")
		case "cocina":
			b.showKitchenSchedule(ctx, userID, user)
		case "roomies", "miembros":
			b.listRoomies(ctx, userID)
		case "asignarcocina":
			b.startKitchenAssignment(ctx, userID, user)
		case "habitos", "gimnasio":
			b.listHabits(ctx, userID)
		case "crearhabito":
			b.setState(userID, StateAwaitingHabitName, make(map[string]interface{}))
			b.sendText(userID, "💪 Ingresa el nombre del hábito o meta (ej: Ir al Gimnasio, Postulaciones de Trabajo, Tomar 2L Agua):")
		case "ayuda":
			b.sendHelp(userID)
		default:
			b.sendText(userID, "❓ Comando desconocido. Escribe /ayuda para ver los comandos disponibles.")
		}
		return
	}

	b.sendText(userID, "🤖 Procesando respuesta inteligente...")
	aiResponse, err := b.appService.ProcessAIChat(ctx, userID, msg.Text)
	if err != nil {
		log.Printf("Error Groq para usuario %d: %v", userID, err)
		b.sendText(userID, "⚠️ Groq no pudo procesar tu mensaje. Escribe un comando o intenta de nuevo.")
		return
	}
	b.sendText(userID, aiResponse)
}

func (b *Bot) handleStateFlow(ctx context.Context, msg *tgbotapi.Message, user *domain.User, state *UserState) {
	userID := msg.From.ID

	switch state.State {
	case StateAwaitingGroupName:
		groupName := strings.TrimSpace(msg.Text)
		if groupName == "" {
			b.sendText(userID, "⚠️ El nombre del grupo no puede estar vacío. Reintenta:")
			return
		}
		g, err := b.appService.JoinOrCreateGroup(ctx, userID, groupName)
		if err != nil {
			b.sendText(userID, "❌ Error al crear el grupo: "+err.Error())
			b.clearState(userID)
			return
		}
		b.clearState(userID)
		b.sendText(userID, fmt.Sprintf("✅ ¡Grupo '%s' creado con éxito! El ID único de tu grupo es:\n`%s`\nCompártelo con tus roomies para que se unan.", g.GroupName, g.ID))
		if freshUser, err := b.appService.RegisterUser(ctx, userID, user.Name); err == nil {
			user = freshUser
		}
		b.sendMainMenu(userID, user)

	case StateAwaitingJoinGroupID:
		groupIDStr := strings.TrimSpace(msg.Text)
		if groupIDStr == "" {
			b.sendText(userID, "⚠️ El ID del grupo no puede estar vacío. Reintenta:")
			return
		}
		err := b.appService.JoinExistingGroup(ctx, userID, groupIDStr)
		if err != nil {
			b.sendText(userID, "❌ Error al unirse: "+err.Error())
			b.clearState(userID)
			return
		}
		b.clearState(userID)
		b.sendText(userID, "✅ ¡Te has unido al grupo con éxito!")
		if freshUser, err := b.appService.RegisterUser(ctx, userID, user.Name); err == nil {
			user = freshUser
		}
		b.sendMainMenu(userID, user)

	case StateAwaitingPaymentAmount:
		amountStr := strings.TrimSpace(msg.Text)
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil || amount <= 0 {
			b.sendText(userID, "⚠️ Monto inválido. Debe ser un número mayor a 0 (ej: 150.00). Reintenta:")
			return
		}
		state.Data["amount"] = amount
		state.State = StateAwaitingPaymentProof
		b.setState(userID, StateAwaitingPaymentProof, state.Data)
		b.sendText(userID, "📸 Por favor, sube una foto o comprobante de transferencia:")

	case StateAwaitingPaymentProof:
		var proofID string
		if len(msg.Photo) > 0 {
			proofID = msg.Photo[len(msg.Photo)-1].FileID
		} else if msg.Document != nil {
			proofID = msg.Document.FileID
		} else {
			b.sendText(userID, "⚠️ Por favor, sube una imagen o archivo de comprobante:")
			return
		}

		amount := state.Data["amount"].(float64)
		p, err := b.appService.CreatePayment(ctx, userID, amount, time.Now())
		if err != nil {
			b.sendText(userID, "❌ Error creando registro de pago: "+err.Error())
			b.clearState(userID)
			return
		}

		_, err = b.appService.UploadPaymentProof(ctx, userID, p.SeqNum, proofID)
		if err != nil {
			b.sendText(userID, "❌ Error guardando comprobante: "+err.Error())
			b.clearState(userID)
			return
		}

		b.clearState(userID)
		b.sendText(userID, fmt.Sprintf("✅ Pago Nro %d registrado con éxito. Esperando validación por el Admin.", p.SeqNum))

		roomies, _ := b.appService.GetUsersInGroup(ctx, userID)
		for _, r := range roomies {
			if r.Role == "admin" {
				msgTxt := fmt.Sprintf("🔔 Nuevo pago Nro %d para validar de %s:\nMonto: $%.2f\nUsa /pagospendientes para revisarlo.", p.SeqNum, user.Name, amount)
				b.sendText(r.TelegramID, msgTxt)
				photoMsg := tgbotapi.NewPhoto(r.TelegramID, tgbotapi.FileID(proofID))
				photoMsg.Caption = fmt.Sprintf("Comprobante de pago Nro %d", p.SeqNum)
				_, _ = b.api.Send(photoMsg)
			}
		}

	case StateAwaitingTaskDesc:
		desc := strings.TrimSpace(msg.Text)
		if desc == "" {
			b.sendText(userID, "⚠️ La descripción no puede estar vacía. Reintenta:")
			return
		}

		state.Data = make(map[string]interface{})
		state.Data["description"] = desc

		roomies, err := b.appService.GetUsersInGroup(ctx, userID)
		if err != nil {
			// Fallback: create task immediately if we can't fetch roomies
			t, err := b.appService.CreateHouseTask(ctx, userID, desc, nil, nil)
			if err != nil {
				b.sendText(userID, "❌ Error creando tarea: "+err.Error())
			} else {
				b.sendText(userID, fmt.Sprintf("✅ Tarea Nro %d ('%s') creada y asignada al grupo.", t.SeqNum, t.Description))
			}
			b.clearState(userID)
			return
		}

		var buttons []tgbotapi.InlineKeyboardButton
		for _, r := range roomies {
			btn := tgbotapi.NewInlineKeyboardButtonData(r.Name, fmt.Sprintf("assignTaskUser_%d", r.TelegramID))
			buttons = append(buttons, btn)
		}
		btnNone := tgbotapi.NewInlineKeyboardButtonData("🤷‍♂️ Sin Asignar", "assignTaskUser_none")
		buttons = append(buttons, btnNone)

		var rows [][]tgbotapi.InlineKeyboardButton
		for _, btn := range buttons {
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
		}
		kb := tgbotapi.NewInlineKeyboardMarkup(rows...)

		state.State = StateAwaitingTaskAssignee
		b.setState(userID, StateAwaitingTaskAssignee, state.Data)

		msgResponse := tgbotapi.NewMessage(userID, "🧹 *Crear Tarea - Paso 2:*\nSelecciona a la persona encargada de la tarea:")
		msgResponse.ParseMode = tgbotapi.ModeMarkdown
		msgResponse.ReplyMarkup = kb
		_, _ = b.api.Send(msgResponse)

	case StateAwaitingTaskAssignee:
		b.sendText(userID, "⚠️ Por favor, selecciona a un encargado usando los botones en pantalla o escribe *cancelar* para salir.")

	case StateAwaitingTaskDueDate:
		dueDateInput := strings.TrimSpace(msg.Text)
		var dueDate *time.Time

		if strings.ToLower(dueDateInput) != "no" {
			parts := strings.Split(dueDateInput, "/")
			if len(parts) != 2 {
				b.sendText(userID, "⚠️ Formato inválido. Debe ser DD/MM (ej: 25/12). Reintenta o escribe 'no':")
				return
			}
			day, errD := strconv.Atoi(parts[0])
			month, errM := strconv.Atoi(parts[1])
			if errD != nil || errM != nil || day < 1 || day > 31 || month < 1 || month > 12 {
				b.sendText(userID, "⚠️ Fecha inválida. Reintenta o escribe 'no':")
				return
			}

			now := time.Now()
			parsedDate := time.Date(now.Year(), time.Month(month), day, 12, 0, 0, 0, time.UTC)
			if parsedDate.Before(now) && now.Sub(parsedDate) > 24*time.Hour {
				parsedDate = time.Date(now.Year()+1, time.Month(month), day, 12, 0, 0, 0, time.UTC)
			}
			dueDate = &parsedDate
		}

		desc := state.Data["description"].(string)
		var assignedToTelegramID *int64
		if val, ok := state.Data["assignedToTelegramID"]; ok && val != nil {
			v := val.(int64)
			assignedToTelegramID = &v
		}

		t, err := b.appService.CreateHouseTask(ctx, userID, desc, assignedToTelegramID, dueDate)
		if err != nil {
			b.sendText(userID, "❌ Error creando tarea: "+err.Error())
			b.clearState(userID)
			return
		}

		b.clearState(userID)

		successText := fmt.Sprintf("✅ Tarea Nro %d ('%s') creada con éxito.", t.SeqNum, t.Description)
		if assignedToTelegramID != nil {
			chefUser, _ := b.appService.RegisterUser(ctx, *assignedToTelegramID, "")
			successText += fmt.Sprintf("\n👤 *Asignado a:* %s", chefUser.Name)
		} else {
			successText += "\n👤 *Asignado a:* Sin asignar"
		}
		if dueDate != nil {
			successText += fmt.Sprintf("\n📅 *Vence:* %s", dueDate.Format("02/01/2006"))
		}
		b.sendText(userID, successText)
		b.sendMainMenu(userID, user)

	case StateAwaitingHabitName:
		habitName := strings.TrimSpace(msg.Text)
		if habitName == "" {
			b.sendText(userID, "⚠️ El nombre del hábito no puede estar vacío. Reintenta:")
			return
		}
		state.Data["name"] = habitName
		state.State = StateAwaitingHabitDays
		b.setState(userID, StateAwaitingHabitDays, state.Data)
		b.sendText(userID, "🗓️ Ingresa los días de la semana separados por comas (ej: Lunes,Miercoles,Viernes):")

	case StateAwaitingHabitDays:
		days := strings.TrimSpace(msg.Text)
		if days == "" {
			b.sendText(userID, "⚠️ Los días no pueden estar vacíos. Reintenta:")
			return
		}
		state.Data["days"] = days
		state.State = StateAwaitingHabitTime
		b.setState(userID, StateAwaitingHabitTime, state.Data)
		b.sendText(userID, "🕒 Ingresa la hora de recordatorio en formato HH:MM de 24 horas (ej. 08:30, 21:00) o escribe 'no' para no recibir alertas:")

	case StateAwaitingHabitTime:
		timeInput := strings.TrimSpace(msg.Text)
		var reminderTime *string

		if strings.ToLower(timeInput) != "no" {
			parts := strings.Split(timeInput, ":")
			if len(parts) != 2 {
				b.sendText(userID, "⚠️ Formato inválido. Debe ser HH:MM (ej: 07:15). Reintenta o escribe 'no':")
				return
			}
			hour, errH := strconv.Atoi(parts[0])
			min, errM := strconv.Atoi(parts[1])
			if errH != nil || errM != nil || hour < 0 || hour > 23 || min < 0 || min > 59 || len(parts[0]) != 2 || len(parts[1]) != 2 {
				b.sendText(userID, "⚠️ Hora o minutos inválidos (debe ser de 00:00 a 23:59). Reintenta o escribe 'no':")
				return
			}
			reminderTime = &timeInput
		}

		habitName := state.Data["name"].(string)
		days := state.Data["days"].(string)

		h, err := b.appService.AddPersonalHabit(ctx, userID, habitName, days, reminderTime, "America/Bogota")
		if err != nil {
			b.sendText(userID, "❌ Error creando hábito: "+err.Error())
			b.clearState(userID)
			return
		}
		b.clearState(userID)
		if reminderTime != nil {
			b.sendText(userID, fmt.Sprintf("✅ ¡Hábito Nro %d '%s' registrado para los días [%s] a las %s (Hora local)! ⏰", h.SeqNum, habitName, days, *reminderTime))
		} else {
			b.sendText(userID, fmt.Sprintf("✅ ¡Hábito Nro %d '%s' registrado para los días [%s] sin recordatorios!", h.SeqNum, habitName, days))
		}
	}
}

func (b *Bot) handleCallbackQuery(ctx context.Context, cb *tgbotapi.CallbackQuery) {
	userID := cb.From.ID
	data := cb.Data

	callbackResp := tgbotapi.NewCallback(cb.ID, "")
	_, _ = b.api.Request(callbackResp)

	user, err := b.appService.RegisterUser(ctx, userID, cb.From.FirstName)
	if err != nil {
		return
	}

	parts := strings.Split(data, "_")
	if len(parts) < 2 {
		return
	}

	action := parts[0]
	targetSeqNum, _ := strconv.Atoi(parts[1])

	switch action {
	case "completeTask":
		err := b.appService.CompleteHouseTask(ctx, userID, targetSeqNum)
		if err != nil {
			b.sendText(userID, "❌ Error al completar la tarea: "+err.Error())
		} else {
			b.sendText(userID, "✅ ¡Tarea marcada como completada!")
			editMsg := tgbotapi.NewEditMessageText(userID, cb.Message.MessageID, fmt.Sprintf("✅ Tarea Nro %d completada exitosamente.", targetSeqNum))
			_, _ = b.api.Send(editMsg)
		}
	case "deleteTask":
		err := b.appService.DeleteHouseTask(ctx, userID, targetSeqNum)
		if err != nil {
			b.sendText(userID, "❌ Error al eliminar la tarea: "+err.Error())
		} else {
			b.sendText(userID, "✅ ¡Tarea eliminada!")
			editMsg := tgbotapi.NewEditMessageText(userID, cb.Message.MessageID, fmt.Sprintf("🗑️ Tarea Nro %d eliminada.", targetSeqNum))
			_, _ = b.api.Send(editMsg)
		}
	case "approvePayment":
		err := b.appService.ApprovePayment(ctx, userID, targetSeqNum)
		if err != nil {
			b.sendText(userID, "❌ Error al aprobar pago: "+err.Error())
		} else {
			b.sendText(userID, "✅ Pago aprobado correctamente.")
			editMsg := tgbotapi.NewEditMessageText(userID, cb.Message.MessageID, fmt.Sprintf("✅ Pago Nro %d APROBADO por el Administrador.", targetSeqNum))
			_, _ = b.api.Send(editMsg)
		}
	case "rejectPayment":
		err := b.appService.RejectPayment(ctx, userID, targetSeqNum)
		if err != nil {
			b.sendText(userID, "❌ Error al rechazar pago: "+err.Error())
		} else {
			b.sendText(userID, "❌ Pago rechazado.")
			editMsg := tgbotapi.NewEditMessageText(userID, cb.Message.MessageID, fmt.Sprintf("❌ Pago Nro %d RECHAZADO por el Administrador.", targetSeqNum))
			_, _ = b.api.Send(editMsg)
		}
	case "deleteHabit":
		err := b.appService.DeletePersonalHabit(ctx, userID, targetSeqNum)
		if err != nil {
			b.sendText(userID, "❌ Error al borrar hábito: "+err.Error())
		} else {
			b.sendText(userID, "✅ Hábito eliminado.")
			editMsg := tgbotapi.NewEditMessageText(userID, cb.Message.MessageID, fmt.Sprintf("🗑️ Hábito Nro %d eliminado.", targetSeqNum))
			_, _ = b.api.Send(editMsg)
		}
	case "updateHabit":
		err := b.appService.LogHabitCheckIn(ctx, userID, targetSeqNum, "completed")
		if err != nil {
			b.sendText(userID, "❌ Error al registrar el hábito: "+err.Error())
		} else {
			b.sendText(userID, "🎉 ¡Felicidades por completar tu hábito de hoy!")
			b.listHabits(ctx, userID)
		}
	case "setChef":
		if len(parts) < 3 {
			return
		}
		day, _ := strconv.Atoi(parts[1])
		mealType := parts[2]
		err := b.appService.SetMealChef(ctx, userID, day, mealType, userID)
		if err != nil {
			b.sendText(userID, "❌ Error al asignarte como cocinero: "+err.Error())
		} else {
			b.sendText(userID, "🍳 Te has asignado como chef para este turno.")
			b.showKitchenSchedule(ctx, userID, user)
		}
	case "assignCookDay":
		if len(parts) < 2 {
			return
		}
		dayNum := parts[1]
		
		btnBreakfast := tgbotapi.NewInlineKeyboardButtonData("🥞 Desayuno", fmt.Sprintf("assignCookMeal_%s_breakfast", dayNum))
		btnLunch := tgbotapi.NewInlineKeyboardButtonData("🍛 Almuerzo", fmt.Sprintf("assignCookMeal_%s_lunch", dayNum))
		btnDinner := tgbotapi.NewInlineKeyboardButtonData("🍕 Cena", fmt.Sprintf("assignCookMeal_%s_dinner", dayNum))
		kb := tgbotapi.NewInlineKeyboardMarkup([]tgbotapi.InlineKeyboardButton{btnBreakfast, btnLunch, btnDinner})

		daysSp := []string{"", "Lunes", "Martes", "Miércoles", "Jueves", "Viernes", "Sábado", "Domingo"}
		dIdx, _ := strconv.Atoi(dayNum)
		dayName := daysSp[dIdx]

		editMsg := tgbotapi.NewEditMessageText(userID, cb.Message.MessageID, fmt.Sprintf("🍳 *Asignar Cocina - Paso 2:*\nHas elegido el *%s*.\nSelecciona la comida:", dayName))
		editMsg.ParseMode = tgbotapi.ModeMarkdown
		editMsg.ReplyMarkup = &kb
		_, _ = b.api.Send(editMsg)

	case "assignCookMeal":
		if len(parts) < 3 {
			return
		}
		dayNum := parts[1]
		mealType := parts[2]

		roomies, err := b.appService.GetUsersInGroup(ctx, userID)
		if err != nil {
			b.sendText(userID, "❌ Error al obtener miembros: "+err.Error())
			return
		}

		var buttons []tgbotapi.InlineKeyboardButton
		for _, r := range roomies {
			btn := tgbotapi.NewInlineKeyboardButtonData(r.Name, fmt.Sprintf("assignCookChef_%s_%s_%d", dayNum, mealType, r.TelegramID))
			buttons = append(buttons, btn)
		}

		var rows [][]tgbotapi.InlineKeyboardButton
		for _, btn := range buttons {
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
		}
		kb := tgbotapi.NewInlineKeyboardMarkup(rows...)

		daysSp := []string{"", "Lunes", "Martes", "Miércoles", "Jueves", "Viernes", "Sábado", "Domingo"}
		dIdx, _ := strconv.Atoi(dayNum)
		dayName := daysSp[dIdx]
		mealNamesSp := map[string]string{
			"breakfast": "Desayuno 🥞",
			"lunch":     "Almuerzo 🍛",
			"dinner":    "Cena 🍕",
		}

		editMsg := tgbotapi.NewEditMessageText(userID, cb.Message.MessageID, fmt.Sprintf("🍳 *Asignar Cocina - Paso 3:*\nHas elegido *%s* para el *%s*.\nSelecciona al chef responsable:", mealNamesSp[mealType], dayName))
		editMsg.ParseMode = tgbotapi.ModeMarkdown
		editMsg.ReplyMarkup = &kb
		_, _ = b.api.Send(editMsg)

	case "assignCookChef":
		if len(parts) < 4 {
			return
		}
		dayNum, _ := strconv.Atoi(parts[1])
		mealType := parts[2]
		chefTelegramID, _ := strconv.ParseInt(parts[3], 10, 64)

		err := b.appService.SetMealChef(ctx, userID, dayNum, mealType, chefTelegramID)
		if err != nil {
			b.sendText(userID, "❌ Error al asignar cocinero: "+err.Error())
			return
		}

		daysSp := []string{"", "Lunes", "Martes", "Miércoles", "Jueves", "Viernes", "Sábado", "Domingo"}
		mealNamesSp := map[string]string{
			"breakfast": "Desayuno 🥞",
			"lunch":     "Almuerzo 🍛",
			"dinner":    "Cena 🍕",
		}

		chefUser, _ := b.appService.RegisterUser(ctx, chefTelegramID, "")
		editMsg := tgbotapi.NewEditMessageText(userID, cb.Message.MessageID, fmt.Sprintf("✅ ¡Turno asignado con éxito!\n*%s* cocinará el *%s* (%s).", chefUser.Name, daysSp[dayNum], mealNamesSp[mealType]))
		editMsg.ParseMode = tgbotapi.ModeMarkdown
		_, _ = b.api.Send(editMsg)

	case "assignTaskUser":
		if len(parts) < 2 {
			return
		}
		assigneeVal := parts[1]

		b.statesMu.RLock()
		state, hasState := b.states[userID]
		b.statesMu.RUnlock()

		if !hasState || state.State != StateAwaitingTaskAssignee {
			b.sendText(userID, "⚠️ Sesión de creación de tarea inválida o expirada.")
			return
		}

		if assigneeVal != "none" {
			chefTelegramID, _ := strconv.ParseInt(assigneeVal, 10, 64)
			state.Data["assignedToTelegramID"] = chefTelegramID
		} else {
			state.Data["assignedToTelegramID"] = nil
		}

		state.State = StateAwaitingTaskDueDate
		b.setState(userID, StateAwaitingTaskDueDate, state.Data)

		editMsg := tgbotapi.NewEditMessageText(userID, cb.Message.MessageID, "🧹 *Crear Tarea - Paso 3:*\nIngresa la fecha de vencimiento en formato *DD/MM* (ej: 18/06) o escribe *no* para no definir fecha:")
		editMsg.ParseMode = tgbotapi.ModeMarkdown
		_, _ = b.api.Send(editMsg)
	}
}

// Helper methods to list and show UI
func (b *Bot) startKitchenAssignment(ctx context.Context, userID int64, user *domain.User) {
	if user.TenantID == nil {
		b.sendText(userID, "⚠️ Primero debes crear o unirte a un grupo con /creargrupo o /unirse.")
		return
	}
	if user.Role != "admin" {
		b.sendText(userID, "⚠️ Solo los administradores pueden asignar turnos de cocina.")
		return
	}

	days := []string{"Lunes", "Martes", "Miércoles", "Jueves", "Viernes", "Sábado", "Domingo"}
	var rows [][]tgbotapi.InlineKeyboardButton
	for i, d := range days {
		dayNum := i + 1
		btn := tgbotapi.NewInlineKeyboardButtonData(d, fmt.Sprintf("assignCookDay_%d", dayNum))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
	}
	kb := tgbotapi.NewInlineKeyboardMarkup(rows...)

	msg := tgbotapi.NewMessage(userID, "🍳 *Asignar Cocina - Paso 1:*\nSelecciona el día de la semana:")
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = kb
	_, _ = b.api.Send(msg)
}

func (b *Bot) listRoomies(ctx context.Context, userID int64) {
	roomies, err := b.appService.GetUsersInGroup(ctx, userID)
	if err != nil {
		b.sendText(userID, "⚠️ "+err.Error())
		return
	}

	if len(roomies) == 0 {
		b.sendText(userID, "🤷‍♂️ No hay otros miembros en tu grupo.")
		return
	}

	// Fetch group name
	groupName := "Tu grupo"
	user, err := b.appService.RegisterUser(ctx, userID, "")
	if err == nil && user.TenantID != nil {
		group, err := b.appService.GetGroupByID(ctx, *user.TenantID)
		if err == nil {
			groupName = group.GroupName
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("👤 *Integrantes del grupo %s:*\n\n", groupName))
	for idx, r := range roomies {
		sb.WriteString(fmt.Sprintf("%d. *%s* (%s)\n", idx+1, r.Name, r.Role))
	}
	b.sendText(userID, sb.String())
}

func (b *Bot) listTasks(ctx context.Context, userID int64, user *domain.User) {
	tasks, err := b.appService.GetPendingTasks(ctx, userID)
	if err != nil {
		b.sendText(userID, "⚠️ "+err.Error())
		return
	}

	if len(tasks) == 0 {
		b.sendText(userID, "✨ ¡Enhorabuena! No hay tareas de limpieza pendientes en el grupo.")
		return
	}

	b.sendText(userID, "🧹 *Tareas de Limpieza Pendientes:*")
	for _, t := range tasks {
		text := fmt.Sprintf("📌 *Tarea Nro %d:* %s\n", t.SeqNum, t.Description)
		if t.DueDate != nil {
			text += fmt.Sprintf("📅 *Vencimiento:* %s\n", t.DueDate.Format("02-01-2006"))
		}

		btnDone := tgbotapi.NewInlineKeyboardButtonData("✅ Completada", fmt.Sprintf("completeTask_%d", t.SeqNum))
		btnDel := tgbotapi.NewInlineKeyboardButtonData("🗑️ Eliminar", fmt.Sprintf("deleteTask_%d", t.SeqNum))
		kb := tgbotapi.NewInlineKeyboardMarkup([]tgbotapi.InlineKeyboardButton{btnDone, btnDel})

		msg := tgbotapi.NewMessage(userID, text)
		msg.ParseMode = tgbotapi.ModeMarkdown
		msg.ReplyMarkup = kb
		_, _ = b.api.Send(msg)
	}
}

func (b *Bot) listPayments(ctx context.Context, userID int64, user *domain.User) {
	payments, err := b.appService.GetPendingPayments(ctx, userID)
	if err != nil {
		b.sendText(userID, "⚠️ "+err.Error())
		return
	}

	if len(payments) == 0 {
		b.sendText(userID, "✨ No hay pagos pendientes para validar.")
		return
	}

	b.sendText(userID, "💵 *Pagos Pendientes:*")
	for _, p := range payments {
		text := fmt.Sprintf("💰 *Pago Nro %d:*\nMonto: $%.2f\n📅 *Fecha Cobro:* %s\n", p.SeqNum, p.Amount, p.BillingDate.Format("02-01-2006"))

		if user.Role == "admin" {
			btnApprove := tgbotapi.NewInlineKeyboardButtonData("✅ Aprobar", fmt.Sprintf("approvePayment_%d", p.SeqNum))
			btnReject := tgbotapi.NewInlineKeyboardButtonData("❌ Rechazar", fmt.Sprintf("rejectPayment_%d", p.SeqNum))
			kb := tgbotapi.NewInlineKeyboardMarkup([]tgbotapi.InlineKeyboardButton{btnApprove, btnReject})

			msg := tgbotapi.NewMessage(userID, text)
			msg.ParseMode = tgbotapi.ModeMarkdown
			msg.ReplyMarkup = kb
			_, _ = b.api.Send(msg)
		} else {
			text += "⏳ Estado: Esperando aprobación del Admin."
			b.sendText(userID, text)
		}
	}
}

func (b *Bot) showKitchenSchedule(ctx context.Context, userID int64, user *domain.User) {
	meals, err := b.appService.GetMealSchedule(ctx, userID)
	if err != nil {
		b.sendText(userID, "⚠️ "+err.Error())
		return
	}

	roomies, _ := b.appService.GetUsersInGroup(ctx, userID)
	userMap := make(map[string]string)
	for _, r := range roomies {
		userMap[r.ID] = r.Name
	}

	days := []string{"", "Lunes", "Martes", "Miércoles", "Jueves", "Viernes", "Sábado", "Domingo"}
	mealTypes := []string{"breakfast", "lunch", "dinner"}
	mealNamesSp := map[string]string{
		"breakfast": "Desayuno 🥞",
		"lunch":     "Almuerzo 🍛",
		"dinner":    "Cena 🍕",
	}

	chefMatrix := make(map[string]string)
	for _, m := range meals {
		key := fmt.Sprintf("%d_%s", m.DayOfWeek, m.MealType)
		if m.ChefID != nil {
			if name, ok := userMap[*m.ChefID]; ok {
				chefMatrix[key] = name
			}
		}
	}

	b.sendText(userID, "🍳 *Rol de Cocina Compartido:*")

	for d := 1; d <= 7; d++ {
		var dayText strings.Builder
		dayText.WriteString(fmt.Sprintf("📅 *%s:*\n", days[d]))

		for _, mt := range mealTypes {
			key := fmt.Sprintf("%d_%s", d, mt)
			chef, hasChef := chefMatrix[key]

			if hasChef {
				dayText.WriteString(fmt.Sprintf("  • %s: Cocina *%s*\n", mealNamesSp[mt], chef))
			} else {
				dayText.WriteString(fmt.Sprintf("  • %s: *Sin asignar* /cocinar\n", mealNamesSp[mt]))
			}
		}

		var buttons []tgbotapi.InlineKeyboardButton
		for _, mt := range mealTypes {
			key := fmt.Sprintf("%d_%s", d, mt)
			if _, ok := chefMatrix[key]; !ok {
				buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(
					fmt.Sprintf("Yo %s", mealNamesSp[mt]),
					fmt.Sprintf("setChef_%d_%s", d, mt),
				))
			}
		}

		msg := tgbotapi.NewMessage(userID, dayText.String())
		msg.ParseMode = tgbotapi.ModeMarkdown
		if len(buttons) > 0 {
			msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons)
		}
		_, _ = b.api.Send(msg)
	}
}

func (b *Bot) listHabits(ctx context.Context, userID int64) {
	habitsWithStreak, err := b.appService.GetPersonalHabitsWithStreak(ctx, userID)
	if err != nil {
		b.sendText(userID, "⚠️ "+err.Error())
		return
	}

	if len(habitsWithStreak) == 0 {
		b.sendText(userID, "🏋️‍♂️ No tienes hábitos personales registrados. Crea uno con /crearhabito.")
		return
	}

	b.sendText(userID, "🏋️‍♂️ *Tus Hábitos y Metas Personales:*")
	for _, hs := range habitsWithStreak {
		text := fmt.Sprintf("💪 *Hábito Nro %d:* %s\n🗓️ *Días:* %s\n🔥 *Racha:* %d días\n📈 *Progreso:* %s\n", 
			hs.Habit.SeqNum, hs.Habit.ActivityType, hs.Habit.ScheduledDays, hs.Streak, hs.Habit.ProgressStatus)

		btnDone := tgbotapi.NewInlineKeyboardButtonData("🔥 Completar Hoy", fmt.Sprintf("updateHabit_%d", hs.Habit.SeqNum))
		btnDel := tgbotapi.NewInlineKeyboardButtonData("🗑️ Borrar", fmt.Sprintf("deleteHabit_%d", hs.Habit.SeqNum))
		kb := tgbotapi.NewInlineKeyboardMarkup([]tgbotapi.InlineKeyboardButton{btnDone, btnDel})

		msg := tgbotapi.NewMessage(userID, text)
		msg.ParseMode = tgbotapi.ModeMarkdown
		msg.ReplyMarkup = kb
		_, _ = b.api.Send(msg)
	}
}

// Menu layout helpers
func (b *Bot) sendWelcome(userID int64, user *domain.User) {
	welcome := `👋 ¡Hola, %s! Bienvenido al Asistente de Gestión del Hogar y Disciplina.

Este bot te ayudará a ti y a tus roomies a coordinar tareas, pagos de alquiler y turnos de cocina, además de ayudarte con tu disciplina personal (gimnasio, comidas y metas de empleo).

🤖 Puedes chatear conmigo libremente en cualquier momento para pedir resúmenes de tareas, consejos de productividad, o informarme algo.

Para empezar, selecciona una opción del menú o escribe un comando.`

	b.sendText(userID, fmt.Sprintf(welcome, user.Name))
	b.sendMainMenu(userID, user)
}

func (b *Bot) sendMainMenu(userID int64, user *domain.User) {
	text := "📌 *Menú Principal:*"
	if user.TenantID == nil {
		text += "\n\n⚠️ Actualmente no estás en ningún grupo de roomies. Elige una opción:"
		btnCreate := tgbotapi.NewKeyboardButton("/creargrupo")
		btnJoin := tgbotapi.NewKeyboardButton("/unirse")
		btnHelp := tgbotapi.NewKeyboardButton("/ayuda")
		kb := tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(btnCreate, btnJoin),
			tgbotapi.NewKeyboardButtonRow(btnHelp),
		)
		msg := tgbotapi.NewMessage(userID, text)
		msg.ParseMode = tgbotapi.ModeMarkdown
		msg.ReplyMarkup = kb
		_, _ = b.api.Send(msg)
	} else {
		groupName := "Cargando..."
		group, err := b.appService.GetGroupByID(context.Background(), *user.TenantID)
		if err == nil {
			groupName = group.GroupName
		}
		text += fmt.Sprintf("\n\n🏠 *Grupo:* %s\n🔑 *Código para Roomies:* `%s`\n👤 *Rol:* %s", groupName, *user.TenantID, user.Role)
		btnTasks := tgbotapi.NewKeyboardButton("/tareas")
		btnPayments := tgbotapi.NewKeyboardButton("/pagospendientes")
		btnKitchen := tgbotapi.NewKeyboardButton("/cocina")
		btnHabits := tgbotapi.NewKeyboardButton("/habitos")
		btnHelp := tgbotapi.NewKeyboardButton("/ayuda")

		kb := tgbotapi.NewReplyKeyboard(
			tgbotapi.NewKeyboardButtonRow(btnTasks, btnPayments, btnKitchen),
			tgbotapi.NewKeyboardButtonRow(btnHabits, btnHelp),
		)
		msg := tgbotapi.NewMessage(userID, text)
		msg.ParseMode = tgbotapi.ModeMarkdown
		msg.ReplyMarkup = kb
		_, _ = b.api.Send(msg)
	}
}

func (b *Bot) sendHelp(userID int64) {
	helpText := `📖 *Comandos Disponibles:*

🏠 *Grupo y Roomies:*
/creargrupo - Crea un nuevo grupo de roomies
/unirse - Únete a un grupo existente usando su UUID
/roomies - Listar miembros de tu grupo (También /miembros)

🧹 *Tareas de Limpieza:*
/tareas - Listar tareas pendientes (permite completarlas y eliminarlas)
/creartarea - Crear una nueva tarea interactiva (asignando roomie y vencimiento)

💰 *Pagos y Servicios:*
/pagar - Subir un comprobante de pago
/pagospendientes - Ver pagos pendientes por validar

🍳 *Cocina:*
/cocina - Ver y organizar los turnos de cocina de la semana
/asignarcocina - Asignar turnos de cocina a roomies específicos (Solo Admin)

🏋️‍♂️ *Disciplina y Hábitos:*
/habitos - Ver tus hábitos personales, rachas (permite completarlos y eliminarlos)
/crearhabito - Crear un nuevo hábito con alertas horarias

🤖 *Groq AI:*
Simplemente envíame un mensaje de texto normal para charlar sobre tus tareas, finanzas o solicitar recordatorios y planes.

❌ Escribe *cancelar* en cualquier momento para salir del asistente de un comando.`

	b.sendText(userID, helpText)
}

func (b *Bot) sendText(userID int64, text string) {
	msg := tgbotapi.NewMessage(userID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	_, err := b.api.Send(msg)
	if err != nil {
		log.Printf("Error enviando mensaje a %d: %v", userID, err)
	}
}

// FSM Helper methods
func (b *Bot) setState(userID int64, state int, data map[string]interface{}) {
	b.statesMu.Lock()
	defer b.statesMu.Unlock()
	b.states[userID] = &UserState{
		State: state,
		Data:  data,
	}
}

func (b *Bot) clearState(userID int64) {
	b.statesMu.Lock()
	defer b.statesMu.Unlock()
	delete(b.states, userID)
}

func (b *Bot) SendNotification(tgUserID int64, text string) {
	b.sendText(tgUserID, text)
}
