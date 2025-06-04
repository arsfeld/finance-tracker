package main

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// Test the struct parsing to verify our fix
type SimpleFinAccount struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	Currency     string                `json:"currency"`
	Balance      string                `json:"balance"`      // SimpleFin Bridge returns as string
	BalanceDate  string                `json:"balance_date"` // SimpleFin Bridge returns as string
	Org          SimpleFinOrganization `json:"org"`
}

type SimpleFinOrganization struct {
	Name string `json:"name"`
}

type SimpleFinResponse struct {
	Accounts []SimpleFinAccount `json:"accounts"`
}

func main() {
	// Test JSON that would have failed before our fix
	testJSON := `{
		"accounts": [
			{
				"id": "test123",
				"name": "Test Account",
				"currency": "USD",
				"balance": "123456",
				"balance_date": "1704067200",
				"org": {
					"name": "Test Bank"
				}
			}
		]
	}`

	var response SimpleFinResponse
	err := json.Unmarshal([]byte(testJSON), &response)
	if err != nil {
		fmt.Printf("ERROR: Failed to parse JSON: %v\n", err)
		return
	}

	fmt.Printf("SUCCESS: Parsed %d accounts\n", len(response.Accounts))
	
	if len(response.Accounts) > 0 {
		account := response.Accounts[0]
		fmt.Printf("Account: %s\n", account.Name)
		fmt.Printf("Raw balance: %s\n", account.Balance)
		
		// Test conversion
		balanceInt, err := strconv.ParseInt(account.Balance, 10, 64)
		if err != nil {
			fmt.Printf("ERROR: Failed to parse balance: %v\n", err)
		} else {
			balanceFloat := float64(balanceInt) / 10000
			fmt.Printf("Converted balance: %.2f %s\n", balanceFloat, account.Currency)
		}
	}
}