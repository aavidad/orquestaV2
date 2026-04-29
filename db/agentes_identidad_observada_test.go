package db

import (
	"strings"
	"testing"
)

func TestRenderAgentesIdentidadObservadaSchemaForDriverPostgresConvierteDDL(t *testing.T) {
	ddls := renderAgentesIdentidadObservadaSchemaForDriver("postgres")
	if len(ddls) != 3 {
		t.Fatalf("esperaba 3 sentencias, tengo %d", len(ddls))
	}
	if strings.Contains(ddls[0], " DATETIME") {
		t.Fatalf("ddl identidad observada conserva sintaxis sqlite: %s", ddls[0])
	}
	if !strings.Contains(ddls[0], " TIMESTAMP") {
		t.Fatalf("ddl identidad observada no adapta timestamps: %s", ddls[0])
	}
}
