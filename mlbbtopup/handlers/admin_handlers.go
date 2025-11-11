package handlers

import (
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

type AdminHandler struct {
	bot      *tgbotapi.BotAPI
	db       *database.DBManager
	config   *models.Config
}

func NewAdminHandler(bot *tgbotapi.BotAPI, db *database.DBManager, config *models.Config) *AdminHandler {
	return &AdminHandler{
		bot:    bot,
		db:     db,
		config: config,
	}
}

func (h *AdminHandler) HandleApprove(message *tgbotapi.Message, args string) {
	userID := strconv.FormatInt(message.From.ID, 10)
	
	if !h.isAdmin(userID) {
		h.sendNotAdminMessage(message.Chat.ID)
		return
	}

	adminName := utils.GetUserDisplayName(message.From)
	argList := strings.Fields(args)
	
	if len(argList) != 2 {
		h.sendInvalidFormatMessage(message.Chat.ID, "/approve user_id amount")
		return
	}

	targetUserID := argList[0]
	amount, err := strconv.Atoi(argList[1])
	if err != nil {
		h.sendInvalidAmountMessage(message.Chat.ID)
		return
	}

	// Find and approve pending topup
	topupID, err := h.findPendingTopup(targetUserID, amount)
	if err != nil {
		h.sendTopupNotFoundMessage(message.Chat.ID, targetUserID, amount)
		return
	}

	updates := bson.M{
		"status":      "approved",
		"approved_by": adminName,
		"approved_at": time.Now(),
	}

	approvedUserID, err := h.db.FindAndUpdateTopup(topupID, updates)
	if err != nil {
		h.sendApprovalErrorMessage(message.Chat.ID)
		return
	}

	// Clear user state if exists
	// delete(userStates, targetUserID)

	// Notify user
	h.notifyUserAboutApproval(approvedUserID, amount, adminName)

	// Send confirmation to admin
	h.sendApprovalConfirmation(message.Chat.ID, targetUserID, amount)
}

func (h *AdminHandler) HandleDeduct(message *tgbotapi.Message, args string) {
	userID := strconv.FormatInt(message.From.ID, 10)
	
	if !h.isAdmin(userID) {
		h.sendNotAdminMessage(message.Chat.ID)
		return
	}

	argList := strings.Fields(args)
	if len(argList) != 2 {
		h.sendInvalidFormatMessage(message.Chat.ID, "/deduct user_id amount")
		return
	}

	targetUserID := argList[0]
	amount, err := strconv.Atoi(argList[1])
	if err != nil || amount <= 0 {
		h.sendInvalidAmountMessage(message.Chat.ID)
		return
	}

	userDoc, err := h.db.GetUser(targetUserID)
	if err != nil || userDoc == nil {
		h.sendUserNotFoundMessage(message.Chat.ID, targetUserID)
		return
	}

	if userDoc.Balance < amount {
		h.sendInsufficientBalanceForDeduction(message.Chat.ID, amount, userDoc.Balance)
		return
	}

	err = h.db.UpdateBalance(targetUserID, -amount)
	if err != nil {
		h.sendDeductionErrorMessage(message.Chat.ID)
		return
	}

	newBalance := userDoc.Balance - amount

	// Notify user
	h.notifyUserAboutDeduction(targetUserID, amount, newBalance)

	// Send confirmation to admin
	h.sendDeductionConfirmation(message.Chat.ID, targetUserID, amount, newBalance)
}

func (h *AdminHandler) HandleBan(message *tgbotapi.Message, args string) {
	userID := strconv.FormatInt(message.From.ID, 10)
	
	if !h.isAdmin(userID) {
		h.sendNotAdminMessage(message.Chat.ID)
		return
	}

	adminName := utils.GetUserDisplayName(message.From)
	argList := strings.Fields(args)
	
	if len(argList) != 1 {
		h.sendInvalidFormatMessage(message.Chat.ID, "/ban user_id")
		return
	}

	targetUserID := argList[0]
	authorizedUsers, err := h.db.LoadAuthorizedUsers()
	if err != nil {
		log.Printf("Error loading authorized users: %v", err)
		return
	}

	if !authorizedUsers[targetUserID] {
		h.sendUserNotAuthorizedMessage(message.Chat.ID)
		return
	}

	err = h.db.RemoveAuthorizedUser(targetUserID)
	if err != nil {
		h.sendBanErrorMessage(message.Chat.ID)
		return
	}

	// Notify user
	h.notifyUserAboutBan(targetUserID)

	// Notify admins
	h.notifyAdminsAboutBan(adminName, targetUserID)

	// Send confirmation
	h.sendBanConfirmation(message.Chat.ID, targetUserID)
}

func (h *AdminHandler) HandleUnban(message *tgbotapi.Message, args string) {
	userID := strconv.FormatInt(message.From.ID, 10)
	
	if !h.isAdmin(userID) {
		h.sendNotAdminMessage(message.Chat.ID)
		return
	}

	adminName := utils.GetUserDisplayName(message.From)
	argList := strings.Fields(args)
	
	if len(argList) != 1 {
		h.sendInvalidFormatMessage(message.Chat.ID, "/unban user_id")
		return
	}

	targetUserID := argList[0]
	authorizedUsers, err := h.db.LoadAuthorizedUsers()
	if err != nil {
		log.Printf("Error loading authorized users: %v", err)
		return
	}

	if authorizedUsers[targetUserID] {
		h.sendUserAlreadyAuthorizedMessage(message.Chat.ID)
		return
	}

	err = h.db.AddAuthorizedUser(targetUserID)
	if err != nil {
		h.sendUnbanErrorMessage(message.Chat.ID)
		return
	}

	// Clear user state if exists
	// delete(userStates, targetUserID)

	// Notify user
	h.notifyUserAboutUnban(targetUserID)

	// Notify admins
	h.notifyAdminsAboutUnban(adminName, targetUserID)

	// Send confirmation
	h.sendUnbanConfirmation(message.Chat.ID, targetUserID)
}

func (h *AdminHandler) HandleSetPrice(message *tgbotapi.Message, args string) {
	userID := strconv.FormatInt(message.From.ID, 10)
	
	if !h.isAdmin(userID) {
		h.sendNotAdminMessage(message.Chat.ID)
		return
	}

	argList := strings.Fields(args)
	if len(argList) < 2 {
		h.sendSetPriceHelpMessage(message.Chat.ID)
		return
	}

	item := strings.ToLower(argList[0])
	customPrices, err := h.db.LoadPrices()
	if err != nil {
		log.Printf("Error loading prices: %v", err)
		customPrices = make(map[string]interface{})
	}

	// Handle batch updates for normal diamonds
	if item == "normal" {
		h.handleNormalDiamondsBatchUpdate(message.Chat.ID, argList[1:], customPrices)
		return
	}

	// Handle batch updates for 2x diamonds
	if item == "2x" {
		h.handle2xDiamondsBatchUpdate(message.Chat.ID, argList[1:], customPrices)
		return
	}

	// Single item update
	if len(argList) != 2 {
		h.sendInvalidFormatMessage(message.Chat.ID, "/setprice item price")
		return
	}

	price, err := strconv.Atoi(argList[1])
	if err != nil || price < 0 {
		h.sendInvalidAmountMessage(message.Chat.ID)
		return
	}

	// Handle weekly pass auto-update
	if strings.HasPrefix(item, "wp") {
		weekNum, err := strconv.Atoi(item[2:])
		if err == nil && weekNum >= 1 && weekNum <= 10 {
			h.handleWeeklyPassUpdate(message.Chat.ID, weekNum, price, customPrices)
			return
		}
	}

	// Single item update
	customPrices[item] = price
	err = h.db.SavePrices(customPrices)
	if err != nil {
		h.sendPriceUpdateErrorMessage(message.Chat.ID)
		return
	}

	h.sendPriceUpdateConfirmation(message.Chat.ID, item, price)
}

func (h *AdminHandler) HandleMaintenance(message *tgbotapi.Message, args string) {
	userID := strconv.FormatInt(message.From.ID, 10)
	
	if !h.isAdmin(userID) {
		h.sendNotAdminMessage(message.Chat.ID)
		return
	}

	argList := strings.Fields(args)
	if len(argList) != 2 {
		h.sendMaintenanceHelpMessage(message.Chat.ID)
		return
	}

	feature := strings.ToLower(argList[0])
	status := strings.ToLower(argList[1])

	if !h.isValidFeature(feature) {
		h.sendInvalidFeatureMessage(message.Chat.ID)
		return
	}

	if !h.isValidStatus(status) {
		h.sendInvalidStatusMessage(message.Chat.ID)
		return
	}

	newStatus := (status == "on")
	settingKey := fmt.Sprintf("maintenance.%s", feature)

	err := h.db.UpdateSetting(settingKey, newStatus)
	if err != nil {
		h.sendMaintenanceUpdateErrorMessage(message.Chat.ID)
		return
	}

	h.sendMaintenanceUpdateConfirmation(message.Chat.ID, feature, newStatus)
}

// Helper methods
func (h *AdminHandler) isAdmin(userID string) bool {
	userIDInt, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return false
	}
	return userIDInt == h.config.AdminID
}

func (h *AdminHandler) findPendingTopup(userID string, amount int) (string, error) {
	userDoc, err := h.db.GetUser(userID)
	if err != nil {
		return "", err
	}

	if userDoc == nil {
		return "", fmt.Errorf("user not found")
	}

	for _, topup := range userDoc.Topups {
		if topup.Status == "pending" && topup.Amount == amount {
			return topup.TopupID, nil
		}
	}

	return "", fmt.Errorf("pending topup not found")
}

func (h *AdminHandler) isValidFeature(feature string) bool {
	validFeatures := []string{"orders", "topups", "general"}
	for _, validFeature := range validFeatures {
		if feature == validFeature {
			return true
		}
	}
	return false
}

func (h *AdminHandler) isValidStatus(status string) bool {
	return status == "on" || status == "off"
}

func (h *AdminHandler) handleNormalDiamondsBatchUpdate(chatID int64, prices []string, customPrices map[string]interface{}) {
	normalDiamonds := []string{"11", "22", "33", "56", "86", "112", "172", "257", "343",
		"429", "514", "600", "706", "878", "963", "1049", "1135",
		"1412", "2195", "3688", "5532", "9288", "12976"}

	if len(prices) != len(normalDiamonds) {
		h.sendInvalidBatchPriceCountMessage(chatID, len(normalDiamonds))
		return
	}

	updatedItems := []string{}
	for i, diamond := range normalDiamonds {
		price, err := strconv.Atoi(prices[i])
		if err != nil || price < 0 {
			h.sendInvalidPriceInBatchMessage(chatID, diamond)
			return
		}
		customPrices[diamond] = price
		updatedItems = append(updatedItems, fmt.Sprintf("%s=%d", diamond, price))
	}

	err := h.db.SavePrices(customPrices)
	if err != nil {
		h.sendPriceUpdateErrorMessage(chatID)
		return
	}

	h.sendBatchPriceUpdateConfirmation(chatID, "Normal Diamonds", updatedItems)
}

func (h *AdminHandler) handle2xDiamondsBatchUpdate(chatID int64, prices []string, customPrices map[string]interface{}) {
	doublePass := []string{"55", "165", "275", "565"}

	if len(prices) != len(doublePass) {
		h.sendInvalidBatchPriceCountMessage(chatID, len(doublePass))
		return
	}

	updatedItems := []string{}
	for i, diamond := range doublePass {
		price, err := strconv.Atoi(prices[i])
		if err != nil || price < 0 {
			h.sendInvalidPriceInBatchMessage(chatID, diamond)
			return
		}
		customPrices[diamond] = price
		updatedItems = append(updatedItems, fmt.Sprintf("%s=%d", diamond, price))
	}

	err := h.db.SavePrices(customPrices)
	if err != nil {
		h.sendPriceUpdateErrorMessage(chatID)
		return
	}

	h.sendBatchPriceUpdateConfirmation(chatID, "2X Diamonds", updatedItems)
}

func (h *AdminHandler) handleWeeklyPassUpdate(chatID int64, weekNum int, price int, customPrices map[string]interface{}) {
	basePricePerWeek := float64(price) / float64(weekNum)
	updatedItems := []string{}

	for i := 1; i <= 10; i++ {
		wpKey := fmt.Sprintf("wp%d", i)
		wpPrice := int(basePricePerWeek * float64(i))
		customPrices[wpKey] = wpPrice
		updatedItems = append(updatedItems, fmt.Sprintf("%s=%d", wpKey, wpPrice))
	}

	err := h.db.SavePrices(customPrices)
	if err != nil {
		h.sendPriceUpdateErrorMessage(chatID)
		return
	}

	h.sendWeeklyPassUpdateConfirmation(chatID, int(basePricePerWeek), updatedItems)
}

// Message sending methods
func (h *AdminHandler) sendNotAdminMessage(chatID int64) {
	text := "❌ သင်သည် admin မဟုတ်ပါ!"
	utils.SendMessage(h.bot, chatID, text, "Markdown")
}

func (h *AdminHandler) sendApprovalConfirmation(chatID int64, userID string, amount int) {
	text := fmt.Sprintf("✅ ***Approve အောင်မြင်ပါပြီ!***\n\n👤 ***User ID:*** `%s`\n💰 ***Amount:*** `%d MMK`", 
		userID, amount)
	utils.SendMessage(h.bot, chatID, text, "Markdown")
}

func (h *AdminHandler) sendDeductionConfirmation(chatID int64, userID string, amount int, newBalance int) {
	text := fmt.Sprintf("✅ ***Balance နှုတ်ခြင်း အောင်မြင်ပါပြီ!***\n\n👤 User ID: `%s`\n💰 ***နှုတ်ခဲ့တဲ့ပမာဏ***: `%d MMK`\n💳 ***User လက်ကျန်ငွေ***: `%d MMK`", 
		userID, amount, newBalance)
	utils.SendMessage(h.bot, chatID, text, "Markdown")
}

func (h *AdminHandler) sendBanConfirmation(chatID int64, userID string) {
	authorizedUsers, _ := h.db.LoadAuthorizedUsers()
	text := fmt.Sprintf("✅ User Ban အောင်မြင်ပါပြီ!\n\n👤 User ID: `%s`\n📝 Total authorized users: %d", 
		userID, len(authorizedUsers))
	utils.SendMessage(h.bot, chatID, text, "Markdown")
}

func (h *AdminHandler) sendUnbanConfirmation(chatID int64, userID string) {
	authorizedUsers, _ := h.db.LoadAuthorizedUsers()
	text := fmt.Sprintf("✅ User Unban အောင်မြင်ပါပြီ!\n\n👤 User ID: `%s`\n📝 Total authorized users: %d", 
		userID, len(authorizedUsers))
	utils.SendMessage(h.bot, chatID, text, "Markdown")
}

func (h *AdminHandler) sendPriceUpdateConfirmation(chatID int64, item string, price int) {
	text := fmt.Sprintf("✅ ***ဈေးနှုန်း ပြောင်းလဲပါပြီ!***\n\n💎 Item: `%s`\n💰 New Price: `%d MMK`\n\n📝 Users တွေ /price နဲ့ အသစ်တွေ့မယ်။", 
		item, price)
	utils.SendMessage(h.bot, chatID, text, "Markdown")
}

func (h *AdminHandler) sendMaintenanceUpdateConfirmation(chatID int64, feature string, status bool) {
	statusText := "🟢 ***ဖွင့်ထား***"
	if !status {
		statusText = "🔴 ***ပိတ်ထား***"
	}

	featureText := map[string]string{
		"orders":  "***အော်ဒါလုပ်ဆောင်ချက်***",
		"topups":  "***ငွေဖြည့်လုပ်ဆောင်ချက်***", 
		"general": "***ယေဘူယျလုပ်ဆောင်ချက်***",
	}

	text := fmt.Sprintf("✅ ***Maintenance Mode ပြောင်းလဲပါပြီ!***\n\n🔧 Feature: %s\n📊 Status: %s",
		featureText[feature], statusText)
	utils.SendMessage(h.bot, chatID, text, "Markdown")
}

// Notification methods
func (h *AdminHandler) notifyUserAboutApproval(userID string, amount int, adminName string) {
	userDoc, err := h.db.GetUser(userID)
	if err != nil {
		return
	}

	userBalance := userDoc.Balance
	chatID, _ := strconv.ParseInt(userID, 10, 64)

	keyboard := utils.CreateInlineKeyboard([][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonURL("💎 Order တင်မယ်", 
				fmt.Sprintf("https://t.me/%s?start=order", h.bot.Self.UserName)),
		},
	})

	text := fmt.Sprintf("✅ ***ငွေဖြည့်မှု အတည်ပြုပါပြီ!*** 🎉\n\n💰 ***ပမာဏ:*** `%d MMK`\n💳 ***လက်ကျန်ငွေ:*** `%d MMK`\n👤 ***Approved by:*** %s\n\n🎉 ***ယခုအခါ diamonds များ ဝယ်ယူနိုင်ပါပြီ!***\n🔓 ***Bot လုပ်ဆောင်ချက်များ ပြန်လည် အသုံးပြုနိုင်ပါပြီ!***",
		amount, userBalance, adminName)

	utils.SendMessageWithKeyboard(h.bot, chatID, text, "Markdown", keyboard)
}

func (h *AdminHandler) notifyUserAboutDeduction(userID string, amount int, newBalance int) {
	chatID, _ := strconv.ParseInt(userID, 10, 64)
	text := fmt.Sprintf("⚠️ ***လက်ကျန်ငွေ နှုတ်ခံရမှု***\n\n💰 ***နှုတ်ခံရတဲ့ပမာဏ***: `%d MMK`\n💳 ***လက်ကျန်ငွေ***: `%d MMK`\n📞 မေးခွန်းရှိရင် admin ကို ဆက်သွယ်ပါ။",
		amount, newBalance)
	utils.SendMessage(h.bot, chatID, text, "Markdown")
}

func (h *AdminHandler) notifyUserAboutBan(userID string) {
	chatID, _ := strconv.ParseInt(userID, 10, 64)
	text := "🚫 Bot အသုံးပြုခွင့် ပိတ်ပင်ခံရမှု\n\n❌ Admin က သင့်ကို ban လုပ်လိုက်ပါပြီ။\n\n📞 အကြောင်းရင်း သိရှိရန် Admin ကို ဆက်သွယ်ပါ။"
	utils.SendMessage(h.bot, chatID, text, "Markdown")
}

func (h *AdminHandler) notifyUserAboutUnban(userID string) {
	chatID, _ := strconv.ParseInt(userID, 10, 64)
	text := "🎉 *Bot အသုံးပြုခွင့် ပြန်လည်ရရှိပါပြီ!*\n\n✅ Admin က သင့် ban ကို ဖြုတ်ပေးလိုက်ပါပြီ။\n\n🚀 ယခုအခါ /start နှိပ်ပြီး bot ကို အသုံးပြုနိုင်ပါပြီ!"
	utils.SendMessage(h.bot, chatID, text, "Markdown")
}
