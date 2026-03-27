package db

import (
	"database/sql"
	"testing"
)

func TestFusionarAgentesMueveReferenciasYResuelveColisionDeVotos(t *testing.T) {
	prepararDBTemporal(t)
	if err := RegistrarAgente("codex1", "programador"); err != nil {
		t.Fatalf("registrar codex1: %v", err)
	}
	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}

	tareaID, err := CrearTarea(&Tarea{
		Titulo:    "Fusion de agentes",
		Modulo:    "orquestador",
		Prioridad: PrioridadAlta,
		CreadoPor: "codex1",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := TomarTarea(tareaID, "codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}

	propuestaID, err := CrearPropuesta(&Propuesta{
		Codigo:       "OP-998",
		Titulo:       "Fusion test",
		Tipo:         "arquitectura",
		PropuestoPor: "codex1",
		Distribuidor: "claude",
	})
	if err != nil {
		t.Fatalf("crear propuesta: %v", err)
	}
	if _, err := Votar(propuestaID, "codex1", VotoAcuerdo, "origen"); err != nil {
		t.Fatalf("votar origen: %v", err)
	}
	if _, err := Votar(propuestaID, "Codex1", VotoDesacuerdo, "destino"); err != nil {
		t.Fatalf("votar destino: %v", err)
	}
	Audit("codex1", "nota_test", "sistema", 0, "audit")

	resultado, err := FusionarAgentes("codex1", "Codex1")
	if err != nil {
		t.Fatalf("FusionarAgentes: %v", err)
	}
	if resultado.VotosDescartados != 1 {
		t.Fatalf("votos descartados inesperados: %d", resultado.VotosDescartados)
	}
	if _, err := GetAgente("codex1"); err == nil {
		t.Fatalf("el agente origen deberia haberse eliminado")
	} else if err != sql.ErrNoRows {
		t.Fatalf("get agente origen: %v", err)
	}

	tarea, err := GetTarea(1)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Agente == nil || *tarea.Agente != "Codex1" {
		t.Fatalf("agente de tarea inesperado: %+v", tarea.Agente)
	}
	if tarea.CreadoPor != "Codex1" {
		t.Fatalf("creado_por inesperado: %s", tarea.CreadoPor)
	}

	propuesta, err := GetPropuesta("OP-998")
	if err != nil {
		t.Fatalf("get propuesta: %v", err)
	}
	if propuesta.PropuestoPor != "Codex1" {
		t.Fatalf("propuesto_por inesperado: %s", propuesta.PropuestoPor)
	}

	var votos int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM votos WHERE propuesta_id = ? AND agente = ?`, propuestaID, "Codex1").Scan(&votos); err != nil {
		t.Fatalf("count votos destino: %v", err)
	}
	if votos != 1 {
		t.Fatalf("conteo de votos destino inesperado: %d", votos)
	}
	var auditorias int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM audit_log WHERE agente = ?`, "codex1").Scan(&auditorias); err != nil {
		t.Fatalf("count audit origen: %v", err)
	}
	if auditorias != 0 {
		t.Fatalf("quedaron auditorias con origen antiguo: %d", auditorias)
	}
}

func TestFusionarAgentesFallaSiOrigenTieneSesionActiva(t *testing.T) {
	prepararDBTemporal(t)
	if err := RegistrarAgente("codex2", "programador"); err != nil {
		t.Fatalf("registrar codex2: %v", err)
	}
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	if _, err := IniciarSesion("codex2"); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}

	if _, err := FusionarAgentes("codex2", "Codex2"); err == nil {
		t.Fatalf("se esperaba error por sesion activa")
	}
}
