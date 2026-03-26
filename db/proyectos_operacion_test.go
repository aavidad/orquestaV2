package db

import (
	"path/filepath"
	"testing"
)

func TestBloquearYDesbloquearTareaActualizaEstadoOperativoProyecto(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Esperar respuesta humana",
		Descripcion: "Bloqueo de proyecto",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
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

	if err := BloquearTarea(tareaID, "Codex1", "esperando validacion humana"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}
	op, err := GetProyectoOperacion(proyectoID)
	if err != nil {
		t.Fatalf("get proyecto operacion tras bloqueo: %v", err)
	}
	if op.EstadoOperativo != ProyectoOperativoEsperandoHumano {
		t.Fatalf("estado operativo inesperado tras bloqueo: %+v", op)
	}
	if op.Motivo != "esperando validacion humana" {
		t.Fatalf("motivo inesperado tras bloqueo: %+v", op)
	}

	if err := DesbloquearTarea(tareaID, "Codex1", "respuesta recibida"); err != nil {
		t.Fatalf("desbloquear tarea: %v", err)
	}
	op, err = GetProyectoOperacion(proyectoID)
	if err != nil {
		t.Fatalf("get proyecto operacion tras desbloqueo: %v", err)
	}
	if op.EstadoOperativo != ProyectoOperativoActivo {
		t.Fatalf("estado operativo inesperado tras desbloqueo: %+v", op)
	}
	if op.Motivo != "" {
		t.Fatalf("motivo deberia quedar vacio tras desbloqueo: %+v", op)
	}
}
