package handlers

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.mongodb.org/mongo-driver/bson"

	"mlbbtopup/database"
	"mlbbtopup/models"
	"mlbbtopup/utils"
)

type UserHandler struct {
	bot      *tgbotapi.BotAPI
	db       *database.DBManager
	config   *models.Config
}

func NewUserHandler(bot *tgbotapi.BotAPI, db *database.DBManager, config *models.Config) *UserHandler {
	return &UserHandler{
		bot:    bot,
		db:     db,
		config: config,
	}
}

func (h *UserHandler) HandleStart(message *tgbotapi.Message, args string) {
	user := message.From
	userID := strconv.FormatInt(user.ID, 10)
	username := user.UserName
	if username == "" {
		username = "-"
	}
	name := utils.GetUserDisplayName(user)

	// Check authorization
	authorizedUsers, err := h.db.LoadAuthorizedUsers()
	if err != nil {
		log.Printf("Error loading authorized users: %v", err)
		return
	}

	if !authorizedUsers[userID] && userID != strconv.FormatInt(h.config.AdminID, 10) {
		h.handleRegistrationRequest(user)
		return
	}

	// Handle referrer ID if provided
	var referrerID *string
	if args != "" {
		refID := args
		if refID != userID {
			referrerID = &refID
		}
	}

	// Check for pending topups
	hasPending, err := utils.HasPendingTopup(h.db, userID)
	if err != nil {
		log.Printf("Error checking pending topup: %v", err)
	}
	if hasPending {
		h.sendPendingTopupWarning(message.Chat.ID)
		return
	}

	// Create or update user
	userDoc, err := h.db.GetUser(userID)
	if err != nil {
		log.Printf("Error getting user: %v", err)
		return
	}

	if userDoc == nil {
		// New user
		err = h.db.CreateUser(userID, name, username, referrerID)
		if err != nil {
			log.Printf("Error creating user: %v", err)
		}

		// Notify referrer
		if referrerID != nil {
			h.notifyReferrer(*referrerID, name, userID)
		}
	} else {
		// Update existing user profile
		err = h.db.UpdateUserProfile(userID, name, username)
		if err != nil {
			log.Printf("Error updating user profile: %v", err)
		}
	}

	// Clear user state
	// Note: In Go implementation, we might use a different approach for user states

	// Send welcome message
	h.sendWelcomeMessage(message.Chat.ID, userID, name)
}

func (h *UserHandler) HandleMmb(message *tgbotapi.Message, args string) {
	userID := strconv.FormatInt(message.From.ID, 10)

	// Authorization check
	authorizedUsers, err := h.db.LoadAuthorizedUsers()
	if err != nil {
		log.Printf("Error loading authorized users: %v", err)
		return
	}

	if !authorizedUsers[userID] && userID != strconv.FormatInt(h.config.AdminID, 10) {
		h.sendNotAuthorizedMessage(message.Chat.ID)
		return
	}

	// Maintenance check
	settings, err := h.db.LoadSettings(
		map[string]interface{}{}, // defaultPayment
		map[string]interface{}{}, // defaultMaintenance  
		map[string]interface{}{}, // defaultAffiliate
		map[string]interface{}{}, // defaultAutoDelete
	)
	if err != nil {
		log.Printf("Error loading settings: %v", err)
	}

	if maintenance, ok := settings["maintenance"].(map[string]interface{}); ok {
		if orders, ok := maintenance["orders"].(bool); ok && !orders {
			h.sendMaintenanceMessage(message.Chat.ID, "orders")
			return
		}
	}

	// User state check (simplified in Go version)
	// if userStates[userID] == "waiting_approval" {
	//     h.sendWaitingApprovalMessage(message.Chat.ID)
	//     return
	// }

	// Pending topup check
	hasPending, err := utils.HasPendingTopup(h.db, userID)
	if err != nil {
		log.Printf("Error checking pending topup: %v", err)
	}
	if hasPending {
		h.sendPendingTopupWarning(message.Chat.ID)
		return
	}

	// Parse arguments
	argList := strings.Fields(args)
	if len(argList) != 3 {
		h.sendInvalidFormatMessage(message.Chat.ID, "/mmb gameid serverid amount")
		return
	}

	gameID, serverID, amount := argList[0], argList[1], argList[2]

	// Validation
	if !utils.ValidateGameID(gameID) {
		h.sendInvalidGameIDMessage(message.Chat.ID)
		return
	}

	if !utils.ValidateServerID(serverID) {
		h.sendInvalidServerIDMessage(message.Chat.ID)
		return
	}

	if utils.IsBannedAccount(gameID) {
		h.sendBannedAccountMessage(message.Chat.ID, gameID)
		h.notifyAdminsAboutBannedAccount(message.From, gameID, serverID, amount)
		return
	}

	// Load custom prices
	customPrices, err := h.db.LoadPrices()
	if err != nil {
		log.Printf("Error loading prices: %v", err)
		customPrices = make(map[string]interface{})
	}

	price := utils.GetPrice(amount, customPrices)
	if price == 0 {
		h.sendInvalidAmountMessage(message.Chat.ID)
		return
	}

	// Check balance
	userDoc, err := h.db.GetUser(userID)
	if err != nil {
		log.Printf("Error getting user: %v", err)
		return
	}

	if userDoc.Balance < price {
		h.sendInsufficientBalanceMessage(message.Chat.ID, price, userDoc.Balance)
		return
	}

	// Create order
	orderID := utils.GenerateOrderID()
	order := models.Order{
		OrderID:   orderID,
		GameID:    gameID,
		ServerID:  serverID,
		Amount:    amount,
		Price:     price,
		Status:    "pending",
		Timestamp: time.Now(),
		UserID:    userID,
		ChatID:    message.Chat.ID,
	}

	// Convert order to BSON for storage
	orderBSON := utils.ConvertOrderToBSON(order)

	// Update balance and add order
	err = h.db.UpdateBalance(userID, -price)
	if err != nil {
		log.Printf("Error updating balance: %v", err)
		return
	}

	err = h.db.AddOrder(userID, orderBSON)
	if err != nil {
		log.Printf("Error adding order: %v", err)
		return
	}

	newBalance := userDoc.Balance - price

	// Notify admins
	h.notifyAdminsAboutNewOrder(order, message.From, newBalance)

	// Send confirmation to user
	h.sendOrderConfirmation(message.Chat.ID, orderID, gameID, serverID, amount, price, newBalance)
}

func (h *UserHandler) HandleBalance(message *tgbotapi.Message) {
	userID := strconv.FormatInt(message.From.ID, 10)

	// Authorization check
	authorizedUsers, err := h.db.LoadAuthorizedUsers()
	if err != nil {
		log.Printf("Error loading authorized users: %v", err)
		return
	}

	if !authorizedUsers[userID] && userID != strconv.FormatInt(h.config.AdminID, 10) {
		h.sendNotAuthorizedMessage(message.Chat.ID)
		return
	}

	// User state checks (simplified in Go version)
	// if userStates[userID] == "waiting_approval" {
	//     h.sendWaitingApprovalMessage(message.Chat.ID)
	//     return
	// }

	// Pending topup check
	// if hasPendingTopup(userID) {
	//     h.sendPendingTopupProcessMessage(message.Chat.ID)
	//     return
	// }

	// Get user data
	userDoc, err := h.db.GetUser(userID)
	if err != nil {
		log.Printf("Error getting user: %v", err)
		return
	}

	if userDoc == nil {
		h.sendStartFirstMessage(message.Chat.ID)
		return
	}

	// Update user profile
	name := utils.GetUserDisplayName(message.From)
	username := message.From.UserName
	if username == "" {
		username = "-"
	}
	h.db.UpdateUserProfile(userID, name, username)

	// Calculate pending topups
	pendingCount := 0
	pendingAmount := 0
	for _, topup := range userDoc.Topups {
		if topup.Status == "pending" {
			pendingCount++
			pendingAmount += topup.Amount
		}
	}

	// Send balance information
	h.sendBalanceInfo(message.Chat.ID, userDoc, pendingCount, pendingAmount)
}

func (h *UserHandler) HandleTopup(message *tgbotapi.Message, args string) {
	userID := strconv.FormatInt(message.From.ID, 10)

	// Authorization check
	authorizedUsers, err := h.db.LoadAuthorizedUsers()
	if err != nil {
		log.Printf("Error loading authorized users: %v", err)
		return
	}

	if !authorizedUsers[userID] && userID != strconv.FormatInt(h.config.AdminID, 10) {
		h.sendNotAuthorizedMessage(message.Chat.ID)
		return
	}

	// Maintenance check
	settings, err := h.db.LoadSettings(
		map[string]interface{}{}, // defaultPayment
		map[string]interface{}{}, // defaultMaintenance  
		map[string]interface{}{}, // defaultAffiliate
		map[string]interface{}{}, // defaultAutoDelete
	)
	if err != nil {
		log.Printf("Error loading settings: %v", err)
	}

	if maintenance, ok := settings["maintenance"].(map[string]interface{}); ok {
		if topups, ok := maintenance["topups"].(bool); ok && !topups {
			h.sendMaintenanceMessage(message.Chat.ID, "topups")
			return
		}
	}

	// User state checks (simplified in Go version)
	// if userStates[userID] == "waiting_approval" {
	//     h.sendWaitingApprovalMessage(message.Chat.ID)
	//     return
	// }

	// Pending topup check
	hasPending, err := utils.HasPendingTopup(h.db, userID)
	if err != nil {
		log.Printf("Error checking pending topup: %v", err)
	}
	if hasPending {
		h.sendPendingTopupProcessMessage(message.Chat.ID)
		return
	}

	// Parse amount
	argList := strings.Fields(args)
	if len(argList) != 1 {
		h.sendInvalidFormatMessage(message.Chat.ID, "/topup amount")
		return
	}

	amount, err := strconv.Atoi(argList[0])
	if err != nil || amount < 1000 {
		h.sendInvalidAmountMessage(message.Chat.ID)
		return
	}

	// Store pending topup (in-memory, in production you might want to use Redis)
	// pendingTopups[userID] = &models.PendingTopup{
	//     Amount:    amount,
	//     Timestamp: time.Now(),
	// }

	// Send payment method selection
	h.sendPaymentMethodSelection(message.Chat.ID, amount)
}

// Message sending helper methods
func (h *UserHandler) sendWelcomeMessage(chatID int64, userID string, name string) {
	text := fmt.Sprintf("👋 ***မင်္ဂလာပါ*** [%s](tg://user?id=%s)!\\n\\n"+
		"💎 ***SASUKE MLBB TOP UP BOT*** မှ ကြိုဆိုပါတယ်\\.\\n\\n"+
		"***အသုံးပြုနိုင်တဲ့ command များ***:\\n"+
		"➤ /mmb gameid serverid amount\\n"+
		"➤ /balance \\- ဘယ်လောက်လက်ကျန်ရှိလဲ စစ်မယ်\\n"+
		"➤ /topup amount \\- ငွေဖြည့်မယ် \\(screenshot တင်ပါ\\)\\n"+
		"➤ /price \\- Diamond များရဲ့ ဈေးနှုန်းများ\\n"+
		"➤ /history \\- အော်ဒါမှတ်တမ်းကြည့်မယ်\\n\\n"+
		"***📌 ဥပမာ***:\\n"+
		"`/mmb 123456789 12345 wp1`\\n\\n"+
		"***လိုအပ်တာရှိရင် Owner ကို ဆက်သွယ်နိုင်ပါတယ်\\.***",
		utils.EscapeMarkdown(name), userID)

	utils.SendMessage(h.bot, chatID, text, "MarkdownV2")
}

func (h *UserHandler) sendNotAuthorizedMessage(chatID int64) {
	keyboard := utils.CreateInlineKeyboard([][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonURL("👑 Contact Owner", 
				fmt.Sprintf("tg://user?id=%d", h.config.AdminID)),
		},
	})

	text := "🚫 အသုံးပြုခွင့် မရှိပါ!\\n\\nOwner ထံ bot အသုံးပြုခွင့် တောင်းဆိုပါ\\."
	utils.SendMessageWithKeyboard(h.bot, chatID, text, "MarkdownV2", keyboard)
}

func (h *UserHandler) sendMaintenanceMessage(chatID int64, commandType string) {
	var text string
	
	switch commandType {
	case "orders":
		text = "⏸️ ***Bot အော်ဒါတင်ခြင်းအား ခေတ္တ ယာယီပိတ်ထားပါသည်*** ⏸️\\n\\n" +
			"***🔄 Admin မှ ပြန်လည်ဖွင့်ပေးမှ အသုံးပြုနိုင်ပါမည်\\.***"
	case "topups":
		text = "⏸️ ***Bot ငွေဖြည့်ခြင်းအား ခေတ္တ ယာယီပိတ်ထားပါသည်*** ⏸️\\n\\n" +
			"***🔄 Admin မှ ပြန်လည်ဖွင့်ပေးမှ အသုံးပြုနိုင်ပါမည်\\.***"
	default:
		text = "⏸️ ***Bot အား ခေတ္တ ယာယီပိတ်ထားပါသည်*** ⏸️\\n\\n" +
			"***🔄 Admin မှ ပြန်လည်ဖွင့်ပေးမှ အသုံးပြုနိုင်ပါမည်\\.***"
	}
	
	utils.SendMessage(h.bot, chatID, text, "MarkdownV2")
}

// Additional helper methods would be implemented here...
// (sendPendingTopupWarning, sendInvalidFormatMessage, sendInvalidGameIDMessage, etc.)

// Note: The complete implementation would include all the message sending methods
// and notification methods from the Python version.
