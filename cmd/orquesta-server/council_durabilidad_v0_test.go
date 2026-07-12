package main

import (
	"os"
	"path/filepath"
	"testing"

	orquestamcp "orquesta/modulos/orquesta-mcp"
)

// Una decision que se pierde al reiniciar no es una decision, es una opinion. El
// recibo tiene que sobrevivir al proceso, y reconvocar el mismo council_ref debe
// devolver LO DECIDIDO, no un veredicto nuevo que podria contradecir al anterior.
func TestCouncilDecisionEsDurableEIdempotenteV0(t *testing.T) {
	stack := buildCanonicalMCPBootstrapStackForTestV0(t)
	handler, err := buildServerAppHandlerV0(stack)
	if err != nil {
		t.Fatalf("buildServerAppHandlerV0: %v", err)
	}
	server := newLocalHTTPServerForTestV0(t, handler)
	t.Cleanup(server.Close)
	stateDir := os.Getenv(envServerStateDirV0)

	decidir := func(councilRef string, votos ...string) orquestamcp.MCPCouncilToolResultV0 {
		miembros := []string{"sobrado", "justito", "medio"}
		ballots := make([]map[string]any, 0, len(miembros))
		for idx, miembro := range miembros {
			ballots = append(ballots, map[string]any{"member_ref": miembro, "vote": votos[idx]})
		}
		return llamarCouncilV0(t, server.URL, map[string]any{
			"action": orquestamcp.MCPCouncilActionDecideV0, "council_ref": councilRef,
			"author_ref": "autor", "members": miembrosConsejoParaTestV0(),
			"ballots": ballots,
		})
	}

	primera := decidir("consejo-durable", "approve", "approve", "approve")
	if primera.Outcome != "council_decision_accepted" {
		t.Fatalf("tres aprobaciones deben aceptar: %+v", primera)
	}

	// El recibo esta en disco: sobrevive al proceso.
	recibo := filepath.Join(stateDir, councilReceiptsDirNameV0, "consejo-durable.json")
	if _, err := os.Stat(recibo); err != nil {
		t.Fatalf("la decision no quedo en disco: %v", err)
	}

	// Idempotencia: reconvocar el MISMO consejo con votos distintos NO cambia el
	// veredicto. Lo decidido, decidido esta.
	segunda := decidir("consejo-durable", "rework", "rework", "rework")
	if segunda.Outcome != "council_decision_accepted" {
		t.Fatalf("reconvocar no puede reabrir una decision tomada: %+v", segunda)
	}

	// Y un consejo distinto sigue decidiendo de cero.
	// Un solo voto a favor de tres no llega a dos tercios.
	otro := decidir("consejo-nuevo", "approve", "rework", "rework")
	if otro.Outcome != "council_decision_rework" {
		t.Fatalf("un consejo nuevo debe decidir de cero: %+v", otro)
	}
}

// El council_ref viene de fuera: no puede escribir fuera de su directorio.
func TestCouncilRefNoEscapaDelDirectorioDeRecibosV0(t *testing.T) {
	store, err := newCouncilReceiptStoreV0(t.TempDir())
	if err != nil {
		t.Fatalf("newCouncilReceiptStoreV0: %v", err)
	}
	for _, ref := range []string{"../fuera", "sub/dir", "..", ""} {
		if err := store.SaveV0(councilReceiptV0{CouncilRef: ref, Outcome: "council_decision_accepted"}); err == nil {
			t.Fatalf("council_ref %q escapo del directorio de recibos", ref)
		}
	}
}
