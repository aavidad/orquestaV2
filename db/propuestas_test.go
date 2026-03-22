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

func abrirDBTemporalPropuestas(t *testing.T) {
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

func TestReabrirPropuestaReiniciaVotosPendientes(t *testing.T) {
	abrirDBTemporalPropuestas(t)

	if err := RetirarAgente("claude"); err != nil {
		t.Fatalf("RetirarAgente claude: %v", err)
	}
	if err := RetirarAgente("antigravity"); err != nil {
		t.Fatalf("RetirarAgente antigravity: %v", err)
	}
	if err := RegistrarAgente("codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente codex1: %v", err)
	}
	if err := RegistrarAgente("codex2", "programador"); err != nil {
		t.Fatalf("RegistrarAgente codex2: %v", err)
	}

	id, err := CrearPropuesta(&Propuesta{
		Codigo:       "OP-901",
		Titulo:       "Propuesta de prueba",
		Descripcion:  "Reabrir y reiniciar votos",
		Tipo:         "arquitectura",
		PropuestoPor: "alberto",
		Distribuidor: "claude",
	})
	if err != nil {
		t.Fatalf("CrearPropuesta: %v", err)
	}

	if consenso, err := Votar(id, "codex1", VotoAcuerdo, "ok"); err != nil {
		t.Fatalf("Votar codex1: %v", err)
	} else if consenso {
		t.Fatalf("no deberia haber consenso aun")
	}
	if consenso, err := Votar(id, "codex2", VotoAcuerdo, "ok"); err != nil {
		t.Fatalf("Votar codex2: %v", err)
	} else if !consenso {
		t.Fatalf("se esperaba consenso")
	}

	p, err := GetPropuesta("OP-901")
	if err != nil {
		t.Fatalf("GetPropuesta: %v", err)
	}
	if p.Estado != PropuestaConsenso {
		t.Fatalf("estado inesperado antes de reabrir: %s", p.Estado)
	}

	reparados, err := ReabrirPropuesta("OP-901", "alberto")
	if err != nil {
		t.Fatalf("ReabrirPropuesta: %v", err)
	}
	if reparados != 2 {
		t.Fatalf("se esperaban 2 votos reiniciados, got %d", reparados)
	}

	p, err = GetPropuesta("OP-901")
	if err != nil {
		t.Fatalf("GetPropuesta reopen: %v", err)
	}
	if p.Estado != PropuestaAbierta {
		t.Fatalf("estado inesperado tras reabrir: %s", p.Estado)
	}
	if p.CerradaAt != nil {
		t.Fatalf("cerrada_at deberia quedar nulo")
	}

	votos, err := VotosDePropuesta(id)
	if err != nil {
		t.Fatalf("VotosDePropuesta: %v", err)
	}
	if len(votos) != 2 {
		t.Fatalf("se esperaban 2 votos: %+v", votos)
	}
	for _, v := range votos {
		if v.Posicion != VotoPendiente {
			t.Fatalf("voto no reiniciado: %+v", v)
		}
		if v.Comentario != "" {
			t.Fatalf("comentario no limpiado: %+v", v)
		}
	}
}

func TestRepararVotosPendientesSoloRellenaLosFaltantes(t *testing.T) {
	abrirDBTemporalPropuestas(t)

	if err := RetirarAgente("claude"); err != nil {
		t.Fatalf("RetirarAgente claude: %v", err)
	}
	if err := RetirarAgente("antigravity"); err != nil {
		t.Fatalf("RetirarAgente antigravity: %v", err)
	}
	if err := RegistrarAgente("codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente codex1: %v", err)
	}
	if err := RegistrarAgente("codex2", "programador"); err != nil {
		t.Fatalf("RegistrarAgente codex2: %v", err)
	}

	id, err := CrearPropuesta(&Propuesta{
		Codigo:       "OP-902",
		Titulo:       "Propuesta de prueba 2",
		Descripcion:  "Reparar faltantes",
		Tipo:         "arquitectura",
		PropuestoPor: "alberto",
		Distribuidor: "claude",
	})
	if err != nil {
		t.Fatalf("CrearPropuesta: %v", err)
	}

	if consenso, err := Votar(id, "codex1", VotoAcuerdo, "ok"); err != nil {
		t.Fatalf("Votar codex1: %v", err)
	} else if consenso {
		t.Fatalf("no deberia haber consenso aun")
	}

	if _, err := DB.Exec(`DELETE FROM votos WHERE propuesta_id=? AND agente=?`, id, "codex2"); err != nil {
		t.Fatalf("borrar voto codex2: %v", err)
	}

	reparados, err := RepararVotosPendientes(id, "alberto", false)
	if err != nil {
		t.Fatalf("RepararVotosPendientes: %v", err)
	}
	if reparados != 1 {
		t.Fatalf("se esperaba 1 voto reparado, got %d", reparados)
	}

	votos, err := VotosDePropuesta(id)
	if err != nil {
		t.Fatalf("VotosDePropuesta: %v", err)
	}
	if len(votos) != 2 {
		t.Fatalf("se esperaban 2 votos: %+v", votos)
	}
	for _, v := range votos {
		switch v.Agente {
		case "codex1":
			if v.Posicion != VotoAcuerdo {
				t.Fatalf("codex1 no deberia haberse tocado: %+v", v)
			}
		case "codex2":
			if v.Posicion != VotoPendiente {
				t.Fatalf("codex2 deberia ser pendiente: %+v", v)
			}
		default:
			t.Fatalf("agente inesperado: %+v", v)
		}
	}
}
