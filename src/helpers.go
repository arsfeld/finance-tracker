package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
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

// stripMarkdown removes markdown formatting from text using regex
func stripMarkdown(text string) string {
	// Remove bold formatting (**text** or __text__)
	text = regexp.MustCompile(`\*\*(.*?)\*\*|__(.*?)__`).ReplaceAllString(text, "$1$2")

	// Remove italic formatting (*text* or _text_)
	text = regexp.MustCompile(`\*(.*?)\*|_(.*?)_`).ReplaceAllString(text, "$1$2")

	// Remove heading markers (# text)
	text = regexp.MustCompile(`^#+\s+`).ReplaceAllString(text, "")

	return text
}

// convertMarkdownToHTML converts markdown text to HTML
func convertMarkdownToHTML(md string) string {
	// Create markdown parser with common extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs
	p := parser.NewWithExtensions(extensions)

	// Parse markdown into AST
	node := p.Parse([]byte(md))

	// Create HTML renderer with common flags
	opts := html.RendererOptions{
		Flags: html.CommonFlags | html.HrefTargetBlank,
	}
	renderer := html.NewRenderer(opts)

	// Render HTML and remove newlines
	html := string(markdown.Render(node, renderer))
	return strings.ReplaceAll(html, "\n", "")
}
