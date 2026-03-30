package db

import (
	"strings"
	"testing"
)

func TestVerificarPersistenciaActualSQLiteSana(t *testing.T) {
	prepararDBTemporal(t)

	informe, err := VerificarPersistenciaActual()
	if err != nil {
		t.Fatalf("VerificarPersistenciaActual: %v", err)
	}
	if informe == nil {
		t.Fatalf("informe nil")
	}
	if !informe.Sano {
		t.Fatalf("la verificacion no deberia marcar error: %+v", informe.Comprobaciones)
	}
	if informe.Driver != "sqlite" {
		t.Fatalf("driver inesperado: %s", informe.Driver)
	}
	if strings.TrimSpace(informe.Target) == "" {
		t.Fatalf("target vacio")
	}
	if len(informe.Comprobaciones) < 3 {
		t.Fatalf("faltan comprobaciones: %+v", informe.Comprobaciones)
	}
}
