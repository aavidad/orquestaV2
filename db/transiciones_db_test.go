package db

import (
	"testing"
)

type testTaskTransitionCoordinator struct {
	tomar       int
	iniciar     int
	completar   int
	cancelar    int
	bloquear    int
	desbloquear int

	lastTomarAgente           string
	lastTomarTareaID          int64
	lastIniciarAgente         string
	lastIniciarTareaID        int64
	lastCompletarAgente       string
	lastCompletarTareaID      int64
	lastCompletarCommit       string
	lastCancelarAgente        string
	lastCancelarTareaID       int64
	lastCancelarMotivo        string
	lastBloquearAgente        string
	lastBloquearTareaID       int64
	lastBloquearMotivo        string
	lastDesbloquearAgente     string
	lastDesbloquearTareaID    int64
	lastDesbloquearResolucion string
}

func (c *testTaskTransitionCoordinator) AfterTomarTarea(t *Tarea, agente string) error {
	c.tomar++
	c.lastTomarAgente = agente
	if t != nil {
		c.lastTomarTareaID = t.ID
	}
	return nil
}

func (c *testTaskTransitionCoordinator) AfterIniciarTarea(t *Tarea, agente string) error {
	c.iniciar++
	c.lastIniciarAgente = agente
	if t != nil {
		c.lastIniciarTareaID = t.ID
	}
	return nil
}

func (c *testTaskTransitionCoordinator) AfterCompletarTarea(t *Tarea, agente, commit string) error {
	c.completar++
	c.lastCompletarAgente = agente
	c.lastCompletarCommit = commit
	if t != nil {
		c.lastCompletarTareaID = t.ID
	}
	return nil
}

func (c *testTaskTransitionCoordinator) AfterCancelarTarea(t *Tarea, agente, motivo string) error {
	c.cancelar++
	c.lastCancelarAgente = agente
	c.lastCancelarMotivo = motivo
	if t != nil {
		c.lastCancelarTareaID = t.ID
	}
	return nil
}

func (c *testTaskTransitionCoordinator) AfterBloquearTarea(t *Tarea, agente, motivo string) error {
	c.bloquear++
	c.lastBloquearAgente = agente
	c.lastBloquearMotivo = motivo
	if t != nil {
		c.lastBloquearTareaID = t.ID
	}
	return nil
}

func (c *testTaskTransitionCoordinator) AfterDesbloquearTarea(t *Tarea, agente, resolucion string) error {
	c.desbloquear++
	c.lastDesbloquearAgente = agente
	c.lastDesbloquearResolucion = resolucion
	if t != nil {
		c.lastDesbloquearTareaID = t.ID
	}
	return nil
}

type testAsignacionTransitionCoordinator struct {
	activar           int
	pausar            int
	activarAgente     string
	pausarAgente      string
	activarProyectoID int64
	pausarProyectoID  int64
	activarNota       string
	pausarNota        string
}

func (c *testAsignacionTransitionCoordinator) AfterActivarAsignacion(agente string, proyectoID int64, nota string) error {
	c.activar++
	c.activarAgente = agente
	c.activarProyectoID = proyectoID
	c.activarNota = nota
	return nil
}

func (c *testAsignacionTransitionCoordinator) AfterPausarAsignacion(agente string, proyectoID int64, nota string) error {
	c.pausar++
	c.pausarAgente = agente
	c.pausarProyectoID = proyectoID
	c.pausarNota = nota
	return nil
}

func TestCrearTarea(t *testing.T) {
	prepararDBTemporal(t)

	id, err := CrearTarea(&Tarea{
		Titulo:      "Tarea de regresion",
		Descripcion: "Valida ruta de creación",
		Modulo:      "db",
		Prioridad:   PrioridadMedia,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("CrearTarea: %v", err)
	}
	if id == 0 {
		t.Fatalf("id inesperado: %d", id)
	}
}

func TestBloquearTarea(t *testing.T) {
	prepararDBTemporal(t)
	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}

	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "regresion-bloqueo",
		Nombre:  "Regresion bloqueo",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Tarea bloqueable",
		Descripcion: "Valida bloqueo de proyecto",
		ProyectoID:  &proyectoID,
		Modulo:      "db",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("CrearTarea: %v", err)
	}
	if err := TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("TomarTarea: %v", err)
	}
	if err := IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("IniciarTarea: %v", err)
	}

	if err := BloquearTarea(tareaID, "Codex1", "esperando validacion humana"); err != nil {
		t.Fatalf("BloquearTarea: %v", err)
	}
	tarea, err := GetTarea(tareaID)
	if err != nil {
		t.Fatalf("GetTarea: %v", err)
	}
	if tarea.Estado != TareaBloqueada {
		t.Fatalf("estado tarea inesperado: %s", tarea.Estado)
	}
	op, err := GetProyectoOperacion(proyectoID)
	if err != nil {
		t.Fatalf("GetProyectoOperacion: %v", err)
	}
	if op.EstadoOperativo != ProyectoOperativoActivo {
		t.Fatalf("estado operativo inesperado: %+v", op)
	}
}

func TestDesbloquearTarea(t *testing.T) {
	prepararDBTemporal(t)
	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}

	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "regresion-desbloqueo",
		Nombre:  "Regresion desbloqueo",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Tarea bloqueable",
		Descripcion: "Valida desbloqueo de proyecto",
		ProyectoID:  &proyectoID,
		Modulo:      "db",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("CrearTarea: %v", err)
	}
	if err := TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("TomarTarea: %v", err)
	}
	if err := IniciarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("IniciarTarea: %v", err)
	}
	if err := BloquearTarea(tareaID, "Codex1", "esperando validacion humana"); err != nil {
		t.Fatalf("BloquearTarea: %v", err)
	}
	if err := DesbloquearTarea(tareaID, "Codex1", "resuelto"); err != nil {
		t.Fatalf("DesbloquearTarea: %v", err)
	}
	op, err := GetProyectoOperacion(proyectoID)
	if err != nil {
		t.Fatalf("GetProyectoOperacion: %v", err)
	}
	if op.EstadoOperativo != ProyectoOperativoActivo {
		t.Fatalf("estado operativo inesperado tras desbloqueo: %+v", op)
	}
}

func TestCrearRegla(t *testing.T) {
	prepararDBTemporal(t)
	id, err := CrearRegla("Codex2", &Regla{
		TipoAgente:  "programador",
		Categoria:   "arquitectura",
		Titulo:      "Regla de regresion",
		Descripcion: "Valida creación de reglas",
		Activa:      true,
	})
	if err != nil {
		t.Fatalf("CrearRegla: %v", err)
	}
	if id == 0 {
		t.Fatalf("id inesperado: %d", id)
	}
}

func TestCrearWorkflow(t *testing.T) {
	prepararDBTemporal(t)
	id, err := CrearWorkflow("Codex2", &Workflow{
		TipoAgente:  "programador",
		Nombre:      "workflow-regresion",
		Descripcion: "Flujo para pruebas de regresión",
		Pasos:       "[\"uno\"]",
		Activo:      true,
	})
	if err != nil {
		t.Fatalf("CrearWorkflow: %v", err)
	}
	if id == 0 {
		t.Fatalf("id inesperado: %d", id)
	}
}

func TestTransicionesTareaDisparanCoordinador(t *testing.T) {
	prepararDBTemporal(t)
	coordinator := &testTaskTransitionCoordinator{}
	SetTaskTransitionCoordinator(coordinator)
	t.Cleanup(func() {
		SetTaskTransitionCoordinator(nil)
	})

	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "transiciones-tarea-proyecto",
		Nombre:  "Proyecto transiciones tarea",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}

	tomaID, err := CrearTarea(&Tarea{
		Titulo:      "tarea tomar",
		Descripcion: "Verifica coordinator en TomarTarea",
		ProyectoID:  &proyectoID,
		Modulo:      "db",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("CrearTarea tomar: %v", err)
	}
	if err := TomarTarea(tomaID, "Codex1"); err != nil {
		t.Fatalf("TomarTarea: %v", err)
	}
	if coordinator.tomar != 1 || coordinator.lastTomarAgente != "Codex1" || coordinator.lastTomarTareaID != tomaID {
		t.Fatalf("TomarTarea no delega correctamente: %+v", coordinator)
	}

	inicioID, err := CrearTarea(&Tarea{
		Titulo:      "tarea iniciar",
		Descripcion: "Verifica coordinator en IniciarTarea",
		ProyectoID:  &proyectoID,
		Modulo:      "db",
		Prioridad:   PrioridadMedia,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("CrearTarea iniciar: %v", err)
	}
	if err := TomarTarea(inicioID, "Codex1"); err != nil {
		t.Fatalf("TomarTarea iniciar: %v", err)
	}
	if err := IniciarTarea(inicioID, "Codex1"); err != nil {
		t.Fatalf("IniciarTarea: %v", err)
	}
	if coordinator.iniciar != 1 || coordinator.lastIniciarAgente != "Codex1" || coordinator.lastIniciarTareaID != inicioID {
		t.Fatalf("IniciarTarea no delega correctamente: %+v", coordinator)
	}

	completarID, err := CrearTarea(&Tarea{
		Titulo:      "tarea completar",
		Descripcion: "Verifica coordinator en CompletarTarea",
		ProyectoID:  &proyectoID,
		Modulo:      "db",
		Prioridad:   PrioridadMedia,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("CrearTarea completar: %v", err)
	}
	if err := TomarTarea(completarID, "Codex1"); err != nil {
		t.Fatalf("TomarTarea completar: %v", err)
	}
	if err := CompletarTarea(completarID, "Codex1", "c1"); err != nil {
		t.Fatalf("CompletarTarea: %v", err)
	}
	if coordinator.completar != 1 || coordinator.lastCompletarAgente != "Codex1" || coordinator.lastCompletarTareaID != completarID || coordinator.lastCompletarCommit != "c1" {
		t.Fatalf("CompletarTarea no delega correctamente: %+v", coordinator)
	}

	cancelID, err := CrearTarea(&Tarea{
		Titulo:      "tarea cancelar",
		Descripcion: "Verifica coordinator en CancelarTarea",
		Modulo:      "db",
		Prioridad:   PrioridadMedia,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("CrearTarea cancelar: %v", err)
	}
	if err := CancelarTarea(cancelID, "Codex1", "motivo validación"); err != nil {
		t.Fatalf("CancelarTarea: %v", err)
	}
	if coordinator.cancelar != 1 || coordinator.lastCancelarAgente != "Codex1" || coordinator.lastCancelarTareaID != cancelID || coordinator.lastCancelarMotivo != "motivo validación" {
		t.Fatalf("CancelarTarea no delega correctamente: %+v", coordinator)
	}

	bloquearID, err := CrearTarea(&Tarea{
		Titulo:      "tarea bloquear",
		Descripcion: "Verifica coordinator en Bloquear/DesbloquearTarea",
		ProyectoID:  &proyectoID,
		Modulo:      "db",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("CrearTarea bloquear: %v", err)
	}
	if err := TomarTarea(bloquearID, "Codex1"); err != nil {
		t.Fatalf("TomarTarea bloquear: %v", err)
	}
	if err := BloquearTarea(bloquearID, "Codex1", "esperando validacion humana"); err != nil {
		t.Fatalf("BloquearTarea: %v", err)
	}
	if coordinator.bloquear != 1 || coordinator.lastBloquearAgente != "Codex1" || coordinator.lastBloquearTareaID != bloquearID || coordinator.lastBloquearMotivo != "esperando validacion humana" {
		t.Fatalf("BloquearTarea no delega correctamente: %+v", coordinator)
	}
	if err := DesbloquearTarea(bloquearID, "Codex1", "resuelto"); err != nil {
		t.Fatalf("DesbloquearTarea: %v", err)
	}
	if coordinator.desbloquear != 1 || coordinator.lastDesbloquearAgente != "Codex1" || coordinator.lastDesbloquearTareaID != bloquearID || coordinator.lastDesbloquearResolucion != "resuelto" {
		t.Fatalf("DesbloquearTarea no delega correctamente: %+v", coordinator)
	}
}

func TestTransicionesAsignacionDisparanCoordinador(t *testing.T) {
	prepararDBTemporal(t)
	coordinator := &testAsignacionTransitionCoordinator{}
	SetAsignacionTransitionCoordinator(coordinator)
	t.Cleanup(func() {
		SetAsignacionTransitionCoordinator(nil)
	})

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "asignacion-coordinator",
		Nombre:  "Proyecto asignación coordinator",
		RutaAbs: t.TempDir(),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}

	if err := ActivarAsignacion("Codex1", proyectoID, "activar"); err != nil {
		t.Fatalf("ActivarAsignacion: %v", err)
	}
	if coordinator.activar != 1 || coordinator.activarAgente != "Codex1" || coordinator.activarProyectoID != proyectoID || coordinator.activarNota != "activar" {
		t.Fatalf("ActivarAsignacion no delega correctamente: %+v", coordinator)
	}

	if err := PausarAsignacion("Codex1", proyectoID, "pausar"); err != nil {
		t.Fatalf("PausarAsignacion: %v", err)
	}
	if coordinator.pausar != 1 || coordinator.pausarAgente != "Codex1" || coordinator.pausarProyectoID != proyectoID || coordinator.pausarNota != "pausar" {
		t.Fatalf("PausarAsignacion no delega correctamente: %+v", coordinator)
	}

	// Sin estado activo, no debe volver a disparar coordinador de pausa.
	if err := PausarAsignacion("Codex1", proyectoID, "sin cambio"); err != nil {
		t.Fatalf("PausarAsignacion (sin cambio): %v", err)
	}
	if coordinator.pausar != 1 {
		t.Fatalf("PausarAsignacion sin cambio no debe invocar coordinación: %+v", coordinator)
	}

	if err := ActivarAsignacion("Codex1", proyectoID, "reactivar"); err != nil {
		t.Fatalf("ActivarAsignacion reactiva: %v", err)
	}
	if coordinator.activar != 2 || coordinator.activarNota != "reactivar" {
		t.Fatalf("ActivarAsignacion reactiva no delega correctamente: %+v", coordinator)
	}
}
