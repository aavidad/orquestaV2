package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
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

	// Idempotencia de reintento: la MISMA convocatoria devuelve LO DECIDIDO, sin
	// volver a votar.
	reintento := decidir("consejo-durable", "approve", "approve", "approve")
	if reintento.Outcome != "council_decision_accepted" {
		t.Fatalf("un reintento identico debe devolver lo decidido: %+v", reintento)
	}

	// Pero reconvocar el mismo council_ref con OTROS VOTOS no es un reintento: es
	// un choque. Devolver el veredicto viejo en silencio seria mentir, y aceptar
	// el nuevo seria reabrir una decision cerrada.
	choque := decidir("consejo-durable", "rework", "rework", "rework")
	if choque.Estado != orquestamcp.MCPCouncilEstadoErrorV0 {
		t.Fatalf("reutilizar el consejo con otra convocatoria debe chocar: %+v", choque)
	}
	if len(choque.ErroresPublicos) == 0 || choque.ErroresPublicos[0].Code != "council_receipt_conflict" {
		t.Fatalf("el choque no trae codigo publico tipado: %+v", choque.ErroresPublicos)
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
		if _, err := store.SaveV0(reciboValidoParaTestV0(ref, "sha256:x", 3)); err == nil {
			t.Fatalf("council_ref %q escapo del directorio de recibos", ref)
		}
	}
}

// Dos decisiones concurrentes sobre el mismo consejo no pueden pisarse: gana UNA
// y la otra reconoce su veredicto. Un Rename que sobrescribe dejaria ganar al
// ultimo en escribir, que es una decision distinta segun el reloj.
func TestCouncilReceiptNoSePisaEnConcurrenciaV0(t *testing.T) {
	store, err := newCouncilReceiptStoreV0(t.TempDir())
	if err != nil {
		t.Fatalf("newCouncilReceiptStoreV0: %v", err)
	}

	const escritores = 8
	resultados := make(chan councilReceiptV0, escritores)
	errores := make(chan error, escritores)
	var arranque sync.WaitGroup
	arranque.Add(1)
	var fin sync.WaitGroup
	for idx := 0; idx < escritores; idx++ {
		fin.Add(1)
		go func(idx int) {
			defer fin.Done()
			arranque.Wait()
			guardado, err := store.SaveV0(reciboValidoParaTestV0("concurrente", "sha256:misma-convocatoria", idx))
			if err != nil {
				errores <- err
				return
			}
			resultados <- guardado
		}(idx)
	}
	arranque.Done()
	fin.Wait()
	close(resultados)
	close(errores)

	for err := range errores {
		t.Fatalf("misma convocatoria concurrente no debe dar error: %v", err)
	}
	var ganador *councilReceiptV0
	for receipt := range resultados {
		if ganador == nil {
			copia := receipt
			ganador = &copia
			continue
		}
		if receipt.Approvals != ganador.Approvals {
			t.Fatalf("dos veredictos distintos sobrevivieron: %d y %d", ganador.Approvals, receipt.Approvals)
		}
	}
	if ganador == nil {
		t.Fatal("ningun escritor guardo la decision")
	}
}

// Reutilizar el council_ref con OTRA convocatoria no es un reintento: es un
// choque. Devolver el recibo viejo en silencio seria mentir.
func TestCouncilReceiptRechazaMismoRefConOtraConvocatoriaV0(t *testing.T) {
	store, err := newCouncilReceiptStoreV0(t.TempDir())
	if err != nil {
		t.Fatalf("newCouncilReceiptStoreV0: %v", err)
	}
	if _, err := store.SaveV0(reciboValidoParaTestV0("c", "sha256:a", 3)); err != nil {
		t.Fatalf("primera decision: %v", err)
	}
	_, err = store.SaveV0(reciboValidoParaTestV0("c", "sha256:DISTINTA", 1))
	if !errors.Is(err, ErrCouncilReceiptConflictV0) {
		t.Fatalf("reutilizar el council_ref con otra convocatoria debe chocar: %v", err)
	}
}

// Un recibo ilegible NO es "no existe": tratarlo asi seria fail-open y permitiria
// abrir la puerta del gate rompiendo un fichero.
func TestCouncilReceiptCorruptoNoAbreLaPuertaV0(t *testing.T) {
	dir := t.TempDir()
	store, err := newCouncilReceiptStoreV0(dir)
	if err != nil {
		t.Fatalf("newCouncilReceiptStoreV0: %v", err)
	}
	corrupto := filepath.Join(dir, councilReceiptsDirNameV0, "roto.json")
	if err := os.WriteFile(corrupto, []byte("{no soy json"), 0o600); err != nil {
		t.Fatalf("escribiendo recibo corrupto: %v", err)
	}
	if _, _, err := store.LoadV0("roto"); !errors.Is(err, ErrCouncilReceiptCorruptV0) {
		t.Fatalf("un recibo corrupto debe fallar, no pasar por inexistente: %v", err)
	}
	aceptada, err := store.CouncilDecisionAcceptedV0(context.Background(), "roto")
	if err == nil || aceptada {
		t.Fatalf("el gate debe fallar CERRADO ante un recibo ilegible: aceptada=%v err=%v", aceptada, err)
	}
}

// Un JSON minimo no es una decision del consejo. Sin validacion, dejar caer dos
// lineas en el directorio de estado abria el gate de creacion.
func TestCouncilReciboFalsificadoNoAbreLaPuertaV0(t *testing.T) {
	dir := t.TempDir()
	store, err := newCouncilReceiptStoreV0(dir)
	if err != nil {
		t.Fatalf("newCouncilReceiptStoreV0: %v", err)
	}
	falsificados := map[string]string{
		"minimo":       `{"outcome":"council_decision_accepted"}`,
		"sin-huella":   `{"schema_version":"orquesta_council_receipt.v0","council_ref":"sin-huella","outcome":"council_decision_accepted","total":3}`,
		"otro-consejo": `{"schema_version":"orquesta_council_receipt.v0","council_ref":"otro","input_fingerprint":"sha256:x","outcome":"council_decision_accepted","total":3,"seats":[{"role":"revisor","member_ref":"a","material":"diff_crudo"}]}`,
		"sin-votantes": `{"schema_version":"orquesta_council_receipt.v0","council_ref":"sin-votantes","input_fingerprint":"sha256:x","outcome":"council_decision_accepted","total":0}`,
	}
	for ref, contenido := range falsificados {
		path := filepath.Join(dir, councilReceiptsDirNameV0, ref+".json")
		if err := os.WriteFile(path, []byte(contenido), 0o600); err != nil {
			t.Fatalf("escribiendo %s: %v", ref, err)
		}
		aceptada, err := store.CouncilDecisionAcceptedV0(context.Background(), ref)
		if err == nil || aceptada {
			t.Fatalf("el recibo falsificado %q abrio la puerta: aceptada=%v err=%v", ref, aceptada, err)
		}
	}
}

// La huella tiene que ser estable ante el orden: overrides que salen de un map y
// miembros en distinto orden son la MISMA convocatoria.
func TestCouncilHuellaEsEstableAnteElOrdenV0(t *testing.T) {
	uno := orquestamcp.MCPCouncilToolInputV0{
		AuthorRef: "autor",
		Members: []orquestamcp.MCPCouncilMemberV0{
			{MemberRef: "a"}, {MemberRef: "b"},
		},
		Overrides: []orquestamcp.MCPCouncilOverrideV0{
			{Role: "revisor", MemberRef: "a"}, {Role: "consultor", MemberRef: "b"},
		},
	}
	otro := orquestamcp.MCPCouncilToolInputV0{
		AuthorRef: "autor",
		Members: []orquestamcp.MCPCouncilMemberV0{
			{MemberRef: "b"}, {MemberRef: "a"},
		},
		Overrides: []orquestamcp.MCPCouncilOverrideV0{
			{Role: "consultor", MemberRef: "b"}, {Role: "revisor", MemberRef: "a"},
		},
	}
	if councilInputFingerprintV0(uno) != councilInputFingerprintV0(otro) {
		t.Fatal("la misma convocatoria en distinto orden dio huellas distintas: un reintento legitimo chocaria")
	}
}

func reciboValidoParaTestV0(councilRef string, fingerprint string, approvals int) councilReceiptV0 {
	return councilReceiptV0{
		CouncilRef:       councilRef,
		InputFingerprint: fingerprint,
		Outcome:          councilOutcomeAcceptedV0,
		Approvals:        approvals,
		Total:            3,
		Seats: []orquestamcp.MCPCouncilSeatV0{
			{Role: "revisor", MemberRef: "a", Material: "diff_crudo"},
			{Role: "consultor", MemberRef: "b", Material: "masticado"},
			{Role: "adversario", MemberRef: "c", Material: "diff_crudo"},
		},
	}
}
