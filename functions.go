package migrate

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// isValidMigrationID validates that the migration ID follows the format YYYY_MM_DD_HHMM_description
func isValidMigrationID(id string) bool {
	// Check total length to prevent excessively long IDs
	if len(id) > 255 {
		return false
	}

	// Use simple validation without regex for simplicity
	parts := strings.Split(id, "_")
	if len(parts) < 5 {
		return false
	}

	// Check date parts: YYYY, MM, DD
	if len(parts[0]) != 4 || len(parts[1]) != 2 || len(parts[2]) != 2 {
		return false
	}

	// Check time part: HHMM (4 digits)
	if len(parts[3]) != 4 {
		return false
	}

	// Check if date and time parts are numeric
	for i := range 4 {
		if _, err := strconv.Atoi(parts[i]); err != nil {
			return false
		}
	}

	// Check month range (1-12)
	month, _ := strconv.Atoi(parts[1])
	if month < 1 || month > 12 {
		return false
	}

	// Check day range (1-31)
	day, _ := strconv.Atoi(parts[2])
	if day < 1 || day > 31 {
		return false
	}

	// Validate actual calendar date (e.g., reject February 30)
	dateStr := fmt.Sprintf("%s-%s-%s", parts[0], parts[1], parts[2])
	_, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return false
	}

	// Validate time part: HH (0-23) and MM (0-59)
	hour, _ := strconv.Atoi(parts[3][:2])
	minute, _ := strconv.Atoi(parts[3][2:])
	if hour < 0 || hour > 23 {
		return false
	}
	if minute < 0 || minute > 59 {
		return false
	}

	// Check that description exists and is not empty
	description := strings.Join(parts[4:], "_")
	if len(description) == 0 {
		return false
	}

	// Check description length (prevent excessively long descriptions)
	if len(description) > 200 {
		return false
	}

	return true
}
