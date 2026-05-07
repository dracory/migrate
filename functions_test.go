package migrate

import (
	"testing"
)

func TestIsValidMigrationID(t *testing.T) {
	t.Run("accepts valid migration IDs", func(t *testing.T) {
		validIDs := []string{
			"2026_03_21_create_users_table",
			"2022_01_01_000_create_schema_migrations",
			"2026_12_31_add_holiday_table",
			"2026_06_15_update_user_profiles",
		}

		for _, id := range validIDs {
			if !isValidMigrationID(id) {
				t.Errorf("Expected valid ID %s to be accepted", id)
			}
		}
	})

	t.Run("rejects invalid migration IDs", func(t *testing.T) {
		invalidIDs := []struct {
			id       string
			expected string
		}{
			{"invalid_format", "must follow format YYYY_MM_DD_description"},
			{"2026_13_01_create_table", "must follow format YYYY_MM_DD_description"},
			{"2026_03_32_create_table", "must follow format YYYY_MM_DD_description"},
			{"2026_03_21", "must follow format YYYY_MM_DD_description"},
			{"2026_03_21_", "must follow format YYYY_MM_DD_description"},
			{"", "must follow format YYYY_MM_DD_description"},
		}

		for _, test := range invalidIDs {
			if isValidMigrationID(test.id) {
				t.Errorf("Expected invalid ID %s to be rejected", test.id)
			}
		}
	})

	t.Run("validates month range", func(t *testing.T) {
		invalidMonths := []string{"2026_00_01_test", "2026_13_01_test", "2026_99_01_test"}
		for _, id := range invalidMonths {
			if isValidMigrationID(id) {
				t.Errorf("Expected invalid month in ID %s to be rejected", id)
			}
		}
	})

	t.Run("validates day range", func(t *testing.T) {
		invalidDays := []string{"2026_03_00_test", "2026_03_32_test", "2026_03_99_test"}
		for _, id := range invalidDays {
			if isValidMigrationID(id) {
				t.Errorf("Expected invalid day in ID %s to be rejected", id)
			}
		}
	})
}
