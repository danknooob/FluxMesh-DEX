package migrations

import (
	_ "embed"

	"gorm.io/gorm"
)

//go:embed stored_procedures.sql
var storedProceduresSQL string

// RunStoredProcedures creates (or replaces) all PL/pgSQL
// functions used by the API and Indexer repositories.
// Idempotent — safe to call on every startup.
func RunStoredProcedures(db *gorm.DB) error {
	return db.Exec(storedProceduresSQL).Error
}
