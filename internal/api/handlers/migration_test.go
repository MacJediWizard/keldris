package handlers

import "testing"

func TestMigration(t *testing.T) {
	t.Skip("migration handler uses concrete migration.Exporter/Importer + 30-method MigrationStore + real SessionStore; integration-tested via migration_e2e_test.go")
}
