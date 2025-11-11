package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.mongodb.org/mongo-driver/bson"

	"mlbbtopup/database"
	"mlbbtopup/models"
)

func FormatNumber(number int) string {
	return fmt.Sprintf("%d", number)
}

func FormatCurrency(amount int) string {
	return fmt.Sprintf("%,d MMK", amount)
}

func EscapeMarkdown(text string) string {
	chars := []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}
	for _, char := range chars {
		text = strings.ReplaceAll(text, char, "\\"+char)
	}
	return text
}

func GetUserDisplayName(user *tgbotapi.User) string {
	name := user.FirstName
	if user.LastName != "" {
		name += " " + user.LastName
	}
	return name
}

func ConvertOrderToBSON(order models.Order) bson.M {
	return bson.M{
		"order_id":   order.OrderID,
		"game_id":    order.GameID,
		"server_id":  order.ServerID,
		"amount":     order.Amount,
		"price":      order.Price,
		"status":     order.Status,
		"timestamp":  order.Timestamp,
		"user_id":    order.UserID,
		"chat_id":    order.ChatID,
	}
}

func ConvertTopupToBSON(topup models.Topup) bson.M {
	return bson.M{
		"topup_id":       topup.TopupID,
		"amount":         topup.Amount,
		"payment_method": topup.PaymentMethod,
		"status":         topup.Status,
		"timestamp":      topup.Timestamp,
		"user_id":        topup.UserID,
		"chat_id":        topup.ChatID,
	}
}

func HasPendingTopup(db *database.DBManager, userID string) (bool, error) {
	user, err := db.GetUser(userID)
	if err != nil {
		return false, err
	}
	
	if user == nil {
		return false, nil
	}
	
	for _, topup := range user.Topups {
		if topup.Status == "pending" {
			return true, nil
		}
	}
	return false, nil
}

func SimpleReply(messageText string) string {
	messageLower := strings.ToLower(messageText)

	// Greetings
	if strings.Contains(messageLower, "hello") || strings.Contains(messageLower, "hi") || 
	   strings.Contains(messageLower, "မင်္ဂလာပါ") || strings.Contains(messageLower, "ဟယ်လို") ||
	   strings.Contains(messageLower, "ဟိုင်း") || strings.Contains(messageLower, "ကောင်းလား") {
		return "👋 မင်္ဂလာပါ! 𝗦𝗔𝗦𝗨𝗞𝗘 𝗠𝗟𝗕𝗕 𝗧𝗢𝗣 𝗨𝗣 𝗕𝗢𝗧 မှ ကြိုဆိုပါတယ်!\n\n📱 Bot commands များ သုံးရန် /start နှိပ်ပါ\n"
	}

	// Help requests
	if strings.Contains(messageLower, "help") || strings.Contains(messageLower, "ကူညီ") || 
	   strings.Contains(messageLower, "အကူအညီ") || strings.Contains(messageLower, "မသိ") ||
	   strings.Contains(messageLower, "လမ်းညွှန်") {
		return "📱 ***အသုံးပြုနိုင်တဲ့ commands:***\n\n" +
			"• /start - Bot စတင်အသုံးပြုရန်\n" +
			"• /mmb gameid serverid amount - Diamond ဝယ်ယူရန်\n" +
			"• /balance - လက်ကျန်ငွေ စစ်ရန်\n" +
			"• /topup amount - ငွေဖြည့်ရန်\n" +
			"• /price - ဈေးနှုန်းများ ကြည့်ရန်\n" +
			"• /history - မှတ်တမ်းများ ကြည့်ရန်\n\n" +
			"💡 အသေးစိတ် လိုအပ်ရင် admin ကို ဆက်သွယ်ပါ!"
	}

	// Default response
	return "📱 ***MLBB Diamond Top-up Bot***\n\n" +
		"💎 ***Diamond ဝယ်ယူရန် /mmb command သုံးပါ။***\n" +
		"💰 ***ဈေးနှုန်းများ သိရှိရန် /price နှိပ်ပါ။***\n" +
		"🆘 ***အကူအညီ လိုရင် /start နှိပ်ပါ။***"
}
