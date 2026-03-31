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

func TestVerificarPersistenciaActualSQLiteFallaSiSaleDeWAL(t *testing.T) {
	prepararDBTemporal(t)

	if _, err := DB.Exec(`PRAGMA journal_mode = DELETE`); err != nil {
		t.Fatalf("cambiar journal_mode a DELETE: %v", err)
	}

	informe, err := VerificarPersistenciaActual()
	if err != nil {
		t.Fatalf("VerificarPersistenciaActual: %v", err)
	}
	if informe == nil {
		t.Fatalf("informe nil")
	}
	if informe.Sano {
		t.Fatalf("la verificacion deberia marcar error si journal_mode no es wal: %+v", informe.Comprobaciones)
	}
	encontro := false
	for _, comprobacion := range informe.Comprobaciones {
		if comprobacion.Nombre == "config_sqlite" && comprobacion.Estado == "error" {
			encontro = true
			if !strings.Contains(strings.ToLower(comprobacion.Detalle), "journal_mode=delete") {
				t.Fatalf("detalle inesperado: %q", comprobacion.Detalle)
			}
		}
	}
	if !encontro {
		t.Fatalf("faltó la comprobacion de config_sqlite en error: %+v", informe.Comprobaciones)
	}
}
