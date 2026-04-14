package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"orquesta/capacidadapp"
	"orquesta/coordinacion"
	"orquesta/db"
	"orquesta/gitoperaciones"
	"orquesta/progresoapp"
	"orquesta/runtimesapp"
	"orquesta/supervisionapp"
	"orquesta/tareasapp"
)

const (
	microcicloRefactorTituloDefault = "Micro-refactorización cíclica del control plane: runtime mailbox/session_resume"
	microcicloRefactorNotasTag      = "autonomia:microrefactor_loop"
)

type proyectoMicrocicloRequest struct {
	Agente               string `json:"agente"`
	ObjetivoGeneral      string `json:"objetivo_general"`
	DefinitionOfDoneJSON string `json:"definition_of_done_json"`
	Titulo               string `json:"titulo"`
	Descripcion          string `json:"descripcion"`
	Modulo               string `json:"modulo"`
	Notas                string `json:"notas"`
	LimpiarPruebas       bool   `json:"limpiar_pruebas"`
}

type proyectoMicrocicloResult struct {
	Proyecto            *db.Proyecto                                      `json:"proyecto"`
	Operacion           *db.ProyectoOperacion                             `json:"operacion"`
	Policy              *db.ProyectoAutonomia                             `json:"policy"`
	Fase                *db.FaseProyecto                                  `json:"fase"`
	Tarea               *db.Tarea                                         `json:"tarea"`
	Dispatch            *capacidadapp.ResultadoEjecucionPasoPipelineLocal `json:"dispatch,omitempty"`
	NudgeRuntimeOrderID int64                                             `json:"nudge_runtime_order_id,omitempty"`
	TareaReutilizada    bool                                              `json:"tarea_reutilizada"`
	RunnerMailboxWake   bool                                              `json:"runner_mailbox_wake"`
	RunnerOrdersWake    bool                                              `json:"runner_orders_wake"`
	RunnerWarmWake      bool                                              `json:"runner_warm_wake"`
}

func activarMicrocicloProyecto(ref string, req proyectoMicrocicloRequest) (*proyectoMicrocicloResult, error) {
	proyecto, err := db.GetProyecto(strings.TrimSpace(ref))
	if err != nil {
		return nil, err
	}
	agente, err := validarAgenteMicrociclo(req.Agente)
	if err != nil {
		return nil, err
	}
	if req.LimpiarPruebas {
		if err := limpiarEntornoPruebaMicrociclo(proyecto, agente); err != nil {
			return nil, err
		}
	}
	if err := asegurarAsignacionExclusivaMicrociclo(proyecto, agente); err != nil {
		return nil, err
	}
	operacion, err := activarOperacionMicrociclo(proyecto.ID)
	if err != nil {
		return nil, err
	}
	policy, err := activarPoliticaMicrociclo(proyecto.Slug, agente, req)
	if err != nil {
		return nil, err
	}
	fase, err := activarFaseImplementacionMicrociclo(proyecto.Slug)
	if err != nil {
		return nil, err
	}
	reutilizarTarea := !req.LimpiarPruebas
	if bloqueado, _, err := agenteBloqueadoPorCuotaPipeline(agente); err != nil {
		return nil, err
	} else if bloqueado {
		reutilizarTarea = true
	}
	tarea, reutilizada, err := asegurarTareaMicrociclo(proyecto, agente, req, reutilizarTarea)
	if err != nil {
		return nil, err
	}
	if err := escribirInboxMicrociclo(proyecto, agente, tarea); err != nil {
		return nil, err
	}
	dispatch, dispatchErr := capacidadService.EjecutarSiguientePasoPipelineLocalDeterminista(proyecto.Slug)
	if dispatchErr != nil {
		db.Audit("orquesta", "microrefactor_loop_dispatch_error", "proyecto", proyecto.ID, dispatchErr.Error())
	}
	dispatch, ensureErr := asegurarDespachoMicrociclo(proyecto, agente, tarea, dispatch)
	if ensureErr != nil {
		db.Audit("orquesta", "microrefactor_loop_dispatch_error", "proyecto", proyecto.ID, ensureErr.Error())
		dispatch = nil
	}
	_, _ = db.RegistrarAutonomiaCiclo(&db.AutonomiaCiclo{
		ProyectoID:   proyecto.ID,
		Kind:         "microrefactor_loop_activation",
		Agente:       agente,
		InputJSON:    fmt.Sprintf(`{"agente":%q,"fase":"implementacion","tarea_id":%d}`, agente, tarea.ID),
		DecisionJSON: `{"modo":"microrefactor_loop","dispatch":"pipeline_local"}`,
		Resultado:    "activado",
	})
	return &proyectoMicrocicloResult{
		Proyecto:            proyecto,
		Operacion:           operacion,
		Policy:              policy,
		Fase:                fase,
		Tarea:               tarea,
		Dispatch:            dispatch,
		NudgeRuntimeOrderID: runtimeOrderIDFromDispatch(dispatch),
		TareaReutilizada:    reutilizada,
		RunnerMailboxWake:   wakeControlPlaneRuntimeMailbox(),
		RunnerOrdersWake:    wakeControlPlaneRuntimeOrders(),
		RunnerWarmWake:      wakeControlPlaneWarm(),
	}, nil
}

func limpiarEntornoPruebaMicrociclo(proyecto *db.Proyecto, agente string) error {
	if proyecto == nil {
		return fmt.Errorf("proyecto obligatorio")
	}
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return fmt.Errorf("agente obligatorio")
	}
	if _, err := tareasService.CleanActiveFront(tareasapp.CleanActiveFrontInput{
		Agente:   agente,
		Proyecto: proyecto.Slug,
	}); err != nil {
		return err
	}
	if err := detenerRuntimeActivoMicrociclo(agente); err != nil {
		return err
	}
	pendiente := "pendiente"
	mailbox, err := runtimesService.ListRuntimeMailbox(db.FiltroRuntimeMailbox{
		ToAgente:   &agente,
		ProyectoID: &proyecto.ID,
		Estado:     &pendiente,
	})
	if err != nil {
		return err
	}
	for _, msg := range mailbox {
		if msg == nil || msg.ID <= 0 {
			continue
		}
		if err := runtimesService.MarkRuntimeMailboxConsumed(msg.ID); err != nil {
			return err
		}
	}
	if _, err := runtimesService.PurgeTerminalRuntimeOrders(runtimesapp.RuntimeOrderPurgeRequest{
		Agente:           agente,
		Proyecto:         proyecto.Slug,
		Estados:          []string{"completada", "fallida", "expirada", "cancelada"},
		OlderThanMinutes: 0,
		Actor:            "microciclo limpio",
	}); err != nil {
		return err
	}
	if _, err := runtimesService.PurgeInactiveRuntimeHandles(runtimesapp.RuntimeHandlePurgeRequest{
		Agente:   agente,
		Proyecto: proyecto.Slug,
		Estados:  []string{"cerrado", "fallido"},
		Actor:    "microciclo limpio",
	}); err != nil {
		return err
	}
	if err := limpiarWorktreesActivasMicrociclo(proyecto, agente); err != nil {
		return err
	}
	if err := prepararWorktreeFrescaMicrociclo(proyecto, agente); err != nil {
		return err
	}
	return nil
}

func limpiarWorktreesActivasMicrociclo(proyecto *db.Proyecto, agente string) error {
	if proyecto == nil {
		return fmt.Errorf("proyecto obligatorio")
	}
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return fmt.Errorf("agente obligatorio")
	}
	svc := newCoordinationService()
	state := coordinacion.WorktreeActive
	worktrees, err := svc.ListWorktrees(coordinacion.WorktreeFilter{
		ProjectID: &proyecto.ID,
		Agent:     &agente,
		State:     &state,
	})
	if err != nil {
		return err
	}
	for _, worktree := range worktrees {
		if worktree == nil || worktree.ID <= 0 {
			continue
		}
		if _, err := svc.CloseWorktree(worktree.ID, true, "microciclo_limpio"); err != nil {
			if _, fallbackErr := svc.CloseWorktree(worktree.ID, false, "microciclo_limpio"); fallbackErr != nil {
				return err
			}
		}
	}
	return nil
}

func prepararWorktreeFrescaMicrociclo(proyecto *db.Proyecto, agente string) error {
	if proyecto == nil {
		return fmt.Errorf("proyecto obligatorio")
	}
	if err := eliminarColisionWorktreeMicrociclo(proyecto, agente); err != nil {
		return err
	}
	if !proyectoPareceRepoGit(proyecto.RutaAbs) {
		return nil
	}
	worktree, err := (worktreeRuntimeService{}).EnsureActiveWorktree(strings.TrimSpace(proyecto.Slug), strings.TrimSpace(agente))
	if err != nil || worktree == nil {
		return err
	}
	if err := sincronizarWorkspaceProyectoEnWorktree(proyecto, worktree); err != nil {
		return err
	}
	return nil
}

func eliminarColisionWorktreeMicrociclo(proyecto *db.Proyecto, agente string) error {
	if proyecto == nil {
		return fmt.Errorf("proyecto obligatorio")
	}
	ruta := rutaEsperadaWorktreeMicrociclo(proyecto.RutaAbs, proyecto.Slug, agente)
	if ruta != "" {
		if _, err := os.Stat(ruta); err == nil {
			if removeErr := os.RemoveAll(ruta); removeErr != nil {
				return removeErr
			}
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	if proyectoPareceRepoGit(proyecto.RutaAbs) {
		if err := (gitoperaciones.WorktreeManager{}).PruneWorktrees(strings.TrimSpace(proyecto.RutaAbs)); err != nil {
			return err
		}
	}
	return nil
}

func rutaEsperadaWorktreeMicrociclo(rutaProyecto, slugProyecto, agente string) string {
	rutaProyecto = strings.TrimSpace(rutaProyecto)
	if rutaProyecto == "" {
		return ""
	}
	return filepath.Join(rutaProyecto, ".orquesta-worktrees", nombreWorktreeMicrociclo(slugProyecto, agente))
}

func nombreWorktreeMicrociclo(slugProyecto, agente string) string {
	replacer := strings.NewReplacer(" ", "-", "_", "-", "/", "-", "\\", "-", ":", "-", "@", "-", "..", "-")
	base := strings.TrimSpace(strings.ToLower(strings.TrimSpace(slugProyecto) + "-" + strings.TrimSpace(agente)))
	base = replacer.Replace(base)
	base = strings.Trim(base, "-")
	if base == "" {
		return "work"
	}
	return base
}

func proyectoPareceRepoGit(ruta string) bool {
	ruta = strings.TrimSpace(ruta)
	if ruta == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(ruta, ".git"))
	if err != nil {
		return false
	}
	return info != nil
}

func detenerRuntimeActivoMicrociclo(agente string) error {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return fmt.Errorf("agente obligatorio")
	}
	handle, err := runtimesService.ResolveControlHandle(agente, nil)
	if err != nil {
		return err
	}
	if handle == nil {
		return db.AparcarSesionActiva(agente, nil)
	}
	if _, _, err := runtimesService.EnqueueAgentControl(runtimesapp.AgentControlRequest{
		Agente: agente,
		Accion: agenteControlAccionStop,
		Motivo: "microciclo limpio",
		Por:    "orquesta",
	}); err != nil {
		return err
	}
	_ = wakeControlPlaneRuntimeOrders()
	_ = wakeControlPlaneWarm()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(250 * time.Millisecond)
		handle, err := runtimesService.ResolveControlHandle(agente, nil)
		if err != nil {
			return err
		}
		if handle == nil {
			return db.AparcarSesionActiva(agente, nil)
		}
	}
	if err := db.AparcarSesionActiva(agente, nil); err != nil {
		return err
	}
	return fmt.Errorf("timeout esperando parada de runtime activo para %s", agente)
}

func asegurarAsignacionExclusivaMicrociclo(proyecto *db.Proyecto, agente string) error {
	if proyecto == nil {
		return fmt.Errorf("proyecto obligatorio")
	}
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return fmt.Errorf("agente obligatorio")
	}
	return db.ActivarAsignacion(agente, proyecto.ID, "microciclo_exclusivo")
}

func asegurarRuntimeAgenteMicrociclo(proyecto *db.Proyecto, agente string) error {
	if proyecto == nil {
		return fmt.Errorf("proyecto obligatorio")
	}
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return fmt.Errorf("agente obligatorio")
	}
	if handle, err := runtimesService.GetOperationalRuntimeHandleAgentProject(agente, &proyecto.ID); err != nil {
		return err
	} else if handle != nil {
		return nil
	}
	if handle, err := runtimesService.GetActiveRuntimeHandleAgentProject(agente, &proyecto.ID); err != nil {
		return err
	} else if handle != nil {
		return nil
	}
	return encolarControlAutonomiaProyecto(agente, proyecto, agenteControlAccionStart, "microciclo_activation")
}

func validarAgenteMicrociclo(raw string) (string, error) {
	agente := strings.TrimSpace(raw)
	if agente == "" {
		agente = "Codex1"
	}
	info, err := db.GetAgente(agente)
	if err != nil {
		return "", err
	}
	if info == nil {
		return "", fmt.Errorf("agente %s no encontrado", agente)
	}
	if !info.Habilitado {
		return "", fmt.Errorf("agente %s está retirado o deshabilitado", agente)
	}
	return strings.TrimSpace(info.Nombre), nil
}

func activarOperacionMicrociclo(proyectoID int64) (*db.ProyectoOperacion, error) {
	op, err := db.GetProyectoOperacion(proyectoID)
	if err != nil {
		return nil, err
	}
	op.EstadoOperativo = db.ProyectoOperativoActivo
	op.Motivo = "microrefactor_loop"
	op.MinAgentes = 1
	op.MaxAgentes = 1
	op.ResumeAutomatico = true
	if op.ObjetivoPct <= 0 {
		op.ObjetivoPct = 100
	}
	if err := db.UpsertProyectoOperacion(op); err != nil {
		return nil, err
	}
	return db.GetProyectoOperacion(proyectoID)
}

func activarPoliticaMicrociclo(proyectoSlug, agente string, req proyectoMicrocicloRequest) (*db.ProyectoAutonomia, error) {
	return supervisionService.UpsertProjectPolicy(proyectoSlug, supervisionapp.PolicyInput{
		Enabled:              true,
		ObjetivoGeneral:      firstNonEmpty(strings.TrimSpace(req.ObjetivoGeneral), objetivoGeneralMicrocicloDefault()),
		DefinitionOfDoneJSON: firstNonEmpty(strings.TrimSpace(req.DefinitionOfDoneJSON), definitionOfDoneMicrocicloDefault()),
		MaxWorkers:           1,
		SupervisorAgente:     agente,
		ReviewerAgente:       "",
		ReserveReviewer:      false,
		ReserveSupervisor:    false,
		ReviewRequired:       false,
		AutoCreateTasks:      false,
		AutoCloseProject:     false,
		EstadoAutonomia:      db.AutonomiaProyectoActiva,
	})
}

func activarFaseImplementacionMicrociclo(proyectoSlug string) (*db.FaseProyecto, error) {
	fases, err := progresoService.ListPhases(proyectoSlug)
	if err != nil {
		return nil, err
	}
	var implementacion *db.FaseProyecto
	for _, fase := range fases {
		if fase == nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(fase.Nombre), "implementacion") {
			implementacion = fase
			continue
		}
		if strings.EqualFold(strings.TrimSpace(fase.Estado), "activa") {
			estado := "pendiente"
			if _, err := progresoService.UpdatePhase(progresoapp.UpdatePhaseInput{ID: fase.ID, Estado: &estado}); err != nil {
				return nil, err
			}
		}
	}
	if implementacion == nil {
		_, fase, err := progresoService.RegisterPhase(progresoapp.RegisterPhaseInput{
			Proyecto:    proyectoSlug,
			Nombre:      "implementacion",
			Descripcion: "Micro-refactorizacion ciclica guiada por Orquesta",
			Orden:       2,
			Peso:        35,
			Estado:      "activa",
		})
		return fase, err
	}
	if !strings.EqualFold(strings.TrimSpace(implementacion.Estado), "activa") {
		estado := "activa"
		return progresoService.UpdatePhase(progresoapp.UpdatePhaseInput{ID: implementacion.ID, Estado: &estado})
	}
	return implementacion, nil
}

func asegurarTareaMicrociclo(proyecto *db.Proyecto, agente string, req proyectoMicrocicloRequest, reutilizarExistente bool) (*db.Tarea, bool, error) {
	if reutilizarExistente {
		tarea, err := buscarTareaMicrocicloAbierta(proyecto.ID)
		if err != nil {
			return nil, false, err
		}
		if tarea != nil {
			actual := ""
			if tarea.Agente != nil {
				actual = strings.TrimSpace(*tarea.Agente)
			}
			estado := strings.TrimSpace(string(tarea.Estado))
			if actual != agente {
				switch strings.ToLower(estado) {
				case string(db.TareaBacklog), string(db.TareaLibre):
					if err := tareasService.Take(tarea.ID, agente); err != nil {
						return nil, true, err
					}
				default:
					if err := tareasService.Reassign(tarea.ID, agente); err != nil {
						return nil, true, err
					}
				}
			}
			if strings.ToLower(estado) != string(db.TareaEnProgreso) {
				if err := tareasService.Start(tarea.ID, agente); err != nil {
					return nil, true, err
				}
			}
			recargada, err := tareasService.Get(tarea.ID)
			return recargada, true, err
		}
	}
	id, err := tareasService.Create(tareasapp.CreateTaskInput{
		Titulo:      firstNonEmpty(strings.TrimSpace(req.Titulo), microcicloRefactorTituloDefault),
		Descripcion: firstNonEmpty(strings.TrimSpace(req.Descripcion), descripcionMicrocicloDefault(proyecto)),
		Modulo:      firstNonEmpty(strings.TrimSpace(req.Modulo), "controlplane"),
		Prioridad:   db.PrioridadAlta,
		CreadoPor:   "orquesta",
		Agente:      agente,
		Proyecto:    proyecto.Slug,
		Notas:       notasMicrociclo(req.Notas),
	})
	if err != nil {
		return nil, false, err
	}
	if err := tareasService.Start(id, agente); err != nil {
		return nil, false, err
	}
	tarea, err := tareasService.Get(id)
	return tarea, false, err
}

func buscarTareaMicrocicloAbierta(proyectoID int64) (*db.Tarea, error) {
	tareas, err := tareasService.List(db.FiltroTareas{ProyectoID: &proyectoID})
	if err != nil {
		return nil, err
	}
	for _, tarea := range tareas {
		if tarea == nil {
			continue
		}
		switch tarea.Estado {
		case db.TareaCompletada, db.TareaCancelada:
			continue
		}
		if strings.EqualFold(strings.TrimSpace(tarea.Titulo), microcicloRefactorTituloDefault) || strings.Contains(strings.TrimSpace(tarea.Notas), microcicloRefactorNotasTag) {
			return tarea, nil
		}
	}
	return nil, nil
}

func objetivoGeneralMicrocicloDefault() string {
	return "Micro-refactorizar de forma ciclica el control plane de Orquesta sobre codigo real, reduciendo mezcla inline y deuda estructural sin cambiar el comportamiento observable."
}

func definitionOfDoneMicrocicloDefault() string {
	return `{"estado":"microrefactor_loop_activo","criterios":["siguiente frente pequeno y util","sin cambiar semantica observable","tests afectados en verde","sin abrir carriles paralelos"]}`
}

func descripcionMicrocicloDefault(proyecto *db.Proyecto) string {
	partes := []string{
		"Frente actual: cerrar el carril premium runtime mailbox/session_resume del control plane.",
		"Objetivo inmediato: seguir eliminando huecos entre bootstrap, mailbox y recibo util para workers premium sin cambiar semantica observable fuera de ese carril.",
		"Objetivo exacto de este frente: endurecer el primer ciclo premium para que bootstrap, mailbox inicial y receipt no diverjan.",
		"Regla arquitectonica de este frente: para runtimes interactivos premium el carril canonico es tmux_cli_session; process_pty_cli/pty_broker quedan relegados a legado de recuperacion o tests y no deben abrir trabajo nuevo ni dictar arquitectura.",
		"Simbolos foco: procesarRuntimeMailboxSessionResumeBatchConMailbox, resolverBootstrapRuntimeLeasePendiente y helpers inmediatos del mismo slice.",
		"Write-set exclusivo: cmd/controlplane_support.go, cmd/controlplane_support_test.go, db/controlplane_entities.go, db/controlplane_entities_test.go, runtimeagente/driver.go y runtimeagente/driver_test.go. Trabaja solo dentro de ese write_set; si el slice exigiera tocar algo fuera, para y reporta BLOQUEO.",
		"Tests minimos del slice: go test ./cmd -run 'TestProcesarRuntimeMailboxSessionResumeBatch.*' -count=1 y go test ./db -run 'TestResolverBootstrapRuntimeLeasePendiente.*' -count=1.",
		"Antes de ampliar validacion, producir broad scans o correr tests colindantes, intenta primero el patch mas pequeno y seguro dentro de simbolos foco y write-set.",
		"Si el slice actual ya queda verde dentro del write-set, no esperes una microtarea nueva: identifica y ejecuta el siguiente caso adyacente mas pequeno y verificable del mismo slice usando codigo real, tests colindantes o un hueco defensible en simbolos foco.",
		"Trabaja en un slice pequeno y verificable dentro de ese frente; no abras otro carril ni inventes una arquitectura nueva.",
		"No abras arquitectura nueva ni cambies comportamiento observable salvo bug claro con prueba.",
		"No reabras ni refuerces process_pty_cli, pty_broker ni fallbacks PTY-first en este frente salvo bug de compatibilidad ya existente y acotado por prueba.",
		"No abras shims de compatibilidad, firmas variadicas ni helpers genericos para cubrir call-sites hipoteticos; solo hazlo si un rg del arbol activo demuestra el uso real y actual.",
		"No ensanches firmas ni APIs del nucleo para tapar deuda ajena al frente actual. Si no hay call-site real en este arbol, no es un bloqueo valido.",
		"Antes de cerrar, deja tests del slice en verde.",
	}
	if proyecto != nil && strings.TrimSpace(proyecto.RutaAbs) != "" {
		partes = append(partes, "Proyecto: "+strings.TrimSpace(proyecto.RutaAbs)+".")
	}
	return strings.Join(partes, " ")
}

func escribirInboxMicrociclo(proyecto *db.Proyecto, agente string, tarea *db.Tarea) error {
	if proyecto == nil || tarea == nil {
		return nil
	}
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return nil
	}
	base := strings.TrimSpace(proyecto.RutaAbs)
	worktree, err := (worktreeRuntimeService{}).ResolveActiveWorktree(strings.TrimSpace(proyecto.Slug), agente)
	if err == nil && worktree != nil && strings.TrimSpace(worktree.RutaAbs) != "" {
		base = strings.TrimSpace(worktree.RutaAbs)
	}
	if base == "" {
		return nil
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		return err
	}
	path := filepath.Join(base, ".orquesta-inbox.md")
	return os.WriteFile(path, []byte(construirInboxMicrocicloMarkdown(proyecto, tarea)), 0o644)
}

func construirInboxMicrocicloMarkdown(proyecto *db.Proyecto, tarea *db.Tarea) string {
	if tarea == nil {
		return ""
	}
	lines := []string{
		"# Microtarea Activa de Orquesta",
		"",
		fmt.Sprintf("- Proyecto: `%s`", strings.TrimSpace(func() string {
			if proyecto == nil {
				return ""
			}
			return proyecto.Slug
		}())),
		fmt.Sprintf("- Tarea: `#%d %s`", tarea.ID, strings.TrimSpace(tarea.Titulo)),
		"- Regla: trabaja solo dentro de este frente y de este write-set.",
		"",
		"## Alcance",
		strings.TrimSpace(tarea.Descripcion),
		"",
		"## Ejecucion",
		"- Aplica un unico slice pequeno y verificable.",
		"- No amplíes validación ni búsquedas laterales antes del primer patch pequeño dentro del write-set.",
		"- Si este slice ya queda verde, no esperes otra microtarea: encuentra el siguiente caso adyacente mas pequeno y verificable dentro del mismo write-set y siguelo de inmediato.",
		"- No reabras doctrina ni otros frentes si aqui ya esta el contrato operativo.",
		"- Mantente en TMUX como carril canonico premium; no abras ni refuerces process_pty_cli ni PTY-first en este frente.",
		"- No abras shims de compatibilidad ni ensanches firmas del nucleo salvo que exista un call-site real en este arbol y lo hayas comprobado antes.",
		"- No cierres diciendo que esperas otra microtarea si aun queda un hueco verificable dentro de simbolos foco y write-set.",
		"- Antes de terminar, deja en verde los tests minimos del slice.",
	}
	return strings.TrimSpace(strings.Join(lines, "\n")) + "\n"
}

func notasMicrociclo(extra string) string {
	extra = strings.TrimSpace(extra)
	if extra == "" {
		return microcicloRefactorNotasTag
	}
	return microcicloRefactorNotasTag + ";" + extra
}

func runtimeOrderIDFromDispatch(resultado *capacidadapp.ResultadoEjecucionPasoPipelineLocal) int64 {
	if resultado == nil || resultado.DispatchRuntime == nil || resultado.DispatchRuntime.RuntimeOrderID == nil {
		return 0
	}
	return *resultado.DispatchRuntime.RuntimeOrderID
}

func asegurarDespachoMicrociclo(proyecto *db.Proyecto, agente string, tarea *db.Tarea, resultado *capacidadapp.ResultadoEjecucionPasoPipelineLocal) (*capacidadapp.ResultadoEjecucionPasoPipelineLocal, error) {
	if proyecto == nil || tarea == nil {
		return resultado, nil
	}
	agente = strings.TrimSpace(agente)
	if resultado == nil {
		resultado = &capacidadapp.ResultadoEjecucionPasoPipelineLocal{}
	}
	writeSet := capacidadapp.ExtraerWriteSetTextoPipelineLocal(strings.TrimSpace(tarea.Descripcion))
	simbolosFoco := capacidadapp.ExtraerSimbolosFocoTextoPipelineLocal(strings.TrimSpace(tarea.Descripcion))
	testsMinimos := capacidadapp.ExtraerTestsMinimosTextoPipelineLocal(strings.TrimSpace(tarea.Descripcion))
	if resultado.Despacho == nil {
		resultado.Despacho = &capacidadapp.DespachoPipelineLocal{
			ProyectoSlug:     strings.TrimSpace(proyecto.Slug),
			Fase:             "implementacion",
			AccionTarea:      "continuar_trabajo",
			PerfilTarea:      "implementacion",
			ModoEjecucion:    "premium_worktree",
			Carril:           "premium_worktree",
			EntregaCanonica:  "git_worktree",
			RequiereWorktree: true,
			RequiereModelo:   true,
			TareaObjetivoID:  tarea.ID,
			TareaObjetivo:    strings.TrimSpace(tarea.Titulo),
			AgenteTarea:      agente,
			WriteSet:         append([]string(nil), writeSet...),
			SimbolosFoco:     strings.TrimSpace(simbolosFoco),
			TestsMinimos:     strings.TrimSpace(testsMinimos),
			AgenteSugerido:   agente,
			Motivo:           "microrefactor_loop",
		}
	} else {
		if resultado.Despacho.TareaObjetivoID <= 0 {
			resultado.Despacho.TareaObjetivoID = tarea.ID
		}
		if strings.TrimSpace(resultado.Despacho.TareaObjetivo) == "" {
			resultado.Despacho.TareaObjetivo = strings.TrimSpace(tarea.Titulo)
		}
		if strings.TrimSpace(resultado.Despacho.AgenteTarea) == "" {
			resultado.Despacho.AgenteTarea = agente
		}
		if len(resultado.Despacho.WriteSet) == 0 && len(writeSet) > 0 {
			resultado.Despacho.WriteSet = append([]string(nil), writeSet...)
		}
		if strings.TrimSpace(resultado.Despacho.SimbolosFoco) == "" {
			resultado.Despacho.SimbolosFoco = strings.TrimSpace(simbolosFoco)
		}
		if strings.TrimSpace(resultado.Despacho.TestsMinimos) == "" {
			resultado.Despacho.TestsMinimos = strings.TrimSpace(testsMinimos)
		}
		if debeForzarAgenteMicrociclo(resultado.Despacho, agente) {
			resultado.Despacho.AgenteSugerido = agente
		}
	}
	if resultado.DispatchRuntime != nil && !strings.EqualFold(strings.TrimSpace(resultado.DispatchRuntime.Estado), "sin_agente") {
		return resultado, nil
	}
	dispatchRuntime, err := (despachadorPipelineOperativo{}).DespacharPipeline(capacidadapp.SolicitudDespachoPipeline{
		ProyectoSlug: strings.TrimSpace(proyecto.Slug),
		Despacho:     resultado.Despacho,
	})
	if err != nil {
		return resultado, err
	}
	resultado.DispatchRuntime = dispatchRuntime
	return resultado, nil
}

func debeForzarAgenteMicrociclo(despacho *capacidadapp.DespachoPipelineLocal, agente string) bool {
	if despacho == nil || strings.TrimSpace(agente) == "" {
		return false
	}
	sugerido := strings.TrimSpace(despacho.AgenteSugerido)
	if sugerido == "" {
		return true
	}
	switch strings.ToLower(sugerido) {
	case "worker_premium", "worker_revisor", "worker_local_mini":
		return true
	default:
		return false
	}
}

func resolverProyectoMicrocicloPorAPI(ref string) (*db.Proyecto, bool, error) {
	ref = strings.TrimSpace(ref)
	if ref != "" {
		return cargarProyectoDesdeAPI(ref)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, false, err
	}
	cwd, err = filepath.Abs(cwd)
	if err != nil {
		return nil, false, err
	}
	if proyecto, ok, err := cargarProyectoDesdeAPI(cwd); ok && err == nil && proyecto != nil {
		return proyecto, ok, nil
	}
	proyectos, ok, err := descubrirProyectosPorAPI(cwd)
	if !ok || err != nil {
		return nil, ok, err
	}
	for _, proyecto := range proyectos {
		if proyecto != nil && strings.EqualFold(filepath.Clean(strings.TrimSpace(proyecto.RutaAbs)), filepath.Clean(cwd)) {
			return proyecto, true, nil
		}
	}
	return nil, true, fmt.Errorf("no se pudo resolver el proyecto actual desde %s", cwd)
}

func resolverProyectoMicrocicloLocal(ref string) (*db.Proyecto, error) {
	ref = strings.TrimSpace(ref)
	if ref != "" {
		return db.GetProyecto(ref)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	cwd, err = filepath.Abs(cwd)
	if err != nil {
		return nil, err
	}
	if proyecto, err := db.GetProyecto(cwd); err == nil && proyecto != nil {
		return proyecto, nil
	}
	proyectos, err := db.DescubrirProyectos(cwd)
	if err != nil {
		return nil, err
	}
	for _, proyecto := range proyectos {
		if proyecto != nil && strings.EqualFold(filepath.Clean(strings.TrimSpace(proyecto.RutaAbs)), filepath.Clean(cwd)) {
			return proyecto, nil
		}
	}
	return nil, fmt.Errorf("no se pudo resolver el proyecto actual desde %s", cwd)
}
