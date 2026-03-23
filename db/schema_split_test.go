package db

import (
	"strings"
	"testing"
)

func TestSchemaSeSeparaEnDDLYSeeds(t *testing.T) {
	t.Parallel()

	ddl := schemaDDLForDriver("sqlite")
	seeds := schemaSeedDataForDriver("sqlite")
	if ddl == "" {
		t.Fatalf("ddl vacio")
	}
	if seeds == "" {
		t.Fatalf("seeds vacio")
	}
	if strings.Contains(ddl, "INSERT OR IGNORE INTO agentes") {
		t.Fatalf("ddl no deberia contener semillas")
	}
	if strings.Contains(seeds, "CREATE TABLE IF NOT EXISTS agentes") {
		t.Fatalf("seeds no deberia contener ddl")
	}
	for _, required := range []string{
		"CREATE TABLE IF NOT EXISTS agentes",
		"CREATE TABLE IF NOT EXISTS sesiones",
		"CREATE TABLE IF NOT EXISTS runtime_handles",
		"CREATE TRIGGER IF NOT EXISTS trig_runtime_orders_updated",
	} {
		if !strings.Contains(ddl, required) {
			t.Fatalf("ddl no contiene %q", required)
		}
	}
	for _, required := range []string{
		"INSERT OR IGNORE INTO agentes",
		"INSERT OR IGNORE INTO config",
		"INSERT OR IGNORE INTO conectores",
		"INSERT OR IGNORE INTO reglas",
		"INSERT OR IGNORE INTO skills",
		"INSERT OR IGNORE INTO workflows",
	} {
		if !strings.Contains(seeds, required) {
			t.Fatalf("seeds no contiene %q", required)
		}
	}
}

func TestSchemaSeedDataForDriverMySQLUsaInsertIgnore(t *testing.T) {
	t.Parallel()

	seeds := schemaSeedDataForDriver("mysql")
	if seeds == "" {
		t.Fatalf("seeds mysql vacio")
	}
	if !strings.Contains(seeds, "INSERT IGNORE INTO agentes") {
		t.Fatalf("mysql deberia usar INSERT IGNORE")
	}
	if strings.Contains(seeds, "INSERT OR IGNORE INTO agentes") {
		t.Fatalf("mysql no deberia conservar INSERT OR IGNORE")
	}
}
