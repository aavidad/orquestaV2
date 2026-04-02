package db

import "testing"

func TestGetAgenteUsaIdentidadObservadaComoFallback(t *testing.T) {
	prepararDBTemporal(t)
	if err := RegistrarAgente("Codex7", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if err := UpsertAgenteIdentidadObservada("Codex7", "berserk@avidad.com", "berserk", "manual_observed_identity", nil); err != nil {
		t.Fatalf("upsert identidad observada: %v", err)
	}
	agente, err := GetAgente("Codex7")
	if err != nil {
		t.Fatalf("get agente: %v", err)
	}
	if agente.CuentaEmail != "berserk@avidad.com" {
		t.Fatalf("cuenta email inesperada: %+v", agente)
	}
	if agente.CuentaUsuario != "berserk" {
		t.Fatalf("cuenta usuario inesperada: %+v", agente)
	}
	if agente.CuentaFuente != "manual_observed_identity" {
		t.Fatalf("fuente inesperada: %+v", agente)
	}
}
