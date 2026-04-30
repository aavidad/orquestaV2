package db

import (
	"os"
	"path/filepath"
	"testing"

	"orquesta/coordinacion"
	"orquesta/internal/controlruntime"
)

func stubRuntimeHygieneTMUX(t *testing.T) {
	t.Helper()
	prevCommand := runtimeTMUXCommandPathFn
	prevList := runtimeTMUXListSessionsFn
	prevKill := runtimeTMUXKillSessionFn
	t.Cleanup(func() {
		runtimeTMUXCommandPathFn = prevCommand
		runtimeTMUXListSessionsFn = prevList
		runtimeTMUXKillSessionFn = prevKill
	})
	runtimeTMUXCommandPathFn = func() (string, error) { return "", nil }
	runtimeTMUXListSessionsFn = func(string) ([]controlruntime.TMUXSessionInfo, error) { return nil, nil }
	runtimeTMUXKillSessionFn = func(string, string) error { return nil }
}

func TestEnviarRuntimeMailboxConsumePendienteSupersedidaPorTaskID(t *testing.T) {
	prepararDBTemporal(t)

	for _, agente := range []string{"Codex1", "Codex2"} {
		if err := RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("registrar agente %s: %v", agente, err)
		}
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "mailbox-demo",
		Nombre:  "Mailbox Demo",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}

	firstID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "Codex1",
		ToAgente:    "Codex2",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","tarea_id":41}`,
	})
	if err != nil {
		t.Fatalf("enviar mailbox 1: %v", err)
	}
	secondID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "Codex1",
		ToAgente:    "Codex2",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","tarea_id":41}`,
	})
	if err != nil {
		t.Fatalf("enviar mailbox 2: %v", err)
	}

	first, err := GetRuntimeMailbox(firstID)
	if err != nil {
		t.Fatalf("get mailbox 1: %v", err)
	}
	second, err := GetRuntimeMailbox(secondID)
	if err != nil {
		t.Fatalf("get mailbox 2: %v", err)
	}
	if first == nil || first.Estado != "consumido" {
		t.Fatalf("mailbox 1 deberia quedar consumida: %+v", first)
	}
	if second == nil || second.Estado != "pendiente" {
		t.Fatalf("mailbox 2 deberia seguir pendiente: %+v", second)
	}
}

func TestProcesarHigieneRuntimesAutonomosBatchConsumeMailboxPendientePorTareaNoAccionable(t *testing.T) {
	prepararDBTemporal(t)
	stubRuntimeHygieneTMUX(t)

	for _, agente := range []string{"Codex1", "Codex2"} {
		if err := RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("registrar agente %s: %v", agente, err)
		}
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "mailbox-hygiene",
		Nombre:  "Mailbox Hygiene",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Frente cerrado",
		Descripcion: "demo",
		ProyectoID:  &proyectoID,
		Prioridad:   PrioridadAlta,
		CreadoPor:   "tester",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := TomarTarea(tareaID, "Codex2"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := IniciarTarea(tareaID, "Codex2"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := CompletarTarea(tareaID, "Codex2", ""); err != nil {
		t.Fatalf("completar tarea: %v", err)
	}

	msgID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "Codex1",
		ToAgente:    "Codex2",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","tarea_id":` + jsonNumber(tareaID) + `}`,
	})
	if err != nil {
		t.Fatalf("enviar mailbox: %v", err)
	}

	n, err := ProcesarHigieneRuntimesAutonomosBatch()
	if err != nil {
		t.Fatalf("procesar higiene: %v", err)
	}
	if n < 1 {
		t.Fatalf("deberia consumir al menos una mailbox obsoleta, got=%d", n)
	}
	msg, err := GetRuntimeMailbox(msgID)
	if err != nil {
		t.Fatalf("get mailbox: %v", err)
	}
	if msg == nil || msg.Estado != "consumido" {
		t.Fatalf("mailbox deberia quedar consumida: %+v", msg)
	}
}

func TestProcesarHigieneRuntimesAutonomosBatchConsumeMailboxEntregadaPorTareaNoAccionable(t *testing.T) {
	prepararDBTemporal(t)
	stubRuntimeHygieneTMUX(t)

	for _, agente := range []string{"Codex1", "Codex2"} {
		if err := RegistrarAgente(agente, "programador"); err != nil {
			t.Fatalf("registrar agente %s: %v", agente, err)
		}
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "mailbox-hygiene-delivered",
		Nombre:  "Mailbox Hygiene Delivered",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Frente libre",
		Descripcion: "demo",
		ProyectoID:  &proyectoID,
		Prioridad:   PrioridadAlta,
		CreadoPor:   "tester",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := TomarTarea(tareaID, "Codex2"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := IniciarTarea(tareaID, "Codex2"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if _, err := DB.Exec(`UPDATE tareas SET estado='libre', agente=NULL WHERE id=?`, tareaID); err != nil {
		t.Fatalf("liberar tarea: %v", err)
	}

	msgID, err := EnviarRuntimeMailbox(&RuntimeMailboxMessage{
		FromAgente:  "Codex1",
		ToAgente:    "Codex2",
		ProyectoID:  &proyectoID,
		Kind:        "autonomia",
		PayloadJSON: `{"accion":"continuar_trabajo","tarea_id":` + jsonNumber(tareaID) + `}`,
	})
	if err != nil {
		t.Fatalf("enviar mailbox: %v", err)
	}
	if err := MarcarRuntimeMailboxEntregado(msgID); err != nil {
		t.Fatalf("marcar mailbox entregado: %v", err)
	}

	n, err := ProcesarHigieneRuntimesAutonomosBatch()
	if err != nil {
		t.Fatalf("procesar higiene: %v", err)
	}
	if n < 1 {
		t.Fatalf("deberia consumir al menos una mailbox entregada obsoleta, got=%d", n)
	}
	msg, err := GetRuntimeMailbox(msgID)
	if err != nil {
		t.Fatalf("get mailbox: %v", err)
	}
	if msg == nil || msg.Estado != "consumido" {
		t.Fatalf("mailbox entregada deberia quedar consumida: %+v", msg)
	}
}

func TestProcesarHigieneRuntimesAutonomosBatchCierraWorktreeDuplicadaPorTarea(t *testing.T) {
	tmp := prepararDBTemporal(t)
	stubRuntimeHygieneTMUX(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "worktree-hygiene",
		Nombre:  "Worktree Hygiene",
		RutaAbs: tmp,
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Frente activo",
		Descripcion: "demo",
		ProyectoID:  &proyectoID,
		Prioridad:   PrioridadAlta,
		CreadoPor:   "tester",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}

	root := filepath.Join(tmp, ".orquesta-worktrees")
	wt1 := filepath.Join(root, "wt-1")
	wt2 := filepath.Join(root, "wt-2")
	if err := os.MkdirAll(wt1, 0o755); err != nil {
		t.Fatalf("mkdir wt1: %v", err)
	}
	if err := os.MkdirAll(wt2, 0o755); err != nil {
		t.Fatalf("mkdir wt2: %v", err)
	}
	repo := CoordinationWorktreeSQLRepository{}
	first, err := repo.Create(&coordinacion.Worktree{
		ProjectID: proyectoID,
		TaskID:    &tareaID,
		Agent:     "Codex1",
		Name:      "wt-1",
		Path:      wt1,
		Branch:    "orq/worktree/1",
		BaseRef:   "main",
		State:     coordinacion.WorktreeActive,
		Reason:    "primera",
	})
	if err != nil {
		t.Fatalf("crear worktree 1: %v", err)
	}
	second, err := repo.Create(&coordinacion.Worktree{
		ProjectID: proyectoID,
		TaskID:    &tareaID,
		Agent:     "Codex1",
		Name:      "wt-2",
		Path:      wt2,
		Branch:    "orq/worktree/2",
		BaseRef:   "main",
		State:     coordinacion.WorktreeActive,
		Reason:    "duplicada",
	})
	if err != nil {
		t.Fatalf("crear worktree 2: %v", err)
	}

	n, err := ProcesarHigieneRuntimesAutonomosBatch()
	if err != nil {
		t.Fatalf("procesar higiene: %v", err)
	}
	if n < 1 {
		t.Fatalf("deberia cerrar al menos una worktree duplicada, got=%d", n)
	}

	activeState := coordinacion.WorktreeActive
	active, err := repo.ListRaw(coordinacion.WorktreeFilter{
		ProjectID: &proyectoID,
		State:     &activeState,
	})
	if err != nil {
		t.Fatalf("listar activas: %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("deberia quedar una sola worktree activa: %+v", active)
	}
	if active[0] == nil || active[0].ID != second.ID {
		t.Fatalf("deberia conservar la worktree mas reciente: %+v", active)
	}
	closed, err := repo.GetByID(first.ID)
	if err != nil {
		t.Fatalf("get worktree cerrada: %v", err)
	}
	if closed == nil || closed.State != coordinacion.WorktreeClosed {
		t.Fatalf("worktree duplicada deberia quedar cerrada: %+v", closed)
	}
	if _, err := os.Stat(wt1); !os.IsNotExist(err) {
		t.Fatalf("la ruta de la worktree cerrada deberia eliminarse: %v", err)
	}
}
