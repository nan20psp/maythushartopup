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

type CallbackHandler struct {
	bot      *tgbotapi.BotAPI
	db       *database.DBManager
	config   *models.Config
}

func NewCallbackHandler(bot *tgbotapi.BotAPI, db *database.DBManager, config *models.Config) *CallbackHandler {
	return &CallbackHandler{
		bot:    bot,
		db:     db,
		config: config,
	}
}

func (h *CallbackHandler) HandleCallback(callback *tgbotapi.CallbackQuery) {
	userID := strconv.FormatInt(callback.From.ID, 10)
	data := callback.Data

	// Answer callback query immediately
	callbackConfig := tgbotapi.NewCallback(callback.ID, "")
	h.bot.Send(callbackConfig)

	// Handle different callback types
	switch {
	case strings.HasPrefix(data, "topup_pay_"):
		h.handleTopupPaymentMethod(callback, data)
	case strings.HasPrefix(data, "order_confirm_"):
		h.handleOrderConfirm(callback, data)
	case strings.HasPrefix(data, "order_cancel_"):
		h.handleOrderCancel(callback, data)
	case strings.HasPrefix(data, "topup_approve_"):
		h.handleTopupApprove(callback, data)
	case strings.HasPrefix(data, "topup_reject_"):
		h.handleTopupReject(callback, data)
	case data == "topup_cancel":
		h.handleTopupCancel(callback)
	case data == "request_register":
		h.handleRegisterRequest(callback)
	case strings.HasPrefix(data, "register_approve_"):
		h.handleRegisterApprove(callback, data)
	case strings.HasPrefix(data, "register_reject_"):
		h.handleRegisterReject(callback, data)
	default:
		log.Printf("Unknown callback data: %s", data)
	}
}

func (h *CallbackHandler) handleTopupPaymentMethod(callback *tgbotapi.CallbackQuery, data string) {
	userID := strconv.FormatInt(callback.From.ID, 10)
	parts := strings.Split(data, "_")
	
	if len(parts) != 4 {
		return
	}

	paymentMethod := parts[2]
	amount, err := strconv.Atoi(parts[3])
	if err != nil {
		return
	}

	// In production, you would store this in a proper state management system
	// For now, we'll simulate with a simple approach
	// pendingTopups[userID].PaymentMethod = paymentMethod

	settings, err := h.db.LoadSettings(
		map[string]interface{}{},
		map[string]interface{}{}, 
		map[string]interface{}{},
		map[string]interface{}{},
	)
	if err != nil {
		log.Printf("Error loading settings: %v", err)
		return
	}

	paymentInfo, _ := settings["payment_info"].(map[string]interface{})

	var paymentName, paymentNum, paymentAccName string
	var paymentQr interface{}

	if paymentMethod == "kpay" {
		paymentName = "KBZ Pay"
		paymentNum, _ = paymentInfo["kpay_number"].(string)
		paymentAccName, _ = paymentInfo["kpay_name"].(string)
		paymentQr = paymentInfo["kpay_image"]
	} else {
		paymentName = "Wave Money" 
		paymentNum, _ = paymentInfo["wave_number"].(string)
		paymentAccName, _ = paymentInfo["wave_name"].(string)
		paymentQr = paymentInfo["wave_image"]
	}

	// Send QR code if available
	if paymentQr != nil {
		if qrFileID, ok := paymentQr.(string); ok {
			caption := fmt.Sprintf("📱 **%s QR Code**\n📞 နံပါတ်: `%s`\n👤 နာမည်: %s",
				paymentName, paymentNum, paymentAccName)
			utils.SendPhoto(h.bot, callback.Message.Chat.ID, qrFileID, caption, "Markdown")
		}
	}

	// Update message with payment instructions
	text := fmt.Sprintf("💳 ***ငွေဖြည့်လုပ်ငန်းစဉ်***\n\n"+
		"✅ ***ပမာဏ:*** `%d MMK`\n"+
		"✅ ***Payment:*** %s\n\n"+
		"***အဆင့် 3: ငွေလွှဲပြီး Screenshot တင်ပါ။***\n\n"+
		"📱 %s\n"+
		"📞 ***နံပါတ်:*** `%s`\n"+
		"👤 ***အမည်:*** %s\n\n"+
		"⚠️ ***အရေးကြီးသော သတိပေးချက်:***\n"+
		"***ငွေလွှဲ note/remark မှာ သင့်ရဲ့ %s အကောင့်နာမည်ကို ရေးပေးပါ။***\n\n"+
		"💡 ***ငွေလွှဲပြီးရင် screenshot ကို ဒီမှာ တင်ပေးပါ။***\n"+
		"ℹ️ ***ပယ်ဖျက်ရန် /cancel နှိပ်ပါ***",
		amount, paymentName, paymentName, paymentNum, paymentAccName, paymentName)

	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	edit.ParseMode = "Markdown"
	h.bot.Send(edit)
}

func (h *CallbackHandler) handleOrderConfirm(callback *tgbotapi.CallbackQuery, data string) {
	userID := strconv.FormatInt(callback.From.ID, 10)
	
	if !h.isAdmin(userID) {
		return
	}

	adminName := utils.GetUserDisplayName(callback.From)
	orderID := strings.TrimPrefix(data, "order_confirm_")

	updates := bson.M{
		"status":       "confirmed",
		"confirmed_by": adminName,
		"confirmed_at": time.Now(),
	}

	targetUserID, err := h.db.FindAndUpdateOrder(orderID, updates)
	if err != nil {
		return
	}

	// Update message
	originalText := callback.Message.Text
	updatedText := strings.Replace(originalText, "⏳ စောင့်ဆိုင်းနေသည်", 
		fmt.Sprintf("✅ လက်ခံပြီး (by %s)", adminName), 1)

	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, updatedText)
	edit.ParseMode = "Markdown"
	h.bot.Send(edit)

	// Remove inline keyboard
	editReplyMarkup := tgbotapi.NewEditMessageReplyMarkup(callback.Message.Chat.ID, callback.Message.MessageID, tgbotapi.InlineKeyboardMarkup{})
	h.bot.Send(editReplyMarkup)

	// Notify other admins
	h.notifyAdminsAboutOrderConfirmation(orderID, adminName, targetUserID)

	// Notify user
	h.notifyUserAboutOrderConfirmation(targetUserID, orderID)
}

func (h *CallbackHandler) handleOrderCancel(callback *tgbotapi.CallbackQuery, data string) {
	userID := strconv.FormatInt(callback.From.ID, 10)
	
	if !h.isAdmin(userID) {
		return
	}

	adminName := utils.GetUserDisplayName(callback.From)
	orderID := strings.TrimPrefix(data, "order_cancel_")

	// Get order details first
	order, err := h.getOrderByID(orderID)
	if err != nil {
		return
	}

	if order.Status != "pending" {
		return
	}

	refundAmount := order.Price
	updates := bson.M{
		"status":       "cancelled", 
		"cancelled_by": adminName,
		"cancelled_at": time.Now(),
	}

	targetUserID, err := h.db.FindAndUpdateOrder(orderID, updates)
	if err != nil {
		return
	}

	// Refund balance
	h.db.UpdateBalance(targetUserID, refundAmount)

	// Update message
	originalText := callback.Message.Text
	updatedText := strings.Replace(originalText, "⏳ စောင့်ဆိုင်းနေသည်",
		fmt.Sprintf("❌ ငြင်းပယ်ပြီး (by %s)", adminName), 1)

	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, updatedText)
	edit.ParseMode = "Markdown"
	h.bot.Send(edit)

	// Remove inline keyboard  
	editReplyMarkup := tgbotapi.NewEditMessageReplyMarkup(callback.Message.Chat.ID, callback.Message.MessageID, tgbotapi.InlineKeyboardMarkup{})
	h.bot.Send(editReplyMarkup)

	// Notify other admins
	h.notifyAdminsAboutOrderCancellation(orderID, adminName, refundAmount)

	// Notify user
	h.notifyUserAboutOrderCancellation(targetUserID, orderID, refundAmount)
}

func (h *CallbackHandler) handleTopupApprove(callback *tgbotapi.CallbackQuery, data string) {
	userID := strconv.FormatInt(callback.From.ID, 10)
	
	if !h.isAdmin(userID) {
		return
	}

	adminName := utils.GetUserDisplayName(callback.From)
	topupID := strings.TrimPrefix(data, "topup_approve_")

	updates := bson.M{
		"status":      "approved",
		"approved_by": adminName,
		"approved_at": time.Now(),
	}

	targetUserID, err := h.db.FindAndUpdateTopup(topupID, updates)
	if err != nil {
		return
	}

	// Update message caption if it's a photo message
	if callback.Message.Photo != nil {
		originalCaption := callback.Message.Caption
		updatedCaption := strings.Replace(originalCaption, "⏳ စောင့်ဆိုင်းနေသည်", "✅ လက်ခံပြီး", 1)
		updatedCaption += fmt.Sprintf("\n\n✅ Approved by: %s", adminName)

		edit := tgbotapi.NewEditMessageCaption(callback.Message.Chat.ID, callback.Message.MessageID, updatedCaption)
		edit.ParseMode = "Markdown"
		h.bot.Send(edit)
	}

	// Remove inline keyboard
	editReplyMarkup := tgbotapi.NewEditMessageReplyMarkup(callback.Message.Chat.ID, callback.Message.MessageID, tgbotapi.InlineKeyboardMarkup{})
	h.bot.Send(editReplyMarkup)

	// Notify user
	h.notifyUserAboutTopupApproval(targetUserID, topupID, adminName)

	// Process affiliate commission
	h.processAffiliateCommission(targetUserID, topupID)
}

func (h *CallbackHandler) handleTopupReject(callback *tgbotapi.CallbackQuery, data string) {
	userID := strconv.FormatInt(callback.From.ID, 10)
	
	if !h.isAdmin(userID) {
		return
	}

	adminName := utils.GetUserDisplayName(callback.From)
	topupID := strings.TrimPrefix(data, "topup_reject_")

	updates := bson.M{
		"status":      "rejected", 
		"rejected_by": adminName,
		"rejected_at": time.Now(),
	}

	targetUserID, err := h.db.FindAndUpdateTopup(topupID, updates)
	if err != nil {
		return
	}

	// Update message caption if it's a photo message
	if callback.Message.Photo != nil {
		originalCaption := callback.Message.Caption
		updatedCaption := strings.Replace(originalCaption, "⏳ စောင့်ဆိုင်းနေသည်", "❌ ငြင်းပယ်ပြီး", 1)
		updatedCaption += fmt.Sprintf("\n\n❌ Rejected by: %s", adminName)

		edit := tgbotapi.NewEditMessageCaption(callback.Message.Chat.ID, callback.Message.MessageID, updatedCaption)
		edit.ParseMode = "Markdown"
		h.bot.Send(edit)
	}

	// Remove inline keyboard
	editReplyMarkup := tgbotapi.NewEditMessageReplyMarkup(callback.Message.Chat.ID, callback.Message.MessageID, tgbotapi.InlineKeyboardMarkup{})
	h.bot.Send(editReplyMarkup)

	// Notify user
	h.notifyUserAboutTopupRejection(targetUserID, topupID, adminName)
}

func (h *CallbackHandler) handleTopupCancel(callback *tgbotapi.CallbackQuery) {
	userID := strconv.FormatInt(callback.From.ID, 10)
	
	// In production, you would remove from pending topups
	// delete(pendingTopups, userID)

	text := "✅ ***ငွေဖြည့်ခြင်း ပယ်ဖျက်ပါပြီ!***\n\n💡 ***ပြန်ဖြည့်ချင်ရင်*** /topup ***နှိပ်ပါ။***"
	
	edit := tgbotapi.NewEditMessageText(callback.Message.Chat.ID, callback.Message.MessageID, text)
	edit.ParseMode = "Markdown"
	h.bot.Send(edit)
}

// Helper methods
func (h *CallbackHandler) isAdmin(userID string) bool {
	userIDInt, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return false
	}
	return userIDInt == h.config.AdminID
}

func (h *CallbackHandler) getOrderByID(orderID string) (*models.Order, error) {
	// This would require implementing a method to find order by ID
	// For now, return a mock implementation
	return &models.Order{}, nil
}

func (h *CallbackHandler) notifyAdminsAboutOrderConfirmation(orderID string, adminName string, targetUserID string) {
	// Notify other admins about order confirmation
	text := fmt.Sprintf("✅ ***Order Confirmed!***\n📝 ***Order ID:*** `%s`\n👤 ***Confirmed by:*** %s",
		orderID, adminName)
	
	// In production, you would send this to all admins except the one who confirmed
	utils.SendMessage(h.bot, h.config.AdminGroupID, text, "Markdown")
}

func (h *CallbackHandler) notifyUserAboutOrderConfirmation(userID string, orderID string) {
	chatID, _ := strconv.ParseInt(userID, 10, 64)
	text := fmt.Sprintf("✅ ***Order လက်ခံပြီးပါပြီ!***\n\n📝 ***Order ID:*** `%s`\n📊 Status: ✅ ***လက်ခံပြီး***\n\n💎 ***Diamonds များကို ထည့်သွင်းပေးလိုက်ပါပြီ။***",
		orderID)
	utils.SendMessage(h.bot, chatID, text, "Markdown")
}

func (h *CallbackHandler) processAffiliateCommission(userID string, topupID string) {
	// Implement affiliate commission logic here
	// This would involve:
	// 1. Getting the topup amount
	// 2. Finding the referrer
	// 3. Calculating commission
	// 4. Updating balances
	// 5. Sending notifications
}
