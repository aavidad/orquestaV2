package db

import (
	"strings"
	"testing"
)

func TestDetectarCarenciaSkillEncuentraEquivalente(t *testing.T) {
	prepararDBTemporal(t)
	if _, err := CrearSkill("Codex1", &Skill{
		TipoAgente:       "programador",
		Nombre:           "rg",
		Descripcion:      "busqueda rapida",
		CuandoUsar:       "buscar texto",
		Escenario:        "investigacion",
		Prioridad:        10,
		AliasesJSON:      `["ripgrep"]`,
		HerramientasJSON: `["rg"]`,
		Activa:           true,
	}); err != nil {
		t.Fatalf("CrearSkill: %v", err)
	}

	resultado, err := DetectarCarenciaSkill(&SolicitudDeteccionSkill{
		TipoAgente:       "programador",
		Nombre:           "rip-grep",
		Descripcion:      "buscar rapido",
		CuandoUsar:       "buscar texto",
		Escenario:        "investigacion",
		HerramientasJSON: `["rg"]`,
	})
	if err != nil {
		t.Fatalf("DetectarCarenciaSkill: %v", err)
	}
	if resultado.Falta {
		t.Fatalf("no deberia marcar falta: %+v", resultado)
	}
	if resultado.Equivalente == nil || resultado.Equivalente.Nombre != "rg" {
		t.Fatalf("equivalente inesperada: %+v", resultado.Equivalente)
	}
	if resultado.InvocacionCreador != "" {
		t.Fatalf("no deberia proponer skill creator si ya existe equivalente")
	}
}

func TestDetectarCarenciaSkillProponeBorradorYCandidatas(t *testing.T) {
	prepararDBTemporal(t)
	if _, err := CrearSkill("Codex1", &Skill{
		TipoAgente:       "programador",
		Nombre:           "gofmt",
		Descripcion:      "formatear codigo go",
		CuandoUsar:       "formatear archivos go",
		Escenario:        "codigo",
		Prioridad:        5,
		HerramientasJSON: `["gofmt"]`,
		Activa:           true,
	}); err != nil {
		t.Fatalf("CrearSkill gofmt: %v", err)
	}
	if _, err := CrearSkill("Codex1", &Skill{
		TipoAgente:       "programador",
		Nombre:           "golangci-lint",
		Descripcion:      "analisis estatico",
		CuandoUsar:       "revisar errores de codigo",
		Escenario:        "codigo",
		Prioridad:        8,
		HerramientasJSON: `["golangci-lint"]`,
		Activa:           true,
	}); err != nil {
		t.Fatalf("CrearSkill golangci-lint: %v", err)
	}

	resultado, err := DetectarCarenciaSkill(&SolicitudDeteccionSkill{
		TipoAgente:       "programador",
		Nombre:           "goimports",
		Descripcion:      "ordenar imports y formatear go",
		CuandoUsar:       "corregir imports y formato en codigo go",
		Escenario:        "codigo",
		HerramientasJSON: `["goimports"]`,
	})
	if err != nil {
		t.Fatalf("DetectarCarenciaSkill: %v", err)
	}
	if !resultado.Falta {
		t.Fatalf("deberia marcar skill faltante: %+v", resultado)
	}
	if resultado.Borrador == nil || resultado.Borrador.Nombre != "goimports" {
		t.Fatalf("borrador inesperado: %+v", resultado.Borrador)
	}
	if len(resultado.Candidatas) == 0 {
		t.Fatalf("deberia proponer candidatas cercanas")
	}
	if !strings.Contains(resultado.InvocacionCreador, "$skill-creator") {
		t.Fatalf("deberia preparar invocacion al skill creator: %s", resultado.InvocacionCreador)
	}
	if len(resultado.RecursosSugeridos) < 2 {
		t.Fatalf("deberia sugerir recursos por capas: %+v", resultado.RecursosSugeridos)
	}
}
