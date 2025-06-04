package simplefin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"finance_tracker/src/providers"
)

// SimpleFin provider implementation
type SimpleFin struct {
	httpClient *http.Client
}

// NewSimpleFin creates a new SimpleFin provider
func NewSimpleFin() *SimpleFin {
	return &SimpleFin{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetProviderType returns the provider type
func (s *SimpleFin) GetProviderType() string {
	return "simplefin"
}

// ValidateCredentials validates SimpleFin credentials
func (s *SimpleFin) ValidateCredentials(credentials map[string]string) error {
	// Check for setup token (initial setup)
	if setupToken, ok := credentials["setup_token"]; ok && setupToken != "" {
		// Validate it's a valid base64 string
		if _, err := base64.StdEncoding.DecodeString(setupToken); err != nil {
			return &providers.ProviderError{
				Code:    "INVALID_CREDENTIALS",
				Message: "setup_token must be a valid base64 encoded string",
				Retry:   false,
			}
		}
		return nil
	}
	
	// Check for access URL (established connection)
	if accessURL, ok := credentials["access_url"]; ok && accessURL != "" {
		if !strings.HasPrefix(accessURL, "https://") {
			return &providers.ProviderError{
				Code:    "INVALID_CREDENTIALS",
				Message: "access_url must be HTTPS",
				Retry:   false,
			}
		}
		return nil
	}

	return &providers.ProviderError{
		Code:    "MISSING_CREDENTIALS",
		Message: "Either setup_token or access_url is required",
		Retry:   false,
	}
}

// SetupTokenExchange exchanges a setup token for access URL
func (s *SimpleFin) SetupTokenExchange(ctx context.Context, setupToken string) (accessURL string, err error) {
	// Decode the setup token to get the claim URL
	claimURLBytes, err := base64.StdEncoding.DecodeString(setupToken)
	if err != nil {
		return "", &providers.ProviderError{
			Code:    "INVALID_TOKEN",
			Message: fmt.Sprintf("failed to decode setup token: %v", err),
			Retry:   false,
		}
	}
	
	claimURL := string(claimURLBytes)
	
	// Validate the claim URL
	if !strings.HasPrefix(claimURL, "https://") {
		return "", &providers.ProviderError{
			Code:    "INVALID_CLAIM_URL",
			Message: "decoded claim URL must be HTTPS",
			Retry:   false,
		}
	}

	// Make the claim request (POST with Content-Length: 0)
	req, err := http.NewRequestWithContext(ctx, "POST", claimURL, nil)
	if err != nil {
		return "", &providers.ProviderError{
			Code:    "REQUEST_ERROR",
			Message: fmt.Sprintf("failed to create claim request: %v", err),
			Retry:   true,
		}
	}
	
	// Set Content-Length header as required by SimpleFin Bridge
	req.Header.Set("Content-Length", "0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", &providers.ProviderError{
			Code:    "NETWORK_ERROR",
			Message: fmt.Sprintf("failed to make claim request: %v", err),
			Retry:   true,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", &providers.ProviderError{
			Code:    "CLAIM_ERROR",
			Message: fmt.Sprintf("claim failed with status %d: %s", resp.StatusCode, string(body)),
			Retry:   resp.StatusCode >= 500,
		}
	}

	// The response body should contain the access URL
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", &providers.ProviderError{
			Code:    "READ_ERROR",
			Message: fmt.Sprintf("failed to read claim response: %v", err),
			Retry:   true,
		}
	}

	accessURL = strings.TrimSpace(string(body))
	
	fmt.Printf("DEBUG: SimpleFin SetupTokenExchange - Raw response body: %s\n", string(body))
	fmt.Printf("DEBUG: SimpleFin SetupTokenExchange - Trimmed access URL: %s\n", accessURL)
	
	// Validate the access URL
	if !strings.HasPrefix(accessURL, "https://") {
		return "", &providers.ProviderError{
			Code:    "INVALID_ACCESS_URL",
			Message: "received access URL must be HTTPS",
			Retry:   false,
		}
	}

	fmt.Printf("DEBUG: SimpleFin SetupTokenExchange - Final valid access URL: %s\n", accessURL)
	return accessURL, nil
}

// ListAccounts fetches all accounts from SimpleFin
func (s *SimpleFin) ListAccounts(ctx context.Context, credentials map[string]string) ([]providers.ProviderAccount, error) {
	if err := s.ValidateCredentials(credentials); err != nil {
		return nil, err
	}

	// Handle setup token (needs to be exchanged for access URL first)
	if setupToken, ok := credentials["setup_token"]; ok && setupToken != "" {
		return nil, &providers.ProviderError{
			Code:    "SETUP_REQUIRED",
			Message: "Setup token must be exchanged for access URL first. Use SetupTokenExchange method.",
			Retry:   false,
		}
	}

	// Use access URL for requests (SimpleFin Bridge doesn't use separate token)
	accessURL := credentials["access_url"]
	
	// Append /accounts to get account data
	accountsURL := strings.TrimSuffix(accessURL, "/") + "/accounts"
	
	fmt.Printf("DEBUG: SimpleFin ListAccounts - Original access URL: %s\n", accessURL)
	fmt.Printf("DEBUG: SimpleFin ListAccounts - Constructed accounts URL: %s\n", accountsURL)
	
	req, err := http.NewRequestWithContext(ctx, "GET", accountsURL, nil)
	if err != nil {
		return nil, &providers.ProviderError{
			Code:    "REQUEST_ERROR",
			Message: fmt.Sprintf("failed to create request: %v", err),
			Retry:   true,
		}
	}

	// SimpleFin Bridge uses the access URL directly without additional authentication
	fmt.Printf("DEBUG: SimpleFin ListAccounts - Making GET request to: %s\n", req.URL.String())
	fmt.Printf("DEBUG: SimpleFin ListAccounts - Request headers: %+v\n", req.Header)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		fmt.Printf("DEBUG: SimpleFin ListAccounts - Network error: %v\n", err)
		return nil, &providers.ProviderError{
			Code:    "NETWORK_ERROR",
			Message: fmt.Sprintf("failed to make request: %v", err),
			Retry:   true,
		}
	}
	defer resp.Body.Close()

	fmt.Printf("DEBUG: SimpleFin ListAccounts - Response status: %d %s\n", resp.StatusCode, resp.Status)
	fmt.Printf("DEBUG: SimpleFin ListAccounts - Response headers: %+v\n", resp.Header)

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("DEBUG: SimpleFin ListAccounts - Error response body: %s\n", string(body))
		return nil, &providers.ProviderError{
			Code:    "API_ERROR",
			Message: fmt.Sprintf("SimpleFin API returned status %d: %s", resp.StatusCode, string(body)),
			Retry:   resp.StatusCode >= 500,
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &providers.ProviderError{
			Code:    "READ_ERROR",
			Message: fmt.Sprintf("failed to read response: %v", err),
			Retry:   true,
		}
	}

	fmt.Printf("DEBUG: SimpleFin ListAccounts - Success response body: %s\n", string(body))

	var sfResponse SimpleFinResponse
	if err := json.Unmarshal(body, &sfResponse); err != nil {
		return nil, &providers.ProviderError{
			Code:    "PARSE_ERROR",
			Message: fmt.Sprintf("failed to parse response: %v", err),
			Retry:   false,
		}
	}

	// Convert SimpleFin accounts to provider accounts
	accounts := make([]providers.ProviderAccount, len(sfResponse.Accounts))
	for i, sfAccount := range sfResponse.Accounts {
		// Parse balance from decimal string
		balanceFloat, err := strconv.ParseFloat(sfAccount.Balance, 64)
		if err != nil {
			fmt.Printf("DEBUG: SimpleFin ListAccounts - Failed to parse balance '%s': %v\n", sfAccount.Balance, err)
			balanceFloat = 0
		}
		
		// Balance date is already an int64 from JSON
		balanceDateInt := sfAccount.BalanceDate
		
		accounts[i] = providers.ProviderAccount{
			ID:          sfAccount.ID,
			Name:        sfAccount.Name,
			Institution: sfAccount.Org.Name,
			Type:        "", // SimpleFin doesn't provide account type
			Balance:     balanceFloat, // SimpleFin Bridge returns actual decimal value
			Currency:    sfAccount.Currency,
			LastUpdated: time.Unix(balanceDateInt, 0),
			Metadata: map[string]interface{}{
				"simplefin_id":           sfAccount.ID,
				"simplefin_balance_date": sfAccount.BalanceDate,
			},
		}
	}

	return accounts, nil
}

// GetAccount fetches a single account from SimpleFin
func (s *SimpleFin) GetAccount(ctx context.Context, credentials map[string]string, accountID string) (*providers.ProviderAccount, error) {
	accounts, err := s.ListAccounts(ctx, credentials)
	if err != nil {
		return nil, err
	}

	for _, account := range accounts {
		if account.ID == accountID {
			return &account, nil
		}
	}

	return nil, &providers.ProviderError{
		Code:    "NOT_FOUND",
		Message: fmt.Sprintf("account %s not found", accountID),
		Retry:   false,
	}
}

// GetTransactions fetches transactions for an account from SimpleFin
func (s *SimpleFin) GetTransactions(ctx context.Context, credentials map[string]string, accountID string, startDate, endDate time.Time) ([]providers.ProviderTransaction, error) {
	accounts, err := s.ListAccounts(ctx, credentials)
	if err != nil {
		return nil, err
	}

	var targetAccount *providers.ProviderAccount
	for _, account := range accounts {
		if account.ID == accountID {
			targetAccount = &account
			break
		}
	}

	if targetAccount == nil {
		return nil, &providers.ProviderError{
			Code:    "NOT_FOUND",
			Message: fmt.Sprintf("account %s not found", accountID),
			Retry:   false,
		}
	}

	// Get the SimpleFin account data with transactions
	accessURL := credentials["access_url"]
	
	// Append /accounts to get account data with transactions
	accountsURL := strings.TrimSuffix(accessURL, "/") + "/accounts"
	
	req, err := http.NewRequestWithContext(ctx, "GET", accountsURL, nil)
	if err != nil {
		return nil, &providers.ProviderError{
			Code:    "REQUEST_ERROR",
			Message: fmt.Sprintf("failed to create request: %v", err),
			Retry:   true,
		}
	}

	// SimpleFin Bridge uses the access URL directly without additional authentication

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, &providers.ProviderError{
			Code:    "NETWORK_ERROR",
			Message: fmt.Sprintf("failed to make request: %v", err),
			Retry:   true,
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &providers.ProviderError{
			Code:    "READ_ERROR",
			Message: fmt.Sprintf("failed to read response: %v", err),
			Retry:   true,
		}
	}

	var sfResponse SimpleFinResponse
	if err := json.Unmarshal(body, &sfResponse); err != nil {
		return nil, &providers.ProviderError{
			Code:    "PARSE_ERROR",
			Message: fmt.Sprintf("failed to parse response: %v", err),
			Retry:   false,
		}
	}

	// Find the target account in the response
	var sfAccount *SimpleFinAccount
	for _, acc := range sfResponse.Accounts {
		if acc.ID == accountID {
			sfAccount = &acc
			break
		}
	}

	if sfAccount == nil {
		return nil, &providers.ProviderError{
			Code:    "NOT_FOUND",
			Message: fmt.Sprintf("account %s not found in response", accountID),
			Retry:   false,
		}
	}

	// Convert SimpleFin transactions to provider transactions
	var transactions []providers.ProviderTransaction
	for _, sfTxn := range sfAccount.Transactions {
		// Posted date is already an int64 from JSON
		txnDate := time.Unix(sfTxn.Posted, 0)
		
		// Filter by date range
		if txnDate.Before(startDate) || txnDate.After(endDate) {
			continue
		}

		// Parse amount from decimal string
		amountFloat, err := strconv.ParseFloat(sfTxn.Amount, 64)
		if err != nil {
			fmt.Printf("DEBUG: SimpleFin GetTransactions - Failed to parse amount '%s': %v\n", sfTxn.Amount, err)
			continue
		}

		postedDate := time.Unix(sfTxn.Posted, 0)
		transactions = append(transactions, providers.ProviderTransaction{
			ID:          sfTxn.ID,
			AccountID:   accountID,
			Amount:      amountFloat, // SimpleFin Bridge returns actual decimal value
			Description: sfTxn.Description,
			Merchant:    "", // SimpleFin doesn't separate merchant
			Date:        txnDate,
			PostedDate:  &postedDate,
			Pending:     false, // SimpleFin transactions are posted
			Type:        s.determineTransactionType(int64(amountFloat * 100)), // Convert to cents for type determination
			Metadata: map[string]interface{}{
				"simplefin_id":     sfTxn.ID,
				"simplefin_posted": sfTxn.Posted,
			},
		})
	}

	return transactions, nil
}

// HealthCheck performs a health check on the SimpleFin connection
func (s *SimpleFin) HealthCheck(ctx context.Context, credentials map[string]string) error {
	if err := s.ValidateCredentials(credentials); err != nil {
		return err
	}

	// Try to list accounts as a health check
	_, err := s.ListAccounts(ctx, credentials)
	return err
}

// determineTransactionType determines if a transaction is debit or credit
func (s *SimpleFin) determineTransactionType(amount int64) string {
	if amount < 0 {
		return "debit"
	}
	return "credit"
}

// SimpleFin API response structures
type SimpleFinResponse struct {
	Accounts []SimpleFinAccount `json:"accounts"`
}

type SimpleFinAccount struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	Currency     string                `json:"currency"`
	Balance      string                `json:"balance"`       // SimpleFin Bridge returns as decimal string
	BalanceDate  int64                 `json:"balance-date"`  // SimpleFin Bridge uses hyphen, returns as int
	Org          SimpleFinOrganization `json:"org"`
	Transactions []SimpleFinTransaction `json:"transactions"`
}

type SimpleFinOrganization struct {
	Name string `json:"name"`
}

type SimpleFinTransaction struct {
	ID          string `json:"id"`
	Posted      int64  `json:"posted"`     // SimpleFin Bridge returns as int64
	Amount      string `json:"amount"`     // SimpleFin Bridge returns as decimal string
	Description string `json:"description"`
}