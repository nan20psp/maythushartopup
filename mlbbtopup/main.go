package main

import (
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"mlbbtopup/config"
	"mlbbtopup/database"
	"mlbbtopup/handlers"
	"mlbbtopup/models"
)

var (
	userHandler    *handlers.UserHandler
	adminHandler   *handlers.AdminHandler
	callbackHandler *handlers.CallbackHandler
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize database
	db, err := database.NewDBManager(cfg.MongoURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize Telegram bot
	bot, err := tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		log.Fatalf("Failed to initialize bot: %v", err)
	}

	bot.Debug = true
	log.Printf("Authorized on account %s", bot.Self.UserName)

	// Setup special user
	setupSpecialUser(db, cfg)

	// Initialize handlers
	appConfig := &models.Config{
		BotToken:     cfg.BotToken,
		AdminID:      cfg.AdminID,
		MongoURL:     cfg.MongoURL,
		AdminGroupID: cfg.AdminGroupID,
	}

	userHandler = handlers.NewUserHandler(bot, db, appConfig)
	adminHandler = handlers.NewAdminHandler(bot, db, appConfig)
	callbackHandler = handlers.NewCallbackHandler(bot, db, appConfig)

	// Start bot
	startBot(bot)
}

func setupSpecialUser(db *database.DBManager, cfg *config.Config) {
	specialUserID := "7499503874"
	initialBalance := 5000

	user, err := db.GetUser(specialUserID)
	if err != nil {
		log.Printf("Error checking special user: %v", err)
		return
	}

	if user == nil {
		// Create user
		err = db.CreateUser(specialUserID, "Special User", "N/A", nil)
		if err != nil {
			log.Printf("Error creating special user: %v", err)
			return
		}

		// Set balance
		err = db.SetBalance(specialUserID, initialBalance)
		if err != nil {
			log.Printf("Error setting balance for special user: %v", err)
		}
		log.Printf("Created special user %s with balance %d", specialUserID, initialBalance)
	} else if user.Balance == 0 && len(user.Orders) == 0 && len(user.Topups) == 0 {
		// Set initial balance if user has no activity
		err = db.SetBalance(specialUserID, initialBalance)
		if err != nil {
			log.Printf("Error setting balance for special user: %v", err)
		}
	}

	// Ensure user is authorized
	authorizedUsers, err := db.LoadAuthorizedUsers()
	if err != nil {
		log.Printf("Error loading authorized users: %v", err)
		return
	}

	if !authorizedUsers[specialUserID] {
		err = db.AddAuthorizedUser(specialUserID)
		if err != nil {
			log.Printf("Error authorizing special user: %v", err)
		}
	}
}

func startBot(bot *tgbotapi.BotAPI) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	log.Println("🤖 Bot is now running...")

	for update := range updates {
		handleUpdate(update)
	}
}

func handleUpdate(update tgbotapi.Update) {
	if update.Message != nil {
		handleMessage(update.Message)
	} else if update.CallbackQuery != nil {
		handleCallbackQuery(update.CallbackQuery)
	}
}

func handleMessage(message *tgbotapi.Message) {
	if message.IsCommand() {
		handleCommand(message)
		return
	}

	// Handle photos (payment screenshots)
	if message.Photo != nil && len(message.Photo) > 0 {
		handlePhoto(message)
		return
	}

	// Handle other text messages
	if message.Text != "" {
		handleTextMessage(message)
	}
}

func handleCommand(message *tgbotapi.Message) {
	command := message.Command()
	args := message.CommandArguments()
	userID := strconv.FormatInt(message.From.ID, 10)

	// Check if user is admin for admin commands
	isAdmin := isUserAdmin(userID)

	switch command {
	case "start":
		userHandler.HandleStart(message, args)
	case "mmb":
		userHandler.HandleMmb(message, args)
	case "balance":
		userHandler.HandleBalance(message)
	case "topup":
		userHandler.HandleTopup(message, args)
	case "price":
		handlePriceCommand(message)
	case "history":
		handleHistoryCommand(message)
	case "register":
		handleRegisterCommand(message)
	case "affiliate":
		handleAffiliateCommand(message)

	// Admin commands
	case "approve":
		if isAdmin {
			adminHandler.HandleApprove(message, args)
		} else {
			sendNotAdminMessage(message.Chat.ID)
		}
	case "deduct":
		if isAdmin {
			adminHandler.HandleDeduct(message, args)
		} else {
			sendNotAdminMessage(message.Chat.ID)
		}
	case "ban":
		if isAdmin {
			adminHandler.HandleBan(message, args)
		} else {
			sendNotAdminMessage(message.Chat.ID)
		}
	case "unban":
		if isAdmin {
			adminHandler.HandleUnban(message, args)
		} else {
			sendNotAdminMessage(message.Chat.ID)
		}
	case "setprice":
		if isAdmin {
			adminHandler.HandleSetPrice(message, args)
		} else {
			sendNotAdminMessage(message.Chat.ID)
		}
	case "maintenance":
		if isAdmin {
			adminHandler.HandleMaintenance(message, args)
		} else {
			sendNotAdminMessage(message.Chat.ID)
		}
	default:
		handleUnknownCommand(message)
	}
}

func handleCallbackQuery(callback *tgbotapi.CallbackQuery) {
	callbackHandler.HandleCallback(callback)
}

func handlePhoto(message *tgbotapi.Message) {
	// Handle payment screenshot
	// This would involve:
	// 1. Checking if user has pending topup
	// 2. Creating topup record
	// 3. Notifying admins
	// 4. Setting user state
	log.Printf("Received photo from user %d", message.From.ID)
}

func handleTextMessage(message *tgbotapi.Message) {
	// Handle non-command text messages
	// This could include:
	// 1. Auto-calculator functionality
	// 2. Simple replies to common questions
	// 3. Other text-based interactions
	reply := utils.SimpleReply(message.Text)
	if reply != "" {
		userID := strconv.FormatInt(message.From.ID, 10)
		// Check if user is authorized before sending reply
		authorizedUsers, err := userHandler.db.LoadAuthorizedUsers()
		if err == nil && (authorizedUsers[userID] || userID == strconv.FormatInt(userHandler.config.AdminID, 10)) {
			utils.SendMessage(userHandler.bot, message.Chat.ID, reply, "Markdown")
		}
	}
}

// Additional command handlers
func handlePriceCommand(message *tgbotapi.Message) {
	// Implement price command
	userID := strconv.FormatInt(message.From.ID, 10)
	authorizedUsers, err := userHandler.db.LoadAuthorizedUsers()
	if err != nil || (!authorizedUsers[userID] && userID != strconv.FormatInt(userHandler.config.AdminID, 10)) {
		userHandler.sendNotAuthorizedMessage(message.Chat.ID)
		return
	}

	// Load and send prices
	customPrices, err := userHandler.db.LoadPrices()
	if err != nil {
		log.Printf("Error loading prices: %v", err)
		customPrices = make(map[string]interface{})
	}

	priceMessage := generatePriceMessage(customPrices)
	utils.SendMessage(userHandler.bot, message.Chat.ID, priceMessage, "Markdown")
}

func handleHistoryCommand(message *tgbotapi.Message) {
	// Implement history command
	userID := strconv.FormatInt(message.From.ID, 10)
	
	authorizedUsers, err := userHandler.db.LoadAuthorizedUsers()
	if err != nil || (!authorizedUsers[userID] && userID != strconv.FormatInt(userHandler.config.AdminID, 10)) {
		userHandler.sendNotAuthorizedMessage(message.Chat.ID)
		return
	}

	userDoc, err := userHandler.db.GetUser(userID)
	if err != nil || userDoc == nil {
		userHandler.sendStartFirstMessage(message.Chat.ID)
		return
	}

	historyMessage := generateHistoryMessage(userDoc)
	utils.SendMessage(userHandler.bot, message.Chat.ID, historyMessage, "Markdown")
}

func handleRegisterCommand(message *tgbotapi.Message) {
	// Implement register command
	user := message.From
	userID := strconv.FormatInt(user.ID, 10)
	
	authorizedUsers, err := userHandler.db.LoadAuthorizedUsers()
	if err == nil && authorizedUsers[userID] {
		text := "✅ သင်သည် အသုံးပြုခွင့် ရပြီးသား ဖြစ်ပါတယ်!\n\n🚀 /start နှိပ်ပါ။"
		utils.SendMessage(userHandler.bot, message.Chat.ID, text, "Markdown")
		return
	}

	// Send registration request to admins
	userHandler.handleRegistrationRequest(user)
	
	// Send confirmation to user
	text := fmt.Sprintf("✅ ***Registration တောင်းဆိုမှု ပို့ပြီးပါပြီ!***\n\n🆔 ***သင့် User ID:*** `%s`\n\n⏳ ***Owner က approve လုပ်တဲ့အထိ စောင့်ပါ။***", userID)
	utils.SendMessage(userHandler.bot, message.Chat.ID, text, "Markdown")
}

func handleAffiliateCommand(message *tgbotapi.Message) {
	// Implement affiliate command
	userID := strconv.FormatInt(message.From.ID, 10)
	
	authorizedUsers, err := userHandler.db.LoadAuthorizedUsers()
	if err != nil || (!authorizedUsers[userID] && userID != strconv.FormatInt(userHandler.config.AdminID, 10)) {
		userHandler.sendNotAuthorizedMessage(message.Chat.ID)
		return
	}

	userDoc, err := userHandler.db.GetUser(userID)
	if err != nil || userDoc == nil {
		userHandler.sendStartFirstMessage(message.Chat.ID)
		return
	}

	affiliateMessage := generateAffiliateMessage(userDoc, userHandler.bot.Self.UserName)
	utils.SendMessage(userHandler.bot, message.Chat.ID, affiliateMessage, "Markdown")
}

func handleUnknownCommand(message *tgbotapi.Message) {
	text := "❌ ***မသိသော command ဖြစ်ပါတယ်!***\n\n💡 ***အသုံးပြုနိုင်သော commands များကို ကြည့်ရန် /start နှိပ်ပါ။***"
	utils.SendMessage(userHandler.bot, message.Chat.ID, text, "Markdown")
}

func isUserAdmin(userID string) bool {
	userIDInt, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return false
	}
	return userIDInt == userHandler.config.AdminID
}

func sendNotAdminMessage(chatID int64) {
	text := "❌ သင်သည် admin မဟုတ်ပါ!"
	utils.SendMessage(userHandler.bot, chatID, text, "Markdown")
}

// Helper functions for generating messages
func generatePriceMessage(customPrices map[string]interface{}) string {
	// Implement price message generation
	// This would combine default prices with custom prices
	// and format them nicely
	return "💎 ***MLBB Diamond ဈေးနှုန်းများ***\n\n(Price list implementation)"
}

func generateHistoryMessage(user *models.User) string {
	// Implement history message generation
	// This would show user's orders and topups
	return "📋 သင့်ရဲ့ မှတ်တမ်းများ\n\n(History implementation)" 
}

func generateAffiliateMessage(user *models.User, botUsername string) string {
	// Implement affiliate message generation
	referralLink := fmt.Sprintf("https://t.me/%s?start=%s", botUsername, user.UserID)
	
	return fmt.Sprintf("💸 ***Affiliate Program***\n\n"+
		"ဒီ bot လေးကို သူငယ်ချင်းတွေဆီ မျှဝေပြီး commission ရယူလိုက်ပါ။\n\n"+
		"**သင်၏ Referral Link:**\n"+
		"`%s`\n\n"+
		"💰 **စုစုပေါင်း ရရှိငွေ:** `%d MMK`",
		referralLink, user.ReferralEarnings)
}
