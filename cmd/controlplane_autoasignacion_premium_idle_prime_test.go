package cmd

import (
	"path/filepath"
	"testing"

	"orquesta/agentesapp"
	"orquesta/db"
)

func TestProcesarAutoasignacionPremiumSinSesionBatchOmitePrimeIdleSinTareaActiva(t *testing.T) {
	tmp := prepararDBTemporalCmd(t)

	if err := db.RegistrarAgente("CodexPg1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := db.UpsertProyecto(&db.Proyecto{
		Slug:    "orquestador-premium-prime-idle-skip",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    db.ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := db.UpsertProyectoOperacion(&db.ProyectoOperacion{
		ProyectoID:       proyectoID,
		EstadoOperativo:  db.ProyectoOperativoActivo,
		Motivo:           "microrefactor_loop",
		ObjetivoPct:      100,
		MinAgentes:       1,
		MaxAgentes:       3,
		ResumeAutomatico: true,
	}); err != nil {
		t.Fatalf("upsert proyecto operacion: %v", err)
	}
	if err := db.ActivarAsignacion("CodexPg1", proyectoID, "frente prime"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := db.CrearTarea(&db.Tarea{
		Titulo:      microcicloRefactorTituloDefault,
		Descripcion: descripcionMicrocicloDefault(&db.Proyecto{Slug: "orquestador-premium-prime-idle-skip", RutaAbs: filepath.Join(tmp, "orquestador")}),
		ProyectoID:  &proyectoID,
		Modulo:      "controlplane",
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Notas:       microcicloRefactorNotasTag,
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}

	n, err := procesarAutoasignacionPremiumSinSesionBatch([]agentesapp.Row{{
		Agente:          &db.Agente{Nombre: "CodexPg1", Habilitado: true},
		EstadoOperativo: "sin_tarea",
	}}, map[string][]*db.Tarea{})
	if err != nil {
		t.Fatalf("procesarAutoasignacionPremiumSinSesionBatch: %v", err)
	}
	if n != 0 {
		t.Fatalf("prime idle no deberia autoasignar trabajo premium normal, got=%d", n)
	}

	tarea, err := db.GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea == nil || tarea.Agente != nil || tarea.Estado != db.TareaLibre {
		t.Fatalf("la tarea premium normal deberia seguir libre para prime idle: %+v", tarea)
	}

	agente := "CodexPg1"
	estado := "pendiente"
	orders, err := db.ListarRuntimeOrders(db.FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("prime idle no deberia encolar ordenes nuevas por premium_idle_autoassigned: %+v", orders)
	}
}
