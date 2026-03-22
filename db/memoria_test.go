/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package db

import (
	"os"
	"path/filepath"
	"testing"
)

func abrirDBTemporalMemoria(t *testing.T) {
	t.Helper()

	if DB != nil {
		Close()
	}

	prev := os.Getenv("ORQUESTA_DB")
	ruta := filepath.Join(t.TempDir(), "orquesta.db")
	if err := os.Setenv("ORQUESTA_DB", ruta); err != nil {
		t.Fatalf("setenv ORQUESTA_DB: %v", err)
	}
	t.Cleanup(func() {
		Close()
		if prev == "" {
			_ = os.Unsetenv("ORQUESTA_DB")
			return
		}
		_ = os.Setenv("ORQUESTA_DB", prev)
	})

	if err := Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
}

func TestGuardarYLeerMemoriaProyecto(t *testing.T) {
	abrirDBTemporalMemoria(t)

	id, err := GuardarMemoriaProyecto(&MemoriaProyecto{
		Proyecto:          "orquestador",
		Resumen:           "Resumen inicial",
		Contexto:          "Contexto vivo",
		PreguntasAbiertas: "Pendiente definir puertos",
		ActualizadoPor:    "Codex3",
	})
	if err != nil {
		t.Fatalf("GuardarMemoriaProyecto: %v", err)
	}
	if id == 0 {
		t.Fatalf("id inesperado: %d", id)
	}

	memoria, err := GetMemoriaProyecto("orquestador")
	if err != nil {
		t.Fatalf("GetMemoriaProyecto: %v", err)
	}
	if memoria.Resumen != "Resumen inicial" {
		t.Fatalf("resumen inesperado: %s", memoria.Resumen)
	}
	if memoria.Contexto != "Contexto vivo" {
		t.Fatalf("contexto inesperado: %s", memoria.Contexto)
	}
	if memoria.PreguntasAbiertas != "Pendiente definir puertos" {
		t.Fatalf("preguntas inesperadas: %s", memoria.PreguntasAbiertas)
	}

	lista, err := ListarMemoriaProyectos()
	if err != nil {
		t.Fatalf("ListarMemoriaProyectos: %v", err)
	}
	if len(lista) != 1 || lista[0].Proyecto != "orquestador" {
		t.Fatalf("lista inesperada: %+v", lista)
	}
}

func TestRegistrarFuentesHallazgosYDerivas(t *testing.T) {
	abrirDBTemporalMemoria(t)

	fuenteID, err := RegistrarFuenteMemoria(&FuenteMemoria{
		Proyecto:      "orquestador",
		Tipo:          "documentacion",
		Referencia:    "ARQUITECTURA.md",
		Titulo:        "Arquitectura objetivo",
		Confianza:     "alta",
		RegistradoPor: "Codex3",
	})
	if err != nil {
		t.Fatalf("RegistrarFuenteMemoria: %v", err)
	}

	hallazgoID, err := RegistrarHallazgoMemoria(&HallazgoMemoria{
		Proyecto:      "orquestador",
		FuenteID:      &fuenteID,
		Tipo:          "hecho",
		Titulo:        "La memoria aun no tiene modelo propio",
		Descripcion:   "Solo existe documentacion y logica dispersa",
		Impacto:       "alto",
		Confianza:     "alta",
		RegistradoPor: "Codex3",
	})
	if err != nil {
		t.Fatalf("RegistrarHallazgoMemoria: %v", err)
	}

	derivaID, err := RegistrarDerivaMemoria(&DerivaMemoria{
		Proyecto:     "orquestador",
		HallazgoID:   &hallazgoID,
		Tipo:         "arquitectonica",
		Severidad:    "media",
		Estado:       DerivaAbierta,
		Descripcion:  "La arquitectura objetivo documenta puertos, pero el codigo sigue centrado en db/",
		Evidencia:    "db/schema.go y db/*.go concentran logica y persistencia",
		DetectadaPor: "Codex3",
	})
	if err != nil {
		t.Fatalf("RegistrarDerivaMemoria: %v", err)
	}
	if derivaID == 0 {
		t.Fatalf("deriva id inesperado: %d", derivaID)
	}

	fuentes, err := ListarFuentesMemoria("orquestador")
	if err != nil {
		t.Fatalf("ListarFuentesMemoria: %v", err)
	}
	if len(fuentes) != 1 || fuentes[0].ID != fuenteID {
		t.Fatalf("fuentes inesperadas: %+v", fuentes)
	}

	hallazgos, err := ListarHallazgosMemoria("orquestador")
	if err != nil {
		t.Fatalf("ListarHallazgosMemoria: %v", err)
	}
	if len(hallazgos) != 1 || hallazgos[0].ID != hallazgoID {
		t.Fatalf("hallazgos inesperados: %+v", hallazgos)
	}
	if hallazgos[0].FuenteID == nil || *hallazgos[0].FuenteID != fuenteID {
		t.Fatalf("fuente relacionada inesperada: %+v", hallazgos[0].FuenteID)
	}

	derivas, err := ListarDerivasMemoria("orquestador", true)
	if err != nil {
		t.Fatalf("ListarDerivasMemoria abiertas: %v", err)
	}
	if len(derivas) != 1 || derivas[0].ID != derivaID {
		t.Fatalf("derivas inesperadas: %+v", derivas)
	}

	if err := ResolverDerivaMemoria(derivaID, "Codex3", string(DerivaResuelta), "Primera version del bloque implementada"); err != nil {
		t.Fatalf("ResolverDerivaMemoria: %v", err)
	}

	derivasAbiertas, err := ListarDerivasMemoria("orquestador", true)
	if err != nil {
		t.Fatalf("ListarDerivasMemoria tras resolver: %v", err)
	}
	if len(derivasAbiertas) != 0 {
		t.Fatalf("esperaba 0 derivas abiertas, hay %d", len(derivasAbiertas))
	}

	fuentesN, hallazgosN, derivasN, abiertasN, err := ContarMemoriaProyecto("orquestador")
	if err != nil {
		t.Fatalf("ContarMemoriaProyecto: %v", err)
	}
	if fuentesN != 1 || hallazgosN != 1 || derivasN != 1 || abiertasN != 0 {
		t.Fatalf("conteos inesperados: fuentes=%d hallazgos=%d derivas=%d abiertas=%d", fuentesN, hallazgosN, derivasN, abiertasN)
	}
}
