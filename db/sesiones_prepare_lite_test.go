package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetAgentePrepareLiteUsaReadOnlySQLiteIndependienteDelHandlePrincipal(t *testing.T) {
	Close()
	path := filepath.Join(t.TempDir(), "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv ORQUESTA_DB: %v", err)
	}
	defer func() {
		if prev == "" {
			_ = os.Unsetenv("ORQUESTA_DB")
		} else {
			_ = os.Setenv("ORQUESTA_DB", prev)
		}
		Close()
	}()
	if err := Open(); err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := RegistrarAgente("QwenCoder1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	Close()
	agente, err := GetAgentePrepareLite("QwenCoder1")
	if err != nil {
		t.Fatalf("GetAgentePrepareLite: %v", err)
	}
	if agente == nil {
		t.Fatalf("GetAgentePrepareLite devolvio nil")
	}
	if agente.Nombre != "QwenCoder1" {
		t.Fatalf("nombre inesperado: %q", agente.Nombre)
	}
	if agente.Rol != "programador" {
		t.Fatalf("rol inesperado: %q", agente.Rol)
	}
}

func TestGetAgentePrepareLiteCanonicalizaAliasCodex(t *testing.T) {
	prepararDBTemporal(t)
	for _, nombre := range []string{"Codex97", "codex97"} {
		if err := RegistrarAgente(nombre, "programador"); err != nil {
			t.Fatalf("RegistrarAgente %s: %v", nombre, err)
		}
	}
	agente, err := GetAgentePrepareLite("codex97")
	if err != nil {
		t.Fatalf("GetAgentePrepareLite: %v", err)
	}
	if agente == nil || agente.Nombre != "Codex97" {
		t.Fatalf("agente inesperado: %+v", agente)
	}
}
