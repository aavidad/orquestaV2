package db

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCupoDeseadoProyectoRespetaObjetivoMinYMax(t *testing.T) {
	t.Run("objetivo_pct", func(t *testing.T) {
		op := &ProyectoOperacion{ObjetivoPct: 40}
		if got := cupoDeseadoProyecto(op, 5); got != 2 {
			t.Fatalf("cupo deseado=%d, want 2", got)
		}
	})

	t.Run("min_agentes", func(t *testing.T) {
		op := &ProyectoOperacion{ObjetivoPct: 10, MinAgentes: 2}
		if got := cupoDeseadoProyecto(op, 3); got != 2 {
			t.Fatalf("cupo deseado=%d, want 2", got)
		}
	})

	t.Run("max_agentes", func(t *testing.T) {
		op := &ProyectoOperacion{ObjetivoPct: 100, MaxAgentes: 1}
		if got := cupoDeseadoProyecto(op, 4); got != 1 {
			t.Fatalf("cupo deseado=%d, want 1", got)
		}
	})
}

func TestMejorProyectoAutomaticoPriorizaDeficitRealAntesQueCargaBruta(t *testing.T) {
	candidatoA := &candidatoProyectoAutomatico{
		Proyecto:   &Proyecto{ID: 1},
		Operacion:  &ProyectoOperacion{Prioridad: 100},
		Activos:    2,
		Deseados:   4,
		Deficit:    2,
		CargaRatio: cargaProyecto(2, 4),
	}
	candidatoB := &candidatoProyectoAutomatico{
		Proyecto:   &Proyecto{ID: 2},
		Operacion:  &ProyectoOperacion{Prioridad: 200},
		Activos:    1,
		Deseados:   2,
		Deficit:    1,
		CargaRatio: cargaProyecto(1, 2),
	}
	if !mejorProyectoAutomatico(candidatoA, candidatoB) {
		t.Fatalf("deberia priorizar mayor deficit real frente a prioridad/carga")
	}
}

func TestMejorProyectoAutomaticoPriorizaTrabajoMicrodirigidoEnPoolCompartido(t *testing.T) {
	candidatoA := &candidatoProyectoAutomatico{
		Proyecto:        &Proyecto{ID: 1},
		Operacion:       &ProyectoOperacion{Prioridad: 100},
		Activos:         1,
		Deseados:        2,
		Deficit:         1,
		CargaRatio:      cargaProyecto(1, 2),
		MicroCerrada:    true,
		ContratoCerrado: true,
	}
	candidatoB := &candidatoProyectoAutomatico{
		Proyecto:        &Proyecto{ID: 2},
		Operacion:       &ProyectoOperacion{Prioridad: 200},
		Activos:         0,
		Deseados:        3,
		Deficit:         3,
		CargaRatio:      cargaProyecto(0, 3),
		MicroCerrada:    false,
		ContratoCerrado: false,
	}
	if !mejorProyectoAutomatico(candidatoA, candidatoB) {
		t.Fatalf("deberia priorizar trabajo microdirigido antes que deficit bruto cuando hay pool compartido")
	}
}

func restringirPlanificadorATestAgentes(t *testing.T, permitidos ...string) {
	t.Helper()
	args := make([]any, 0, len(permitidos))
	placeholders := make([]string, 0, len(permitidos))
	for _, nombre := range permitidos {
		placeholders = append(placeholders, "?")
		args = append(args, nombre)
	}
	query := `UPDATE agentes SET habilitado=0 WHERE rol='programador'`
	if len(placeholders) > 0 {
		query += ` AND nombre NOT IN (` + strings.Join(placeholders, ",") + `)`
	}
	if _, err := DB.Exec(query, args...); err != nil {
		t.Fatalf("restringir agentes planificables: %v", err)
	}
}

func configurarPoolAutobootstrapTest(t *testing.T, proyectoSlug string, workers ...string) {
	t.Helper()
	if err := ConfigSet("server_autobootstrap_project_slug", strings.TrimSpace(proyectoSlug)); err != nil {
		t.Fatalf("config project slug: %v", err)
	}
	if err := ConfigSet("server_autobootstrap_worker_agents", strings.Join(workers, ",")); err != nil {
		t.Fatalf("config worker agents: %v", err)
	}
}

func registrarHandleCuentaCompartidaPausado(t *testing.T, agente, email, usuario string) {
	t.Helper()
	meta := `{"driver":"tmux_cli_session","account_email":"` + strings.TrimSpace(email) + `","account_user":"` + strings.TrimSpace(usuario) + `"}`
	if _, err := DB.Exec(`
		INSERT INTO runtime_handles (
			agente, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
		) VALUES (?,?,?,?,?,?,CURRENT_TIMESTAMP)`,
		strings.TrimSpace(agente), "tmux", "session", "orq-"+strings.ToLower(strings.TrimSpace(agente)), "pausado", meta,
	); err != nil {
		t.Fatalf("registrar handle de cuenta compartida: %v", err)
	}
	runtimeHandleHotReset()
}

func TestPlanificarTareasAutomaticamenteAutoasignaYEncolaStart(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "Codex1")
	configurarPoolAutobootstrapTest(t, "orquestador", "Codex1")

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar agente disponible: %v", err)
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
	if err := ActivarAsignacion("Codex1", proyectoID, "asignacion test"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Implementar arranque autonomo",
		Descripcion: "Cobertura del planificador",
		ProyectoID:  &proyectoID,
		Modulo:      "orquestador",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}

	if err := PlanificarTareasAutomaticamente(); err != nil {
		t.Fatalf("planificar: %v", err)
	}

	tarea, err := GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Agente == nil || *tarea.Agente != "Codex1" || tarea.Estado != TareaAsignada {
		t.Fatalf("tarea no autoasignada correctamente: %+v", tarea)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "start" {
		t.Fatalf("runtime orders inesperadas: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"proyecto":"orquestador"`) {
		t.Fatalf("payload start inesperado: %s", orders[0].PayloadJSON)
	}
}

func TestListarAgentesPlanificablesAgrupaAgentesDeLaMismaCuenta(t *testing.T) {
	prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "Codex1", "Codex5")
	if err := ConfigSet("runtime_shared_account_active_ceiling", "1"); err != nil {
		t.Fatalf("config shared account ceiling: %v", err)
	}
	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("RegistrarAgente Codex1: %v", err)
	}
	if err := RegistrarAgente("Codex5", "programador"); err != nil {
		t.Fatalf("RegistrarAgente Codex5: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre IN ('Codex1','Codex5')`); err != nil {
		t.Fatalf("marcar agentes disponibles: %v", err)
	}
	registrarHandleCuentaCompartidaPausado(t, "Codex1", "maritere@avidad.com", "maritere")
	registrarHandleCuentaCompartidaPausado(t, "Codex5", "maritere@avidad.com", "maritere")

	agentes, err := ListarAgentesPlanificables()
	if err != nil {
		t.Fatalf("ListarAgentesPlanificables: %v", err)
	}
	if len(agentes) != 1 {
		t.Fatalf("deberia quedar solo un planificable por cuenta compartida, got=%d (%+v)", len(agentes), agentes)
	}
	if nombre := strings.TrimSpace(agentes[0].Nombre); nombre != "Codex1" && nombre != "Codex5" {
		t.Fatalf("agente inesperado: %+v", agentes[0])
	}
}

func TestListarAgentesPlanificablesPriorizaAfinidadEnPoolLocalCompartido(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "GemmaA", "GemmaB")

	if err := RegistrarAgente("GemmaA", "programador"); err != nil {
		t.Fatalf("RegistrarAgente GemmaA: %v", err)
	}
	if err := RegistrarAgente("GemmaB", "programador"); err != nil {
		t.Fatalf("RegistrarAgente GemmaB: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible', estado_cuota='activo' WHERE nombre IN ('GemmaA','GemmaB')`); err != nil {
		t.Fatalf("marcar agentes disponibles: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	if _, err := GuardarPool(&PoolCapacidad{
		Slug:                "ollama-gemma4",
		Proveedor:           "Ollama",
		Runtime:             "ollama",
		Plan:                "local",
		CapacidadTotal:      1,
		PermiteModelosMulti: true,
		PoliticaHandoff:     "preventivo",
		FuenteTelemetria:    "manual",
		MetadataJSON:        `{"conector_canonico":"ollama_pool_local","modelo_preferente":"gemma4:26b","slots_maximos":1}`,
		Activo:              true,
	}); err != nil {
		t.Fatalf("GuardarPool: %v", err)
	}
	if _, err := GuardarPoolModelo("ollama-gemma4", &PoolModelo{ModelSlug: "gemma4:26b", Activo: true, Prioridad: 10, CosteRelativo: 1}); err != nil {
		t.Fatalf("GuardarPoolModelo: %v", err)
	}
	if _, err := GuardarPoliticaModelo(&PoliticaModelo{
		ScopeTipo:       "perfil",
		ScopeRef:        "implementacion",
		PerfilTarea:     "implementacion",
		PoolSlug:        "ollama-gemma4",
		ModelSlug:       "gemma4:26b",
		ReasoningEffort: "high",
		Prioridad:       10,
		Activa:          true,
	}); err != nil {
		t.Fatalf("GuardarPoliticaModelo: %v", err)
	}
	if err := ActivarAsignacion("GemmaA", proyectoID, "afinidad_activa"); err != nil {
		t.Fatalf("ActivarAsignacion GemmaA: %v", err)
	}
	if _, err := DB.Exec(`INSERT INTO asignaciones (agente, proyecto_id, estado, nota) VALUES (?,?,?,?)`, "GemmaB", proyectoID, "pausada", "afinidad_pausada"); err != nil {
		t.Fatalf("insert asignacion pausada GemmaB: %v", err)
	}
	if _, err := CrearTarea(&Tarea{
		Titulo:      "Trabajo libre pool local",
		Descripcion: "Debe favorecer al agente con afinidad activa",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	}); err != nil {
		t.Fatalf("CrearTarea: %v", err)
	}

	agentes, err := ListarAgentesPlanificables()
	if err != nil {
		t.Fatalf("ListarAgentesPlanificables: %v", err)
	}
	if len(agentes) != 1 {
		t.Fatalf("deberia quedar un unico planificable por slot efectivo, got=%d (%+v)", len(agentes), agentes)
	}
	if strings.TrimSpace(agentes[0].Nombre) != "GemmaA" {
		t.Fatalf("deberia priorizar GemmaA por afinidad activa, got=%+v", agentes[0])
	}
}

func TestBuscarSiguienteTareaLibreParaAgentePoolLocalPriorizaMicroprogramacion(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "GemmaA")
	if err := RegistrarAgente("GemmaA", "programador"); err != nil {
		t.Fatalf("RegistrarAgente GemmaA: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible', estado_cuota='activo' WHERE nombre='GemmaA'`); err != nil {
		t.Fatalf("marcar agente disponible: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto: %v", err)
	}
	if _, err := GuardarPool(&PoolCapacidad{
		Slug:                "ollama-gemma4",
		Proveedor:           "Ollama",
		Runtime:             "ollama",
		Plan:                "local",
		CapacidadTotal:      1,
		PermiteModelosMulti: true,
		PoliticaHandoff:     "preventivo",
		FuenteTelemetria:    "manual",
		MetadataJSON:        `{"conector_canonico":"ollama_pool_local","modelo_preferente":"gemma4:26b","slots_maximos":1}`,
		Activo:              true,
	}); err != nil {
		t.Fatalf("GuardarPool: %v", err)
	}
	if _, err := GuardarPoolModelo("ollama-gemma4", &PoolModelo{ModelSlug: "gemma4:26b", Activo: true, Prioridad: 10, CosteRelativo: 1}); err != nil {
		t.Fatalf("GuardarPoolModelo: %v", err)
	}
	if _, err := GuardarPoliticaModelo(&PoliticaModelo{
		ScopeTipo:       "perfil",
		ScopeRef:        "implementacion",
		PerfilTarea:     "implementacion",
		PoolSlug:        "ollama-gemma4",
		ModelSlug:       "gemma4:26b",
		ReasoningEffort: "high",
		Prioridad:       10,
		Activa:          true,
	}); err != nil {
		t.Fatalf("GuardarPoliticaModelo: %v", err)
	}
	if err := ActivarAsignacion("GemmaA", proyectoID, "afinidad_activa"); err != nil {
		t.Fatalf("ActivarAsignacion GemmaA: %v", err)
	}
	tareaLibreID, err := CrearTarea(&Tarea{
		Titulo:      "Tarea libre sin contrato",
		Descripcion: "Deberia perder frente a microprogramacion",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("CrearTarea libre: %v", err)
	}
	tareaMicroID, err := CrearTarea(&Tarea{
		Titulo:      "Tarea microprogramada",
		Descripcion: "Debe ganar por especificacion activa",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   PrioridadMedia,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("CrearTarea micro: %v", err)
	}
	if _, err := DB.Exec(`UPDATE tareas SET contrato_definido=1 WHERE id=?`, tareaMicroID); err != nil {
		t.Fatalf("marcar contrato definido: %v", err)
	}
	if _, err := DB.Exec(`
		INSERT INTO especificaciones_funcion (
			tarea_id, proyecto_id, titulo, archivo_objetivo, simbolo_objetivo, descripcion,
			tests_obligatorios_json, write_set_json, estado, creado_por
		) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		tareaMicroID, proyectoID, "Spec micro", "identidad/normalizar.go", "NormalizarIdentificador", "Solo microprogramacion",
		`["go test ./identidad -run TestNormalizarIdentificador -count=1"]`,
		`["identidad/normalizar.go"]`,
		"activa", "alberto",
	); err != nil {
		t.Fatalf("insert especificacion activa: %v", err)
	}

	tarea, err := buscarSiguienteTareaLibreParaAgente("GemmaA", proyectoID)
	if err != nil {
		t.Fatalf("buscarSiguienteTareaLibreParaAgente: %v", err)
	}
	if tarea == nil || tarea.ID != tareaMicroID {
		t.Fatalf("deberia priorizar tarea microprogramada, got=%+v want=%d libre=%d", tarea, tareaMicroID, tareaLibreID)
	}
}

func TestSeleccionarProyectoAutomaticoDisponiblePoolLocalPriorizaProyectoConMicroprogramacion(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "GemmaA")
	if err := RegistrarAgente("GemmaA", "programador"); err != nil {
		t.Fatalf("RegistrarAgente GemmaA: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible', estado_cuota='activo' WHERE nombre='GemmaA'`); err != nil {
		t.Fatalf("marcar agente disponible: %v", err)
	}
	if _, err := GuardarPool(&PoolCapacidad{
		Slug:                "ollama-gemma4",
		Proveedor:           "Ollama",
		Runtime:             "ollama",
		Plan:                "local",
		CapacidadTotal:      1,
		PermiteModelosMulti: true,
		PoliticaHandoff:     "preventivo",
		FuenteTelemetria:    "manual",
		MetadataJSON:        `{"conector_canonico":"ollama_pool_local","modelo_preferente":"gemma4:26b","slots_maximos":1}`,
		Activo:              true,
	}); err != nil {
		t.Fatalf("GuardarPool: %v", err)
	}
	if _, err := GuardarPoolModelo("ollama-gemma4", &PoolModelo{ModelSlug: "gemma4:26b", Activo: true, Prioridad: 10, CosteRelativo: 1}); err != nil {
		t.Fatalf("GuardarPoolModelo: %v", err)
	}
	if _, err := GuardarPoliticaModelo(&PoliticaModelo{
		ScopeTipo:       "perfil",
		ScopeRef:        "implementacion",
		PerfilTarea:     "implementacion",
		PoolSlug:        "ollama-gemma4",
		ModelSlug:       "gemma4:26b",
		ReasoningEffort: "high",
		Prioridad:       10,
		Activa:          true,
	}); err != nil {
		t.Fatalf("GuardarPoliticaModelo: %v", err)
	}
	proyectoMicroID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador",
		Nombre:  "Orquestador",
		RutaAbs: filepath.Join(tmp, "orquestador"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto micro: %v", err)
	}
	proyectoAbiertoID, err := UpsertProyecto(&Proyecto{
		Slug:    "secundario",
		Nombre:  "Secundario",
		RutaAbs: filepath.Join(tmp, "secundario"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("UpsertProyecto abierto: %v", err)
	}
	if err := ActivarAsignacion("GemmaA", proyectoAbiertoID, "arranque sobre proyecto abierto"); err != nil {
		t.Fatalf("ActivarAsignacion proyecto abierto: %v", err)
	}
	if _, err := DB.Exec(`INSERT INTO asignaciones (agente, proyecto_id, estado, nota) VALUES (?,?,?,?)`, "GemmaA", proyectoMicroID, "pausada", "reactivable micro"); err != nil {
		t.Fatalf("insert asignacion pausada proyecto micro: %v", err)
	}
	if _, err := GuardarPoliticaModelo(&PoliticaModelo{
		ScopeTipo:       "proyecto",
		ScopeRef:        "secundario",
		PerfilTarea:     "implementacion",
		PoolSlug:        "ollama-gemma4",
		ModelSlug:       "gemma4:26b",
		ReasoningEffort: "high",
		Prioridad:       10,
		Activa:          true,
	}); err != nil {
		t.Fatalf("GuardarPoliticaModelo secundario: %v", err)
	}
	if _, err := GuardarPoliticaModelo(&PoliticaModelo{
		ScopeTipo:       "proyecto",
		ScopeRef:        "orquestador",
		PerfilTarea:     "implementacion",
		PoolSlug:        "ollama-gemma4",
		ModelSlug:       "gemma4:26b",
		ReasoningEffort: "high",
		Prioridad:       10,
		Activa:          true,
	}); err != nil {
		t.Fatalf("GuardarPoliticaModelo orquestador: %v", err)
	}
	tareaAbiertaID, err := CrearTarea(&Tarea{
		Titulo:      "Trabajo abierto",
		Descripcion: "No deberia ganar frente a microprogramacion",
		ProyectoID:  &proyectoAbiertoID,
		Modulo:      "core",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("CrearTarea abierta: %v", err)
	}
	_ = tareaAbiertaID
	tareaMicroID, err := CrearTarea(&Tarea{
		Titulo:      "Trabajo microprogramado",
		Descripcion: "Debe priorizarse por ser acotado",
		ProyectoID:  &proyectoMicroID,
		Modulo:      "core",
		Prioridad:   PrioridadMedia,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("CrearTarea micro: %v", err)
	}
	if _, err := DB.Exec(`UPDATE tareas SET contrato_definido=1 WHERE id=?`, tareaMicroID); err != nil {
		t.Fatalf("marcar contrato definido: %v", err)
	}
	if _, err := DB.Exec(`
		INSERT INTO especificaciones_funcion (
			tarea_id, proyecto_id, titulo, archivo_objetivo, simbolo_objetivo, descripcion,
			tests_obligatorios_json, write_set_json, estado, creado_por
		) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		tareaMicroID, proyectoMicroID, "Spec micro", "identidad/normalizar.go", "NormalizarIdentificador", "Solo microprogramacion",
		`["go test ./identidad -run TestNormalizarIdentificador -count=1"]`,
		`["identidad/normalizar.go"]`,
		"activa", "alberto",
	); err != nil {
		t.Fatalf("insert especificacion activa: %v", err)
	}

	proyectoID, err := seleccionarProyectoAutomaticoDisponible("GemmaA")
	if err != nil {
		t.Fatalf("seleccionarProyectoAutomaticoDisponible: %v", err)
	}
	if proyectoID != proyectoMicroID {
		t.Fatalf("deberia priorizar proyecto con microprogramacion: got=%d want=%d", proyectoID, proyectoMicroID)
	}
}

func TestEncolarStartAutomaticoSiHaceFaltaRetieneCuentaCompartidaOcupada(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "CodexActivo", "CodexNuevo")
	configurarPoolAutobootstrapTest(t, "orquestador", "CodexActivo", "CodexNuevo")
	if err := ConfigSet("runtime_shared_account_active_ceiling", "1"); err != nil {
		t.Fatalf("config shared account ceiling: %v", err)
	}
	if err := RegistrarAgente("CodexActivo", "programador"); err != nil {
		t.Fatalf("RegistrarAgente CodexActivo: %v", err)
	}
	if err := RegistrarAgente("CodexNuevo", "programador"); err != nil {
		t.Fatalf("RegistrarAgente CodexNuevo: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre IN ('CodexActivo','CodexNuevo')`); err != nil {
		t.Fatalf("marcar agentes disponibles: %v", err)
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
	proyecto, err := GetProyecto(jsonNumber(proyectoID))
	if err != nil {
		t.Fatalf("GetProyecto: %v", err)
	}
	if _, err := DB.Exec(`
		INSERT INTO runtime_handles (
			agente, proyecto_id, transporte, handle_kind, handle_ref, estado, metadata_json, last_seen_at
		) VALUES (?,?,?,?,?,?,?,CURRENT_TIMESTAMP)`,
		"CodexActivo", proyectoID, "tmux", "session", "orq-codexactivo", "activo",
		`{"driver":"tmux_cli_session","account_email":"maritere@avidad.com","account_user":"maritere"}`,
	); err != nil {
		t.Fatalf("registrar handle activo: %v", err)
	}
	registrarHandleCuentaCompartidaPausado(t, "CodexNuevo", "maritere@avidad.com", "maritere")

	if err := EncolarStartAutomaticoSiHaceFalta("CodexNuevo", proyecto, "prueba_cuenta_compartida"); err != nil {
		t.Fatalf("EncolarStartAutomaticoSiHaceFalta: %v", err)
	}

	agente := "CodexNuevo"
	estado := "pendiente"
	orders, err := ListarRuntimeOrdersVivas(FiltroRuntimeOrders{Agente: &agente, Estado: &estado})
	if err != nil {
		t.Fatalf("ListarRuntimeOrdersVivas: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia encolar start si la cuenta compartida ya está ocupada: %+v", orders)
	}
}

func TestEncolarStartAutomaticoSiHaceFaltaOmitePoolLocalCompartidoSinCapacidad(t *testing.T) {
	tmp := prepararDBTemporal(t)
	configurarPoolAutobootstrapTest(t, "orquestador", "GemmaBusy", "GemmaNuevo")

	if err := RegistrarAgente("GemmaBusy", "programador"); err != nil {
		t.Fatalf("RegistrarAgente GemmaBusy: %v", err)
	}
	if err := RegistrarAgente("GemmaNuevo", "programador"); err != nil {
		t.Fatalf("RegistrarAgente GemmaNuevo: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre IN ('GemmaBusy','GemmaNuevo')`); err != nil {
		t.Fatalf("marcar agentes disponibles: %v", err)
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
	proyecto, err := GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("GetProyecto: %v", err)
	}
	if _, err := GuardarPool(&PoolCapacidad{
		Slug:                "ollama-gemma4",
		Proveedor:           "Ollama",
		Runtime:             "ollama",
		Plan:                "local",
		CapacidadTotal:      1,
		CapacidadReservada:  0,
		PermiteModelosMulti: true,
		PoliticaHandoff:     "preventivo",
		FuenteTelemetria:    "manual",
		MetadataJSON:        `{"conector_canonico":"ollama_pool_local","modelo_preferente":"gemma4:26b","slots_maximos":1}`,
		Activo:              true,
	}); err != nil {
		t.Fatalf("GuardarPool: %v", err)
	}
	if _, err := GuardarPoolModelo("ollama-gemma4", &PoolModelo{ModelSlug: "gemma4:26b", Activo: true, Prioridad: 10, CosteRelativo: 1}); err != nil {
		t.Fatalf("GuardarPoolModelo: %v", err)
	}
	if _, err := GuardarPoliticaModelo(&PoliticaModelo{
		ScopeTipo:       "perfil",
		ScopeRef:        "implementacion",
		PerfilTarea:     "implementacion",
		PoolSlug:        "ollama-gemma4",
		ModelSlug:       "gemma4:26b",
		ReasoningEffort: "high",
		Prioridad:       10,
		Activa:          true,
	}); err != nil {
		t.Fatalf("GuardarPoliticaModelo: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "GemmaBusy",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "gemma-busy"),
		Herramienta: "ollama_pool_local",
	})
	if err != nil {
		t.Fatalf("IniciarSesionContexto GemmaBusy: %v", err)
	}
	pool, err := GetPool("ollama-gemma4")
	if err != nil {
		t.Fatalf("GetPool: %v", err)
	}
	if _, err := DB.Exec(`UPDATE sesiones SET pool_id = ? WHERE id = ?`, pool.ID, sesion.ID); err != nil {
		t.Fatalf("update sesiones.pool_id: %v", err)
	}

	if err := EncolarStartAutomaticoSiHaceFalta("GemmaNuevo", proyecto, "pool_local_sin_capacidad"); err != nil {
		t.Fatalf("EncolarStartAutomaticoSiHaceFalta: %v", err)
	}

	agente := "GemmaNuevo"
	estado := "pendiente"
	orders, err := ListarRuntimeOrdersVivas(FiltroRuntimeOrders{Agente: &agente, Estado: &estado})
	if err != nil {
		t.Fatalf("ListarRuntimeOrdersVivas: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia encolar start si el pool local compartido no tiene slots: %+v", orders)
	}
}

func TestEncolarStartAutomaticoSiHaceFaltaOmitePoolLocalCompartidoConReservaPendiente(t *testing.T) {
	tmp := prepararDBTemporal(t)
	configurarPoolAutobootstrapTest(t, "orquestador", "GemmaA", "GemmaB")

	if err := RegistrarAgente("GemmaA", "programador"); err != nil {
		t.Fatalf("RegistrarAgente GemmaA: %v", err)
	}
	if err := RegistrarAgente("GemmaB", "programador"); err != nil {
		t.Fatalf("RegistrarAgente GemmaB: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre IN ('GemmaA','GemmaB')`); err != nil {
		t.Fatalf("marcar agentes disponibles: %v", err)
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
	proyecto, err := GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("GetProyecto: %v", err)
	}
	if _, err := GuardarPool(&PoolCapacidad{
		Slug:                "ollama-gemma4",
		Proveedor:           "Ollama",
		Runtime:             "ollama",
		Plan:                "local",
		CapacidadTotal:      1,
		PermiteModelosMulti: true,
		PoliticaHandoff:     "preventivo",
		FuenteTelemetria:    "manual",
		MetadataJSON:        `{"conector_canonico":"ollama_pool_local","modelo_preferente":"gemma4:26b","slots_maximos":1}`,
		Activo:              true,
	}); err != nil {
		t.Fatalf("GuardarPool: %v", err)
	}
	if _, err := GuardarPoolModelo("ollama-gemma4", &PoolModelo{ModelSlug: "gemma4:26b", Activo: true, Prioridad: 10, CosteRelativo: 1}); err != nil {
		t.Fatalf("GuardarPoolModelo: %v", err)
	}
	if _, err := GuardarPoliticaModelo(&PoliticaModelo{
		ScopeTipo:       "perfil",
		ScopeRef:        "implementacion",
		PerfilTarea:     "implementacion",
		PoolSlug:        "ollama-gemma4",
		ModelSlug:       "gemma4:26b",
		ReasoningEffort: "high",
		Prioridad:       10,
		Activa:          true,
	}); err != nil {
		t.Fatalf("GuardarPoliticaModelo: %v", err)
	}
	payloadJSON, err := json.Marshal(map[string]any{
		"accion":   "start",
		"proyecto": "orquestador",
		"motivo":   "reserva_previa",
		"por":      "test",
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	if _, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "GemmaA",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		PayloadJSON: string(payloadJSON),
	}); err != nil {
		t.Fatalf("EncolarRuntimeOrder GemmaA: %v", err)
	}

	if err := EncolarStartAutomaticoSiHaceFalta("GemmaB", proyecto, "pool_local_reserva_pendiente"); err != nil {
		t.Fatalf("EncolarStartAutomaticoSiHaceFalta: %v", err)
	}

	agente := "GemmaB"
	estado := "pendiente"
	orders, err := ListarRuntimeOrdersVivas(FiltroRuntimeOrders{Agente: &agente, Estado: &estado})
	if err != nil {
		t.Fatalf("ListarRuntimeOrdersVivas: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia encolar start si otro agente ya reservo el slot del pool: %+v", orders)
	}
}

func TestPlanificarTareasAutomaticamenteEncolaStartParaTrabajoYaAsignado(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "Codex1")
	configurarPoolAutobootstrapTest(t, "orquestador", "Codex1")

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar agente disponible: %v", err)
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
	if err := ActivarAsignacion("Codex1", proyectoID, "asignacion test"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Retomar trabajo ya asignado",
		Descripcion: "Cobertura de autoarranque",
		ProyectoID:  &proyectoID,
		Modulo:      "orquestador",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}

	if err := PlanificarTareasAutomaticamente(); err != nil {
		t.Fatalf("planificar: %v", err)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "start" {
		t.Fatalf("runtime orders inesperadas: %+v", orders)
	}
}

func TestPlanificarTareasAutomaticamenteRecuperaTareaHuerfanaYLaReasigna(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "CodexStale", "CodexNuevo")
	configurarPoolAutobootstrapTest(t, "orquestador", "CodexStale", "CodexNuevo")

	if err := RegistrarAgente("CodexStale", "programador"); err != nil {
		t.Fatalf("registrar agente stale: %v", err)
	}
	if err := RegistrarAgente("CodexNuevo", "programador"); err != nil {
		t.Fatalf("registrar agente nuevo: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre IN ('CodexStale','CodexNuevo')`); err != nil {
		t.Fatalf("marcar agentes disponibles: %v", err)
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
	if err := ActivarAsignacion("CodexNuevo", proyectoID, "frente libre"); err != nil {
		t.Fatalf("activar asignacion nueva: %v", err)
	}
	if err := ConfigSet("orphan_task_recovery_grace_seconds", "0"); err != nil {
		t.Fatalf("config grace stale: %v", err)
	}
	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:      "CodexStale",
		CWD:         filepath.Join(tmp, "stale"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion stale: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Recuperar frente huérfano",
		Descripcion: "Debe volver al pool y reasignarse",
		ProyectoID:  &proyectoID,
		Modulo:      "orquestador",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := TomarTarea(tareaID, "CodexStale"); err != nil {
		t.Fatalf("tomar tarea stale: %v", err)
	}

	if err := PlanificarTareasAutomaticamente(); err != nil {
		t.Fatalf("planificar: %v", err)
	}

	tarea, err := GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	agenteAsignado := "<nil>"
	if tarea.Agente != nil {
		agenteAsignado = *tarea.Agente
	}
	if tarea.Agente == nil || *tarea.Agente != "CodexNuevo" || tarea.Estado != TareaAsignada {
		t.Fatalf("la tarea huérfana debería reasignarse al nuevo agente, agente=%s estado=%s notas=%q", agenteAsignado, tarea.Estado, tarea.Notas)
	}
	agente := "CodexNuevo"
	estado := "pendiente"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "start" {
		t.Fatalf("debería encolar start para el nuevo agente: %+v", orders)
	}
}

func TestBuscarSiguienteTareaLibreParaAgenteEvitaModuloYaOcupadoSiHayAlternativa(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
	}
	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
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
	tareaOcupada, err := CrearTarea(&Tarea{
		Titulo:      "Trabajo runtime en curso",
		Descripcion: "ocupado por otro agente",
		ProyectoID:  &proyectoID,
		Modulo:      "runtime",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea ocupada: %v", err)
	}
	if err := TomarTarea(tareaOcupada, "Codex2"); err != nil {
		t.Fatalf("tomar tarea ocupada: %v", err)
	}
	if err := IniciarTarea(tareaOcupada, "Codex2"); err != nil {
		t.Fatalf("iniciar tarea ocupada: %v", err)
	}
	if _, err := CrearTarea(&Tarea{
		Titulo:      "Nueva tarea runtime",
		Descripcion: "mismo modulo ya ocupado",
		ProyectoID:  &proyectoID,
		Modulo:      "runtime",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	}); err != nil {
		t.Fatalf("crear tarea libre runtime: %v", err)
	}
	tareaWeb, err := CrearTarea(&Tarea{
		Titulo:      "Nueva tarea web",
		Descripcion: "modulo libre",
		ProyectoID:  &proyectoID,
		Modulo:      "web",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea libre web: %v", err)
	}

	tarea, err := buscarSiguienteTareaLibreParaAgente("Codex1", proyectoID)
	if err != nil {
		t.Fatalf("buscarSiguienteTareaLibreParaAgente: %v", err)
	}
	if tarea == nil || tarea.ID != tareaWeb {
		t.Fatalf("deberia evitar el modulo ya ocupado si hay alternativa, got=%+v want=%d", tarea, tareaWeb)
	}
}

func TestPlanificarTareasAutomaticamenteNoRecuperaTareaManualFueraDeFlota(t *testing.T) {
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
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Frente manual del core",
		Descripcion: "Tomado por operador manual fuera de la flota",
		ProyectoID:  &proyectoID,
		Modulo:      "runtime-adapters",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := TomarTarea(tareaID, "alberto"); err != nil {
		t.Fatalf("tomar tarea manual: %v", err)
	}
	if err := IniciarTarea(tareaID, "alberto"); err != nil {
		t.Fatalf("iniciar tarea manual: %v", err)
	}
	if _, err := DB.Exec(`UPDATE tareas SET updated_at=? WHERE id=?`, time.Now().UTC().Add(-10*time.Minute), tareaID); err != nil {
		t.Fatalf("forzar updated_at antiguo: %v", err)
	}

	if err := PlanificarTareasAutomaticamente(); err != nil {
		t.Fatalf("planificar: %v", err)
	}

	tarea, err := GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Agente == nil || *tarea.Agente != "alberto" {
		got := "<nil>"
		if tarea.Agente != nil {
			got = *tarea.Agente
		}
		t.Fatalf("la tarea manual no deberia recuperarse del operador fuera de flota, agente=%s estado=%s notas=%q", got, tarea.Estado, tarea.Notas)
	}
	if tarea.Estado != TareaEnProgreso {
		t.Fatalf("la tarea manual deberia seguir en progreso, estado=%s notas=%q", tarea.Estado, tarea.Notas)
	}
	if strings.Contains(strings.ToLower(tarea.Notas), "continuidad huérfana") || strings.Contains(strings.ToLower(tarea.Notas), "continuidad huerfana") {
		t.Fatalf("no deberia anotar recuperación de continuidad huérfana: %q", tarea.Notas)
	}
}

func TestBuscarSiguienteTareaLibreParaAgentePrefiereAfinidadDeModuloSinSolape(t *testing.T) {
	tmp := prepararDBTemporal(t)

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar Codex1: %v", err)
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
	tareaHist, err := CrearTarea(&Tarea{
		Titulo:      "Frente web previo",
		Descripcion: "afinidad previa",
		ProyectoID:  &proyectoID,
		Modulo:      "web",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea historica: %v", err)
	}
	if err := TomarTarea(tareaHist, "Codex1"); err != nil {
		t.Fatalf("tomar tarea historica: %v", err)
	}
	if err := IniciarTarea(tareaHist, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea historica: %v", err)
	}
	if err := CompletarTarea(tareaHist, "Codex1", "sha-test"); err != nil {
		t.Fatalf("completar tarea historica: %v", err)
	}
	tareaWeb, err := CrearTarea(&Tarea{
		Titulo:      "Seguir web",
		Descripcion: "deberia respetar afinidad",
		ProyectoID:  &proyectoID,
		Modulo:      "web",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea web: %v", err)
	}
	if _, err := CrearTarea(&Tarea{
		Titulo:      "Trabajo api alternativo",
		Descripcion: "misma prioridad, distinto modulo",
		ProyectoID:  &proyectoID,
		Modulo:      "api",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	}); err != nil {
		t.Fatalf("crear tarea api: %v", err)
	}

	tarea, err := buscarSiguienteTareaLibreParaAgente("Codex1", proyectoID)
	if err != nil {
		t.Fatalf("buscarSiguienteTareaLibreParaAgente: %v", err)
	}
	if tarea == nil || tarea.ID != tareaWeb {
		t.Fatalf("deberia preferir afinidad de modulo, got=%+v want=%d", tarea, tareaWeb)
	}
}

func TestPlanificarTareasAutomaticamenteNoRecuperaTareaRecienTomada(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "CodexStale", "CodexNuevo")

	if err := RegistrarAgente("CodexStale", "programador"); err != nil {
		t.Fatalf("registrar agente stale: %v", err)
	}
	if err := RegistrarAgente("CodexNuevo", "programador"); err != nil {
		t.Fatalf("registrar agente nuevo: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre IN ('CodexStale','CodexNuevo')`); err != nil {
		t.Fatalf("marcar agentes disponibles: %v", err)
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
	if err := ActivarAsignacion("CodexNuevo", proyectoID, "frente libre"); err != nil {
		t.Fatalf("activar asignacion nueva: %v", err)
	}
	if err := ConfigSet("orphan_task_recovery_grace_seconds", "300"); err != nil {
		t.Fatalf("config grace recent: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Mantener tarea recién tomada",
		Descripcion: "No debe recuperarse durante la ventana de gracia",
		ProyectoID:  &proyectoID,
		Modulo:      "orquestador",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := TomarTarea(tareaID, "CodexStale"); err != nil {
		t.Fatalf("tomar tarea reciente: %v", err)
	}

	if err := PlanificarTareasAutomaticamente(); err != nil {
		t.Fatalf("planificar: %v", err)
	}

	tarea, err := GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Agente == nil || *tarea.Agente != "CodexStale" || tarea.Estado != TareaAsignada {
		t.Fatalf("la tarea recién tomada no debería liberarse todavía: %+v", tarea)
	}
	agente := "CodexNuevo"
	estado := "pendiente"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no debería arrancar un nuevo agente mientras la tarea sigue protegida: %+v", orders)
	}
}

func TestPlanificarTareasAutomaticamenteLiberaTareaAsignadaDeAgenteEnCuotaYLaReasigna(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "CodexAgotado", "CodexNuevo")
	configurarPoolAutobootstrapTest(t, "orquestador", "CodexAgotado", "CodexNuevo")

	if err := RegistrarAgente("CodexAgotado", "programador"); err != nil {
		t.Fatalf("registrar agente agotado: %v", err)
	}
	if err := RegistrarAgente("CodexNuevo", "programador"); err != nil {
		t.Fatalf("registrar agente nuevo: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre IN ('CodexAgotado','CodexNuevo')`); err != nil {
		t.Fatalf("marcar agentes disponibles: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_cuota='enfriamiento', motivo_pausa='presupuesto agotado' WHERE nombre='CodexAgotado'`); err != nil {
		t.Fatalf("marcar agente agotado: %v", err)
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
	if err := ActivarAsignacion("CodexNuevo", proyectoID, "frente libre"); err != nil {
		t.Fatalf("activar asignacion nueva: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Repartir frente retenido por cuota",
		Descripcion: "Debe volver al pool y reasignarse",
		ProyectoID:  &proyectoID,
		Modulo:      "orquestador",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := TomarTarea(tareaID, "CodexAgotado"); err != nil {
		t.Fatalf("tomar tarea agotado: %v", err)
	}

	if err := PlanificarTareasAutomaticamente(); err != nil {
		t.Fatalf("planificar: %v", err)
	}

	tarea, err := GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	agenteAsignado := "<nil>"
	if tarea.Agente != nil {
		agenteAsignado = *tarea.Agente
	}
	if tarea.Agente == nil || *tarea.Agente != "CodexNuevo" || tarea.Estado != TareaAsignada {
		t.Fatalf("la tarea retenida por cuota debería reasignarse al nuevo agente, agente=%s estado=%s notas=%q", agenteAsignado, tarea.Estado, tarea.Notas)
	}
	if !strings.Contains(strings.ToLower(tarea.Notas), "cuota enfriamiento") {
		t.Fatalf("deberia anotar la liberacion por cuota: %q", tarea.Notas)
	}
	agente := "CodexNuevo"
	estado := "pendiente"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "start" {
		t.Fatalf("debería encolar start para el nuevo agente: %+v", orders)
	}
}

func TestPlanificarTareasAutomaticamenteLiberaTareaPorPresupuestoObservadoFresco(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "CodexAgotado", "CodexNuevo")
	configurarPoolAutobootstrapTest(t, "orquestador", "CodexAgotado", "CodexNuevo")

	if err := RegistrarAgente("CodexAgotado", "programador"); err != nil {
		t.Fatalf("registrar agente agotado: %v", err)
	}
	if err := RegistrarAgente("CodexNuevo", "programador"); err != nil {
		t.Fatalf("registrar agente nuevo: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre IN ('CodexAgotado','CodexNuevo')`); err != nil {
		t.Fatalf("marcar agentes disponibles: %v", err)
	}
	if err := ConfigSet("pool_budget_snapshot_observed_max_age_seconds", "3600"); err != nil {
		t.Fatalf("config observed snapshot max age: %v", err)
	}
	sesionID, err := IniciarSesion("CodexAgotado")
	if err != nil {
		t.Fatalf("iniciar sesion agotado: %v", err)
	}
	credits := 0.0
	reset := time.Now().UTC().Add(90 * time.Minute)
	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:         sesionID,
		WindowKind:       "5h",
		ResetAt:          &reset,
		RemainingCredits: &credits,
		BudgetSource:     "codex_token_count_observed",
		CheckedAt:        time.Now().UTC().Add(-10 * time.Minute),
	}); err != nil {
		t.Fatalf("registrar presupuesto agotado: %v", err)
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
	if err := ActivarAsignacion("CodexNuevo", proyectoID, "frente libre"); err != nil {
		t.Fatalf("activar asignacion nueva: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Repartir frente retenido por presupuesto observado",
		Descripcion: "Debe volver al pool y reasignarse",
		ProyectoID:  &proyectoID,
		Modulo:      "orquestador",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := TomarTarea(tareaID, "CodexAgotado"); err != nil {
		t.Fatalf("tomar tarea agotado: %v", err)
	}

	if err := PlanificarTareasAutomaticamente(); err != nil {
		t.Fatalf("planificar: %v", err)
	}

	tarea, err := GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Agente == nil || *tarea.Agente != "CodexNuevo" || tarea.Estado != TareaAsignada {
		t.Fatalf("la tarea retenida por presupuesto observado debería reasignarse: %+v", tarea)
	}
	if !strings.Contains(strings.ToLower(tarea.Notas), "cuota enfriamiento") {
		t.Fatalf("deberia anotar la liberacion por cuota observada: %q", tarea.Notas)
	}
}

func TestListarAgentesPlanificablesExcluyeAgenteConPresupuestoObservadoCritico(t *testing.T) {
	abrirDBTemporalMemoria(t)

	if err := RegistrarAgente("CodexAgotado", "programador"); err != nil {
		t.Fatalf("registrar agente agotado: %v", err)
	}
	if err := RegistrarAgente("CodexNuevo", "programador"); err != nil {
		t.Fatalf("registrar agente nuevo: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre IN ('CodexAgotado','CodexNuevo')`); err != nil {
		t.Fatalf("marcar agentes disponibles: %v", err)
	}
	if err := ConfigSet("pool_budget_snapshot_observed_max_age_seconds", "3600"); err != nil {
		t.Fatalf("config observed snapshot max age: %v", err)
	}
	sesionID, err := IniciarSesion("CodexAgotado")
	if err != nil {
		t.Fatalf("iniciar sesion agotado: %v", err)
	}
	credits := 0.0
	reset := time.Now().UTC().Add(90 * time.Minute)
	if _, err := RegistrarPresupuestoSesion(&PresupuestoSesion{
		SesionID:         sesionID,
		WindowKind:       "5h",
		ResetAt:          &reset,
		RemainingCredits: &credits,
		BudgetSource:     "codex_token_count_observed",
		CheckedAt:        time.Now().UTC().Add(-10 * time.Minute),
	}); err != nil {
		t.Fatalf("registrar presupuesto agotado: %v", err)
	}

	agentes, err := ListarAgentesPlanificables()
	if err != nil {
		t.Fatalf("ListarAgentesPlanificables: %v", err)
	}
	for _, agente := range agentes {
		if agente != nil && agente.Nombre == "CodexAgotado" {
			t.Fatalf("CodexAgotado no deberia ser planificable con presupuesto observado critico: %+v", agentes)
		}
	}
	var foundNuevo bool
	for _, agente := range agentes {
		if agente != nil && agente.Nombre == "CodexNuevo" {
			foundNuevo = true
		}
	}
	if !foundNuevo {
		t.Fatalf("CodexNuevo deberia seguir siendo planificable: %+v", agentes)
	}
}

func TestListarAgentesPlanificablesAgrupaCuentaCompartidaLibre(t *testing.T) {
	abrirDBTemporalMemoria(t)
	restringirPlanificadorATestAgentes(t, "Codex2", "Codex3")

	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	if err := RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar Codex3: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible', estado_cuota='activo' WHERE nombre IN ('Codex2','Codex3')`); err != nil {
		t.Fatalf("marcar agentes disponibles: %v", err)
	}
	now := time.Now().UTC()
	if err := UpsertAgenteIdentidadObservada("Codex2", "shared@example.com", "shared", "test", &now); err != nil {
		t.Fatalf("identidad Codex2: %v", err)
	}
	if err := UpsertAgenteIdentidadObservada("Codex3", "shared@example.com", "shared", "test", &now); err != nil {
		t.Fatalf("identidad Codex3: %v", err)
	}

	agentes, err := ListarAgentesPlanificables()
	if err != nil {
		t.Fatalf("ListarAgentesPlanificables: %v", err)
	}
	if len(agentes) != 1 {
		t.Fatalf("deberia dejar un unico planificable por cuenta compartida libre, got=%d %+v", len(agentes), agentes)
	}
	if agentes[0] == nil || agentes[0].Nombre != "Codex2" {
		t.Fatalf("deberia conservar el primer agente estable del grupo, got=%+v", agentes[0])
	}
}

func TestListarAgentesPlanificablesNoAgrupaEmailsIgualesConAccountIDDistinto(t *testing.T) {
	abrirDBTemporalMemoria(t)
	restringirPlanificadorATestAgentes(t, "Codex2", "Codex3")

	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	if err := RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar Codex3: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible', estado_cuota='activo' WHERE nombre IN ('Codex2','Codex3')`); err != nil {
		t.Fatalf("marcar agentes disponibles: %v", err)
	}
	now := time.Now().UTC()
	if err := UpsertAgenteIdentidadObservadaCanonica("Codex2", "acct-codex2", "shared@example.com", "shared", "test", &now); err != nil {
		t.Fatalf("identidad Codex2: %v", err)
	}
	if err := UpsertAgenteIdentidadObservadaCanonica("Codex3", "acct-codex3", "shared@example.com", "shared", "test", &now); err != nil {
		t.Fatalf("identidad Codex3: %v", err)
	}

	agentes, err := ListarAgentesPlanificables()
	if err != nil {
		t.Fatalf("ListarAgentesPlanificables: %v", err)
	}
	if len(agentes) != 2 {
		t.Fatalf("emails iguales con account_id distinto no deberian agruparse: got=%d %+v", len(agentes), agentes)
	}
}

func TestEncolarStartAutomaticoSiHaceFaltaOmiteCuentaCompartidaOcupada(t *testing.T) {
	tmp := prepararDBTemporal(t)
	configurarPoolAutobootstrapTest(t, "orquestador", "Codex2", "Codex3")

	if err := RegistrarAgente("Codex2", "programador"); err != nil {
		t.Fatalf("registrar Codex2: %v", err)
	}
	if err := RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar Codex3: %v", err)
	}
	now := time.Now().UTC()
	if err := UpsertAgenteIdentidadObservada("Codex2", "shared@example.com", "shared", "test", &now); err != nil {
		t.Fatalf("identidad Codex2: %v", err)
	}
	if err := UpsertAgenteIdentidadObservada("Codex3", "shared@example.com", "shared", "test", &now); err != nil {
		t.Fatalf("identidad Codex3: %v", err)
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
	proyecto, err := GetProyecto("orquestador")
	if err != nil {
		t.Fatalf("get proyecto: %v", err)
	}
	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:            "Codex2",
		ProyectoID:        &proyectoID,
		CWD:               filepath.Join(tmp, "codex2"),
		Herramienta:       "codex-cli",
		ExternalSessionID: "sess-codex2",
	}); err != nil {
		t.Fatalf("iniciar sesion Codex2: %v", err)
	}

	if err := EncolarStartAutomaticoSiHaceFalta("Codex3", proyecto, "shared_account_test"); err != nil {
		t.Fatalf("EncolarStartAutomaticoSiHaceFalta: %v", err)
	}

	agente := "Codex3"
	estado := "pendiente"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia encolar start para una cuenta compartida ya ocupada: %+v", orders)
	}
}

func TestPlanificarTareasAutomaticamenteNoRecuperaTareaConSesionPausada(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "CodexStale", "CodexNuevo")

	if err := RegistrarAgente("CodexStale", "programador"); err != nil {
		t.Fatalf("registrar agente stale: %v", err)
	}
	if err := RegistrarAgente("CodexNuevo", "programador"); err != nil {
		t.Fatalf("registrar agente nuevo: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre IN ('CodexStale','CodexNuevo')`); err != nil {
		t.Fatalf("marcar agentes disponibles: %v", err)
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
	if err := ActivarAsignacion("CodexNuevo", proyectoID, "frente libre"); err != nil {
		t.Fatalf("activar asignacion nueva: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Mantener continuidad pausada",
		Descripcion: "No debe reasignarse si hay sesión pausada viva",
		ProyectoID:  &proyectoID,
		Modulo:      "orquestador",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := TomarTarea(tareaID, "CodexStale"); err != nil {
		t.Fatalf("tomar tarea stale: %v", err)
	}
	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:      "CodexStale",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion stale: %v", err)
	}
	if err := AparcarSesionActiva("CodexStale", &proyectoID); err != nil {
		t.Fatalf("aparcar sesion stale: %v", err)
	}

	if err := PlanificarTareasAutomaticamente(); err != nil {
		t.Fatalf("planificar: %v", err)
	}

	tarea, err := GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Agente == nil || *tarea.Agente != "CodexStale" || tarea.Estado != TareaAsignada {
		t.Fatalf("la tarea no debería reasignarse mientras exista sesión pausada: %+v", tarea)
	}
	agente := "CodexNuevo"
	estado := "pendiente"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no debería arrancar el nuevo agente si la tarea sigue protegida: %+v", orders)
	}
}

func TestPlanificarTareasAutomaticamenteEncolaResumeParaHandlePausado(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "CodexResume")

	if err := RegistrarAgente("CodexResume", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre='CodexResume'`); err != nil {
		t.Fatalf("marcar agente disponible: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "orquesta",
		Nombre:  "orquesta",
		RutaAbs: filepath.Join(tmp, "orquesta"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := ActivarAsignacion("CodexResume", proyectoID, "frente actual"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Retomar frente pausado",
		Descripcion: "El planificador debe encolar resume",
		ProyectoID:  &proyectoID,
		Modulo:      "orquestador",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := TomarTarea(tareaID, "CodexResume"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := IniciarTarea(tareaID, "CodexResume"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}
	sesion, err := IniciarSesionContexto(SesionInicio{
		Agente:      "CodexResume",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquesta"),
		Herramienta: "codex-cli",
	})
	if err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if err := UpsertRuntimeHandleDesdeSesion(sesion); err != nil {
		t.Fatalf("upsert runtime handle: %v", err)
	}
	handle, err := GetRuntimeHandleBySesionID(sesion.ID)
	if err != nil || handle == nil {
		t.Fatalf("obtener handle: handle=%+v err=%v", handle, err)
	}
	if _, err := DB.Exec(`UPDATE sesiones SET estado='pausada' WHERE id=?`, sesion.ID); err != nil {
		t.Fatalf("pausar sesion: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_handles SET estado='pausado' WHERE id=?`, handle.ID); err != nil {
		t.Fatalf("pausar handle: %v", err)
	}

	if err := PlanificarTareasAutomaticamente(); err != nil {
		t.Fatalf("planificar: %v", err)
	}

	agente := "CodexResume"
	estado := "pendiente"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "resume" {
		t.Fatalf("debería encolar resume para el handle pausado: %+v", orders)
	}
	if !strings.Contains(orders[0].PayloadJSON, `"accion":"resume"`) {
		t.Fatalf("payload resume inesperado: %s", orders[0].PayloadJSON)
	}
}

func TestPlanificarTareasAutomaticamenteNoAutoasignaTrabajoASupervisorReservado(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "CodexSupervisor")

	if err := RegistrarAgente("CodexSupervisor", "programador"); err != nil {
		t.Fatalf("registrar supervisor: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre='CodexSupervisor'`); err != nil {
		t.Fatalf("marcar supervisor disponible: %v", err)
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
	if _, err := UpsertProyectoAutonomia(&ProyectoAutonomia{
		ProyectoID:        proyectoID,
		Enabled:           true,
		MaxWorkers:        1,
		SupervisorAgente:  "CodexSupervisor",
		ReserveSupervisor: true,
		EstadoAutonomia:   AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert autonomia: %v", err)
	}
	if err := ActivarAsignacion("CodexSupervisor", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion supervisor: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "No tocar supervisor",
		Descripcion: "Debe quedar libre para otro worker",
		ProyectoID:  &proyectoID,
		Modulo:      "orquestador",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}

	if err := PlanificarTareasAutomaticamente(); err != nil {
		t.Fatalf("planificar: %v", err)
	}

	tarea, err := GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Estado != TareaLibre || tarea.Agente != nil {
		t.Fatalf("el supervisor reservado no deberia autoasignarse trabajo: %+v", tarea)
	}
	agente := "CodexSupervisor"
	estado := "pendiente"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders supervisor: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("supervisor reservado no deberia recibir start por trabajo libre: %+v", orders)
	}
}

func TestPlanificarTareasAutomaticamenteNoCuentaSupervisorReservadoComoWorker(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "CodexSupervisor", "CodexWorker")
	configurarPoolAutobootstrapTest(t, "orquestador", "CodexSupervisor", "CodexWorker")

	if err := RegistrarAgente("CodexSupervisor", "programador"); err != nil {
		t.Fatalf("registrar supervisor: %v", err)
	}
	if err := RegistrarAgente("CodexWorker", "programador"); err != nil {
		t.Fatalf("registrar worker: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre IN ('CodexSupervisor','CodexWorker')`); err != nil {
		t.Fatalf("marcar agentes disponibles: %v", err)
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
	if _, err := UpsertProyectoAutonomia(&ProyectoAutonomia{
		ProyectoID:        proyectoID,
		Enabled:           true,
		MaxWorkers:        1,
		SupervisorAgente:  "CodexSupervisor",
		ReserveSupervisor: true,
		EstadoAutonomia:   AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert autonomia: %v", err)
	}
	if err := ActivarAsignacion("CodexSupervisor", proyectoID, "supervision"); err != nil {
		t.Fatalf("activar asignacion supervisor: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Worker libre",
		Descripcion: "El supervisor reservado no debe consumir el cupo de worker",
		ProyectoID:  &proyectoID,
		Modulo:      "orquestador",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}

	if err := PlanificarTareasAutomaticamente(); err != nil {
		t.Fatalf("planificar: %v", err)
	}

	tarea, err := GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Agente == nil || *tarea.Agente != "CodexWorker" || tarea.Estado != TareaAsignada {
		t.Fatalf("el worker deberia tomar la tarea y no el supervisor reservado: %+v", tarea)
	}
}

func TestPlanificarTareasAutomaticamenteRespetaMaxWorkersAutonomia(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "CodexBusy", "CodexWorker")

	if err := RegistrarAgente("CodexBusy", "programador"); err != nil {
		t.Fatalf("registrar worker ocupado: %v", err)
	}
	if err := RegistrarAgente("CodexWorker", "programador"); err != nil {
		t.Fatalf("registrar worker libre: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre IN ('CodexBusy','CodexWorker')`); err != nil {
		t.Fatalf("marcar agentes disponibles: %v", err)
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
	if _, err := UpsertProyectoAutonomia(&ProyectoAutonomia{
		ProyectoID:      proyectoID,
		Enabled:         true,
		MaxWorkers:      1,
		EstadoAutonomia: AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert autonomia: %v", err)
	}
	if err := ActivarAsignacion("CodexBusy", proyectoID, "worker ocupado"); err != nil {
		t.Fatalf("activar asignacion ocupada: %v", err)
	}
	tareaBusyID, err := CrearTarea(&Tarea{
		Titulo:      "Trabajo ya ocupado",
		Descripcion: "Mantiene el cupo lleno",
		ProyectoID:  &proyectoID,
		Modulo:      "orquestador",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea busy: %v", err)
	}
	if err := TomarTarea(tareaBusyID, "CodexBusy"); err != nil {
		t.Fatalf("tomar tarea busy: %v", err)
	}
	tareaLibreID, err := CrearTarea(&Tarea{
		Titulo:      "No debe entrar otro worker",
		Descripcion: "Max workers = 1",
		ProyectoID:  &proyectoID,
		Modulo:      "orquestador",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea libre: %v", err)
	}

	if err := PlanificarTareasAutomaticamente(); err != nil {
		t.Fatalf("planificar: %v", err)
	}

	tareaLibre, err := GetTarea(tareaLibreID)
	if err != nil {
		t.Fatalf("get tarea libre: %v", err)
	}
	if tareaLibre.Estado != TareaLibre || tareaLibre.Agente != nil {
		t.Fatalf("max_workers deberia impedir un segundo worker: %+v", tareaLibre)
	}
	agente := "CodexWorker"
	estado := "pendiente"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders worker: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("max_workers no deberia encolar trabajo para un segundo worker: %+v", orders)
	}
}

func TestPlanificarTareasAutomaticamenteNoCuentaWorkerEnCuotaParaMaxWorkers(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "CodexBloqueado", "CodexWorker")
	if err := ConfigSet("server_autobootstrap_project_slug", "orquestador"); err != nil {
		t.Fatalf("config project slug: %v", err)
	}
	if err := ConfigSet("server_autobootstrap_worker_agents", "CodexWorker"); err != nil {
		t.Fatalf("config worker agents: %v", err)
	}

	if err := RegistrarAgente("CodexBloqueado", "programador"); err != nil {
		t.Fatalf("registrar worker bloqueado: %v", err)
	}
	if err := RegistrarAgente("CodexWorker", "programador"); err != nil {
		t.Fatalf("registrar worker libre: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre IN ('CodexBloqueado','CodexWorker')`); err != nil {
		t.Fatalf("marcar agentes disponibles: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_cuota='enfriamiento', motivo_pausa='presupuesto observado' WHERE nombre='CodexBloqueado'`); err != nil {
		t.Fatalf("bloquear worker por cuota: %v", err)
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
	if _, err := UpsertProyectoAutonomia(&ProyectoAutonomia{
		ProyectoID:      proyectoID,
		Enabled:         true,
		MaxWorkers:      1,
		EstadoAutonomia: AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert autonomia: %v", err)
	}
	if err := ActivarAsignacion("CodexBloqueado", proyectoID, "worker bloqueado"); err != nil {
		t.Fatalf("activar asignacion bloqueada: %v", err)
	}
	tareaLibreID, err := CrearTarea(&Tarea{
		Titulo:      "Worker disponible pese a cuota ajena",
		Descripcion: "El cupo debe ignorar workers bloqueados por cuota",
		ProyectoID:  &proyectoID,
		Modulo:      "orquestador",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea libre: %v", err)
	}

	if err := PlanificarTareasAutomaticamente(); err != nil {
		t.Fatalf("planificar: %v", err)
	}

	tareaLibre, err := GetTarea(tareaLibreID)
	if err != nil {
		t.Fatalf("get tarea libre: %v", err)
	}
	if tareaLibre.Agente == nil || *tareaLibre.Agente != "CodexWorker" || tareaLibre.Estado != TareaAsignada {
		t.Fatalf("el worker libre deberia tomar la tarea cuando el otro esta en cuota: %+v", tareaLibre)
	}
}

func TestPlanificarTareasAutomaticamenteRestringePoolAutobootstrapAWorkersConfigurados(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "claude1", "Codex3")

	if err := ConfigSet("server_autobootstrap_project_slug", "orquestador"); err != nil {
		t.Fatalf("config project slug: %v", err)
	}
	if err := ConfigSet("server_autobootstrap_worker_agents", "Codex3,Codex4"); err != nil {
		t.Fatalf("config worker agents: %v", err)
	}
	if err := RegistrarAgente("claude1", "programador"); err != nil {
		t.Fatalf("registrar claude1: %v", err)
	}
	if err := RegistrarAgente("Codex3", "programador"); err != nil {
		t.Fatalf("registrar Codex3: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre IN ('claude1','Codex3')`); err != nil {
		t.Fatalf("marcar agentes disponibles: %v", err)
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
	if _, err := UpsertProyectoAutonomia(&ProyectoAutonomia{
		ProyectoID:      proyectoID,
		Enabled:         true,
		MaxWorkers:      1,
		EstadoAutonomia: AutonomiaProyectoActiva,
	}); err != nil {
		t.Fatalf("upsert autonomia: %v", err)
	}
	if err := ActivarAsignacion("claude1", proyectoID, "frente heredado"); err != nil {
		t.Fatalf("activar asignacion claude1: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Solo la flota Codex debe autoasignarse",
		Descripcion: "Los agentes fuera del pool no deben drenar backlog",
		ProyectoID:  &proyectoID,
		Modulo:      "planocontrol",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}

	if err := PlanificarTareasAutomaticamente(); err != nil {
		t.Fatalf("planificar: %v", err)
	}

	tarea, err := GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Agente == nil || *tarea.Agente != "Codex3" || tarea.Estado != TareaAsignada {
		t.Fatalf("la tarea deberia quedarse en la flota configurada Codex, got=%+v", tarea)
	}
}

func TestPlanificarTareasAutomaticamenteNoDuplicaStartNiHandleActivo(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "Codex1")

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar agente disponible: %v", err)
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
	if err := ActivarAsignacion("Codex1", proyectoID, "asignacion test"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "No duplicar start",
		Descripcion: "Cobertura de deduplicacion",
		ProyectoID:  &proyectoID,
		Modulo:      "orquestador",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}

	if _, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "start",
		PayloadJSON: `{"proyecto":"orquestador","motivo":"preexistente"}`,
	}); err != nil {
		t.Fatalf("encolar start previo: %v", err)
	}

	if err := PlanificarTareasAutomaticamente(); err != nil {
		t.Fatalf("planificar con start pendiente: %v", err)
	}

	agente := "Codex1"
	estado := "pendiente"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("se esperaba una sola start pendiente: %+v", orders)
	}

	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		CWD:         filepath.Join(tmp, "orquestador"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion: %v", err)
	}
	if _, err := DB.Exec(`UPDATE runtime_orders SET estado='completada' WHERE agente=? AND proyecto_id=? AND tipo='start'`, "Codex1", proyectoID); err != nil {
		t.Fatalf("cerrar start previa: %v", err)
	}

	if err := PlanificarTareasAutomaticamente(); err != nil {
		t.Fatalf("planificar con handle activo: %v", err)
	}

	orders, err = ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar runtime orders final: %v", err)
	}
	var starts int
	for _, order := range orders {
		if order.Tipo == "start" {
			starts++
		}
	}
	if starts != 1 {
		t.Fatalf("no deberia duplicar starts con handle activo: %+v", orders)
	}
}

func TestPlanificarTareasAutomaticamenteNoDuplicaSiHayBootstrapPendiente(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "Codex1")

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar agente disponible: %v", err)
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
	if err := ActivarAsignacion("Codex1", proyectoID, "asignacion test"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "No duplicar bootstrap",
		Descripcion: "Cobertura de continuidad",
		ProyectoID:  &proyectoID,
		Modulo:      "orquestador",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}

	if _, err := EncolarRuntimeOrder(&RuntimeOrder{
		Agente:      "Codex1",
		ProyectoID:  &proyectoID,
		Tipo:        "resume",
		PayloadJSON: `{"proyecto":"orquestador","motivo":"continuidad_pendiente"}`,
	}); err != nil {
		t.Fatalf("encolar resume previo: %v", err)
	}

	if err := PlanificarTareasAutomaticamente(); err != nil {
		t.Fatalf("planificar con resume pendiente: %v", err)
	}

	agente := "Codex1"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "resume" {
		t.Fatalf("no deberia crear start si ya hay continuidad pendiente: %+v", orders)
	}
}

func TestPlanificarTareasAutomaticamenteReactivaAsignacionPausadaConTrabajo(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "Codex1")

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar agente disponible: %v", err)
	}
	proyectoA, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador-a",
		Nombre:  "Orquestador A",
		RutaAbs: filepath.Join(tmp, "orquestador-a"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto A: %v", err)
	}
	proyectoB, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador-b",
		Nombre:  "Orquestador B",
		RutaAbs: filepath.Join(tmp, "orquestador-b"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto B: %v", err)
	}
	if err := ActivarAsignacion("Codex1", proyectoA, "frente original"); err != nil {
		t.Fatalf("activar asignacion A: %v", err)
	}
	if err := PausarAsignacion("Codex1", proyectoA, "esperando humano"); err != nil {
		t.Fatalf("pausar asignacion A: %v", err)
	}
	if err := ActivarAsignacion("Codex1", proyectoB, "frente temporal"); err != nil {
		t.Fatalf("activar asignacion B: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Retomar frente pausado",
		Descripcion: "El proyecto A vuelve a tener trabajo",
		ProyectoID:  &proyectoA,
		Modulo:      "core",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}

	if err := PlanificarTareasAutomaticamente(); err != nil {
		t.Fatalf("planificar: %v", err)
	}

	proyectoActivoID, err := ObtenerProyectoActivoAgente("Codex1")
	if err != nil {
		t.Fatalf("obtener proyecto activo: %v", err)
	}
	if proyectoActivoID != proyectoA {
		t.Fatalf("deberia haber reactivado el proyecto A, got=%d want=%d", proyectoActivoID, proyectoA)
	}
	tarea, err := GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Agente == nil || *tarea.Agente != "Codex1" || tarea.Estado != TareaAsignada {
		t.Fatalf("tarea no reasignada al frente reactivado: %+v", tarea)
	}
	agente := "Codex1"
	asignaciones, err := ListarAsignaciones(FiltroAsignaciones{Agente: &agente})
	if err != nil {
		t.Fatalf("listar asignaciones: %v", err)
	}
	if len(asignaciones) < 2 {
		t.Fatalf("asignaciones insuficientes: %+v", asignaciones)
	}
	if asignaciones[0].ProyectoID != proyectoA || asignaciones[0].Estado != AsignacionActiva {
		t.Fatalf("asignacion A no reactivada: %+v", asignaciones)
	}
	if asignaciones[1].ProyectoID != proyectoB || asignaciones[1].Estado != AsignacionPausada {
		t.Fatalf("asignacion B no pausada tras retorno: %+v", asignaciones)
	}
}

func TestPlanificarTareasAutomaticamenteNoArrancaSiHaySesionActivaEnOtroProyecto(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "Codex1")

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar agente disponible: %v", err)
	}
	proyectoA, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador-a",
		Nombre:  "Orquestador A",
		RutaAbs: filepath.Join(tmp, "orquestador-a"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto A: %v", err)
	}
	proyectoB, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador-b",
		Nombre:  "Orquestador B",
		RutaAbs: filepath.Join(tmp, "orquestador-b"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto B: %v", err)
	}
	if err := ActivarAsignacion("Codex1", proyectoA, "asignacion principal"); err != nil {
		t.Fatalf("activar asignacion A: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "No desalojar sesion de otro proyecto",
		Descripcion: "Cobertura cross-project",
		ProyectoID:  &proyectoA,
		Modulo:      "orquestador",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := TomarTarea(tareaID, "Codex1"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}

	if _, err := IniciarSesionContexto(SesionInicio{
		Agente:      "Codex1",
		ProyectoID:  &proyectoB,
		CWD:         filepath.Join(tmp, "orquestador-b"),
		Herramienta: "codex-cli",
	}); err != nil {
		t.Fatalf("iniciar sesion otro proyecto: %v", err)
	}

	if err := PlanificarTareasAutomaticamente(); err != nil {
		t.Fatalf("planificar con sesion activa en otro proyecto: %v", err)
	}

	agente := "Codex1"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia crear start con sesion activa en otro proyecto: %+v", orders)
	}
}

func TestPlanificarTareasAutomaticamenteIncluyeAgenteRecienRegistradoSinSesion(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "CodexNuevo")

	if err := RegistrarAgente("CodexNuevo", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "app-demo",
		Nombre:  "App Demo",
		RutaAbs: filepath.Join(tmp, "app-demo"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := ActivarAsignacion("CodexNuevo", proyectoID, "onboarding automatico"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Primera tarea autonoma",
		Descripcion: "Debe entrar en la rueda sin sesion previa",
		ProyectoID:  &proyectoID,
		Modulo:      "bootstrap",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}

	if err := PlanificarTareasAutomaticamente(); err != nil {
		t.Fatalf("planificar: %v", err)
	}

	tarea, err := GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Agente == nil || *tarea.Agente != "CodexNuevo" || tarea.Estado != TareaAsignada {
		t.Fatalf("tarea no autoasignada al agente nuevo: %+v", tarea)
	}

	agente := "CodexNuevo"
	estado := "pendiente"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID, Estado: &estado})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "start" {
		t.Fatalf("start no encolada para agente nuevo: %+v", orders)
	}
}

func TestPlanificarTareasAutomaticamenteNoSeleccionaAgenteEnEnfriamiento(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "CodexPausado")

	if err := RegistrarAgente("CodexPausado", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_cuota='enfriamiento', reanimar_at=CURRENT_TIMESTAMP WHERE nombre='CodexPausado'`); err != nil {
		t.Fatalf("marcar enfriamiento: %v", err)
	}
	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "app-enfriada",
		Nombre:  "App Enfriada",
		RutaAbs: filepath.Join(tmp, "app-enfriada"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := ActivarAsignacion("CodexPausado", proyectoID, "pausa por cuota"); err != nil {
		t.Fatalf("activar asignacion: %v", err)
	}
	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "No arrancar en enfriamiento",
		Descripcion: "Debe esperar a reanimacion",
		ProyectoID:  &proyectoID,
		Modulo:      "scheduler",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}

	if err := PlanificarTareasAutomaticamente(); err != nil {
		t.Fatalf("planificar: %v", err)
	}

	tarea, err := GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Agente != nil || tarea.Estado != TareaLibre {
		t.Fatalf("la tarea no deberia asignarse mientras el agente esta enfriando: %+v", tarea)
	}

	agente := "CodexPausado"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agente, ProyectoID: &proyectoID})
	if err != nil {
		t.Fatalf("listar runtime orders: %v", err)
	}
	if len(orders) != 0 {
		t.Fatalf("no deberia crear runtime orders para agente en enfriamiento: %+v", orders)
	}
}

func TestResolverProyectoPlanificableAgenteReactivaProyectoTrasDesbloqueoHumano(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "Codex1")

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar agente disponible: %v", err)
	}
	proyectoA, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador-a",
		Nombre:  "Orquestador A",
		RutaAbs: filepath.Join(tmp, "orquestador-a"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto A: %v", err)
	}
	proyectoB, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador-b",
		Nombre:  "Orquestador B",
		RutaAbs: filepath.Join(tmp, "orquestador-b"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto B: %v", err)
	}
	if err := ActivarAsignacion("Codex1", proyectoA, "frente original"); err != nil {
		t.Fatalf("activar asignacion A: %v", err)
	}
	tareaA, err := CrearTarea(&Tarea{
		Titulo:      "Esperando desbloqueo humano",
		Descripcion: "Bloqueo del frente A",
		ProyectoID:  &proyectoA,
		Modulo:      "core",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea A: %v", err)
	}
	if err := TomarTarea(tareaA, "Codex1"); err != nil {
		t.Fatalf("tomar tarea A: %v", err)
	}
	if err := IniciarTarea(tareaA, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea A: %v", err)
	}
	if err := BloquearTarea(tareaA, "Codex1", "esperando respuesta humana"); err != nil {
		t.Fatalf("bloquear tarea A: %v", err)
	}
	if err := MarcarProyectoEsperandoHumano(proyectoA, "esperando respuesta humana"); err != nil {
		t.Fatalf("marcar proyecto esperando humano: %v", err)
	}
	if err := PausarAsignacion("Codex1", proyectoA, "bloqueo_humano"); err != nil {
		t.Fatalf("pausar asignacion A: %v", err)
	}
	if err := ActivarAsignacion("Codex1", proyectoB, "frente temporal"); err != nil {
		t.Fatalf("activar asignacion B: %v", err)
	}
	if err := DesbloquearTarea(tareaA, "alberto", "respuesta recibida"); err != nil {
		t.Fatalf("desbloquear tarea A: %v", err)
	}

	proyectoPlanificable, err := ResolverProyectoPlanificableAgente("Codex1")
	if err != nil {
		t.Fatalf("resolver proyecto planificable: %v", err)
	}
	if proyectoPlanificable != proyectoA {
		t.Fatalf("deberia volver al proyecto A tras desbloqueo, got=%d want=%d", proyectoPlanificable, proyectoA)
	}

	asignacionActiva, err := GetAsignacionActivaAgente("Codex1")
	if err != nil {
		t.Fatalf("get asignacion activa: %v", err)
	}
	if asignacionActiva == nil || asignacionActiva.ProyectoID != proyectoA {
		t.Fatalf("asignacion activa inesperada: %+v", asignacionActiva)
	}
	op, err := GetProyectoOperacion(proyectoA)
	if err != nil {
		t.Fatalf("get proyecto operacion: %v", err)
	}
	if op.EstadoOperativo != ProyectoOperativoActivo {
		t.Fatalf("el proyecto A deberia quedar reactivado automaticamente: %+v", op)
	}
}

func TestResolverProyectoPlanificableAgenteAsignaSegunPoliticaOperativaProyecto(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "CodexLibre", "CodexOcupado")

	if err := RegistrarAgente("CodexLibre", "programador"); err != nil {
		t.Fatalf("registrar agente libre: %v", err)
	}
	if err := RegistrarAgente("CodexOcupado", "programador"); err != nil {
		t.Fatalf("registrar agente ocupado: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre IN ('CodexLibre','CodexOcupado')`); err != nil {
		t.Fatalf("marcar agentes disponibles: %v", err)
	}

	proyectoA, err := UpsertProyecto(&Proyecto{
		Slug:    "proyecto-a",
		Nombre:  "Proyecto A",
		RutaAbs: filepath.Join(tmp, "proyecto-a"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto A: %v", err)
	}
	proyectoB, err := UpsertProyecto(&Proyecto{
		Slug:    "proyecto-b",
		Nombre:  "Proyecto B",
		RutaAbs: filepath.Join(tmp, "proyecto-b"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto B: %v", err)
	}
	if err := UpsertProyectoOperacion(&ProyectoOperacion{
		ProyectoID:       proyectoA,
		EstadoOperativo:  ProyectoOperativoActivo,
		ObjetivoPct:      80,
		Prioridad:        200,
		ResumeAutomatico: true,
	}); err != nil {
		t.Fatalf("upsert proyecto operacion A: %v", err)
	}
	if err := UpsertProyectoOperacion(&ProyectoOperacion{
		ProyectoID:       proyectoB,
		EstadoOperativo:  ProyectoOperativoActivo,
		ObjetivoPct:      20,
		Prioridad:        100,
		ResumeAutomatico: true,
	}); err != nil {
		t.Fatalf("upsert proyecto operacion B: %v", err)
	}
	if err := ActivarAsignacion("CodexOcupado", proyectoB, "ocupado en B"); err != nil {
		t.Fatalf("activar asignacion B: %v", err)
	}
	if _, err := CrearTarea(&Tarea{
		Titulo:      "Trabajo A",
		Descripcion: "Proyecto priorizado",
		ProyectoID:  &proyectoA,
		Modulo:      "core",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	}); err != nil {
		t.Fatalf("crear tarea A: %v", err)
	}
	if _, err := CrearTarea(&Tarea{
		Titulo:      "Trabajo B",
		Descripcion: "Proyecto menos prioritario",
		ProyectoID:  &proyectoB,
		Modulo:      "core",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	}); err != nil {
		t.Fatalf("crear tarea B: %v", err)
	}

	proyectoPlanificable, err := ResolverProyectoPlanificableAgente("CodexLibre")
	if err != nil {
		t.Fatalf("resolver proyecto planificable: %v", err)
	}
	if proyectoPlanificable != proyectoA {
		t.Fatalf("deberia elegir el proyecto A por politica operativa, got=%d want=%d", proyectoPlanificable, proyectoA)
	}
	asignacionActiva, err := GetAsignacionActivaAgente("CodexLibre")
	if err != nil {
		t.Fatalf("get asignacion activa libre: %v", err)
	}
	if asignacionActiva == nil || asignacionActiva.ProyectoID != proyectoA {
		t.Fatalf("asignacion activa inesperada tras politicas: %+v", asignacionActiva)
	}
}

func TestResolverProyectoPlanificableAgentePausaFrenteActivoSinTrabajoAlRebalancear(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "CodexLibre", "CodexOcupado")

	if err := RegistrarAgente("CodexLibre", "programador"); err != nil {
		t.Fatalf("registrar agente libre: %v", err)
	}
	if err := RegistrarAgente("CodexOcupado", "programador"); err != nil {
		t.Fatalf("registrar agente ocupado: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre IN ('CodexLibre','CodexOcupado')`); err != nil {
		t.Fatalf("marcar agentes disponibles: %v", err)
	}

	proyectoA, err := UpsertProyecto(&Proyecto{
		Slug:    "proyecto-a",
		Nombre:  "Proyecto A",
		RutaAbs: filepath.Join(tmp, "proyecto-a"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto A: %v", err)
	}
	proyectoB, err := UpsertProyecto(&Proyecto{
		Slug:    "proyecto-b",
		Nombre:  "Proyecto B",
		RutaAbs: filepath.Join(tmp, "proyecto-b"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto B: %v", err)
	}
	if err := UpsertProyectoOperacion(&ProyectoOperacion{
		ProyectoID:       proyectoA,
		EstadoOperativo:  ProyectoOperativoActivo,
		ObjetivoPct:      80,
		Prioridad:        200,
		ResumeAutomatico: true,
	}); err != nil {
		t.Fatalf("upsert proyecto operacion A: %v", err)
	}
	if err := UpsertProyectoOperacion(&ProyectoOperacion{
		ProyectoID:       proyectoB,
		EstadoOperativo:  ProyectoOperativoActivo,
		ObjetivoPct:      20,
		Prioridad:        100,
		ResumeAutomatico: true,
	}); err != nil {
		t.Fatalf("upsert proyecto operacion B: %v", err)
	}
	if err := ActivarAsignacion("CodexLibre", proyectoB, "frente sin trabajo"); err != nil {
		t.Fatalf("activar asignacion B libre: %v", err)
	}
	if err := ActivarAsignacion("CodexOcupado", proyectoB, "ocupado en B"); err != nil {
		t.Fatalf("activar asignacion B ocupado: %v", err)
	}
	if _, err := CrearTarea(&Tarea{
		Titulo:      "Trabajo A",
		Descripcion: "Proyecto priorizado",
		ProyectoID:  &proyectoA,
		Modulo:      "core",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	}); err != nil {
		t.Fatalf("crear tarea A: %v", err)
	}

	proyectoPlanificable, err := ResolverProyectoPlanificableAgente("CodexLibre")
	if err != nil {
		t.Fatalf("resolver proyecto planificable: %v", err)
	}
	if proyectoPlanificable != proyectoA {
		t.Fatalf("deberia reequilibrar hacia el proyecto A, got=%d want=%d", proyectoPlanificable, proyectoA)
	}

	agente := "CodexLibre"
	asignaciones, err := ListarAsignaciones(FiltroAsignaciones{Agente: &agente})
	if err != nil {
		t.Fatalf("listar asignaciones: %v", err)
	}
	if len(asignaciones) < 2 {
		t.Fatalf("asignaciones insuficientes tras rebalanceo: %+v", asignaciones)
	}
	var activaA, pausadaB bool
	for _, asignacion := range asignaciones {
		if asignacion == nil {
			continue
		}
		if asignacion.ProyectoID == proyectoA && asignacion.Estado == AsignacionActiva {
			activaA = true
		}
		if asignacion.ProyectoID == proyectoB && asignacion.Estado == AsignacionPausada && asignacion.Nota == "sin_trabajo_espera_automatica" {
			pausadaB = true
		}
	}
	if !activaA || !pausadaB {
		t.Fatalf("el rebalanceo debe mantener afinidad pausando el frente viejo: %+v", asignaciones)
	}
}

func TestResolverProyectoPlanificableAgenteLiberaAgenteSiSuFrenteActivoNoTieneTrabajo(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "CodexLibre")

	if err := RegistrarAgente("CodexLibre", "programador"); err != nil {
		t.Fatalf("registrar agente libre: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre='CodexLibre'`); err != nil {
		t.Fatalf("marcar agente disponible: %v", err)
	}

	proyectoA, err := UpsertProyecto(&Proyecto{
		Slug:    "proyecto-a",
		Nombre:  "Proyecto A",
		RutaAbs: filepath.Join(tmp, "proyecto-a"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto A: %v", err)
	}
	if err := ActivarAsignacion("CodexLibre", proyectoA, "frente sin trabajo"); err != nil {
		t.Fatalf("activar asignacion A libre: %v", err)
	}

	proyectoPlanificable, err := ResolverProyectoPlanificableAgente("CodexLibre")
	if err != nil {
		t.Fatalf("resolver proyecto planificable: %v", err)
	}
	if proyectoPlanificable != 0 {
		t.Fatalf("sin trabajo en el frente activo deberia liberar al agente, got=%d", proyectoPlanificable)
	}

	agente := "CodexLibre"
	estado := AsignacionPausada
	asignaciones, err := ListarAsignaciones(FiltroAsignaciones{Agente: &agente, Estado: &estado})
	if err != nil {
		t.Fatalf("listar asignaciones pausadas: %v", err)
	}
	if len(asignaciones) != 1 || asignaciones[0].ProyectoID != proyectoA || asignaciones[0].Nota != "sin_trabajo_espera_automatica" {
		t.Fatalf("la asignacion deberia quedar pausada para preservar afinidad: %+v", asignaciones)
	}
}

func TestPlanificarTareasAutomaticamenteReubicaTrabajoTrasRetirarAgente(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "CodexRetirado", "CodexRelevo")

	if err := RegistrarAgente("CodexRetirado", "programador"); err != nil {
		t.Fatalf("registrar agente retirado: %v", err)
	}
	if err := RegistrarAgente("CodexRelevo", "programador"); err != nil {
		t.Fatalf("registrar agente relevo: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre='CodexRelevo'`); err != nil {
		t.Fatalf("marcar relevo disponible: %v", err)
	}

	proyectoID, err := UpsertProyecto(&Proyecto{
		Slug:    "proyecto-relevo",
		Nombre:  "Proyecto Relevo",
		RutaAbs: filepath.Join(tmp, "proyecto-relevo"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto: %v", err)
	}
	if err := ActivarAsignacion("CodexRetirado", proyectoID, "frente original"); err != nil {
		t.Fatalf("activar asignacion original: %v", err)
	}

	tareaID, err := CrearTarea(&Tarea{
		Titulo:      "Tarea que debe sobrevivir al relevo",
		Descripcion: "el orquestador debe recolocarla solo",
		ProyectoID:  &proyectoID,
		Modulo:      "core",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea: %v", err)
	}
	if err := TomarTarea(tareaID, "CodexRetirado"); err != nil {
		t.Fatalf("tomar tarea: %v", err)
	}
	if err := IniciarTarea(tareaID, "CodexRetirado"); err != nil {
		t.Fatalf("iniciar tarea: %v", err)
	}

	if err := RetirarAgente("CodexRetirado"); err != nil {
		t.Fatalf("retirar agente: %v", err)
	}
	if err := PlanificarTareasAutomaticamente(); err != nil {
		t.Fatalf("planificar tras retirada: %v", err)
	}

	tarea, err := GetTarea(tareaID)
	if err != nil {
		t.Fatalf("get tarea: %v", err)
	}
	if tarea.Estado != TareaAsignada || tarea.Agente == nil || *tarea.Agente != "CodexRelevo" {
		t.Fatalf("la tarea deberia reubicarse automaticamente: %+v", tarea)
	}

	asignacionActiva, err := GetAsignacionActivaAgente("CodexRelevo")
	if err != nil {
		t.Fatalf("get asignacion activa relevo: %v", err)
	}
	if asignacionActiva == nil || asignacionActiva.ProyectoID != proyectoID {
		t.Fatalf("asignacion activa del relevo inesperada: %+v", asignacionActiva)
	}

	agenteRetirado := "CodexRetirado"
	estadoPausada := AsignacionPausada
	pausadas, err := ListarAsignaciones(FiltroAsignaciones{Agente: &agenteRetirado, Estado: &estadoPausada})
	if err != nil {
		t.Fatalf("listar asignaciones pausadas del retirado: %v", err)
	}
	if len(pausadas) != 1 || pausadas[0].ProyectoID != proyectoID || !strings.Contains(pausadas[0].Nota, "agente_retirado") {
		t.Fatalf("la afinidad del retirado deberia quedar aparcada: %+v", pausadas)
	}

	agenteRelevo := "CodexRelevo"
	estadoPendiente := "pendiente"
	orders, err := ListarRuntimeOrders(FiltroRuntimeOrders{Agente: &agenteRelevo, ProyectoID: &proyectoID, Estado: &estadoPendiente})
	if err != nil {
		t.Fatalf("listar runtime orders del relevo: %v", err)
	}
	if len(orders) != 1 || orders[0].Tipo != "start" {
		t.Fatalf("runtime orders del relevo inesperadas: %+v", orders)
	}
}

func TestResolverProyectoPlanificableAgenteNoReactivaProyectoSinResumeAutomatico(t *testing.T) {
	tmp := prepararDBTemporal(t)
	restringirPlanificadorATestAgentes(t, "Codex1")

	if err := RegistrarAgente("Codex1", "programador"); err != nil {
		t.Fatalf("registrar agente: %v", err)
	}
	if _, err := DB.Exec(`UPDATE agentes SET estado_sesion='disponible' WHERE nombre='Codex1'`); err != nil {
		t.Fatalf("marcar agente disponible: %v", err)
	}
	proyectoA, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador-a",
		Nombre:  "Orquestador A",
		RutaAbs: filepath.Join(tmp, "orquestador-a"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto A: %v", err)
	}
	proyectoB, err := UpsertProyecto(&Proyecto{
		Slug:    "orquestador-b",
		Nombre:  "Orquestador B",
		RutaAbs: filepath.Join(tmp, "orquestador-b"),
		Tipo:    ProyectoRepo,
		Activo:  true,
	})
	if err != nil {
		t.Fatalf("upsert proyecto B: %v", err)
	}
	if err := UpsertProyectoOperacion(&ProyectoOperacion{
		ProyectoID:       proyectoA,
		EstadoOperativo:  ProyectoOperativoEsperandoHumano,
		Motivo:           "esperando respuesta humana",
		ObjetivoPct:      100,
		Prioridad:        300,
		ResumeAutomatico: false,
	}); err != nil {
		t.Fatalf("upsert proyecto operacion A: %v", err)
	}
	if err := ActivarAsignacion("Codex1", proyectoA, "frente original"); err != nil {
		t.Fatalf("activar asignacion A: %v", err)
	}
	tareaA, err := CrearTarea(&Tarea{
		Titulo:      "Esperando desbloqueo humano",
		Descripcion: "Bloqueo del frente A",
		ProyectoID:  &proyectoA,
		Modulo:      "core",
		Prioridad:   PrioridadAlta,
		CreadoPor:   "alberto",
	})
	if err != nil {
		t.Fatalf("crear tarea A: %v", err)
	}
	if err := TomarTarea(tareaA, "Codex1"); err != nil {
		t.Fatalf("tomar tarea A: %v", err)
	}
	if err := IniciarTarea(tareaA, "Codex1"); err != nil {
		t.Fatalf("iniciar tarea A: %v", err)
	}
	if err := BloquearTarea(tareaA, "Codex1", "esperando respuesta humana"); err != nil {
		t.Fatalf("bloquear tarea A: %v", err)
	}
	if err := PausarAsignacion("Codex1", proyectoA, "bloqueo_humano"); err != nil {
		t.Fatalf("pausar asignacion A: %v", err)
	}
	if err := ActivarAsignacion("Codex1", proyectoB, "frente temporal"); err != nil {
		t.Fatalf("activar asignacion B: %v", err)
	}
	if _, err := CrearTarea(&Tarea{
		Titulo:      "Trabajo B",
		Descripcion: "frente temporal",
		ProyectoID:  &proyectoB,
		Modulo:      "core",
		Prioridad:   PrioridadMedia,
		CreadoPor:   "alberto",
	}); err != nil {
		t.Fatalf("crear tarea B: %v", err)
	}
	if err := DesbloquearTarea(tareaA, "alberto", "respuesta recibida"); err != nil {
		t.Fatalf("desbloquear tarea A: %v", err)
	}

	proyectoPlanificable, err := ResolverProyectoPlanificableAgente("Codex1")
	if err != nil {
		t.Fatalf("resolver proyecto planificable: %v", err)
	}
	if proyectoPlanificable != proyectoB {
		t.Fatalf("no deberia reactivar automaticamente el proyecto A, got=%d want=%d", proyectoPlanificable, proyectoB)
	}

	asignacionActiva, err := GetAsignacionActivaAgente("Codex1")
	if err != nil {
		t.Fatalf("get asignacion activa: %v", err)
	}
	if asignacionActiva == nil || asignacionActiva.ProyectoID != proyectoB {
		t.Fatalf("asignacion activa inesperada: %+v", asignacionActiva)
	}
	op, err := GetProyectoOperacion(proyectoA)
	if err != nil {
		t.Fatalf("get proyecto operacion: %v", err)
	}
	if op.EstadoOperativo != ProyectoOperativoEsperandoHumano {
		t.Fatalf("el proyecto A no deberia reactivarse automaticamente: %+v", op)
	}
}
