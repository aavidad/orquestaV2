package db

import (
	"testing"
	"time"
)

// setHeartbeatStale fuerza el heartbeat de una sesión a un tiempo muy antiguo
// para que sea detectada como candidata a handoff.
func setHeartbeatStale(t *testing.T, sesionID int64) {
	t.Helper()
	old := time.Now().UTC().Add(-2 * time.Hour).Format("2006-01-02 15:04:05")
	if _, err := DB.Exec(`UPDATE sesiones SET heartbeat_at=? WHERE id=?`, old, sesionID); err != nil {
		t.Fatalf("setHeartbeatStale: %v", err)
	}
}

// prepararAgenteConTareaEnProgreso crea un agente con sesión activa y una tarea en progreso.
func prepararAgenteConTareaEnProgreso(t *testing.T, nombre string) (sesionID int64, tareaID int64) {
	t.Helper()
	if err := RegistrarAgente(nombre, "programador"); err != nil {
		t.Fatalf("RegistrarAgente %s: %v", nombre, err)
	}
	sesionID, err := IniciarSesion(nombre)
	if err != nil {
		t.Fatalf("IniciarSesion %s: %v", nombre, err)
	}
	tareaID, err = CrearTarea(&Tarea{
		Titulo:    "Tarea de " + nombre,
		Modulo:    "orquestador",
		Prioridad: PrioridadMedia,
		CreadoPor: "alberto",
	})
	if err != nil {
		t.Fatalf("CrearTarea %s: %v", nombre, err)
	}
	if err := TomarTarea(tareaID, nombre); err != nil {
		t.Fatalf("TomarTarea %s: %v", nombre, err)
	}
	if err := IniciarTarea(tareaID, nombre); err != nil {
		t.Fatalf("IniciarTarea %s: %v", nombre, err)
	}
	return sesionID, tareaID
}

func TestDetectarAgentesAgotadosDevuelveAgentesConHeartbeatAntiguo(t *testing.T) {
	prepararDBTemporal(t)

	sesionID, _ := prepararAgenteConTareaEnProgreso(t, "Codex1")
	setHeartbeatStale(t, sesionID)

	candidatos, err := DetectarAgentesAgotados()
	if err != nil {
		t.Fatalf("DetectarAgentesAgotados: %v", err)
	}
	if len(candidatos) != 1 {
		t.Fatalf("esperaba 1 candidato, got=%d", len(candidatos))
	}
	if candidatos[0].Agente != "Codex1" {
		t.Fatalf("agente inesperado: %s", candidatos[0].Agente)
	}
	if candidatos[0].TareaID == nil {
		t.Fatalf("tarea_id nula en candidato")
	}
	if candidatos[0].SesionID == nil {
		t.Fatalf("sesion_id nula en candidato")
	}
}

func TestDetectarAgentesAgotadosIgnoraHeartbeatReciente(t *testing.T) {
	prepararDBTemporal(t)

	// heartbeat_at por defecto es CURRENT_TIMESTAMP → reciente
	_, _ = prepararAgenteConTareaEnProgreso(t, "Codex1")

	candidatos, err := DetectarAgentesAgotados()
	if err != nil {
		t.Fatalf("DetectarAgentesAgotados: %v", err)
	}
	if len(candidatos) != 0 {
		t.Fatalf("no esperaba candidatos con heartbeat reciente, got=%d", len(candidatos))
	}
}

func TestDetectarAgentesAgotadosExcluyeConHandoffPendiente(t *testing.T) {
	prepararDBTemporal(t)

	sesionID, tareaID := prepararAgenteConTareaEnProgreso(t, "Codex1")
	setHeartbeatStale(t, sesionID)

	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	if _, err := IniciarSesion("Codex2"); err != nil {
		t.Fatalf("IniciarSesion Codex2: %v", err)
	}
	// Crear una orden de handoff pendiente para Codex1
	if _, err := CrearHandoffAgenteVivo("Codex1", "Codex2", &tareaID, "test", "continuar", ""); err != nil {
		t.Fatalf("CrearHandoffAgenteVivo: %v", err)
	}

	candidatos, err := DetectarAgentesAgotados()
	if err != nil {
		t.Fatalf("DetectarAgentesAgotados: %v", err)
	}
	// Codex1 ya tiene handoff pendiente → no debe aparecer
	for _, c := range candidatos {
		if c.Agente == "Codex1" {
			t.Fatalf("Codex1 no debería aparecer: ya tiene handoff pendiente")
		}
	}
}

func TestSeleccionarAgenteReemplazoDevuelveLibre(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	// Codex2 libre (activo=0, habilitado=1 por defecto)

	nombre, err := SeleccionarAgenteReemplazo("Codex1", nil)
	if err != nil {
		t.Fatalf("SeleccionarAgenteReemplazo: %v", err)
	}
	if nombre != "Codex2" {
		t.Fatalf("agente inesperado: %s", nombre)
	}
}

func TestSeleccionarAgenteReemplazoSinDisponibles(t *testing.T) {
	prepararDBTemporal(t)

	// Deshabilitar todos los agentes programadores pre-seeded
	if _, err := DB.Exec(`UPDATE agentes SET habilitado=0 WHERE rol='programador'`); err != nil {
		t.Fatalf("deshabilitar agentes: %v", err)
	}
	if err := RegistrarAgente("SoloAgente", "programador"); err != nil {
		t.Fatalf("registrar SoloAgente: %v", err)
	}

	_, err := SeleccionarAgenteReemplazo("SoloAgente", nil)
	if err == nil {
		t.Fatalf("se esperaba error por no haber reemplazo disponible")
	}
}

func TestSeleccionarAgenteReemplazoExcluyeAgentesOcupados(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	if err := RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar Codex3: %v", err)
	}

	nombre, err := SeleccionarAgenteReemplazo("Codex1", []string{"Codex2"})
	if err != nil {
		t.Fatalf("SeleccionarAgenteReemplazo: %v", err)
	}
	if nombre != "Codex3" {
		t.Fatalf("agente inesperado: %s", nombre)
	}
}

func TestGuardarCheckpointHandoffCreaRegistro(t *testing.T) {
	prepararDBTemporal(t)

	sesionID, tareaID := prepararAgenteConTareaEnProgreso(t, "Codex1")
	c := &HandoffCandidato{
		Agente:      "Codex1",
		TareaID:     &tareaID,
		SesionID:    &sesionID,
		Inactividad: 2 * time.Hour,
		Motivo:      "heartbeat hace 120 min (umbral: 30 min)",
	}

	checkpointID, err := GuardarCheckpointHandoff(c, "")
	if err != nil {
		t.Fatalf("GuardarCheckpointHandoff: %v", err)
	}
	if checkpointID == 0 {
		t.Fatalf("checkpoint id inesperado: %d", checkpointID)
	}
}

func TestGuardarCheckpointHandoffFallaSinSesion(t *testing.T) {
	prepararDBTemporal(t)

	c := &HandoffCandidato{
		Agente:  "Codex1",
		Motivo:  "test",
		SesionID: nil,
	}
	if _, err := GuardarCheckpointHandoff(c, ""); err == nil {
		t.Fatalf("se esperaba error por SesionID nil")
	}
}

func TestProcesarHandoffsBatchSinCandidatos(t *testing.T) {
	prepararDBTemporal(t)

	n, err := ProcesarHandoffsBatch()
	if err != nil {
		t.Fatalf("ProcesarHandoffsBatch: %v", err)
	}
	if n != 0 {
		t.Fatalf("esperaba 0 handoffs, got=%d", n)
	}
}

func TestProcesarHandoffsBatchEjecutaHandoffCompleto(t *testing.T) {
	prepararDBTemporal(t)

	sesionID, _ := prepararAgenteConTareaEnProgreso(t, "Codex1")
	setHeartbeatStale(t, sesionID)

	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	// Codex2 libre (no tiene sesión activa)

	n, err := ProcesarHandoffsBatch()
	if err != nil {
		t.Fatalf("ProcesarHandoffsBatch: %v", err)
	}
	if n != 1 {
		t.Fatalf("esperaba 1 handoff, got=%d", n)
	}

	// Verificar que Codex2 tiene una orden de handoff pendiente
	agente := "Codex2"
	estado := "pendiente"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, Estado: &estado})
	if err != nil {
		t.Fatalf("ListarRuntimeOrders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "handoff" {
		t.Fatalf("orden de handoff no encontrada para Codex2")
	}
}
