package main

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/adrg/xdg"
)

// getCacheFilePath returns the path to the cache file
func getCacheFilePath() string {
	cacheFilePath, err := xdg.CacheFile("finance_tracker/cache.json")
	if err != nil {
		log.Fatal(err)
	}
	return cacheFilePath
}

// initializeCache sets the default values for a new cache
func (c *Cache) initializeCache() {
	c.Version = 2
	c.Accounts = make(map[string]Account)
	c.LastSuccessfulMessage = nil
}

// Load loads the cache from the cache file
func (c *Cache) Load() error {
	cacheFile := getCacheFilePath()
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Initialize with default values for new cache
			c.initializeCache()
			return nil
		}
		return err
	}

	// Try to unmarshal the data
	if err := json.Unmarshal(data, c); err != nil {
		// If unmarshaling fails, initialize with default values
		c.initializeCache()
		return nil
	}

	// If version is not set or is different from 1, initialize with version 1
	if c.Version != 2 {
		c.initializeCache()
	}

	return nil
}

// Save saves the cache to the cache file
func (c *Cache) Save() error {
	// Ensure version is set to 2
	c.Version = 2

	cacheFile := getCacheFilePath()
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, data, 0644)
}

// UpdateAccount updates the account information in the cache
func (c *Cache) UpdateAccount(account Account) {
	if c.Accounts == nil {
		c.Accounts = make(map[string]Account)
	}

	c.Accounts[account.ID] = account
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

	return account.BalanceDate != balanceDate
}

// UpdateLastMessageTime updates the last successful message time to now
func (c *Cache) UpdateLastMessageTime() {
	now := time.Now().Unix()
	c.LastSuccessfulMessage = &now
}
