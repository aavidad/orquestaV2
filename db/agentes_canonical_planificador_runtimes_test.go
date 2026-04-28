package db

import (
	"path/filepath"
	"testing"
	"time"
)

func TestTareaHuerfanaRecuperableCanonicalizaAliasCodex(t *testing.T) {
	tmp := prepararDBTemporal(t)

	for _, nombre := range []string{"Codex71", "codex71"} {
		if err := RegistrarAgente(nombre, "programador"); err != nil {
			t.Fatalf("RegistrarAgente %s: %v", nombre, err)
		}
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orphan-canon",
		Nombre:  "Orphan Canon",
		RutaAbs: filepath.Join(tmp, "orphan-canon"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	mustInsertID(t, `INSERT INTO sesiones (
		agente, proyecto_id, activa, estado, herramienta, host, inicio, heartbeat_at
	) VALUES (?,?,?,?,?,?,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		"Codex71", proyectoID, 0, "pausada", "codex-cli", "test-host")

	ok, err := tareaHuerfanaRecuperable("codex71", proyectoID)
	if err != nil {
		t.Fatalf("tareaHuerfanaRecuperable: %v", err)
	}
	if ok {
		t.Fatalf("no deberia considerar recuperable una tarea con sesion pausada protegida del alias canonico")
	}
}

func TestTareaHuerfanaRecuperableNoLiberaAgenteReservadoAutonomia(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex75", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orphan-reserved",
		Nombre:  "Orphan Reserved",
		RutaAbs: filepath.Join(tmp, "orphan-reserved"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	if _, err := UpsertProyectoAutonomia(&ProyectoAutonomia{
		ProyectoID:        proyectoID,
		Enabled:           true,
		SupervisorAgente:  "Codex75",
		ReserveSupervisor: true,
		EstadoAutonomia:   AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("UpsertProyectoAutonomia: %v", err)
	}

	ok, err := tareaHuerfanaRecuperable("Codex75", proyectoID)
	if err != nil {
		t.Fatalf("tareaHuerfanaRecuperable: %v", err)
	}
	if ok {
		t.Fatalf("un agente reservado de autonomia no deberia pasar por recuperacion huerfana generica")
	}
}

func TestAgenteTieneSesionActivaEnPoolLocalCompartidoCanonicalizaAliasCodex(t *testing.T) {
	prepararDBTemporal(t)

	for _, nombre := range []string{"Codex72", "codex72"} {
		if err := RegistrarAgente(nombre, "programador"); err != nil {
			t.Fatalf("RegistrarAgente %s: %v", nombre, err)
		}
	}
	poolID, err := GuardarPool(&PoolCapacidad{
		Slug:             "pool-canon-codex72",
		Proveedor:        "OpenAI",
		Runtime:          "codex",
		Plan:             "premium",
		CapacidadTotal:   1,
		PoliticaHandoff:  "preventivo",
		FuenteTelemetria: "manual",
		Activo:           true,
	})
	if err != nil {
		t.Fatalf("GuardarPool: %v", err)
	}
	sesionID, err := IniciarSesion("Codex72")
	if err != nil {
		t.Fatalf("IniciarSesion: %v", err)
	}
	if _, err := DB.Exec(`UPDATE sesiones SET pool_id=? WHERE id=?`, poolID, sesionID); err != nil {
		t.Fatalf("UPDATE sesiones pool_id: %v", err)
	}

	ok, err := agenteTieneSesionActivaEnPoolLocalCompartido("codex72", poolID)
	if err != nil {
		t.Fatalf("agenteTieneSesionActivaEnPoolLocalCompartido: %v", err)
	}
	if !ok {
		t.Fatalf("deberia detectar sesion activa en pool por alias canonico")
	}
}

func TestListarRuntimesCanonicalizaAliasCodex(t *testing.T) {
	prepararDBTemporal(t)

	for _, nombre := range []string{"Codex73", "codex73"} {
		if err := RegistrarAgente(nombre, "programador"); err != nil {
			t.Fatalf("RegistrarAgente %s: %v", nombre, err)
		}
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "runtimes-canon",
		Nombre:  "Runtimes Canon",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	if _, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "Codex73",
		ProyectoID:   &proyectoID,
		LogicalState: "esperando_io",
		ProcessState: "running",
	}); err != nil {
		t.Fatalf("RegistrarRuntimeInstance: %v", err)
	}

	agente := "codex73"
	items, err := ListarRuntimes(FiltroRuntimes{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("ListarRuntimes: %v", err)
	}
	if len(items) != 1 || items[0] == nil || items[0].Agente != "Codex73" {
		t.Fatalf("runtimes inesperadas: %+v", items)
	}
}

func TestRuntimePrincipalAgenteCanonicalizaAliasCodex(t *testing.T) {
	prepararDBTemporal(t)

	for _, nombre := range []string{"Codex74", "codex74"} {
		if err := RegistrarAgente(nombre, "programador"); err != nil {
			t.Fatalf("RegistrarAgente %s: %v", nombre, err)
		}
	}
	runtimeID, err := RegistrarRuntimeInstance(&RuntimeInstance{
		Agente:       "Codex74",
		LogicalState: "esperando_io",
		ProcessState: "running",
	})
	if err != nil {
		t.Fatalf("RegistrarRuntimeInstance: %v", err)
	}
	now := time.Now().UTC()
	if _, err := DB.Exec(`UPDATE runtime_instances SET last_event_at=?, last_heartbeat_at=?, updated_at=? WHERE id=?`, now, now, now, runtimeID); err != nil {
		t.Fatalf("UPDATE runtime_instances: %v", err)
	}

	runtime, err := RuntimePrincipalAgente("codex74")
	if err != nil {
		t.Fatalf("RuntimePrincipalAgente: %v", err)
	}
	if runtime == nil || runtime.ID != runtimeID || runtime.Agente != "Codex74" {
		t.Fatalf("runtime inesperada: %+v", runtime)
	}
}
