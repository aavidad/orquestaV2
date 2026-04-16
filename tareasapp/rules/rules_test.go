package rules

import (
	"testing"

	"orquesta/db"
)

func TestTaskTransitionCoordinatorMethodsNoPanic(t *testing.T) {
	t.Parallel()

	c := TaskTransitionCoordinator{}
	if err := c.AfterTomarTarea(&db.Tarea{ID: 1, Titulo: "tarea"}, "agente"); err != nil {
		t.Fatalf("AfterTomarTarea: %v", err)
	}
	if err := c.AfterIniciarTarea(&db.Tarea{ID: 2, Titulo: "tarea"}, "agente"); err != nil {
		t.Fatalf("AfterIniciarTarea: %v", err)
	}
	if err := c.AfterCompletarTarea(&db.Tarea{ID: 3, Titulo: "tarea"}, "agente", "c1"); err != nil {
		t.Fatalf("AfterCompletarTarea: %v", err)
	}
	if err := c.AfterCancelarTarea(&db.Tarea{ID: 4, Titulo: "tarea"}, "agente", "motivo"); err != nil {
		t.Fatalf("AfterCancelarTarea: %v", err)
	}
	if err := c.AfterBloquearTarea(&db.Tarea{ID: 5, Titulo: "tarea"}, "agente", "motivo"); err != nil {
		t.Fatalf("AfterBloquearTarea: %v", err)
	}
	if err := c.AfterDesbloquearTarea(&db.Tarea{ID: 6, Titulo: "tarea"}, "agente", "resolución"); err != nil {
		t.Fatalf("AfterDesbloquearTarea: %v", err)
	}
}

func TestAsignacionTransitionCoordinatorMethodsNoPanic(t *testing.T) {
	t.Parallel()

	c := AsignacionTransitionCoordinator{}
	if err := c.AfterActivarAsignacion("agente", 77, "nota"); err != nil {
		t.Fatalf("AfterActivarAsignacion: %v", err)
	}
	if err := c.AfterPausarAsignacion("agente", 77, "nota"); err != nil {
		t.Fatalf("AfterPausarAsignacion: %v", err)
	}
}

func TestGovernanceTransitionCoordinatorMethodsNoPanic(t *testing.T) {
	t.Parallel()

	c := GovernanceTransitionCoordinator{}
	if err := c.AfterCrearRegla("actor", nil, 1); err != nil {
		t.Fatalf("AfterCrearRegla: %v", err)
	}
	if err := c.AfterActualizarRegla("actor", nil); err != nil {
		t.Fatalf("AfterActualizarRegla: %v", err)
	}
	if err := c.AfterSetReglaActiva("actor", nil, true); err != nil {
		t.Fatalf("AfterSetReglaActiva: %v", err)
	}
	if err := c.AfterCrearSkill("actor", nil, 10); err != nil {
		t.Fatalf("AfterCrearSkill: %v", err)
	}
	if err := c.AfterActualizarSkill("actor", nil); err != nil {
		t.Fatalf("AfterActualizarSkill: %v", err)
	}
	if err := c.AfterSetSkillActiva("actor", nil, false); err != nil {
		t.Fatalf("AfterSetSkillActiva: %v", err)
	}
	if err := c.AfterCrearWorkflow("actor", nil, 20); err != nil {
		t.Fatalf("AfterCrearWorkflow: %v", err)
	}
	if err := c.AfterActualizarWorkflow("actor", nil); err != nil {
		t.Fatalf("AfterActualizarWorkflow: %v", err)
	}
	if err := c.AfterSetWorkflowActivo("actor", nil, true); err != nil {
		t.Fatalf("AfterSetWorkflowActivo: %v", err)
	}
}

func TestInstallIsSafeToCall(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Install no debe hacer panic: %v", r)
		}
	}()
	Install()
	Install()
}
