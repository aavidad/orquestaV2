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

func TestGetProyectoOperacionSinTablaDevuelveDefault(t *testing.T) {
	prepararDBTemporal(t)
	if _, err := DB.Exec(`DROP TABLE proyectos_operacion`); err != nil {
		t.Fatalf("drop proyectos_operacion: %v", err)
	}

	op, err := GetProyectoOperacion(42)
	if err != nil {
		t.Fatalf("get proyecto operacion sin tabla: %v", err)
	}
	if op == nil || op.ProyectoID != 42 || op.EstadoOperativo != ProyectoOperativoActivo {
		t.Fatalf("default inesperado sin tabla: %+v", op)
	}
}

func TestBloquearTareaAutoPorAgenteNoMarcaProyectoEsperandoHumano(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex7", "programador"); err != nil {
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
		Titulo:      "Tarea autobloqueada por degradacion",
		Descripcion: "No deberia bloquear el proyecto entero",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := TomarTarea(tareaID, "Codex7"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := IniciarTarea(tareaID, "Codex7"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}

	motivo := "Agente Codex7 en estado bloqueado_por_runtime: pausado"
	if err := BloquearTarea(tareaID, "Codex7", motivo); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}

	op, err := GetProyectoOperacion(proyectoID)
	if err != nil {
		t.Fatalf("get proyecto operacion: %v", err)
	}
	if op.EstadoOperativo != ProyectoOperativoActivo {
		t.Fatalf("el proyecto no deberia quedar esperando_humano por autobloqueo de agente: %+v", op)
	}
	if op.Motivo != "" {
		t.Fatalf("el motivo del proyecto deberia permanecer vacio: %+v", op)
	}
}

func TestResolverBloqueoProyectoIgnoraAutobloqueosDeAgente(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex7", "programador"); err != nil {
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
		Titulo:      "Tarea autobloqueada por degradacion",
		Descripcion: "No deberia bloquear el proyecto entero",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := TomarTarea(tareaID, "Codex7"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := IniciarTarea(tareaID, "Codex7"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := BloquearTarea(tareaID, "Codex7", "Agente degradado: mailbox pendiente y entrega no convergente"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}

	bloqueado, motivo, err := ResolverBloqueoProyecto(proyectoID)
	if err != nil {
		t.Fatalf("resolver bloqueo proyecto: %v", err)
	}
	if bloqueado {
		t.Fatalf("el proyecto no deberia considerarse bloqueado por autobloqueos de agente, motivo=%q", motivo)
	}
	if motivo != "" {
		t.Fatalf("motivo inesperado: %q", motivo)
	}
}

func TestResolverBloqueoProyectoIgnoraBloqueosPorSobrecargaOperativa(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex8", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador-sobrecarga",
		Nombre:  "Orquestador Sobrecarga",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Tarea retenida por scheduler",
		Descripcion: "No deberia bloquear el proyecto como si esperara humano",
		ProyectoID:  &proyectoID,
		Modulo:      "runtime",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := TomarTarea(tareaID, "Codex8"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := IniciarTarea(tareaID, "Codex8"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	if err := BloquearTarea(tareaID, "Codex8", "Sobrecarga operativa: sin relevo sano disponible"); err != nil {
		t.Fatalf("bloquear tarea: %v", err)
	}

	bloqueado, motivo, err := ResolverBloqueoProyecto(proyectoID)
	if err != nil {
		t.Fatalf("resolver bloqueo proyecto: %v", err)
	}
	if bloqueado {
		t.Fatalf("el proyecto no deberia requerir intervencion humana por sobrecarga operativa, motivo=%q", motivo)
	}
	if motivo != "" {
		t.Fatalf("motivo inesperado: %q", motivo)
	}
}
