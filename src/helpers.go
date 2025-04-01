package main

import (
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// initLogger initializes the Zerolog logger with the appropriate level
func initLogger(verbose bool) {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}

// retryWithBackoff implements a retry mechanism with exponential backoff
func retryWithBackoff[T any](
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
		delay *= 2 // Exponential backoff
	}

	// Return zero value and error if all retries failed
	var zero T
	return zero, fmt.Errorf("all %d retry attempts failed for %s. Last error: %w", maxRetries, operationName, lastErr)
}

// getStringValue safely gets a string value from a pointer
func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
