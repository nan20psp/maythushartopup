package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"mlbbtopup/models"
)

type DBManager struct {
	client                *mongo.Client
	db                    *mongo.Database
	usersCollection       *mongo.Collection
	pricesCollection      *mongo.Collection
	pubgPricesCollection  *mongo.Collection
	authCollection        *mongo.Collection
	adminsCollection      *mongo.Collection
	settingsCollection    *mongo.Collection
	autoDeleteCollection  *mongo.Collection
	allGroupsCollection   *mongo.Collection
}

func NewDBManager(mongoURL string) (*DBManager, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURL))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %v", err)
	}

	db := client.Database("mlbb_bot_db")

	return &DBManager{
		client:               client,
		db:                   db,
		usersCollection:      db.Collection("users"),
		pricesCollection:     db.Collection("prices"),
		pubgPricesCollection: db.Collection("pubg_prices"),
		authCollection:       db.Collection("authorized_users"),
		adminsCollection:     db.Collection("admins"),
		settingsCollection:   db.Collection("settings"),
		autoDeleteCollection: db.Collection("auto_delete_messages"),
		allGroupsCollection:  db.Collection("all_groups"),
	}, nil
}

func (db *DBManager) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return db.client.Disconnect(ctx)
}

// User Functions
func (db *DBManager) GetUser(userID string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	err := db.usersCollection.FindOne(ctx, bson.M{"user_id": userID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (db *DBManager) GetAllUsers() ([]models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := db.usersCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err = cursor.All(ctx, &users); err != nil {
		return nil, err
	}
	return users, nil
}

func (db *DBManager) CreateUser(userID, name, username string, referrerID *string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	userData := bson.M{
		"user_id":           userID,
		"name":              name,
		"username":          username,
		"balance":           0,
		"orders":            []bson.M{},
		"topups":            []bson.M{},
		"joined_at":         time.Now(),
		"referral_earnings": 0,
	}

	if referrerID != nil {
		userData["referred_by"] = *referrerID
	}

	opts := options.Update().SetUpsert(true)
	_, err := db.usersCollection.UpdateOne(
		ctx,
		bson.M{"user_id": userID},
		bson.M{"$setOnInsert": userData},
		opts,
	)
	return err
}

func (db *DBManager) UpdateUserProfile(userID, name, username string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.usersCollection.UpdateOne(
		ctx,
		bson.M{"user_id": userID},
		bson.M{"$set": bson.M{"name": name, "username": username}},
	)
	return err
}

func (db *DBManager) UpdateBalance(userID string, amountChange int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.usersCollection.UpdateOne(
		ctx,
		bson.M{"user_id": userID},
		bson.M{"$inc": bson.M{"balance": amountChange}},
	)
	return err
}

func (db *DBManager) SetBalance(userID string, amount int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.usersCollection.UpdateOne(
		ctx,
		bson.M{"user_id": userID},
		bson.M{"$set": bson.M{"balance": amount}},
	)
	return err
}

func (db *DBManager) UpdateReferralEarnings(userID string, commissionAmount int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.usersCollection.UpdateOne(
		ctx,
		bson.M{"user_id": userID},
		bson.M{"$inc": bson.M{
			"balance":          commissionAmount,
			"referral_earnings": commissionAmount,
		}},
	)
	return err
}

func (db *DBManager) AddOrder(userID string, orderData bson.M) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.usersCollection.UpdateOne(
		ctx,
		bson.M{"user_id": userID},
		bson.M{"$push": bson.M{"orders": orderData}},
	)
	return err
}

func (db *DBManager) AddTopup(userID string, topupData bson.M) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.usersCollection.UpdateOne(
		ctx,
		bson.M{"user_id": userID},
		bson.M{"$push": bson.M{"topups": topupData}},
	)
	return err
}

func (db *DBManager) FindAndUpdateOrder(orderID string, updates bson.M) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	setFields := bson.M{}
	for key, value := range updates {
		setFields["orders.$."+key] = value
	}

	var result struct {
		UserID string `bson:"user_id"`
	}

	err := db.usersCollection.FindOneAndUpdate(
		ctx,
		bson.M{"orders.order_id": orderID, "orders.status": "pending"},
		bson.M{"$set": setFields},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&result)

	if err != nil {
		return "", err
	}
	return result.UserID, nil
}

func (db *DBManager) FindAndUpdateTopup(topupID string, updates bson.M) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First find the user and topup to get the amount
	var user struct {
		UserID string        `bson:"user_id"`
		Topups []models.Topup `bson:"topups"`
	}

	err := db.usersCollection.FindOne(
		ctx,
		bson.M{"topups.topup_id": topupID},
	).Decode(&user)
	if err != nil {
		return "", err
	}

	// Update the topup
	setFields := bson.M{}
	for key, value := range updates {
		setFields["topups.$."+key] = value
	}

	updateResult := db.usersCollection.FindOneAndUpdate(
		ctx,
		bson.M{"topups.topup_id": topupID, "topups.status": "pending"},
		bson.M{"$set": setFields},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	var updatedUser struct {
		UserID string `bson:"user_id"`
	}
	if err := updateResult.Decode(&updatedUser); err != nil {
		return "", err
	}

	// If topup is approved, update balance
	if status, ok := updates["status"]; ok && status == "approved" {
		// Find the topup amount
		var topupAmount int
		for _, topup := range user.Topups {
			if topup.TopupID == topupID {
				topupAmount = topup.Amount
				break
			}
		}
		if topupAmount > 0 {
			err = db.UpdateBalance(updatedUser.UserID, topupAmount)
			if err != nil {
				return updatedUser.UserID, err
			}
		}
	}

	return updatedUser.UserID, nil
}

// Price Functions
func (db *DBManager) LoadPrices() (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result struct {
		Prices map[string]interface{} `bson:"prices"`
	}
	
	err := db.pricesCollection.FindOne(ctx, bson.M{"_id": "custom_prices"}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return make(map[string]interface{}), nil
		}
		return nil, err
	}
	return result.Prices, nil
}

func (db *DBManager) SavePrices(prices map[string]interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.pricesCollection.UpdateOne(
		ctx,
		bson.M{"_id": "custom_prices"},
		bson.M{"$set": bson.M{"prices": prices}},
		options.Update().SetUpsert(true),
	)
	return err
}

// Authorization Functions
func (db *DBManager) LoadAuthorizedUsers() (map[string]bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result struct {
		Users []string `bson:"users"`
	}
	
	err := db.authCollection.FindOne(ctx, bson.M{"_id": "auth_list"}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return make(map[string]bool), nil
		}
		return nil, err
	}

	users := make(map[string]bool)
	for _, user := range result.Users {
		users[user] = true
	}
	return users, nil
}

func (db *DBManager) AddAuthorizedUser(userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.authCollection.UpdateOne(
		ctx,
		bson.M{"_id": "auth_list"},
		bson.M{"$addToSet": bson.M{"users": userID}},
		options.Update().SetUpsert(true),
	)
	return err
}

func (db *DBManager) RemoveAuthorizedUser(userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.authCollection.UpdateOne(
		ctx,
		bson.M{"_id": "auth_list"},
		bson.M{"$pull": bson.M{"users": userID}},
	)
	return err
}

// Settings Functions
func (db *DBManager) LoadSettings(defaultPayment, defaultMaintenance, defaultAffiliate, defaultAutoDelete map[string]interface{}) (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var config map[string]interface{}
	err := db.settingsCollection.FindOne(ctx, bson.M{"_id": "global_config"}).Decode(&config)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Create default settings
			config = map[string]interface{}{
				"_id":          "global_config",
				"payment_info": defaultPayment,
				"maintenance":  defaultMaintenance,
				"affiliate":    defaultAffiliate,
				"auto_delete":  defaultAutoDelete,
			}
			_, err = db.settingsCollection.InsertOne(ctx, config)
			if err != nil {
				return nil, err
			}
			return config, nil
		}
		return nil, err
	}
	return config, nil
}

func (db *DBManager) UpdateSetting(key string, value interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.settingsCollection.UpdateOne(
		ctx,
		bson.M{"_id": "global_config"},
		bson.M{"$set": bson.M{key: value}},
		options.Update().SetUpsert(true),
	)
	return err
}
