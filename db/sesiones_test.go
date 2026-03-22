package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIniciarSesionDevuelveIDPersistido(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	defer func() {
		_ = os.Setenv("ORQUESTA_DB", prev)
		Close()
	}()

	if err := Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}

	if err := RegistrarAgente("codex-sesion", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}

	id, err := IniciarSesion("codex-sesion")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	if id == 0 {
		t.Fatalf("id de sesion no valido: %d", id)
	}

	var persistedID int64
	if err := DB.QueryRow(`
		SELECT id
		FROM sesiones
		WHERE agente = ? AND activa = 1
		ORDER BY id DESC
		LIMIT 1`, "codex-sesion").Scan(&persistedID); err != nil {
		t.Fatalf("select sesion activa: %v", err)
	}
	if persistedID != id {
		t.Fatalf("id devuelto %d distinto del persistido %d", id, persistedID)
	}
}
