package main

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
)

// LoadCategories reads the persistent category store from a JSON file.
// Returns an empty store if the file doesn't exist.
func LoadCategories(path string) (CategoryStore, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Debug().Str("path", path).Msg("Categories file not found, starting with empty store")
			return make(CategoryStore), nil
		}
		return nil, err
	}

	var store CategoryStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}

	// Count total merchants
	total := 0
	for _, merchants := range store {
		total += len(merchants)
	}

	log.Debug().
		Int("categories", len(store)).
		Int("merchants", total).
		Str("path", path).
		Msg("Loaded category store")

	return store, nil
}

// SaveCategories writes the category store to a JSON file with indentation for readability.
func SaveCategories(path string, store CategoryStore) error {
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// LookupCategory finds the category for a given transaction description
// using case-insensitive substring matching against known merchants.
func LookupCategory(store CategoryStore, description string) (string, bool) {
	descLower := strings.ToLower(description)

	for category, merchants := range store {
		for _, merchant := range merchants {
			merchantLower := strings.ToLower(merchant)
			if strings.Contains(descLower, merchantLower) || strings.Contains(merchantLower, descLower) {
				return category, true
			}
		}
	}

	return "", false
}

// AddToCategory adds a merchant description to a category, avoiding duplicates.
func AddToCategory(store CategoryStore, category string, description string) {
	// Check for duplicates
	for _, existing := range store[category] {
		if strings.EqualFold(existing, description) {
			return
		}
	}

	store[category] = append(store[category], description)
}
