/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import "testing"

func TestConsensoNoCierraConSoloUnNoAutorAunqueTodoEsteEnAcuerdo(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("autor", "programador"); err != nil {
		t.Fatalf("registrar autor: %v", err)
	}
	if err := RegistrarAgente("revisor1", "programador"); err != nil {
		t.Fatalf("registrar revisor1: %v", err)
	}

	propuesta := &Propuesta{
		Titulo:       "Arquitectura de prueba",
		Descripcion:  "Validar consenso minimo",
		Tipo:         "arquitectura",
		PropuestoPor: "autor",
		Distribuidor: "codex",
	}
	_, err := CrearPropuesta(propuesta)
	if err != nil {
		t.Fatalf("crear propuesta: %v", err)
	}

	p, err := GetPropuesta(propuesta.Codigo)
	if err != nil {
		t.Fatalf("get propuesta: %v", err)
	}

	if _, err := Votar(p.ID, "autor", VotoAcuerdo, "de acuerdo"); err != nil {
		t.Fatalf("votar autor: %v", err)
	}
	if consenso, err := Votar(p.ID, "revisor1", VotoAcuerdo, "de acuerdo"); err != nil {
		t.Fatalf("votar revisor1: %v", err)
	} else if consenso {
		t.Fatalf("no deberia cerrar consenso con un solo no autor")
	}

	actualizada, err := GetPropuesta(propuesta.Codigo)
	if err != nil {
		t.Fatalf("get propuesta actualizada: %v", err)
	}
	if actualizada.Estado != PropuestaAbierta {
		t.Fatalf("estado inesperado: %s", actualizada.Estado)
	}
}

func TestConsensoCierraConDosNoAutoresYSinPendientes(t *testing.T) {
	prepararDBTemporal(t)

	for _, agente := range []string{"autor", "revisor1", "revisor2"} {
		if err := RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("registrar %s: %v", agente, err)
		}
	}

	propuesta := &Propuesta{
		Titulo:       "Arquitectura aprobable",
		Descripcion:  "Debe cerrar con dos revisores",
		Tipo:         "arquitectura",
		PropuestoPor: "autor",
		Distribuidor: "codex",
	}
	_, err := CrearPropuesta(propuesta)
	if err != nil {
		t.Fatalf("crear propuesta: %v", err)
	}

	p, err := GetPropuesta(propuesta.Codigo)
	if err != nil {
		t.Fatalf("get propuesta: %v", err)
	}

	if _, err := Votar(p.ID, "autor", VotoAcuerdo, "de acuerdo"); err != nil {
		t.Fatalf("votar autor: %v", err)
	}
	if _, err := Votar(p.ID, "revisor1", VotoAcuerdo, "de acuerdo"); err != nil {
		t.Fatalf("votar revisor1: %v", err)
	}
	consenso, err := Votar(p.ID, "revisor2", VotoAcuerdo, "de acuerdo")
	if err != nil {
		t.Fatalf("votar revisor2: %v", err)
	}
	if !consenso {
		t.Fatalf("deberia cerrar consenso con dos no autores y sin pendientes")
	}

	actualizada, err := GetPropuesta(propuesta.Codigo)
	if err != nil {
		t.Fatalf("get propuesta actualizada: %v", err)
	}
	if actualizada.Estado != PropuestaConsenso {
		t.Fatalf("estado inesperado: %s", actualizada.Estado)
	}
}

func TestCerrarPropuestaConsensoFallaSinMinimoDeNoAutores(t *testing.T) {
	prepararDBTemporal(t)

	if err := RegistrarAgente("autor", "programador"); err != nil {
		t.Fatalf("registrar autor: %v", err)
	}
	if err := RegistrarAgente("revisor1", "programador"); err != nil {
		t.Fatalf("registrar revisor1: %v", err)
	}

	propuesta := &Propuesta{
		Titulo:       "Cierre manual insuficiente",
		Descripcion:  "No debe poder cerrarse en consenso",
		Tipo:         "arquitectura",
		PropuestoPor: "autor",
		Distribuidor: "codex",
	}
	_, err := CrearPropuesta(propuesta)
	if err != nil {
		t.Fatalf("crear propuesta: %v", err)
	}

	p, err := GetPropuesta(propuesta.Codigo)
	if err != nil {
		t.Fatalf("get propuesta: %v", err)
	}

	if _, err := Votar(p.ID, "autor", VotoAcuerdo, "de acuerdo"); err != nil {
		t.Fatalf("votar autor: %v", err)
	}
	if _, err := Votar(p.ID, "revisor1", VotoAcuerdo, "de acuerdo"); err != nil {
		t.Fatalf("votar revisor1: %v", err)
	}

	if err := CerrarPropuesta(propuesta.Codigo, string(PropuestaConsenso), "alberto"); err == nil {
		t.Fatalf("esperaba error al cerrar consenso sin dos no autores")
	}
}

func TestReabrirPropuestaReconstruyeVotosPendientes(t *testing.T) {
	prepararDBTemporal(t)

	for _, agente := range []string{"autor", "revisor1", "revisor2"} {
		if err := RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("registrar %s: %v", agente, err)
		}
	}

	propuesta := &Propuesta{
		Titulo:       "Reabrir propuesta",
		Descripcion:  "Debe reconstruir votos pendientes faltantes",
		Tipo:         "implementacion",
		PropuestoPor: "autor",
		Distribuidor: "codex",
	}
	if _, err := CrearPropuesta(propuesta); err != nil {
		t.Fatalf("crear propuesta: %v", err)
	}

	p, err := GetPropuesta(propuesta.Codigo)
	if err != nil {
		t.Fatalf("get propuesta: %v", err)
	}

	if _, err := DB.Exec(`DELETE FROM votos WHERE propuesta_id = ? AND agente = ?`, p.ID, "revisor2"); err != nil {
		t.Fatalf("delete voto faltante: %v", err)
	}
	if err := CerrarPropuesta(propuesta.Codigo, string(PropuestaBacklog), "alberto"); err != nil {
		t.Fatalf("cerrar propuesta: %v", err)
	}

	insertados, err := ReabrirPropuesta(propuesta.Codigo, "alberto")
	if err != nil {
		t.Fatalf("reabrir propuesta: %v", err)
	}
	if insertados != 1 {
		t.Fatalf("insertados esperados=1 obtenidos=%d", insertados)
	}

	actualizada, err := GetPropuesta(propuesta.Codigo)
	if err != nil {
		t.Fatalf("get propuesta actualizada: %v", err)
	}
	if actualizada.Estado != PropuestaAbierta {
		t.Fatalf("estado inesperado: %s", actualizada.Estado)
	}
	if actualizada.CerradaAt != nil {
		t.Fatalf("cerrada_at deberia ser nil tras reabrir")
	}

	var posicion string
	if err := DB.QueryRow(`SELECT posicion FROM votos WHERE propuesta_id = ? AND agente = ?`, p.ID, "revisor2").Scan(&posicion); err != nil {
		t.Fatalf("get voto reinsertado: %v", err)
	}
	if posicion != string(VotoPendiente) {
		t.Fatalf("posicion inesperada: %s", posicion)
	}
}

func TestRepararVotosPendientesInsertaSoloFaltantes(t *testing.T) {
	prepararDBTemporal(t)

	for _, agente := range []string{"autor", "revisor1", "revisor2"} {
		if err := RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("registrar %s: %v", agente, err)
		}
	}

	propuesta := &Propuesta{
		Titulo:       "Reparar votos",
		Descripcion:  "Debe insertar solo votos faltantes",
		Tipo:         "implementacion",
		PropuestoPor: "autor",
		Distribuidor: "codex",
	}
	if _, err := CrearPropuesta(propuesta); err != nil {
		t.Fatalf("crear propuesta: %v", err)
	}

	p, err := GetPropuesta(propuesta.Codigo)
	if err != nil {
		t.Fatalf("get propuesta: %v", err)
	}

	if _, err := DB.Exec(`DELETE FROM votos WHERE propuesta_id = ? AND agente = ?`, p.ID, "revisor1"); err != nil {
		t.Fatalf("delete voto faltante: %v", err)
	}

	insertados, err := RepararVotosPendientesPropuesta(propuesta.Codigo, "alberto")
	if err != nil {
		t.Fatalf("reparar votos: %v", err)
	}
	if insertados != 1 {
		t.Fatalf("insertados esperados=1 obtenidos=%d", insertados)
	}

	var total int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM votos WHERE propuesta_id = ? AND agente = ?`, p.ID, "revisor1").Scan(&total); err != nil {
		t.Fatalf("contar voto reparado: %v", err)
	}
	if total != 1 {
		t.Fatalf("conteo inesperado tras reparar: %d", total)
	}
}
