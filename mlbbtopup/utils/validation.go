package utils

import (
	"regexp"
	"strconv"
	"strings"
)

func ValidateGameID(gameID string) bool {
	if len(gameID) < 6 || len(gameID) > 10 {
		return false
	}
	_, err := strconv.Atoi(gameID)
	return err == nil
}

func ValidateServerID(serverID string) bool {
	if len(serverID) < 3 || len(serverID) > 5 {
		return false
	}
	_, err := strconv.Atoi(serverID)
	return err == nil
}

func ValidatePubgID(playerID string) bool {
	if len(playerID) < 7 || len(playerID) > 11 {
		return false
	}
	_, err := strconv.Atoi(playerID)
	return err == nil
}

func IsBannedAccount(gameID string) bool {
	bannedIDs := []string{"123456789", "000000000", "111111111"}
	for _, bannedID := range bannedIDs {
		if gameID == bannedID {
			return true
		}
	}
	
	// Check if all digits are same
	if len(gameID) > 0 {
		firstChar := gameID[0]
		for i := 1; i < len(gameID); i++ {
			if gameID[i] != firstChar {
				return false
			}
		}
		return true
	}
	return false
}

func GetPrice(diamonds string, customPrices map[string]interface{}) int {
	// Check custom prices first
	if price, ok := customPrices[diamonds].(int); ok {
		return price
	}

	// Default prices
	defaultPrices := map[string]int{
		"11": 950, "22": 1900, "33": 2850, "56": 4200, "112": 8200,
		"86": 5100, "172": 10200, "257": 15300, "343": 20400,
		"429": 25500, "514": 30600, "600": 35700, "706": 40800,
		"878": 51000, "963": 56100, "1049": 61200, "1135": 66300,
		"1412": 81600, "2195": 122400, "3688": 204000,
		"5532": 306000, "9288": 510000, "12976": 714000,
		"55": 3500, "165": 10000, "275": 16000, "565": 33000,
	}

	// Weekly passes
	if strings.HasPrefix(diamonds, "wp") {
		weekNum, err := strconv.Atoi(diamonds[2:])
		if err == nil && weekNum >= 1 && weekNum <= 10 {
			return weekNum * 6000
		}
	}

	return defaultPrices[diamonds]
}

func GetPubgPrice(ucAmount string, customPrices map[string]interface{}) int {
	// Check custom prices first
	if price, ok := customPrices[ucAmount].(int); ok {
		return price
	}

	// Default prices
	defaultPrices := map[string]int{
		"60uc":   1500,
		"325uc":  7500,
		"660uc":  15000,
		"1800uc": 37500,
		"3850uc": 75000,
		"8100uc": 150000,
	}

	return defaultPrices[ucAmount]
}

func GenerateOrderID() string {
	return "ORD" + strconv.FormatInt(time.Now().Unix(), 10)
}

func GenerateTopupID(userID string) string {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	if len(userID) >= 4 {
		return "TOP" + timestamp + userID[len(userID)-4:]
	}
	return "TOP" + timestamp + userID
}

func IsPaymentScreenshot(message *tgbotapi.Message) bool {
	return message.Photo != nil && len(message.Photo) > 0
}
