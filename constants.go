package migrate

const (
	DefaultTableName = "schema_migrations"

	ColumnID          = "id"
	ColumnBatch       = "batch"
	ColumnDescription = "description"
	ColumnStartedAt   = "started_at"
	ColumnCompletedAt = "completed_at"

	DirectionUp   = "up"
	DirectionDown = "down"

	BuiltinMigrationID = "2022_01_01_000_create_schema_migrations"
)
