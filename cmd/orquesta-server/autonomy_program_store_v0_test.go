package main

import (
	"context"
	"errors"
	"testing"

	autonomy "orquesta/modulos/orquesta-autonomy-program"
)

func programaParaTestV0(status autonomy.AutonomyProgramStatusV0) autonomy.AutonomyProgramV0 {
	return autonomy.AutonomyProgramV0{
		ProgramRef: "p", ProjectRef: "proj", RootRef: "root", Status: status,
		Nodes: []autonomy.AutonomyProgramNodeV0{{
			NodeRef: "a", GoalRef: "goal-a", Status: autonomy.AutonomyNodePendingV0,
			WriteSet: []string{"docs/a.md"}, RequiredTests: []string{"go test ./a"},
		}},
	}
}

// Sin CompareAndSwap, dos planificadores concurrentes pisarian el avance del otro
// y relanzarian nodos ya lanzados. El CAS solo escribe si lo que hay en disco es
// EXACTAMENTE lo que el llamante creia.
func TestAutonomyProgramCASNoPisaElAvanceAjenoV0(t *testing.T) {
	store, err := newAutonomyProgramStoreV0(t.TempDir())
	if err != nil {
		t.Fatalf("newAutonomyProgramStoreV0: %v", err)
	}
	ctx := context.Background()

	base, err := autonomy.NewAutonomyProgramV0(programaParaTestV0(autonomy.AutonomyProgramActiveV0))
	if err != nil {
		t.Fatalf("NewAutonomyProgramV0: %v", err)
	}
	if err := store.SaveAutonomyProgramV0(ctx, base); err != nil {
		t.Fatalf("SaveAutonomyProgramV0: %v", err)
	}

	// El planificador A avanza el programa.
	avanzado := base
	avanzado.Status = autonomy.AutonomyProgramCompletedV0
	ok, err := store.CompareAndSwapAutonomyProgramV0(ctx, base, avanzado)
	if err != nil || !ok {
		t.Fatalf("el primer CAS deberia ganar: ok=%v err=%v", ok, err)
	}

	// El planificador B, que aun creia ver el estado viejo, NO puede pisarlo.
	otro := base
	otro.Status = autonomy.AutonomyProgramBlockedV0
	ok, err = store.CompareAndSwapAutonomyProgramV0(ctx, base, otro)
	if err != nil {
		t.Fatalf("CompareAndSwapAutonomyProgramV0: %v", err)
	}
	if ok {
		t.Fatal("el CAS dejo pisar el avance de otro planificador: se relanzarian nodos ya lanzados")
	}

	// Y lo que queda en disco es el avance del primero, no el del segundo.
	actual, err := store.LoadAutonomyProgramV0(ctx, "proj", "root", "p")
	if err != nil {
		t.Fatalf("LoadAutonomyProgramV0: %v", err)
	}
	if actual.Status != autonomy.AutonomyProgramCompletedV0 {
		t.Fatalf("el estado persistido no es el del ganador: %q", actual.Status)
	}
}

// Guardar encima de un programa con OTRA topologia perderia el avance: choca.
func TestAutonomyProgramSaveAceptaReintentoYChocaSiCambiaV0(t *testing.T) {
	store, err := newAutonomyProgramStoreV0(t.TempDir())
	if err != nil {
		t.Fatalf("newAutonomyProgramStoreV0: %v", err)
	}
	ctx := context.Background()
	base, err := autonomy.NewAutonomyProgramV0(programaParaTestV0(autonomy.AutonomyProgramActiveV0))
	if err != nil {
		t.Fatalf("NewAutonomyProgramV0: %v", err)
	}
	if err := store.SaveAutonomyProgramV0(ctx, base); err != nil {
		t.Fatalf("primer save: %v", err)
	}
	// Reintento identico: idempotente.
	if err := store.SaveAutonomyProgramV0(ctx, base); err != nil {
		t.Fatalf("un reintento identico debe aceptarse: %v", err)
	}
	// Otra topologia con el mismo ref: choca.
	distinto := base
	distinto.Nodes = append(append([]autonomy.AutonomyProgramNodeV0(nil), base.Nodes...), autonomy.AutonomyProgramNodeV0{
		NodeRef: "b", GoalRef: "goal-b", Status: autonomy.AutonomyNodePendingV0,
		WriteSet: []string{"docs/b.md"}, RequiredTests: []string{"go test ./b"},
	})
	if err := store.SaveAutonomyProgramV0(ctx, distinto); !errors.Is(err, ErrAutonomyProgramConflictV0) {
		t.Fatalf("guardar otra topologia con el mismo ref debe chocar: %v", err)
	}
}
