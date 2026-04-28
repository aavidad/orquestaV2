package db

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orquesta/internal/controlruntime"
)

func TestProcesarHigieneRuntimesAutonomosBatchIncluyePurgaHistorica(t *testing.T) {
	prepararDBTemporal(t)

	prevCommand := runtimeTMUXCommandPathFn
	prevList := runtimeTMUXListSessionsFn
	prevKill := runtimeTMUXKillSessionFn
	t.Cleanup(func() {
		runtimeTMUXCommandPathFn = prevCommand
		runtimeTMUXListSessionsFn = prevList
		runtimeTMUXKillSessionFn = prevKill
	})
	runtimeTMUXCommandPathFn = func() (string, error) { return "", nil }
	runtimeTMUXListSessionsFn = func(string) ([]controlruntime.TMUXSessionInfo, error) { return nil, nil }
	runtimeTMUXKillSessionFn = func(string, string) error { return nil }

	if err := ConfigSet("runtime_orders_retention_minutes", "30"); err != nil {
		t.Fatalf("config runtime_orders_retention_minutes: %v", err)
	}

	res, err := DB.Exec(`INSERT INTO runtime_orders
		(agente, proyecto_id, tipo, payload_json, estado, created_at, updated_at, finished_at)
		VALUES ('Codex1', 1, 'nudge', '{}', 'completada',
		        datetime('now','-2 hours'),
		        datetime('now','-2 hours'),
		        datetime('now','-2 hours'))`)
	if err != nil {
		t.Fatalf("insert runtime order historica: %v", err)
	}
	orderID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last insert runtime order: %v", err)
	}

	n, err := ProcesarHigieneRuntimesAutonomosBatch()
	if err != nil {
		t.Fatalf("procesar higiene runtimes autonomos: %v", err)
	}
	if n < 1 {
		t.Fatalf("la higiene deberia contar al menos una purga historica, got=%d", n)
	}

	var count int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM runtime_orders WHERE id=?`, orderID).Scan(&count); err != nil {
		t.Fatalf("count runtime order: %v", err)
	}
	if count != 0 {
		t.Fatalf("la runtime order historica deberia purgarse, count=%d", count)
	}
}

func TestProcesarHigieneRuntimesAutonomosBatchEncolaStopParaHandleSupervisadoSinTrabajo(t *testing.T) {
	tmp := prepararDBTemporal(t)

	prevCommand := runtimeTMUXCommandPathFn
	prevList := runtimeTMUXListSessionsFn
	prevKill := runtimeTMUXKillSessionFn
	t.Cleanup(func() {
		runtimeTMUXCommandPathFn = prevCommand
		runtimeTMUXListSessionsFn = prevList
		runtimeTMUXKillSessionFn = prevKill
	})
	runtimeTMUXCommandPathFn = func() (string, error) { return "", nil }
	runtimeTMUXListSessionsFn = func(string) ([]controlruntime.TMUXSessionInfo, error) { return nil, nil }
	runtimeTMUXKillSessionFn = func(string, string) error { return nil }

	if err := RegistrarAgente("CodexPg1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "demo",
		Nombre:  "Demo",
		RutaAbs: filepath.Join(tmp, "demo"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:            "CodexPg1",
		ProyectoID:        &proyectoID,
		CWD:               filepath.Join(tmp, "demo"),
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-higiene-idle",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("handle: %+v err=%v", handle, err)
	}
	oldSeen := time.Now().UTC().Add(-10 * time.Minute)
	metaJSON := `{"driver":"tmux_cli_session","transport":"tmux","supervisor_driver":"local_runtime_supervisor","supervisor_owner_pid":1234}`
	if _, err := DB.Exec(`UPDATE runtime_handles SET estado='activo', metadata_json=?, created_at=?, last_seen_at=?, updated_at=? WHERE id=?`,
		metaJSON, oldSeen, oldSeen, oldSeen, handle.ID); err != nil {
		t.Fatalf("update handle: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_instances SET logical_state='activo', process_state='running', updated_at=? WHERE id=?`,
		oldSeen, handle.RuntimeID); err != nil {
		t.Fatalf("update runtime: %v", err)
	}

	n, err := ProcesarHigieneRuntimesAutonomosBatch()
	if err != nil {
		t.Fatalf("procesar higiene runtimes autonomos: %v", err)
	}
	if n < 1 {
		t.Fatalf("deberia encolar al menos un stop por higiene sin trabajo, got=%d", n)
	}

	estado := "pendiente"
	agente := "CodexPg1"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	found := false
	for _, order := range orders {
		if order != nil && order.Tipo == "stop" && strings.Contains(order.PayloadJSON, "higiene_autonoma:sin_trabajo_pendiente") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("deberia existir stop por higiene sin trabajo, got=%+v", orders)
	}
}

func TestProcesarHigieneRuntimesAutonomosBatchDesactivaAgenteActivoSinVida(t *testing.T) {
	prepararDBTemporal(t)

	prevCommand := runtimeTMUXCommandPathFn
	prevList := runtimeTMUXListSessionsFn
	prevKill := runtimeTMUXKillSessionFn
	t.Cleanup(func() {
		runtimeTMUXCommandPathFn = prevCommand
		runtimeTMUXListSessionsFn = prevList
		runtimeTMUXKillSessionFn = prevKill
	})
	runtimeTMUXCommandPathFn = func() (string, error) { return "", nil }
	runtimeTMUXListSessionsFn = func(string) ([]controlruntime.TMUXSessionInfo, error) { return nil, nil }
	runtimeTMUXKillSessionFn = func(string, string) error { return nil }

	if err := RegistrarAgente("CodexBudget", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET activo=1, estado_sesion='pensando' WHERE nombre='CodexBudget'`); err != nil {
		t.Fatalf("activar agente zombie: %v", err)
	}

	n, err := ProcesarHigieneRuntimesAutonomosBatch()
	if err != nil {
		t.Fatalf("procesar higiene runtimes autonomos: %v", err)
	}
	if n < 1 {
		t.Fatalf("deberia procesar al menos la desactivacion del agente zombie, got=%d", n)
	}

	agente, err := GetAgente("CodexBudget")
	if err != nil {
		t.Fatalf("get agente: %v", err)
	}
	if agente.Activo {
		t.Fatalf("el agente zombie deberia quedar inactivo: %+v", agente)
	}
	if strings.EqualFold(strings.TrimSpace(agente.EstadoSesion), "pensando") {
		t.Fatalf("estado_sesion inesperado: %+v", agente)
	}
}
