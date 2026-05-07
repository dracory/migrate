package migrate

import (
	"strconv"
	"strings"
)

// isValidMigrationID validates that the migration ID follows the format YYYY_MM_DD_description
func isValidMigrationID(id string) bool {
	// Use simple validation without regex for simplicity
	parts := strings.Split(id, "_")
	if len(parts) < 4 {
		return false
	}

	// Check date parts
	if len(parts[0]) != 4 || len(parts[1]) != 2 || len(parts[2]) != 2 {
		return false
	}

	// Check if date parts are numeric
	for i := 0; i < 3; i++ {
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

	// Check that description exists and is not empty
	description := strings.Join(parts[3:], "_")
	if len(description) == 0 {
		return false
	}

	return true
}
