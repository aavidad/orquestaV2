package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"orquesta/db"
)

func TestMemoriaGuardarListarYVer(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if proyectoID == 0 {
		t.Fatalf("proyecto id inesperado")
	}

	if err := memoriaGuardarCmd.Flags().Set("proyecto", "orquestador"); err != nil {
		t.Fatalf("set proyecto guardar: %v", err)
	}
	if err := memoriaGuardarCmd.Flags().Set("valor", `{"version":"v2","grpc":true}`); err != nil {
		t.Fatalf("set valor guardar: %v", err)
	}
	if err := memoriaGuardarCmd.Flags().Set("metadata", `{"fuente":"manual"}`); err != nil {
		t.Fatalf("set metadata guardar: %v", err)
	}
	if err := memoriaGuardarCmd.Flags().Set("verificado-por", "Codex1"); err != nil {
		t.Fatalf("set verificado-por guardar: %v", err)
	}
	t.Cleanup(func() {
		_ = memoriaGuardarCmd.Flags().Set("proyecto", "")
		_ = memoriaGuardarCmd.Flags().Set("valor", "")
		_ = memoriaGuardarCmd.Flags().Set("metadata", "{}")
		_ = memoriaGuardarCmd.Flags().Set("verificado-por", "")
	})

	outGuardar := capturarStdout(t, func() {
		if err := memoriaGuardarCmd.RunE(memoriaGuardarCmd, []string{"Core_API", "api"}); err != nil {
			t.Fatalf("memoria guardar: %v", err)
		}
	})
	if !strings.Contains(outGuardar, "Entidad de memoria #") {
		t.Fatalf("salida guardar inesperada:\n%s", outGuardar)
	}

	if err := memoriaListarCmd.Flags().Set("proyecto", "orquestador"); err != nil {
		t.Fatalf("set proyecto listar: %v", err)
	}
	t.Cleanup(func() {
		_ = memoriaListarCmd.Flags().Set("proyecto", "")
	})
	outListar := capturarStdout(t, func() {
		if err := memoriaListarCmd.RunE(memoriaListarCmd, nil); err != nil {
			t.Fatalf("memoria listar: %v", err)
		}
	})
	if !strings.Contains(outListar, "Core_API") || !strings.Contains(outListar, `"version":"v2"`) {
		t.Fatalf("salida listar inesperada:\n%s", outListar)
	}

	if err := memoriaVerCmd.Flags().Set("proyecto", "orquestador"); err != nil {
		t.Fatalf("set proyecto ver: %v", err)
	}
	t.Cleanup(func() {
		_ = memoriaVerCmd.Flags().Set("proyecto", "")
	})
	outVer := capturarStdout(t, func() {
		if err := memoriaVerCmd.RunE(memoriaVerCmd, []string{"Core_API"}); err != nil {
			t.Fatalf("memoria ver: %v", err)
		}
	})
	if !strings.Contains(outVer, "Entidad:      Core_API") || !strings.Contains(outVer, "Codex1") {
		t.Fatalf("salida ver inesperada:\n%s", outVer)
	}
}
