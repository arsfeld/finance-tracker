package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// getCacheFilePath returns the path to the cache file
func getCacheFilePath() string {
	return filepath.Join(os.Getenv("HOME"), ".finance_tracker_cache.json")
}

// Load loads the cache from the cache file
func (c *Cache) Load() error {
	cacheFile := getCacheFilePath()
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, c)
}

// Save saves the cache to the cache file
func (c *Cache) Save() error {
	cacheFile := getCacheFilePath()
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, data, 0644)
}

// UpdateAccount updates the account information in the cache
func (c *Cache) UpdateAccount(accountID string, balance Balance, balanceDate int64) {
	if c.Accounts == nil {
		c.Accounts = make(map[string]map[string]interface{})
	}

	c.Accounts[accountID] = map[string]interface{}{
		"balance":      float64(balance),
		"balance_date": balanceDate,
	}
}

// IsAccountUpdated checks if the account has been updated since the last time
// it was stored in the cache
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

// UpdateLastMessageTime updates the last successful message time to now
func (c *Cache) UpdateLastMessageTime() {
	now := time.Now().Unix()
	c.LastSuccessfulMessage = &now
}
