package cmd

import (
	"testing"

	"orquesta/db"
)

func TestAgenteCuentaComoConectadoRespetaEstadoCuotaVisible(t *testing.T) {
	if agenteCuentaComoConectado(nil) {
		t.Fatalf("nil no deberia contar como conectado")
	}
	if agenteCuentaComoConectado(&db.Agente{Nombre: "Codex1", Activo: false, EstadoCuota: "activo"}) {
		t.Fatalf("un agente sin sesion activa no deberia contar como conectado")
	}
	if agenteCuentaComoConectado(&db.Agente{Nombre: "Codex2", Activo: true, EstadoCuota: "agotado"}) {
		t.Fatalf("un agente agotado no deberia contar como conectado visible")
	}
	if agenteCuentaComoConectado(&db.Agente{Nombre: "Codex3", Activo: true, EstadoCuota: "enfriamiento"}) {
		t.Fatalf("un agente en enfriamiento no deberia contar como conectado visible")
	}
	if !agenteCuentaComoConectado(&db.Agente{Nombre: "Codex4", Activo: true, EstadoCuota: "activo"}) {
		t.Fatalf("un agente activo con cuota activa deberia contar como conectado")
	}
}
