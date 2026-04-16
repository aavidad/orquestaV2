package db

import (
	"path/filepath"
	"testing"
	"time"
)

func TestProyectoAutonomiaUpsertYListarActivos(t *testing.T) {
	tmp := prepararDBTemporal(t)
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

	item, err := GetProyectoAutonomia(proyectoID)
	if err != nil {
		t.Fatalf("get default autonomia: %v", err)
	}
	if item == nil || item.Enabled {
		t.Fatalf("default autonomia inesperada: %+v", item)
	}

	now := time.Now().UTC().Truncate(time.Second)
	item.Enabled = true
	item.ObjetivoGeneral = "Terminar la app sin intervención humana normal"
	item.DefinitionOfDoneJSON = `{"tests":"green","api":"server-first"}`
	item.MaxWorkers = 4
	item.SupervisorAgente = "CodexSupervisor"
	item.ReviewerAgente = "CodexReviewer"
	item.ReserveReviewer = true
	item.ReserveSupervisor = true
	item.LastSupervisionAt = &now
	if _, err := UpsertProyectoAutonomia(item); err != nil {
		t.Fatalf("upsert autonomia: %v", err)
	}

	got, err := GetProyectoAutonomia(proyectoID)
	if err != nil {
		t.Fatalf("get autonomia: %v", err)
	}
	if got == nil || !got.Enabled || got.MaxWorkers != 4 || got.ObjetivoGeneral == "" {
		t.Fatalf("autonomia persistida inesperada: %+v", got)
	}
	if got.SupervisorAgente != "CodexSupervisor" || got.ReviewerAgente != "CodexReviewer" {
		t.Fatalf("agentes preferidos inesperados: %+v", got)
	}

	enabled := true
	lista, err := ListarProyectosAutonomia(&enabled)
	if err != nil {
		t.Fatalf("listar autonomias activas: %v", err)
	}
	if len(lista) != 1 || lista[0].ProyectoID != proyectoID {
		t.Fatalf("autonomias activas inesperadas: %+v", lista)
	}
}

func TestAutonomiaCyclesRegistrarYFiltrar(t *testing.T) {
	tmp := prepararDBTemporal(t)
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

	if _, err := RegistrarAutonomiaCiclo(&AutonomiaCiclo{
		ProyectoID:   proyectoID,
		Kind:         "supervision",
		Agente:       "Codex3",
		InputJSON:    `{"snapshot":"ok"}`,
		DecisionJSON: `{"accion":"crear_tarea"}`,
		Resultado:    "ok",
	}); err != nil {
		t.Fatalf("registrar ciclo: %v", err)
	}

	kind := "supervision"
	ciclos, err := ListarAutonomiaCiclos(FiltroAutonomiaCiclos{
		ProyectoID: &proyectoID,
		Kind:       &kind,
	})
	if err != nil {
		t.Fatalf("listar ciclos: %v", err)
	}
	if len(ciclos) != 1 || ciclos[0].Agente != "Codex3" {
		t.Fatalf("ciclos inesperados: %+v", ciclos)
	}
}

func TestAutonomiaProyectoReservaYCapacidad(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar Codex3: %v", err)
	}
	if err := RegistrarAgente("Codex4", "programador"); err != nil {
		t.Fatalf("registrar Codex4: %v", err)
	}
	if err := RegistrarAgente("Codex5", "programador"); err != nil {
		t.Fatalf("registrar Codex5: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre IN ('Codex3','Codex4','Codex5')`); err != nil {
		t.Fatalf("marcar agentes disponibles: %v", err)
	}

	proyectoReservaID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador-reserva"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto reserva: %v", err)
	}
	if _, err := UpsertProyectoAutonomia(&ProyectoAutonomia{
		ProyectoID:        proyectoReservaID,
		Enabled:           true,
		SupervisorAgente:  "Codex3",
		ReviewerAgente:    "Codex4",
		ReserveSupervisor: true,
		ReserveReviewer:   true,
		EstadoAutonomia:   AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert autonomia reserva: %v", err)
	}

	if reservado, rol, err := agenteReservadoAutonomiaProyecto("Codex3", proyectoReservaID); err != nil || !reservado || rol != "supervisor" {
		t.Fatalf("reserva supervisor inesperada: reservado=%v rol=%q err=%v", reservado, rol, err)
	}
	if reservado, rol, err := agenteReservadoAutonomiaProyecto("Codex4", proyectoReservaID); err != nil || !reservado || rol != "reviewer" {
		t.Fatalf("reserva reviewer inesperada: reservado=%v rol=%q err=%v", reservado, rol, err)
	}
	if reservado, rol, err := agenteReservadoAutonomiaProyecto("Codex5", proyectoReservaID); err != nil || reservado || rol != "" {
		t.Fatalf("agente no reservado inesperado: reservado=%v rol=%q err=%v", reservado, rol, err)
	}

	proyectoCapacidadID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador-capacidad",
		Nombre:  "Orquestador Capacidad",
		RutaAbs: filepath.Join(tmp, "orquestador-capacidad"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto capacidad: %v", err)
	}
	if _, err := UpsertProyectoAutonomia(&ProyectoAutonomia{
		ProyectoID:      proyectoCapacidadID,
		Enabled:         true,
		MaxWorkers:      1,
		EstadoAutonomia: AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert autonomia capacidad: %v", err)
	}
	if err := ActivarAsignacion("Codex3", proyectoCapacidadID, "worker ocupado"); err != nil {
		t.Fatalf("activar worker ocupado: %v", err)
	}
	admite, err := proyectoAdmiteWorkerAutonomiaParaAgente(proyectoCapacidadID, "Codex2")
	if err != nil {
		t.Fatalf("proyectoAdmiteWorkerAutonomiaParaAgente: %v", err)
	}
	if admite {
		t.Fatalf("max_workers=1 deberia bloquear un worker adicional")
	}

	proyectoPropioID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador-propio",
		Nombre:  "Orquestador Propio",
		RutaAbs: filepath.Join(tmp, "orquestador-propio"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto propio: %v", err)
	}
	if _, err := UpsertProyectoAutonomia(&ProyectoAutonomia{
		ProyectoID:      proyectoPropioID,
		Enabled:         true,
		MaxWorkers:      1,
		EstadoAutonomia: AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert autonomia propio: %v", err)
	}
	if err := ActivarAsignacion("Codex3", proyectoPropioID, "worker propio"); err != nil {
		t.Fatalf("activar worker propio: %v", err)
	}
	admite, err = proyectoAdmiteWorkerAutonomiaParaAgente(proyectoPropioID, "Codex3")
	if err != nil {
		t.Fatalf("proyectoAdmiteWorkerAutonomiaParaAgente propio: %v", err)
	}
	if !admite {
		t.Fatalf("la propia asignacion activa no deberia contarse dos veces")
	}

	proyectoLlenoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador-lleno",
		Nombre:  "Orquestador Lleno",
		RutaAbs: filepath.Join(tmp, "orquestador-lleno"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto lleno: %v", err)
	}
	if _, err := UpsertProyectoAutonomia(&ProyectoAutonomia{
		ProyectoID:      proyectoLlenoID,
		Enabled:         true,
		MaxWorkers:      1,
		EstadoAutonomia: AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert autonomia lleno: %v", err)
	}
	if err := ActivarAsignacion("Codex3", proyectoLlenoID, "worker propio lleno"); err != nil {
		t.Fatalf("activar worker propio lleno: %v", err)
	}
	if err := ActivarAsignacion("Codex4", proyectoLlenoID, "worker extra lleno"); err != nil {
		t.Fatalf("activar worker extra lleno: %v", err)
	}
	admite, err = proyectoAdmiteWorkerAutonomiaParaAgente(proyectoLlenoID, "Codex3")
	if err != nil {
		t.Fatalf("proyectoAdmiteWorkerAutonomiaParaAgente lleno: %v", err)
	}
	if admite {
		t.Fatalf("proyecto lleno deberia bloquear aunque el agente ya tenga asignacion activa")
	}
}

func TestProyectoAutonomiaUpsertPreservaTimestampsOperativosSiNoSeInforman(t *testing.T) {
	tmp := prepararDBTemporal(t)
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

	now := time.Now().UTC().Truncate(time.Second)
	item := &ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app",
		DefinitionOfDoneJSON: `{"done":true}`,
		MaxWorkers:           4,
		SupervisorAgente:     "Codex1",
		EstadoAutonomia:      AutonomiaProyectoActiva,
		LastSupervisionAt:    &now,
	}
	if _, err := UpsertProyectoAutonomia(item); err != nil {
		t.Fatalf("upsert autonomia inicial: %v", err)
	}

	if _, err := UpsertProyectoAutonomia(&ProyectoAutonomia{
		ProyectoID:           proyectoID,
		Enabled:              true,
		ObjetivoGeneral:      "Terminar la app mejor",
		DefinitionOfDoneJSON: `{"done":true}`,
		MaxWorkers:           4,
		SupervisorAgente:     "Codex1",
		EstadoAutonomia:      AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert autonomia sin timestamps: %v", err)
	}

	got, err := GetProyectoAutonomia(proyectoID)
	if err != nil {
		t.Fatalf("get autonomia: %v", err)
	}
	if got == nil || got.LastSupervisionAt == nil {
		t.Fatalf("last_supervision_at perdido: %+v", got)
	}
	if !got.LastSupervisionAt.UTC().Equal(now) {
		t.Fatalf("last_supervision_at inesperado: got=%v want=%v", got.LastSupervisionAt.UTC(), now)
	}
}
