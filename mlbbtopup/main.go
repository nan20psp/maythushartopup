package main

import (
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"mlbbtopup/config"
	"mlbbtopup/database"
	"mlbbtopup/handlers"
	"mlbbtopup/models"
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
	userHandler := handlers.NewUserHandler(bot, db, &models.Config{
		BotToken:     cfg.BotToken,
		AdminID:      cfg.AdminID,
		MongoURL:     cfg.MongoURL,
		AdminGroupID: cfg.AdminGroupID,
	})

	// Start bot
	startBot(bot, userHandler)
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

func startBot(bot *tgbotapi.BotAPI, userHandler *handlers.UserHandler) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		handleUpdate(update, userHandler)
	}
}

func handleUpdate(update tgbotapi.Update, userHandler *handlers.UserHandler) {
	if update.Message != nil {
		handleMessage(update.Message, userHandler)
	} else if update.CallbackQuery != nil {
		handleCallbackQuery(update.CallbackQuery, userHandler)
	}
}

func handleMessage(message *tgbotapi.Message, userHandler *handlers.UserHandler) {
	if message.IsCommand() {
		handleCommand(message, userHandler)
		return
	}

	// Handle photos (payment screenshots)
	if message.Photo != nil && len(message.Photo) > 0 {
		// handlePhoto(message, userHandler)
		return
	}

	// Handle other messages
	// handleTextMessage(message, userHandler)
}

func handleCommand(message *tgbotapi.Message, userHandler *handlers.UserHandler) {
	command := message.Command()
	args := message.CommandArguments()

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
		// userHandler.HandlePrice(message)
	case "history":
		// userHandler.HandleHistory(message)
	case "register":
		// userHandler.HandleRegister(message)
	case "affiliate":
		// userHandler.HandleAffiliate(message)
	default:
		// Handle admin commands or unknown commands
	}
}

func handleCallbackQuery(callback *tgbotapi.CallbackQuery, userHandler *handlers.UserHandler) {
	// Handle callback queries from inline keyboards
	// This would be implemented in a separate callback handler
}
