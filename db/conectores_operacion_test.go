package db

import (
	"path/filepath"
	"testing"
	"time"
)

func TestConectorOperacionAbreYCierraCircuitoPorCooldown(t *testing.T) {
	abrirDBTemporalMemoria(t)

	conectorID, err := UpsertConector(&Conector{
		Slug:       "codex-remote",
		Nombre:     "Codex Remote",
		Transporte: "api",
		Comando:    "http://localhost:9999",
		Activo:     true,
	})
	if err != nil {
		t.Fatalf("UpsertConector: %v", err)
	}
	if err := ConfigSet("connector_circuit_breaker_threshold", "2"); err != nil {
		t.Fatalf("ConfigSet threshold: %v", err)
	}
	if err := ConfigSet("connector_circuit_breaker_cooldown_seconds", "60"); err != nil {
		t.Fatalf("ConfigSet cooldown: %v", err)
	}

	if _, err := RegistrarFalloConector(conectorID, "adapter down 1"); err != nil {
		t.Fatalf("RegistrarFalloConector 1: %v", err)
	}
	disponible, op, err := ConectorDisponibleParaArranque(conectorID)
	if err != nil {
		t.Fatalf("ConectorDisponibleParaArranque 1: %v", err)
	}
	if !disponible || op.EstadoOperativo != ConectorOperativoActivo {
		t.Fatalf("el conector no deberia abrir circuito aun: %+v disponible=%v", op, disponible)
	}

	if _, err := RegistrarFalloConector(conectorID, "adapter down 2"); err != nil {
		t.Fatalf("RegistrarFalloConector 2: %v", err)
	}
	disponible, op, err = ConectorDisponibleParaArranque(conectorID)
	if err != nil {
		t.Fatalf("ConectorDisponibleParaArranque 2: %v", err)
	}
	if disponible || op.EstadoOperativo != ConectorOperativoCircuitoAbierto {
		t.Fatalf("el conector deberia tener circuito abierto: %+v disponible=%v", op, disponible)
	}

	if _, err := DB.Exec(`UPDATE conectores_operacion SET cooldown_until = ? WHERE conector_id = ?`, time.Now().UTC().Add(-time.Minute), conectorID); err != nil {
		t.Fatalf("forzar cooldown expirado: %v", err)
	}
	disponible, op, err = ConectorDisponibleParaArranque(conectorID)
	if err != nil {
		t.Fatalf("ConectorDisponibleParaArranque 3: %v", err)
	}
	if !disponible || op.EstadoOperativo != ConectorOperativoActivo || op.FallosConsecutivos != 0 {
		t.Fatalf("el conector deberia reabrirse tras el cooldown: %+v disponible=%v", op, disponible)
	}
}

func TestProyectoDisponibleParaAutonomiaRespetaCircuitoAbiertoDeConector(t *testing.T) {
	abrirDBTemporalMemoria(t)

	conectorID, err := UpsertConector(&Conector{
		Slug:       "codex-remote",
		Nombre:     "Codex Remote",
		Transporte: "api",
		Comando:    "http://localhost:9999",
		Activo:     true,
	})
	if err != nil {
		t.Fatalf("UpsertConector: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(t.TempDir(), "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	cooldown := time.Now().UTC().Add(time.Minute)
	if err := UpsertConectorOperacion(&ConectorOperacion{
		ConectorID:         conectorID,
		EstadoOperativo:    ConectorOperativoCircuitoAbierto,
		Motivo:             "remote_connector_unavailable",
		FallosConsecutivos: 3,
		CooldownUntil:      &cooldown,
	}); err != nil {
		t.Fatalf("UpsertConectorOperacion: %v", err)
	}
	if err := MarcarProyectoBloqueadoExterno(proyectoID, "conector:codex-remote:circuito_abierto"); err != nil {
		t.Fatalf("MarcarProyectoBloqueadoExterno: %v", err)
	}

	disponible, err := ProyectoDisponibleParaAutonomia(proyectoID)
	if err != nil {
		t.Fatalf("ProyectoDisponibleParaAutonomia circuito abierto: %v", err)
	}
	if disponible {
		t.Fatalf("el proyecto no deberia estar disponible con el circuito abierto")
	}

	if _, err := DB.Exec(`UPDATE conectores_operacion SET cooldown_until = ? WHERE conector_id = ?`, time.Now().UTC().Add(-time.Minute), conectorID); err != nil {
		t.Fatalf("forzar cooldown expirado: %v", err)
	}
	disponible, err = ProyectoDisponibleParaAutonomia(proyectoID)
	if err != nil {
		t.Fatalf("ProyectoDisponibleParaAutonomia tras cooldown: %v", err)
	}
	if !disponible {
		t.Fatalf("el proyecto deberia reactivarse cuando el circuito expira")
	}
	op, err := GetProyectoOperacion(proyectoID)
	if err != nil {
		t.Fatalf("GetProyectoOperacion: %v", err)
	}
	if op.EstadoOperativo != ProyectoOperativoActivo {
		t.Fatalf("estado operativo inesperado tras reactivar: %+v", op)
	}
}
