package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/adrg/xdg"
	"github.com/dgraph-io/badger/v4"
)

// DB represents the database for the application
type DB struct {
	db *badger.DB
}

// getDBDir returns the directory for the BadgerDB database
func getDBDir() string {
	dbDir, err := xdg.CacheFile("finance_tracker/badger")
	if err != nil {
		log.Fatal(err)
	}
	return dbDir
}

// NewDB creates a new DB instance with BadgerDB
func NewDB() (*DB, error) {
	opts := badger.DefaultOptions(getDBDir())
	opts.Logger = nil // Disable BadgerDB's default logger

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("error opening BadgerDB: %w", err)
	}

	return &DB{db: db}, nil
}

// Close closes the BadgerDB database
func (d *DB) Close() error {
	return d.db.Close()
}

// UpdateAccount updates the account information in the database
func (d *DB) UpdateAccount(account Account) error {
	key := []byte(fmt.Sprintf("account:%s", account.ID))
	data, err := json.Marshal(account)
	if err != nil {
		return fmt.Errorf("error marshaling account: %w", err)
	}

	return d.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, data)
	})
}

// IsAccountUpdated checks if the account has been updated since the last time
// it was stored in the database
func (d *DB) IsAccountUpdated(accountID string, balanceDate int64) bool {
	key := []byte(fmt.Sprintf("account:%s", accountID))
	var account Account

	err := d.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &account)
		})
	})

	if err != nil {
		return true // Account not found or error occurred
	}

	return account.BalanceDate != balanceDate
}

// UpdateLastMessageTime updates the last successful message time to now
func (d *DB) UpdateLastMessageTime() error {
	now := time.Now().Unix()
	key := []byte("last_message_time")
	data := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		data[i] = byte(now)
		now >>= 8
	}

	return d.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, data)
	})
}

// GetLastMessageTime returns the last successful message time
func (d *DB) GetLastMessageTime() (*int64, error) {
	key := []byte("last_message_time")
	var timestamp int64

	err := d.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			for i := 0; i < 8; i++ {
				timestamp = (timestamp << 8) | int64(val[i])
			}
			return nil
		})
	})

	if err != nil {
		return nil, nil // No last message time found
	}

	return &timestamp, nil
}
