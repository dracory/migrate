package migrate_test

import (
	"strings"
	"testing"

	"github.com/dracory/migrate"
)

func TestValidateMigrationIDWithFormat_AcceptsValidHHMMFormatIDs(t *testing.T) {
	validIDs := []string{
		"2026_03_21_1200_create_users_table",
		"2022_01_01_0000_create_schema_migrations",
		"2026_12_31_2359_add_holiday_table",
		"2026_06_15_0830_update_user_profiles",
	}

	for _, id := range validIDs {
		if err := migrate.ValidateMigrationID(id, migrate.NamingFormatHHMM); err != nil {
			t.Errorf("Expected valid ID %s to be accepted, got error: %v", id, err)
		}
	}
}

func TestValidateMigrationIDWithFormat_RejectsInvalidHHMMFormatIDs(t *testing.T) {
	invalidIDs := []string{
		"invalid_format",
		"2026_13_01_1200_create_table",
		"2026_03_32_1200_create_table",
		"2026_03_21",
		"2026_03_21_1200",
		"2026_03_21_1200_",
		"",
		"2026_03_21_create_users_table",
		"2026_03_21_12_create_users_table",
	}

	for _, id := range invalidIDs {
		if err := migrate.ValidateMigrationID(id, migrate.NamingFormatHHMM); err == nil {
			t.Errorf("Expected invalid ID %s to be rejected", id)
		}
	}
}

func TestValidateMigrationIDWithFormat_ValidatesMonthRangeForHHMMFormat(t *testing.T) {
	invalidMonths := []string{"2026_00_01_1200_test", "2026_13_01_1200_test", "2026_99_01_1200_test"}
	for _, id := range invalidMonths {
		if err := migrate.ValidateMigrationID(id, migrate.NamingFormatHHMM); err == nil {
			t.Errorf("Expected invalid month in ID %s to be rejected", id)
		}
	}
}

func TestValidateMigrationIDWithFormat_ValidatesDayRangeForHHMMFormat(t *testing.T) {
	invalidDays := []string{"2026_03_00_1200_test", "2026_03_32_1200_test", "2026_03_99_1200_test"}
	for _, id := range invalidDays {
		if err := migrate.ValidateMigrationID(id, migrate.NamingFormatHHMM); err == nil {
			t.Errorf("Expected invalid day in ID %s to be rejected", id)
		}
	}
}

func TestValidateMigrationIDWithFormat_ValidatesTimeRangeForHHMMFormat(t *testing.T) {
	invalidTimes := []string{
		"2026_03_21_2400_test",
		"2026_03_21_1260_test",
		"2026_03_21_9999_test",
	}
	for _, id := range invalidTimes {
		if err := migrate.ValidateMigrationID(id, migrate.NamingFormatHHMM); err == nil {
			t.Errorf("Expected invalid time in ID %s to be rejected", id)
		}
	}
}

func TestValidateMigrationIDWithFormat_RejectsInvalidCalendarDatesForHHMMFormat(t *testing.T) {
	invalidDates := []string{
		"2026_02_30_1200_test",
		"2026_04_31_1200_test",
		"2026_06_31_1200_test",
		"2026_09_31_1200_test",
		"2026_11_31_1200_test",
	}
	for _, id := range invalidDates {
		if err := migrate.ValidateMigrationID(id, migrate.NamingFormatHHMM); err == nil {
			t.Errorf("Expected invalid calendar date %s to be rejected", id)
		}
	}
}

func TestValidateMigrationIDWithFormat_AcceptsValidCalendarDatesIncludingLeapYearsForHHMMFormat(t *testing.T) {
	validDates := []string{
		"2024_02_29_1200_test",
		"2026_02_28_1200_test",
	}
	for _, id := range validDates {
		if err := migrate.ValidateMigrationID(id, migrate.NamingFormatHHMM); err != nil {
			t.Errorf("Expected valid calendar date %s to be accepted, got error: %v", id, err)
		}
	}
}

func TestValidateMigrationIDWithFormat_AcceptsValidNNNFormatIDs(t *testing.T) {
	validIDs := []string{
		"2026_03_21_001_create_users_table",
		"2022_01_01_000_create_schema_migrations",
		"2026_12_31_999_add_holiday_table",
		"2026_06_15_123_update_user_profiles",
	}

	for _, id := range validIDs {
		if err := migrate.ValidateMigrationID(id, migrate.NamingFormatNNN); err != nil {
			t.Errorf("Expected valid NNN format ID %s to be accepted, got error: %v", id, err)
		}
	}
}

func TestValidateMigrationIDWithFormat_RejectsInvalidNNNFormatIDs(t *testing.T) {
	invalidIDs := []string{
		"invalid_format",
		"2026_13_01_001_create_table",
		"2026_03_32_001_create_table",
		"2026_03_21",
		"2026_03_21_001",
		"2026_03_21_001_",
		"",
		"2026_03_21_create_users_table",
		"2026_03_21_12_create_users_table",
		"2026_03_21_1234_create_users_table",
	}

	for _, id := range invalidIDs {
		if err := migrate.ValidateMigrationID(id, migrate.NamingFormatNNN); err == nil {
			t.Errorf("Expected invalid NNN format ID %s to be rejected", id)
		}
	}
}

func TestValidateMigrationIDWithFormat_ValidatesSequenceRangeForNNNFormat(t *testing.T) {
	invalidSequences := []string{
		"2026_03_21_1000_test",
		"2026_03_21_9999_test",
	}
	for _, id := range invalidSequences {
		if err := migrate.ValidateMigrationID(id, migrate.NamingFormatNNN); err == nil {
			t.Errorf("Expected invalid sequence in NNN format ID %s to be rejected", id)
		}
	}
}

func TestValidateMigrationIDWithFormat_NNNFormatRejectsHHMMFormatIDs(t *testing.T) {
	hhmmIDs := []string{
		"2026_03_21_1200_create_users_table",
		"2022_01_01_0000_create_schema_migrations",
	}
	for _, id := range hhmmIDs {
		if err := migrate.ValidateMigrationID(id, migrate.NamingFormatNNN); err == nil {
			t.Errorf("Expected HHMM format ID %s to be rejected in NNN mode", id)
		}
	}
}

func TestValidateMigrationIDWithFormat_HHMMFormatRejectsNNNFormatIDs(t *testing.T) {
	nnnIDs := []string{
		"2026_03_21_001_create_users_table",
		"2022_01_01_000_create_schema_migrations",
	}
	for _, id := range nnnIDs {
		if err := migrate.ValidateMigrationID(id, migrate.NamingFormatHHMM); err == nil {
			t.Errorf("Expected NNN format ID %s to be rejected in HHMM mode", id)
		}
	}
}

func TestValidateMigrationIDWithFormat_InvalidFormatReturnsError(t *testing.T) {
	testID := "2026_03_21_001_create_users_table"
	if err := migrate.ValidateMigrationID(testID, "invalid"); err == nil {
		t.Errorf("Expected ID to be rejected with invalid format")
	}
}

func TestValidateTableName_AcceptsValidTableNames(t *testing.T) {
	validNames := []string{
		"schema_migrations",
		"migrations",
		"custom_table",
		"a",
		"Table123",
		"_underscore",
		"_123",
	}

	for _, name := range validNames {
		if err := migrate.ValidateTableName(name); err != nil {
			t.Errorf("Expected valid name '%s' to be accepted, got: %v", name, err)
		}
	}
}

func TestValidateTableName_RejectsEmptyTableName(t *testing.T) {
	if err := migrate.ValidateTableName(""); err == nil {
		t.Error("Expected error for empty table name")
	}
}

func TestValidateTableName_RejectsTableNameTooLong(t *testing.T) {
	longName := strings.Repeat("a", 65)
	if err := migrate.ValidateTableName(longName); err == nil {
		t.Error("Expected error for table name too long")
	}
}

func TestValidateTableName_RejectsTableNameStartingWithDigit(t *testing.T) {
	invalidNames := []string{
		"123_table",
		"9table",
		"0_migrations",
	}

	for _, name := range invalidNames {
		if err := migrate.ValidateTableName(name); err == nil {
			t.Errorf("Expected invalid name '%s' (starting with digit) to be rejected", name)
		}
	}
}

func TestValidateTableName_RejectsTableNameWithSpecialCharacters(t *testing.T) {
	invalidNames := []string{
		"table-name",
		"table.name",
		"table name",
		"table;name",
		"table'name",
		"table\"name",
		"table*name",
	}

	for _, name := range invalidNames {
		if err := migrate.ValidateTableName(name); err == nil {
			t.Errorf("Expected invalid name '%s' to be rejected", name)
		}
	}
}
