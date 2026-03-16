package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// Client communicates with the OpenRouter API.
type Client struct {
	url    string
	apiKey string
	models []string
}

type openRouterRequest struct {
	Model       string    `json:"model"`
	Models      []string  `json:"models"`
	Messages    []message `json:"messages"`
	Reasoning   reasoning `json:"reasoning"`
	Temperature float64   `json:"temperature,omitempty"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type reasoning struct {
	Exclude bool `json:"exclude"`
}

type openRouterResponse struct {
	ID       string   `json:"id"`
	Choices  []choice `json:"choices"`
	Error    *apiErr  `json:"error,omitempty"`
	Provider string   `json:"provider"`
	Model    string   `json:"model"`
	Usage    usage    `json:"usage"`
}

type choice struct {
	Message      message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type apiErr struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// NewClient creates a new OpenRouter API client.
func NewClient(url, apiKey, modelList string) *Client {
	models := strings.Split(modelList, ",")
	for i := range models {
		models[i] = strings.TrimSpace(models[i])
	}
	return &Client{url: url, apiKey: apiKey, models: models}
}

// Analyze sends a spending analysis prompt to the LLM and returns the response.
func (c *Client) Analyze(systemPrompt, userPrompt string, isComplex bool) (string, string, error) {
	req := openRouterRequest{
		Models:      c.models,
		Temperature: 0.4,
		Messages: []message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Reasoning: reasoning{Exclude: !isComplex},
	}

	content, model, err := c.call(req)
	if err != nil {
		return "", "", err
	}

	return content, model, nil
}

// CategorizeJSON sends a categorization prompt expecting JSON response.
func (c *Client) CategorizeJSON(systemPrompt, userPrompt string) (string, error) {
	req := openRouterRequest{
		Models:      c.models,
		Temperature: 0.1,
		Messages: []message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Reasoning: reasoning{Exclude: true},
	}

	content, _, err := c.call(req)
	if err != nil {
		return "", err
	}

	// Strip markdown code fences if present.
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	} else if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}

	return content, nil
}

func (c *Client) call(req openRouterRequest) (string, string, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", "", fmt.Errorf("error marshaling request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, c.url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", "", fmt.Errorf("error creating request: %w", err)
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 360 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", "", fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("error reading response body: %w", err)
	}

	log.Debug().Int("status_code", resp.StatusCode).Msg("OpenRouter response")

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var orResp openRouterResponse
	if err := json.Unmarshal(bodyBytes, &orResp); err != nil {
		return "", "", fmt.Errorf("error decoding response: %w", err)
	}

	log.Info().Str("model", orResp.Model).Str("provider", orResp.Provider).Msg("OpenRouter response")

	if orResp.Error != nil {
		return "", "", fmt.Errorf("OpenRouter API error: %s (code: %d)", orResp.Error.Message, orResp.Error.Code)
	}

	if len(orResp.Choices) == 0 {
		return "", "", fmt.Errorf("no response from OpenRouter")
	}

	content := orResp.Choices[0].Message.Content
	if content == "" {
		return "", "", fmt.Errorf("received empty response from LLM")
	}

	return content, orResp.Model, nil
}

// RetryWithBackoff retries an operation with exponential backoff.
func RetryWithBackoff[T any](
	operation func() (T, error),
	maxRetries int,
	retryDelay int,
	operationName string,
) (T, error) {
	var result T
	var lastErr error
	delay := time.Duration(retryDelay) * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		result, lastErr = operation()
		if lastErr == nil {
			return result, nil
		}

		log.Warn().
			Err(lastErr).
			Int("attempt", attempt).
			Int("max_retries", maxRetries).
			Dur("delay", delay).
			Str("operation", operationName).
			Msg("Retrying operation after delay")

		time.Sleep(delay)
		delay *= 2
	}

	var zero T
	return zero, fmt.Errorf("all %d retry attempts failed for %s: %w", maxRetries, operationName, lastErr)
}
