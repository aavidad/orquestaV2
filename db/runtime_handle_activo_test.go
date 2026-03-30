package db

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestGetRuntimeHandleActivoAgenteProyectoIgnoraFantasmaMasReciente(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	runtimeVivoID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "Codex1",
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          int64PtrTest(int64(os.Getpid())),
	})
	if err != nil {
		t.Fatalf("crear runtime vivo: %v", err)
	}
	resVivo, err := DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?)`,
		"Codex1", proyectoID, runtimeVivoID, "cli", "process", strconv.Itoa(os.Getpid()), time.Now().UTC().Add(-5*time.Minute))
	if err != nil {
		t.Fatalf("insert handle vivo: %v", err)
	}
	handleVivoID, _ := resVivo.LastInsertId()

	pidFantasma := int64(99999991)
	runtimeFantasmaID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "Codex1",
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
		PID:          &pidFantasma,
	})
	if err != nil {
		t.Fatalf("crear runtime fantasma: %v", err)
	}
	resFantasma, err := DB.Exec(`INSERT INTO runtime_handles (
		agente, proyecto_id, runtime_id, transporte, handle_kind, handle_ref, estado, last_seen_at
	) VALUES (?,?,?,?,?,?, 'activo', ?)`,
		"Codex1", proyectoID, runtimeFantasmaID, "cli", "process", strconv.FormatInt(pidFantasma, 10), time.Now().UTC())
	if err != nil {
		t.Fatalf("insert handle fantasma: %v", err)
	}
	handleFantasmaID, _ := resFantasma.LastInsertId()

	handle, err := GetRuntimeHandleActivoAgenteProyecto("Codex1", &proyectoID)
	if err != nil {
		t.Fatalf("get handle activo: %v", err)
	}
	if handle == nil || handle.ID != handleVivoID || handle.Estado != "activo" {
		t.Fatalf("deberia recuperar el handle vivo y no el fantasma reciente: %+v", handle)
	}

	fantasma, err := GetRuntimeHandle(handleFantasmaID)
	if err != nil {
		t.Fatalf("get handle fantasma: %v", err)
	}
	if fantasma == nil || fantasma.Estado != "fallido" {
		t.Fatalf("el handle fantasma reciente deberia quedar fallido: %+v", fantasma)
	}
}
