package migrate

import (
	"testing"
)

func TestIsValidMigrationID(t *testing.T) {
	t.Run("accepts valid migration IDs", func(t *testing.T) {
		validIDs := []string{
			"2026_03_21_1200_create_users_table",
			"2022_01_01_0000_create_schema_migrations",
			"2026_12_31_2359_add_holiday_table",
			"2026_06_15_0830_update_user_profiles",
		}

		for _, id := range validIDs {
			if !isValidMigrationID(id) {
				t.Errorf("Expected valid ID %s to be accepted", id)
			}
		}
	})

	t.Run("rejects invalid migration IDs", func(t *testing.T) {
		invalidIDs := []string{
			"invalid_format",
			"2026_13_01_1200_create_table",
			"2026_03_32_1200_create_table",
			"2026_03_21",
			"2026_03_21_1200",
			"2026_03_21_1200_",
			"",
			"2026_03_21_create_users_table",    // missing HHMM
			"2026_03_21_12_create_users_table", // HHMM too short
		}

		for _, id := range invalidIDs {
			if isValidMigrationID(id) {
				t.Errorf("Expected invalid ID %s to be rejected", id)
			}
		}
	})

	t.Run("validates month range", func(t *testing.T) {
		invalidMonths := []string{"2026_00_01_1200_test", "2026_13_01_1200_test", "2026_99_01_1200_test"}
		for _, id := range invalidMonths {
			if isValidMigrationID(id) {
				t.Errorf("Expected invalid month in ID %s to be rejected", id)
			}
		}
	})

	t.Run("validates day range", func(t *testing.T) {
		invalidDays := []string{"2026_03_00_1200_test", "2026_03_32_1200_test", "2026_03_99_1200_test"}
		for _, id := range invalidDays {
			if isValidMigrationID(id) {
				t.Errorf("Expected invalid day in ID %s to be rejected", id)
			}
		}
	})

	t.Run("validates time range", func(t *testing.T) {
		invalidTimes := []string{
			"2026_03_21_2400_test", // hour 24 invalid
			"2026_03_21_1260_test", // minute 60 invalid
			"2026_03_21_9999_test", // both invalid
		}
		for _, id := range invalidTimes {
			if isValidMigrationID(id) {
				t.Errorf("Expected invalid time in ID %s to be rejected", id)
			}
		}
	})

	t.Run("rejects invalid calendar dates", func(t *testing.T) {
		invalidDates := []string{
			"2026_02_30_1200_test", // February 30 doesn't exist
			"2026_04_31_1200_test", // April has only 30 days
			"2026_06_31_1200_test", // June has only 30 days
			"2026_09_31_1200_test", // September has only 30 days
			"2026_11_31_1200_test", // November has only 30 days
		}
		for _, id := range invalidDates {
			if isValidMigrationID(id) {
				t.Errorf("Expected invalid calendar date %s to be rejected", id)
			}
		}
	})

	t.Run("accepts valid calendar dates including leap years", func(t *testing.T) {
		validDates := []string{
			"2024_02_29_1200_test", // 2024 is a leap year, Feb 29 is valid
			"2026_02_28_1200_test", // 2026 is not a leap year, Feb 28 is valid
		}
		for _, id := range validDates {
			if !isValidMigrationID(id) {
				t.Errorf("Expected valid calendar date %s to be accepted", id)
			}
		}
	})
}
