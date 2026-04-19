/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import (
	"strings"
	"testing"
)

func TestValidarEntregaWriteSet(t *testing.T) {
	prepararDBTemporal(t)

	// Insertar una especificación con write_set definido
	especID := mustInsertID(t, `
		INSERT INTO especificaciones_funcion
		(titulo, archivo_objetivo, simbolo_objetivo, descripcion, write_set_json, creado_por)
		VALUES (?,?,?,?,?,?)`,
		"Test spec", "db/foo.go", "FooFunc", "test", `["db/foo.go","db/bar.go"]`, "test",
	)

	// Todos los ficheros permitidos
	ok, noPermitidos, err := ValidarEntregaWriteSet(especID, []string{"db/foo.go"})
	if err != nil {
		t.Fatalf("caso permitido: error inesperado: %v", err)
	}
	if !ok || len(noPermitidos) != 0 {
		t.Fatalf("caso permitido: esperaba ok=true, got ok=%v noPermitidos=%v", ok, noPermitidos)
	}

	// Fichero fuera del write-set
	ok, noPermitidos, err = ValidarEntregaWriteSet(especID, []string{"db/foo.go", "cmd/api.go"})
	if err != nil {
		t.Fatalf("caso no permitido: error inesperado: %v", err)
	}
	if ok {
		t.Fatalf("caso no permitido: esperaba ok=false")
	}
	if len(noPermitidos) != 1 || noPermitidos[0] != "cmd/api.go" {
		t.Fatalf("caso no permitido: lista incorrecta: %v", noPermitidos)
	}

	// Spec inexistente → error
	_, _, err = ValidarEntregaWriteSet(9999, []string{"db/foo.go"})
	if err == nil {
		t.Fatal("spec inexistente: esperaba error, got nil")
	}
	if !strings.Contains(err.Error(), "no encontrada") {
		t.Fatalf("spec inexistente: error inesperado: %v", err)
	}
}
