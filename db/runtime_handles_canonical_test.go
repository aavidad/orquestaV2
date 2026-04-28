package db

import (
	"testing"
	"time"
)

func TestListarRuntimeHandlesPasivosCanonicalizaAliasCodex(t *testing.T) {
	prepararDBTemporal(t)

	for _, nombre := range []string{"Codex93", "codex93"} {
		if err := RegistrarAgente(nombre, "programador"); err != nil {
			t.Fatalf("RegistrarAgente %s: %v", nombre, err)
		}
	}
	handleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, transporte, handle_kind, handle_ref, estado, created_at, updated_at, last_seen_at
	) VALUES (?,?,?,?,?,?,?,?)`,
		"Codex93", "tmux", "session", "runtime:93", "activo",
		time.Now().UTC(), time.Now().UTC(), time.Now().UTC(),
	)

	agente := "codex93"
	items, err := ListarRuntimeHandlesPasivos(&agente)
	if err != nil {
		t.Fatalf("ListarRuntimeHandlesPasivos: %v", err)
	}
	if len(items) != 1 || items[0] == nil || items[0].ID != handleID || items[0].Agente != "Codex93" {
		t.Fatalf("handles inesperados: %+v", items)
	}
}

func TestGetRuntimeHandleActivoAgenteProyectoCanonicalizaAliasCodex(t *testing.T) {
	prepararDBTemporal(t)

	for _, nombre := range []string{"Codex96", "codex96"} {
		if err := RegistrarAgente(nombre, "programador"); err != nil {
			t.Fatalf("RegistrarAgente %s: %v", nombre, err)
		}
	}
	proyectoID := mustInsertID(t, `
		INSERT INTO proyectos (slug, nombre, ruta_abs, tipo, activo)
		VALUES (?,?,?,?,1)`,
		"orquestador-canon-runtime", "Orquestador Canon Runtime", t.TempDir(), "repo",
	)
	runtimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "Codex96",
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
	})
	if err != nil {
		t.Fatalf("RegistrarRuntimeInstance: %v", err)
	}
	handleID := mustInsertID(t, `
		INSERT INTO runtime_handles (
			agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, created_at, updated_at, last_seen_at
		) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		"Codex96", proyectoID, runtimeID, "tmux", "session", "runtime:96", "activo",
		time.Now().UTC(), time.Now().UTC(), time.Now().UTC(),
	)

	handle, err := GetRuntimeHandleActivoAgenteProyecto("codex96", &proyectoID)
	if err != nil {
		t.Fatalf("GetRuntimeHandleActivoAgenteProyecto: %v", err)
	}
	if handle == nil || handle.ID != handleID || handle.Agente != "Codex96" {
		t.Fatalf("handle activo inesperado: %+v", handle)
	}
}

func TestListarRuntimeHandlesIdentidadCuentaCanonicalizaAliasCodex(t *testing.T) {
	prepararDBTemporal(t)

	for _, nombre := range []string{"Codex97", "codex97"} {
		if err := RegistrarAgente(nombre, "programador"); err != nil {
			t.Fatalf("RegistrarAgente %s: %v", nombre, err)
		}
	}
	handleID := mustInsertID(t, `INSERT INTO runtime_handles (
		agente, transporte, handle_kind, handle_ref, estado, created_at, updated_at, last_seen_at
	) VALUES (?,?,?,?,?,?,?,?)`,
		"Codex97", "tmux", "session", "runtime:97", "activo",
		time.Now().UTC(), time.Now().UTC(), time.Now().UTC(),
	)

	agente := "codex97"
	items, err := listarRuntimeHandlesIdentidadCuenta(&agente)
	if err != nil {
		t.Fatalf("listarRuntimeHandlesIdentidadCuenta: %v", err)
	}
	if len(items) != 1 || items[0] == nil || items[0].ID != handleID || items[0].Agente != "Codex97" {
		t.Fatalf("handles identidad inesperados: %+v", items)
	}
}
