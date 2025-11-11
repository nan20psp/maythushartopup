package utils

import (
	"fmt"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func SendMessage(bot *tgbotapi.BotAPI, chatID int64, text string, parseMode string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	if parseMode != "" {
		msg.ParseMode = parseMode
	}
	_, err := bot.Send(msg)
	return err
}

func SendMessageWithKeyboard(bot *tgbotapi.BotAPI, chatID int64, text string, parseMode string, keyboard tgbotapi.InlineKeyboardMarkup) error {
	msg := tgbotapi.NewMessage(chatID, text)
	if parseMode != "" {
		msg.ParseMode = parseMode
	}
	msg.ReplyMarkup = keyboard
	_, err := bot.Send(msg)
	return err
}

func SendPhoto(bot *tgbotapi.BotAPI, chatID int64, photoFileID string, caption string, parseMode string) error {
	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileID(photoFileID))
	if caption != "" {
		photo.Caption = caption
	}
	if parseMode != "" {
		photo.ParseMode = parseMode
	}
	_, err := bot.Send(photo)
	return err
}

func EditMessageText(bot *tgbotapi.BotAPI, chatID int64, messageID int, text string, parseMode string) error {
	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	if parseMode != "" {
		edit.ParseMode = parseMode
	}
	_, err := bot.Send(edit)
	return err
}

func CreateInlineKeyboard(buttons [][]tgbotapi.InlineKeyboardButton) tgbotapi.InlineKeyboardMarkup {
	var keyboard [][]tgbotapi.InlineKeyboardButton
	for _, row := range buttons {
		keyboard = append(keyboard, row)
	}
	return tgbotapi.NewInlineKeyboardMarkup(keyboard...)
}

func CreatePaymentMethodsKeyboard(amount int) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📱 KBZ Pay", fmt.Sprintf("topup_pay_kpay_%d", amount)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📱 Wave Money", fmt.Sprintf("topup_pay_wave_%d", amount)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ ငြင်းပယ်မယ်", "topup_cancel"),
		),
	)
}

func CreateOrderActionKeyboard(orderID string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Confirm", fmt.Sprintf("order_confirm_%s", orderID)),
			tgbotapi.NewInlineKeyboardButtonData("❌ Cancel", fmt.Sprintf("order_cancel_%s", orderID)),
		),
	)
}

func CreateTopupActionKeyboard(topupID string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Approve", fmt.Sprintf("topup_approve_%s", topupID)),
			tgbotapi.NewInlineKeyboardButtonData("❌ Reject", fmt.Sprintf("topup_reject_%s", topupID)),
		),
	)
}
