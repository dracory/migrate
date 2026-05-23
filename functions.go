package migrate

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// ValidateTableName ensures the table name contains only safe characters
// This function is exported to allow external validation of table names
// before creating a migrator instance.
//
// Business Logic:
// - Rejects empty table names
// - Enforces maximum length of 64 characters
// - First character must be a letter or underscore (not a digit)
// - All characters must be alphanumeric or underscore
// - Prevents SQL injection and naming conflicts
func ValidateTableName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("table name cannot be empty")
	}
	if len(name) > 64 {
		return fmt.Errorf("table name too long (max 64 characters)")
	}

	// First character must be a letter or underscore (not a digit)
	firstRune := rune(name[0])
	if !unicode.IsLetter(firstRune) && firstRune != '_' {
		return fmt.Errorf("table name must start with a letter or underscore")
	}

	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return fmt.Errorf("table name contains invalid characters (only alphanumeric and underscore allowed)")
		}
	}
	return nil
}

// validateDatePart checks that the date part (YYYY, MM, DD) is valid
//
// Business rules:
// - YYYY must be 4 digits
// - MM must be 2 digits (01-12)
// - DD must be 2 digits (01-31)
// - The date must be valid (e.g., 2025-02-30 is invalid)
// - The date can be any valid date (past, present, or future)
//
// Parameters:
//   - parts: array of strings split by underscore
//
// Returns:
//   - error if the date part is invalid, nil otherwise
func validateDatePart(parts []string) error {
	if len(parts[0]) != 4 || len(parts[1]) != 2 || len(parts[2]) != 2 {
		return fmt.Errorf("date parts must be YYYY_MM_DD format")
	}

	var nums [3]int
	for i := range 3 {
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			return fmt.Errorf("date part %d must be numeric: %w", i, err)
		}
		nums[i] = n
	}

	month := nums[1]
	if month < 1 || month > 12 {
		return fmt.Errorf("month must be between 01 and 12, got %02d", month)
	}

	day := nums[2]
	if day < 1 || day > 31 {
		return fmt.Errorf("day must be between 01 and 31, got %02d", day)
	}

	dateStr := fmt.Sprintf("%s-%s-%s", parts[0], parts[1], parts[2])
	_, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return fmt.Errorf("invalid calendar date: %w", err)
	}
	return nil
}

// validateTimePart checks that the time part (HHMM) is valid (00:00-23:59)
//
// Business Logic:
// - Requires exactly 4 digits
// - Parses as integer to extract hour and minute
// - Hour must be between 00-23
// - Minute must be between 00-59
// - Ensures valid 24-hour time format
func validateTimePart(part string) error {
	if len(part) != 4 {
		return fmt.Errorf("time part must be 4 digits (HHMM)")
	}

	num, err := strconv.Atoi(part)
	if err != nil {
		return fmt.Errorf("time part must be numeric: %w", err)
	}

	hour := num / 100
	minute := num % 100
	if hour < 0 || hour > 23 {
		return fmt.Errorf("hour must be between 00 and 23, got %02d", hour)
	}
	if minute < 0 || minute > 59 {
		return fmt.Errorf("minute must be between 00 and 59, got %02d", minute)
	}
	return nil
}

// validateSequencePart checks that the sequence part (NNN) is valid (000-999)
//
// Business Logic:
// - Requires exactly 3 digits
// - Parses as integer
// - Sequence must be between 000-999
// - Allows up to 999 migrations per day
func validateSequencePart(part string) error {
	if len(part) != 3 {
		return fmt.Errorf("sequence part must be 3 digits (NNN)")
	}

	sequence, err := strconv.Atoi(part)
	if err != nil {
		return fmt.Errorf("sequence part must be numeric: %w", err)
	}

	if sequence < 0 || sequence > 999 {
		return fmt.Errorf("sequence must be between 000 and 999, got %03d", sequence)
	}
	return nil
}

// validateDescription checks that the description exists and is within length limits
//
// Business Logic:
// - Joins all parts after the time/sequence part with underscores
// - Description cannot be empty
// - Maximum length is 200 characters
// - Ensures meaningful migration identifiers
func validateDescription(parts []string) error {
	description := strings.Join(parts[4:], "_")
	if len(description) == 0 {
		return fmt.Errorf("description cannot be empty")
	}
	if len(description) > 200 {
		return fmt.Errorf("description too long (max 200 characters)")
	}
	return nil
}

// ValidateMigrationID validates that the migration ID follows the specified format
// Supported formats:
//   - YYYY_MM_DD_HHMM_description (for HHMM format)
//   - YYYY_MM_DD_NNN_description (for NNN format)
//   - none (no prefix format restriction)
//
// Business Logic:
// - Enforces maximum length of 255 characters
// - Rejects empty IDs
// - For "none" format: only validates length and non-empty
// - For other formats: requires at least 5 underscore-separated parts
// - Validates date part (YYYY_MM_DD) is a valid calendar date
// - Validates time part (HHMM) or sequence part (NNN) based on format
// - Validates description exists and is within length limits
// - Ensures lexicographical ordering by date/time
func ValidateMigrationID(id string, format NamingFormat) error {
	if len(id) > 255 {
		return fmt.Errorf("migration ID too long (max 255 characters)")
	}

	if len(id) == 0 {
		return fmt.Errorf("migration ID cannot be empty")
	}

	// For "none" format, only validate length and non-empty
	if format == NamingFormatPrefixNone {
		return nil
	}

	parts := strings.Split(id, "_")

	// Minimum 5 parts: YYYY, MM, DD, [HHMM|NNN], description
	if len(parts) < 5 {
		return fmt.Errorf("migration ID must have at least 5 parts separated by underscores")
	}

	if err := validateDatePart(parts); err != nil {
		return err
	}

	if format == NamingFormatPrefixYYYY_MM_DD_HHMM {
		if err := validateTimePart(parts[3]); err != nil {
			return err
		}
	} else if format == NamingFormatPrefixYYYY_MM_DD_NNN {
		if err := validateSequencePart(parts[3]); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("invalid naming format: %s", format)
	}

	return validateDescription(parts)
}
