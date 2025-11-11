package models

import (
	"time"
)

type User struct {
	UserID           string    `bson:"user_id"`
	Name             string    `bson:"name"`
	Username         string    `bson:"username"`
	Balance          int       `bson:"balance"`
	Orders           []Order   `bson:"orders"`
	Topups           []Topup   `bson:"topups"`
	JoinedAt         time.Time `bson:"joined_at"`
	ReferredBy       string    `bson:"referred_by,omitempty"`
	ReferralEarnings int       `bson:"referral_earnings"`
}

type Order struct {
	OrderID     string    `bson:"order_id"`
	GameID      string    `bson:"game_id"`
	ServerID    string    `bson:"server_id"`
	Amount      string    `bson:"amount"`
	Price       int       `bson:"price"`
	Status      string    `bson:"status"`
	Timestamp   time.Time `bson:"timestamp"`
	UserID      string    `bson:"user_id"`
	ChatID      int64     `bson:"chat_id"`
	ConfirmedBy string    `bson:"confirmed_by,omitempty"`
	ConfirmedAt time.Time `bson:"confirmed_at,omitempty"`
}

type Topup struct {
	TopupID      string    `bson:"topup_id"`
	Amount       int       `bson:"amount"`
	PaymentMethod string    `bson:"payment_method"`
	Status       string    `bson:"status"`
	Timestamp    time.Time `bson:"timestamp"`
	UserID       string    `bson:"user_id"`
	ChatID       int64     `bson:"chat_id"`
	ApprovedBy   string    `bson:"approved_by,omitempty"`
	ApprovedAt   time.Time `bson:"approved_at,omitempty"`
}

type PendingTopup struct {
	Amount       int       `json:"amount"`
	Timestamp    time.Time `json:"timestamp"`
	PaymentMethod string    `json:"payment_method"`
}

type Settings struct {
	PaymentInfo  map[string]interface{} `bson:"payment_info"`
	Maintenance  map[string]interface{} `bson:"maintenance"`
	Affiliate    map[string]interface{} `bson:"affiliate"`
	AutoDelete   map[string]interface{} `bson:"auto_delete"`
}
