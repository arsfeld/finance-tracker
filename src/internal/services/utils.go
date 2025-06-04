package services

import "time"

// parseTimestamp parses a timestamp string
func parseTimestamp(timestampStr string) (time.Time, error) {
	if timestampStr == "" {
		return time.Time{}, nil
	}
	// Try RFC3339 format first
	if t, err := time.Parse(time.RFC3339, timestampStr); err == nil {
		return t, nil
	}
	// Try with nanoseconds
	if t, err := time.Parse("2006-01-02T15:04:05.999999Z", timestampStr); err == nil {
		return t, nil
	}
	// Try PostgreSQL format
	return time.Parse("2006-01-02 15:04:05.999999-07", timestampStr)
}

// parseDate parses a date string in YYYY-MM-DD format
func parseDate(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, nil
	}
	return time.Parse("2006-01-02", dateStr)
}

// stringPtrToString converts a string pointer to string, returning empty string if nil
func stringPtrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}