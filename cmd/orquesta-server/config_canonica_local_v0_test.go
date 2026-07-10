package main

import (
	"os"
	"path/filepath"
	"testing"
)

// La config canonica local minima del repo (orquesta.config.json en la raiz)
// debe cargar con el loader real: sin secretos, sin Telegram, sin remoto.
func TestOrquestaConfigCanonicaLocalCargaV0(t *testing.T) {
	path := filepath.Join("..", "..", "orquesta.config.json")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("config canonica local no presente: %v", err)
	}

	config, ok, err := loadServerProjectConfigPathV0(path)
	if err != nil {
		t.Fatalf("loadServerProjectConfigPathV0: %v", err)
	}
	if !ok || config.SchemaVersion != serverProjectConfigSchemaVersionV0 {
		t.Fatalf("config canonica invalida: ok=%v config=%+v", ok, config)
	}
	if config.TelegramOperator.Token != nil {
		t.Fatalf("config canonica local no debe llevar token de Telegram")
	}
	if config.ControlPlane.RemoteAccessOptIn == nil || *config.ControlPlane.RemoteAccessOptIn {
		t.Fatalf("config canonica local debe declarar remote_access_opt_in=false: %+v", config.ControlPlane)
	}
}
