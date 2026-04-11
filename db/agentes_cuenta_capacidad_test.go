package db

import (
	"testing"
	"time"
)

func TestCuentaClaveAgentePrefiereAccountIDSobreEmail(t *testing.T) {
	agente := &Agente{
		Nombre:        "Codex2",
		CuentaID:      "acc-codex2",
		CuentaEmail:   "shared@example.com",
		CuentaUsuario: "codex2",
	}
	if got := CuentaClaveAgente(agente); got != "acc-codex2" {
		t.Fatalf("la clave de cuenta debe preferir account_id, got=%q", got)
	}
}

func TestAgenteOcupaCapacidadCuentaIgnoraOrdenVivaStaleSinRuntimeNiSesion(t *testing.T) {
	prepararDBTemporal(t)
	if err := ConfigSet("runtime_shared_account_order_hold_seconds", "300"); err != nil {
		t.Fatalf("ConfigSet hold: %v", err)
	}
	if err := RegistrarAgente("Codex11", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	now := time.Now().UTC()
	if err := UpsertAgenteIdentidadObservadaCanonica("Codex11", "acc-shared", "shared@example.com", "Codex11", "test", &now); err != nil {
		t.Fatalf("UpsertAgenteIdentidadObservadaCanonica: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_cuota='activo' WHERE nombre=?`, "Codex11"); err != nil {
		t.Fatalf("update estado cuota: %v", err)
	}
	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{Agente: "Codex11", Tipo: "handoff"})
	if err != nil {
		t.Fatalf("EncolarRuntimeOrder: %v", err)
	}
	stale := time.Now().UTC().Add(-2 * time.Hour)
	if _, err := DB.Exec(`UPDATE runtime_orders SET created_at=?, updated_at=?, available_at=? WHERE id=?`, stale, stale, stale, orderID); err != nil {
		t.Fatalf("stale order: %v", err)
	}
	resetRuntimeOrdersHotIndex()
	ocupa, err := agenteOcupaCapacidadCuenta("Codex11")
	if err != nil {
		t.Fatalf("agenteOcupaCapacidadCuenta: %v", err)
	}
	if ocupa {
		t.Fatalf("una orden viva pero stale sin runtime/sesion no debe ocupar capacidad")
	}
}

func TestCuentaCompartidaPermiteActivacionSiElOtroSoloTieneOrdenStale(t *testing.T) {
	prepararDBTemporal(t)
	if err := ConfigSet("runtime_shared_account_active_ceiling", "1"); err != nil {
		t.Fatalf("ConfigSet ceiling: %v", err)
	}
	if err := ConfigSet("runtime_shared_account_order_hold_seconds", "300"); err != nil {
		t.Fatalf("ConfigSet hold: %v", err)
	}
	for _, nombre := range []string{"Codex10", "Codex11"} {
		if err := RegistrarAgente(nombre, "programador"); err != nil {
			t.Fatalf("RegistrarAgente %s: %v", nombre, err)
		}
		now := time.Now().UTC()
		if err := UpsertAgenteIdentidadObservadaCanonica(nombre, "acc-shared", "shared@example.com", nombre, "test", &now); err != nil {
			t.Fatalf("UpsertAgenteIdentidadObservadaCanonica %s: %v", nombre, err)
		}
		if _, err := DB.Exec(`UPDATE agentes SET estado_cuota='activo' WHERE nombre=?`, nombre); err != nil {
			t.Fatalf("update estado cuota %s: %v", nombre, err)
		}
	}
	orderID, err := EncolarRuntimeOrder(&RuntimeOrder{Agente: "Codex11", Tipo: "handoff"})
	if err != nil {
		t.Fatalf("EncolarRuntimeOrder: %v", err)
	}
	stale := time.Now().UTC().Add(-2 * time.Hour)
	if _, err := DB.Exec(`UPDATE runtime_orders SET created_at=?, updated_at=?, available_at=? WHERE id=?`, stale, stale, stale, orderID); err != nil {
		t.Fatalf("stale order: %v", err)
	}
	resetRuntimeOrdersHotIndex()
	ok, ocupadoPor, err := CuentaCompartidaPermiteActivacionAgente("Codex10")
	if err != nil {
		t.Fatalf("CuentaCompartidaPermiteActivacionAgente: %v", err)
	}
	if !ok {
		t.Fatalf("no deberia bloquear por orden stale, ocupadoPor=%q", ocupadoPor)
	}
}

func TestCuentaCompartidaIgnoraSesionActivaNoOperativa(t *testing.T) {
	prepararDBTemporal(t)
	if err := ConfigSet("runtime_shared_account_active_ceiling", "1"); err != nil {
		t.Fatalf("ConfigSet ceiling: %v", err)
	}
	for _, nombre := range []string{"Codex10", "Codex11"} {
		if err := RegistrarAgente(nombre, "programador"); err != nil {
			t.Fatalf("RegistrarAgente %s: %v", nombre, err)
		}
		now := time.Now().UTC()
		if err := UpsertAgenteIdentidadObservadaCanonica(nombre, "acc-shared", "shared@example.com", nombre, "test", &now); err != nil {
			t.Fatalf("UpsertAgenteIdentidadObservadaCanonica %s: %v", nombre, err)
		}
		if _, err := DB.Exec(`UPDATE agentes SET estado_cuota='activo' WHERE nombre=?`, nombre); err != nil {
			t.Fatalf("update estado cuota %s: %v", nombre, err)
		}
	}
	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex11",
		CWD:         "/tmp/orquesta-codex11",
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("IniciarSesionContexto: %v", err)
	}
	if _, err := DB.Exec(`DELETE FROM runtime_handles WHERE agente=?`, "Codex11"); err != nil {
		t.Fatalf("delete runtime_handles: %v", err)
	}
	if _, err := DB.Exec(`DELETE FROM runtime_instances WHERE agente=?`, "Codex11"); err != nil {
		t.Fatalf("delete runtime_instances: %v", err)
	}
	runtimeHandleHotReset()
	ok, ocupadoPor, err := CuentaCompartidaPermiteActivacionAgente("Codex10")
	if err != nil {
		t.Fatalf("CuentaCompartidaPermiteActivacionAgente: %v", err)
	}
	if !ok {
		t.Fatalf("una sesion activa sin runtime operativo no debe bloquear, ocupadoPor=%q", ocupadoPor)
	}
}
