package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	BotToken     string
	AdminID      int64
	MongoURL     string
	AdminGroupID int64
}

func LoadConfig() *Config {
	botToken := os.Getenv("BOT_TOKEN")
	adminIDStr := os.Getenv("ADMIN_ID")
	mongoURL := os.Getenv("MONGO_URL")
	adminGroupIDStr := os.Getenv("ADMIN_GROUP_ID")

	if botToken == "" || adminIDStr == "" || mongoURL == "" || adminGroupIDStr == "" {
		log.Fatal("Error: Environment variables (BOT_TOKEN, ADMIN_ID, MONGO_URL, ADMIN_GROUP_ID) are required")
	}

	adminID, err := strconv.ParseInt(adminIDStr, 10, 64)
	if err != nil {
		log.Fatalf("Error: Invalid ADMIN_ID: %v", err)
	}

	adminGroupID, err := strconv.ParseInt(adminGroupIDStr, 10, 64)
	if err != nil {
		log.Fatalf("Error: Invalid ADMIN_GROUP_ID: %v", err)
	}

	return &Config{
		BotToken:     botToken,
		AdminID:      adminID,
		MongoURL:     mongoURL,
		AdminGroupID: adminGroupID,
	}
}
