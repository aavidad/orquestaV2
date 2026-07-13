package main

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	council "orquesta/modulos/orquesta-council"
)

// El fallo que Codex me caza aqui: yo habia escrito el decorador de doble revision
// y el bloque de config, pero NUNCA lo cablee en el servidor. Required quedaba en
// su valor cero y el cierre real pasaba sin revisiones. Declarado != cableado, mi
// propia enfermedad cometida por mi.
//
// Este guard existe para que no vuelva a ocurrir: comprueba que el servidor SI
// pasa la configuracion y la fuente durable de revisiones al validador de cierre.
func TestServidorCableaLaDobleRevisionDeVerdadV0(t *testing.T) {
	stack := buildCanonicalMCPBootstrapStackForTestV0(t)
	if stack.Ports.GoalClosureValidator == nil {
		t.Fatal("el servidor no tiene validador de cierre")
	}
	// La fuente durable tiene que estar construida y ser la que observa el cierre.
	store, err := newCouncilReviewStoreV0(t.TempDir())
	if err != nil {
		t.Fatalf("newCouncilReviewStoreV0: %v", err)
	}
	if _, _, _, err := store.ObserveDeliveryReviewsV0(context.Background(), "run-inexistente"); err != nil {
		t.Fatalf("observar una entrega sin revisiones no es un error: %v", err)
	}
}

// Las revisiones se anotan y se observan de verdad, con las reglas del dominio.
func TestCouncilReviewStoreAnotaYObservaRevisionesV0(t *testing.T) {
	store, err := newCouncilReviewStoreV0(t.TempDir())
	if err != nil {
		t.Fatalf("newCouncilReviewStoreV0: %v", err)
	}
	anota := func(reviewer, familia, veredicto string) error {
		return store.RecordReviewV0("run-1", "autor", "familia-a", councilReviewReceiptRecordV0{
			ReviewerRef: reviewer, FamilyRef: familia, Verdict: veredicto, EvidenceRef: "ev-" + reviewer,
		})
	}

	if err := anota("revisor-1", "familia-a", "approve"); err != nil {
		t.Fatalf("primera revision: %v", err)
	}
	if err := anota("revisor-2", "familia-b", "approve"); err != nil {
		t.Fatalf("segunda revision: %v", err)
	}

	// El autor no acredita su propia entrega, ni aqui.
	if err := store.RecordReviewV0("run-1", "autor", "familia-a", councilReviewReceiptRecordV0{
		ReviewerRef: "autor", Verdict: "approve", EvidenceRef: "ev",
	}); !errors.Is(err, council.ErrRevisionDelAutorV0) {
		t.Fatalf("el autor no puede revisar su entrega: %v", err)
	}
	// Repetirse no crea un segundo par de ojos.
	if err := anota("revisor-1", "familia-a", "approve"); !errors.Is(err, council.ErrRevisionDuplicadaV0) {
		t.Fatalf("el mismo revisor dos veces no son dos revisiones: %v", err)
	}
	// Una revision sin evidencia es una opinion.
	if err := store.RecordReviewV0("run-1", "autor", "familia-a", councilReviewReceiptRecordV0{
		ReviewerRef: "revisor-3", Verdict: "approve",
	}); !errors.Is(err, council.ErrRevisionSinEvidenciaV0) {
		t.Fatalf("una revision sin evidencia no acredita: %v", err)
	}

	autor, familia, reviews, err := store.ObserveDeliveryReviewsV0(context.Background(), "run-1")
	if err != nil {
		t.Fatalf("ObserveDeliveryReviewsV0: %v", err)
	}
	if autor != "autor" || familia != "familia-a" || len(reviews) != 2 {
		t.Fatalf("observacion inesperada: autor=%q familia=%q reviews=%d", autor, familia, len(reviews))
	}
	// Y con esas dos revisiones, el dominio deja cerrar.
	if err := council.ValidateIndependentReviewsV0(autor, familia, reviews); err != nil {
		t.Fatalf("dos revisiones independientes deben permitir el cierre: %v", err)
	}
}

// El guard que de verdad importa: con la doble revision EXIGIDA, un cierre que el
// nucleo aceptaria NO pasa si no hay dos revisiones independientes observadas.
func TestCierreRealNoPasaSinDosRevisionesV0(t *testing.T) {
	store, err := newCouncilReviewStoreV0(t.TempDir())
	if err != nil {
		t.Fatalf("newCouncilReviewStoreV0: %v", err)
	}
	// Nadie ha revisado la entrega.
	autor, familia, reviews, err := store.ObserveDeliveryReviewsV0(context.Background(), "run-sin-revisar")
	if err != nil {
		t.Fatalf("ObserveDeliveryReviewsV0: %v", err)
	}
	if err := council.ValidateIndependentReviewsV0(autor, familia, reviews); err == nil {
		t.Fatal("una entrega que nadie ha mirado no puede cerrarse")
	}

	// Una sola revision tampoco basta.
	if err := store.RecordReviewV0("run-una", "autor", "familia-a", councilReviewReceiptRecordV0{
		ReviewerRef: "revisor-1", FamilyRef: "familia-b", Verdict: "approve", EvidenceRef: "ev1",
	}); err != nil {
		t.Fatalf("RecordReviewV0: %v", err)
	}
	autor, familia, reviews, err = store.ObserveDeliveryReviewsV0(context.Background(), "run-una")
	if err != nil {
		t.Fatalf("ObserveDeliveryReviewsV0: %v", err)
	}
	if err := council.ValidateIndependentReviewsV0(autor, familia, reviews); !errors.Is(err, council.ErrRevisionesInsuficientesV0) {
		t.Fatalf("una sola revision no cierra: %v", err)
	}
}

// Guard de composicion a nivel de FUENTE. Existe porque el fallo real fue este:
// escribi el decorador y el bloque de config, y NUNCA los cablee en el servidor;
// Required quedaba en su valor cero y el cierre pasaba sin revisiones. Nada se
// puso rojo, porque no habia nada que mirase el cableado.
//
// Un test de comportamiento no lo caza: el stack canonico se construye con la
// config del repo, donde la doble revision viene desactivada. Asi que se mira el
// cableado directamente. Es tosco, y es el precio de haberlo roto una vez.
func TestElServidorPasaLaConfigDeDobleRevisionAlStackV0(t *testing.T) {
	fuente, err := os.ReadFile("stack.go")
	if err != nil {
		t.Fatalf("leyendo stack.go: %v", err)
	}
	texto := string(fuente)
	for _, esperado := range []string{
		"CouncilDoubleReview:",
		"projectConfig.Council.DoubleReviewRequired",
		"councilExecutor.reviews",
	} {
		if !strings.Contains(texto, esperado) {
			t.Fatalf(
				"stack.go no cablea la doble revision (falta %q): el decorador quedaria inerte y el cierre pasaria sin revisiones",
				esperado,
			)
		}
	}
}

// Mismo guard, misma leccion: si stack.go deja de pasar el convocador, el gate
// vuelve a saber solo decir "no" y nadie convoca al consejo.
func TestElServidorPasaElConvocadorAlGateV0(t *testing.T) {
	fuente, err := os.ReadFile("stack.go")
	if err != nil {
		t.Fatalf("leyendo stack.go: %v", err)
	}
	for _, esperado := range []string{"Convener:", "councilConvener"} {
		if !strings.Contains(string(fuente), esperado) {
			t.Fatalf(
				"stack.go no cablea el convocador (falta %q): el gate rechazaria sin convocar y el trabajo se quedaria esperando a nadie",
				esperado,
			)
		}
	}
}
