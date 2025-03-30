package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

var cacheFile = filepath.Join(os.Getenv("HOME"), ".finance_tracker_cache.json")

func (c *Cache) Load() error {
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, c)
}

func (c *Cache) Save() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, data, 0644)
}

func (c *Cache) UpdateAccount(accountID string, balance Balance, balanceDate int64) {
	if c.Accounts == nil {
		c.Accounts = make(map[string]map[string]interface{})
	}

	c.Accounts[accountID] = map[string]interface{}{
		"balance":      float64(balance),
		"balance_date": balanceDate,
	}
}

func (c *Cache) IsAccountUpdated(accountID string, balanceDate int64) bool {
	if c.Accounts == nil {
		return true
	}

	account, exists := c.Accounts[accountID]
	if !exists {
		return true
	}

	storedDate, ok := account["balance_date"].(float64)
	if !ok {
		return true
	}

	return int64(storedDate) != balanceDate
}

func (c *Cache) UpdateLastMessageTime() {
	now := time.Now().Unix()
	c.LastSuccessfulMessage = &now
} 