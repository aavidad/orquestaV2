package db

import (
	"testing"
)

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
