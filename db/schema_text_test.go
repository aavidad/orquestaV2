package db

import (
	"strings"
	"testing"
)

func TestSchemaNoIncluyeCoordinacionPorFicherosObsoletos(t *testing.T) {
	t.Parallel()

	for _, prohibido := range []string{
		"Opinion.md",
		"Dudas.md",
		"ContaGrx/orquestacion.md",
		"en Opinion.md o en la BD",
	} {
		if strings.Contains(Schema, prohibido) {
			t.Fatalf("Schema contiene referencia obsoleta: %q", prohibido)
		}
	}
}
