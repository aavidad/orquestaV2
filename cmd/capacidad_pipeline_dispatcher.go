package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"orquesta/capacidadapp"
	"orquesta/db"
	"orquesta/gitaplicacion"
	"orquesta/microprogramacionapp"
	"orquesta/runtimeagente"
	"orquesta/runtimesapp"
)

type despachadorPipelineOperativo struct{}

var pipelineSubagentLaunchFn = func(req supervisorSubagentLaunchRequest) (*supervisorSubagentLaunchResult, error) {
	return launchClaudeSubagentExternal(req)
}

func (despachadorPipelineOperativo) DespacharPipeline(entrada capacidadapp.SolicitudDespachoPipeline) (*capacidadapp.ResultadoDespachoPipeline, error) {
	despacho := entrada.Despacho
	if despacho == nil {
		return &capacidadapp.ResultadoDespachoPipeline{
			Estado: "sin_despacho",
			Motivo: "paso sin despacho asociado",
		}, nil
	}
	switch strings.ToLower(strings.TrimSpace(despacho.Carril)) {
	case "premium_worktree", "revision_diff":
		agente := strings.TrimSpace(despacho.AgenteSugerido)
		if agente == "" {
			return &capacidadapp.ResultadoDespachoPipeline{
				Estado: "sin_agente",
				Motivo: "no hay agente operativo compatible con el carril",
			}, nil
		}
		proyecto, err := runtimesService.GetProject(strings.TrimSpace(entrada.ProyectoSlug))
		if err != nil {
			return nil, err
		}
		if proyecto == nil {
			return nil, fmt.Errorf("proyecto no encontrado: %s", strings.TrimSpace(entrada.ProyectoSlug))
		}
		if bloqueado, estadoCuota, err := agenteBloqueadoPorCuotaPipeline(agente); err != nil {
			return nil, err
		} else if bloqueado {
			if pendiente, err := existeRuntimeMailboxPipelinePendiente(agente, &proyecto.ID); err != nil {
				return nil, err
			} else if pendiente {
				return &capacidadapp.ResultadoDespachoPipeline{
					Estado: "cuota_bloqueada_con_mailbox_pendiente",
					Motivo: fmt.Sprintf("agente %s en cuota (%s) con pipeline_local ya pendiente", agente, estadoCuota),
				}, nil
			}
			return &capacidadapp.ResultadoDespachoPipeline{
				Estado: "cuota_bloqueada",
				Motivo: fmt.Sprintf("agente %s en cuota (%s); no se encola trabajo nuevo", agente, estadoCuota),
			}, nil
		}
		if pendiente, err := existeRuntimeMailboxPipelinePendiente(agente, &proyecto.ID); err != nil {
			return nil, err
		} else if pendiente {
			return &capacidadapp.ResultadoDespachoPipeline{
				Estado: "mailbox_ya_pendiente",
				Motivo: "ya existe pipeline_local pendiente para el agente y proyecto",
			}, nil
		}
		if ambiguo, detalle, err := agenteTieneRuntimeAmbiguoPipeline(agente); err != nil {
			return nil, err
		} else if ambiguo {
			return &capacidadapp.ResultadoDespachoPipeline{
				Estado: "runtime_ambiguo",
				Motivo: detalle,
			}, nil
		}
		if pendienteOtro, detalle, err := existeRuntimeMailboxPipelinePendienteEnOtroProyecto(agente, proyecto.ID); err != nil {
			return nil, err
		} else if pendienteOtro {
			return &capacidadapp.ResultadoDespachoPipeline{
				Estado: "agente_ocupado_otro_proyecto",
				Motivo: detalle,
			}, nil
		}
		if ocupado, detalle, err := agenteOcupadoEnOtroProyectoPipeline(agente, proyecto.ID); err != nil {
			return nil, err
		} else if ocupado {
			return &capacidadapp.ResultadoDespachoPipeline{
				Estado: "agente_ocupado_otro_proyecto",
				Motivo: detalle,
			}, nil
		}
		if err := asegurarOwnershipTareaDespachoPremium(despacho.TareaObjetivoID, agente); err != nil {
			return nil, err
		}
		var forkPayload map[string]any
		payload := map[string]any{
			"from_agente":       "server",
			"to_agente":         agente,
			"kind":              "pipeline_local",
			"accion":            firstNonEmpty(strings.TrimSpace(despacho.AccionTarea), "continuar_trabajo"),
			"texto":             strings.TrimSpace(despacho.Motivo),
			"instruction":       construirInstructionPipeline(*despacho),
			"source":            "pipeline_local",
			"fase":              strings.TrimSpace(despacho.Fase),
			"perfil_tarea":      strings.TrimSpace(despacho.PerfilTarea),
			"carril":            strings.TrimSpace(despacho.Carril),
			"entrega_canonica":  strings.TrimSpace(despacho.EntregaCanonica),
			"tarea_objetivo_id": despacho.TareaObjetivoID,
			"tarea_objetivo":    strings.TrimSpace(despacho.TareaObjetivo),
			"write_set":         append([]string(nil), despacho.WriteSet...),
			"simbolos_foco":     strings.TrimSpace(despacho.SimbolosFoco),
			"tests_minimos":     strings.TrimSpace(despacho.TestsMinimos),
		}
		if fork := cargarForkFuncionPipeline(despacho.TareaObjetivoID); fork != nil {
			forkPayload = fork
			payload["fork_funcion"] = fork
			if variantes := construirVariantesCandidatasFork(fork); len(variantes) > 0 {
				payload["variantes_candidatas"] = variantes
			}
		}
		if despacho.Paralelismo != nil {
			payload["paralelismo"] = map[string]any{
				"puede_abrir_subagentes": despacho.Paralelismo.PuedeAbrirSubagentes,
				"max_subagentes":         despacho.Paralelismo.MaxSubagentes,
				"motivo":                 strings.TrimSpace(despacho.Paralelismo.Motivo),
			}
		}
		payloadJSON, err := jsonMarshalPipelinePayload(payload)
		if err != nil {
			return nil, err
		}
		if _, err := runtimesService.SendRuntimeMailbox(&db.RuntimeMailboxMessage{
			FromAgente:  "server",
			ToAgente:    agente,
			ProyectoID:  &proyecto.ID,
			Kind:        "pipeline_local",
			PayloadJSON: payloadJSON,
		}); err != nil {
			return nil, err
		}
		if forkPayload != nil {
			modelosAudit := stringSliceFromAny(forkPayload["selected_models"])
			if len(modelosAudit) == 0 {
				modelosAudit = stringSliceFromAny(forkPayload["modelos_candidatos"])
			}
			detalle := fmt.Sprintf("slug=%s tarea=%d funcion=%s modelos=%s fork_lines=%d decision_mode=%s",
				strings.TrimSpace(entrada.ProyectoSlug),
				despacho.TareaObjetivoID,
				strings.TrimSpace(stringFromAny(forkPayload["funcion_objetivo"])),
				strings.Join(modelosAudit, ","),
				intFromAny(forkPayload["fork_lines"]),
				strings.TrimSpace(stringFromAny(forkPayload["decision_mode"])),
			)
			db.Audit("server", "pipeline_dispatch_fork_funcion", "tarea", despacho.TareaObjetivoID, detalle)
		}
		subagentsLaunched := lanzarSubagentesForkFuncionSidecar(strings.TrimSpace(entrada.ProyectoSlug), despacho, forkPayload)
		if subagentsLaunched == 0 {
			subagentsLaunched = lanzarSubagentesPipelineSidecar(strings.TrimSpace(entrada.ProyectoSlug), despacho)
		}
		if activo, err := existeRuntimeHandleActivoPipeline(agente, &proyecto.ID); err != nil {
			return nil, err
		} else if activo {
			return &capacidadapp.ResultadoDespachoPipeline{
				Estado:            "mailbox_encolado_runtime_existente",
				Motivo:            "pipeline_local encolado; ya existe un runtime activo para el agente y proyecto",
				SubagentsLaunched: subagentsLaunched,
			}, nil
		}
		conectorStart, modeloStart := resolverConectorYModeloStartPipeline(agente, strings.TrimSpace(despacho.ObjetivoModelo))
		startID, _, err := runtimesService.EnqueueAgentControl(runtimesapp.AgentControlRequest{
			Agente:       agente,
			Proyecto:     strings.TrimSpace(entrada.ProyectoSlug),
			Accion:       "start",
			Conector:     conectorStart,
			Modelo:       modeloStart,
			Perfil:       strings.TrimSpace(despacho.PerfilTarea),
			Por:          "orquesta",
			Motivo:       strings.TrimSpace(despacho.Motivo),
			Razonamiento: razonamientoPorCarril(strings.TrimSpace(despacho.Carril)),
			TareaID:      &despacho.TareaObjetivoID,
		})
		if err != nil {
			return nil, err
		}
		if err := runtimesService.MarcarRuntimeOrderDispatchNotificado(startID, "despacho de pipeline encolado"); err != nil {
			db.Audit("server", "pipeline_dispatch_notify_error", "orden", startID, err.Error())
		}
		return &capacidadapp.ResultadoDespachoPipeline{
			Estado:            "encolado",
			Motivo:            "mailbox y start encolados por la vía canónica",
			StartOrderID:      &startID,
			SubagentsLaunched: subagentsLaunched,
		}, nil
	case "microprogramacion_local":
		if despacho.TareaObjetivoID <= 0 {
			return &capacidadapp.ResultadoDespachoPipeline{
				Estado: "sin_tarea",
				Motivo: "microprogramacion_local requiere una tarea_objetivo_id valida",
			}, nil
		}
		specs, err := microprogramacionService.Listar(microprogramacionapp.FiltroEspecificaciones{
			TareaID: &despacho.TareaObjetivoID,
		})
		if err != nil {
			return nil, err
		}
		var spec *microprogramacionapp.EspecificacionFuncion
		for _, s := range specs {
			if s != nil && strings.EqualFold(strings.TrimSpace(string(s.Estado)), string(microprogramacionapp.EstadoEspecificacionActiva)) {
				spec = s
				break
			}
		}
		if spec == nil {
			return &capacidadapp.ResultadoDespachoPipeline{
				Estado: "requiere_especificacion",
				Motivo: fmt.Sprintf("no hay especificacion activa para la tarea #%d", despacho.TareaObjetivoID),
			}, nil
		}
		emitida, err := microprogramacionService.Emitir(spec.ID, microprogramacionapp.EntradaEmitirMicrotarea{
			Contexto: strings.TrimSpace(despacho.Motivo),
		})
		if err != nil {
			return nil, err
		}
		agente := strings.TrimSpace(despacho.AgenteSugerido)
		if agente == "" {
			return &capacidadapp.ResultadoDespachoPipeline{
				Estado: "sin_agente",
				Motivo: "no hay agente operativo compatible con el carril microprogramacion",
			}, nil
		}
		proyecto, err := runtimesService.GetProject(strings.TrimSpace(entrada.ProyectoSlug))
		if err != nil {
			return nil, err
		}
		if proyecto == nil {
			return nil, fmt.Errorf("proyecto no encontrado: %s", strings.TrimSpace(entrada.ProyectoSlug))
		}
		payload := map[string]any{
			"from_agente":       "server",
			"to_agente":         agente,
			"kind":              "microprogramacion",
			"especificacion_id": spec.ID,
			"tarea_id":          despacho.TareaObjetivoID,
			"texto":             emitida.Mensaje,
			"source":            "pipeline_local",
			"fase":              strings.TrimSpace(despacho.Fase),
		}
		payloadJSON, err := jsonMarshalPipelinePayload(payload)
		if err != nil {
			return nil, err
		}
		if _, err := runtimesService.SendRuntimeMailbox(&db.RuntimeMailboxMessage{
			FromAgente:  "server",
			ToAgente:    agente,
			ProyectoID:  &proyecto.ID,
			Kind:        "microprogramacion",
			PayloadJSON: payloadJSON,
		}); err != nil {
			return nil, err
		}
		if activo, err := existeRuntimeHandleActivoPipeline(agente, &proyecto.ID); err != nil {
			return nil, err
		} else if activo {
			return &capacidadapp.ResultadoDespachoPipeline{
				Estado: "mailbox_encolado_runtime_existente",
				Motivo: fmt.Sprintf("microprogramacion spec #%d encolada sobre runtime ya activo", spec.ID),
			}, nil
		}
		conectorStart, modeloStart := resolverConectorYModeloStartPipeline(agente, strings.TrimSpace(despacho.ObjetivoModelo))
		startID, _, err := runtimesService.EnqueueAgentControl(runtimesapp.AgentControlRequest{
			Agente:       agente,
			Proyecto:     strings.TrimSpace(entrada.ProyectoSlug),
			Accion:       "start",
			Conector:     conectorStart,
			Modelo:       modeloStart,
			Perfil:       strings.TrimSpace(despacho.PerfilTarea),
			Por:          "orquesta",
			Motivo:       "microprogramacion: " + spec.Titulo,
			Razonamiento: "high",
			TareaID:      &despacho.TareaObjetivoID,
		})
		if err != nil {
			return nil, err
		}
		if err := runtimesService.MarcarRuntimeOrderDispatchNotificado(startID, "microprogramacion encolada"); err != nil {
			db.Audit("server", "pipeline_dispatch_notify_error", "orden", startID, err.Error())
		}
		return &capacidadapp.ResultadoDespachoPipeline{
			Estado:       "encolado",
			Motivo:       fmt.Sprintf("microprogramacion spec #%d encolada", spec.ID),
			StartOrderID: &startID,
		}, nil
	case "determinista_app":
		if strings.EqualFold(strings.TrimSpace(despacho.Fase), "integracion") {
			agente := firstNonEmpty(strings.TrimSpace(despacho.AgenteTarea), strings.TrimSpace(despacho.AgenteSugerido))
			if agente == "" {
				return &capacidadapp.ResultadoDespachoPipeline{
					Estado: "sin_agente_origen",
					Motivo: "la integración requiere conocer el agente que produjo la worktree activa",
				}, nil
			}
			worktree, err := gitService.ResolveActiveWorktree(strings.TrimSpace(entrada.ProyectoSlug), agente)
			if err != nil {
				return &capacidadapp.ResultadoDespachoPipeline{
					Estado: "sin_worktree_activa",
					Motivo: err.Error(),
				}, nil
			}
			targetBranch := firstNonEmpty(strings.TrimSpace(worktree.BaseRef), "main")
			pendientes, err := gitService.ListMerges(strings.TrimSpace(entrada.ProyectoSlug), "pendiente")
			if err != nil {
				return nil, err
			}
			for _, item := range pendientes {
				if item == nil {
					continue
				}
				if strings.EqualFold(strings.TrimSpace(item.SourceBranch), strings.TrimSpace(worktree.Branch)) &&
					strings.EqualFold(strings.TrimSpace(item.TargetBranch), targetBranch) {
					id := item.ID
					return &capacidadapp.ResultadoDespachoPipeline{
						Estado:         "merge_ya_pendiente",
						Motivo:         "ya existe una solicitud de merge pendiente para la worktree activa",
						RuntimeOrderID: &id,
					}, nil
				}
			}
			id, err := gitService.CreateMerge(gitaplicacion.CreateMergeInput{
				ProyectoSlug: strings.TrimSpace(entrada.ProyectoSlug),
				SourceBranch: strings.TrimSpace(worktree.Branch),
				TargetBranch: targetBranch,
				RequestedBy:  "orquesta",
				Estado:       "pendiente",
				Notas: fmt.Sprintf("pipeline_local integracion tarea_id=%d tarea=%s agente=%s",
					despacho.TareaObjetivoID,
					strings.TrimSpace(despacho.TareaObjetivo),
					agente,
				),
			})
			if err != nil {
				return nil, err
			}
			return &capacidadapp.ResultadoDespachoPipeline{
				Estado:         "merge_solicitado",
				Motivo:         "solicitud de merge creada por integración determinista",
				RuntimeOrderID: &id,
			}, nil
		}
		return &capacidadapp.ResultadoDespachoPipeline{
			Estado: "determinista_app",
			Motivo: "la fase actual no delega en un agente",
		}, nil
	default:
		return &capacidadapp.ResultadoDespachoPipeline{
			Estado: "carril_no_soportado",
			Motivo: fmt.Sprintf("carril %q no soportado por el despachador operativo", strings.TrimSpace(despacho.Carril)),
		}, nil
	}
}

func cargarForkFuncionPipeline(tareaID int64) map[string]any {
	if tareaID <= 0 {
		return nil
	}
	tarea, err := db.GetTarea(tareaID)
	if err != nil || tarea == nil {
		return nil
	}
	spec := parseRepoFunctionForkSpecFromNotes(strings.TrimSpace(tarea.Notas))
	if spec == nil {
		return nil
	}
	return map[string]any{
		"schema_version":         "fork_funcion_v1",
		"funcion_objetivo":       strings.TrimSpace(spec.FuncionObjetivo),
		"write_set":              append([]string(nil), spec.WriteSet...),
		"modelos_candidatos":     append([]string(nil), spec.ModelosCandidatos...),
		"materia":                strings.TrimSpace(spec.Materia),
		"fork_lines":             spec.ForkLines,
		"selected_models":        append([]string(nil), spec.SelectedModels...),
		"decision_mode":          strings.TrimSpace(spec.DecisionMode),
		"decision_reason":        strings.TrimSpace(spec.DecisionReason),
		"preservar_arquitectura": spec.PreservarArquitectura,
	}
}

func construirVariantesCandidatasFork(fork map[string]any) []map[string]any {
	if len(fork) == 0 {
		return nil
	}
	modelos := stringSliceFromAny(fork["selected_models"])
	if len(modelos) == 0 {
		modelos = stringSliceFromAny(fork["modelos_candidatos"])
	}
	if len(modelos) == 0 {
		return nil
	}
	forkLines := intFromAny(fork["fork_lines"])
	if forkLines > 0 && len(modelos) > forkLines {
		modelos = modelos[:forkLines]
	}
	writeSet := stringSliceFromAny(fork["write_set"])
	funcionObjetivo := strings.TrimSpace(stringFromAny(fork["funcion_objetivo"]))
	preservar := boolFromAny(fork["preservar_arquitectura"])
	materia := strings.TrimSpace(stringFromAny(fork["materia"]))
	decisionMode := strings.TrimSpace(stringFromAny(fork["decision_mode"]))
	decisionReason := strings.TrimSpace(stringFromAny(fork["decision_reason"]))
	out := make([]map[string]any, 0, len(modelos))
	for idx, modelo := range modelos {
		modelo = strings.TrimSpace(modelo)
		if modelo == "" {
			continue
		}
		out = append(out, map[string]any{
			"indice":                 idx + 1,
			"modelo":                 modelo,
			"funcion_objetivo":       funcionObjetivo,
			"write_set":              append([]string(nil), writeSet...),
			"preservar_arquitectura": preservar,
			"materia":                materia,
			"fork_lines":             forkLines,
			"decision_mode":          decisionMode,
			"decision_reason":        decisionReason,
			"estado":                 "pendiente",
		})
	}
	return out
}

func stringSliceFromAny(raw any) []string {
	switch v := raw.(type) {
	case []string:
		return append([]string(nil), v...)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s := strings.TrimSpace(stringFromAny(item)); s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func stringFromAny(raw any) string {
	switch v := raw.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf("%v", raw)
	}
}

func intFromAny(raw any) int {
	switch v := raw.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	case string:
		var out int
		fmt.Sscanf(strings.TrimSpace(v), "%d", &out)
		return out
	default:
		return 0
	}
}

func boolFromAny(raw any) bool {
	switch v := raw.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(strings.TrimSpace(v), "true")
	default:
		return false
	}
}

func lanzarSubagentesPipelineSidecar(proyectoSlug string, despacho *capacidadapp.DespachoPipelineLocal) int {
	if !pipelinePermiteSubagentes(despacho) {
		return 0
	}
	if pipelineSubagentLaunchFn == nil || strings.TrimSpace(resolveClaudeSubagentLauncher()) == "" {
		return 0
	}
	slices := pipelineWriteSetSlices(despacho.WriteSet, despacho.Paralelismo.MaxSubagentes)
	if len(slices) == 0 {
		return 0
	}
	launched := 0
	for idx, slice := range slices {
		req := supervisorSubagentLaunchRequest{
			Supervisor:   "OpenClaw",
			Proyecto:     strings.TrimSpace(proyectoSlug),
			Name:         pipelineSubagentName(proyectoSlug, despacho, idx+1),
			Description:  pipelineSubagentDescription(despacho, idx+1, len(slices)),
			Prompt:       pipelineSubagentPrompt(proyectoSlug, despacho, slice, idx+1, len(slices)),
			SubagentType: "general-purpose",
			Metadata: map[string]any{
				"source":             "pipeline_local_parallel",
				"pipeline_phase":     strings.TrimSpace(despacho.Fase),
				"pipeline_lane":      strings.TrimSpace(despacho.Carril),
				"task_id":            despacho.TareaObjetivoID,
				"task_title":         strings.TrimSpace(despacho.TareaObjetivo),
				"slice_index":        idx + 1,
				"slice_total":        len(slices),
				"write_set_slice":    append([]string(nil), slice...),
				"tests_minimos":      strings.TrimSpace(despacho.TestsMinimos),
				"agente_supervisado": strings.TrimSpace(despacho.AgenteSugerido),
			},
		}
		if _, err := pipelineSubagentLaunchFn(req); err != nil {
			db.Audit("server", "pipeline_subagent_launch_error", "proyecto", 0, fmt.Sprintf("%s: %v", strings.TrimSpace(proyectoSlug), err))
			continue
		}
		launched++
	}
	return launched
}

func lanzarSubagentesForkFuncionSidecar(proyectoSlug string, despacho *capacidadapp.DespachoPipelineLocal, fork map[string]any) int {
	if !pipelinePermiteSubagentesForkFuncion(despacho, fork) {
		return 0
	}
	if pipelineSubagentLaunchFn == nil || strings.TrimSpace(resolveClaudeSubagentLauncher()) == "" {
		return 0
	}
	variantes := construirVariantesCandidatasFork(fork)
	if len(variantes) < 2 {
		return 0
	}
	writeSet := stringSliceFromAny(fork["write_set"])
	if len(writeSet) == 0 {
		writeSet = append([]string(nil), despacho.WriteSet...)
	}
	total := len(variantes)
	launched := 0
	for idx, variante := range variantes {
		modelo := strings.TrimSpace(stringFromAny(variante["modelo"]))
		if modelo == "" {
			continue
		}
		req := supervisorSubagentLaunchRequest{
			Supervisor:   "OpenClaw",
			Proyecto:     strings.TrimSpace(proyectoSlug),
			Name:         pipelineForkSubagentName(proyectoSlug, despacho, idx+1, modelo),
			Description:  pipelineForkSubagentDescription(despacho, idx+1, total, modelo),
			Prompt:       pipelineForkSubagentPrompt(proyectoSlug, despacho, fork, variante, writeSet, idx+1, total),
			SubagentType: "general-purpose",
			Model:        modelo,
			Metadata: map[string]any{
				"source":             "pipeline_local_parallel",
				"parallel_mode":      "fork_funcion",
				"pipeline_phase":     strings.TrimSpace(despacho.Fase),
				"pipeline_lane":      strings.TrimSpace(despacho.Carril),
				"task_id":            despacho.TareaObjetivoID,
				"task_title":         strings.TrimSpace(despacho.TareaObjetivo),
				"slice_index":        idx + 1,
				"slice_total":        total,
				"write_set_slice":    append([]string(nil), writeSet...),
				"tests_minimos":      strings.TrimSpace(despacho.TestsMinimos),
				"agente_supervisado": strings.TrimSpace(despacho.AgenteSugerido),
				"fork_function":      strings.TrimSpace(stringFromAny(fork["funcion_objetivo"])),
				"fork_model":         modelo,
				"fork_decision_mode": strings.TrimSpace(stringFromAny(fork["decision_mode"])),
				"fork_lines":         intFromAny(fork["fork_lines"]),
				"fork_materia":       strings.TrimSpace(stringFromAny(fork["materia"])),
			},
		}
		if _, err := pipelineSubagentLaunchFn(req); err != nil {
			db.Audit("server", "pipeline_fork_subagent_launch_error", "tarea", despacho.TareaObjetivoID, fmt.Sprintf("%s modelo=%s: %v", strings.TrimSpace(proyectoSlug), modelo, err))
			continue
		}
		launched++
	}
	if launched > 0 {
		db.Audit("server", "pipeline_fork_subagents_launched", "tarea", despacho.TareaObjetivoID, fmt.Sprintf("slug=%s funcion=%s modelos=%d", strings.TrimSpace(proyectoSlug), strings.TrimSpace(stringFromAny(fork["funcion_objetivo"])), launched))
	}
	return launched
}

func pipelinePermiteSubagentes(despacho *capacidadapp.DespachoPipelineLocal) bool {
	if despacho == nil || despacho.Paralelismo == nil {
		return false
	}
	if !despacho.Paralelismo.PuedeAbrirSubagentes || despacho.Paralelismo.MaxSubagentes <= 0 {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(despacho.Fase), "implementacion") {
		return false
	}
	if len(despacho.WriteSet) < 2 || strings.TrimSpace(despacho.TestsMinimos) == "" {
		return false
	}
	return true
}

func pipelinePermiteSubagentesForkFuncion(despacho *capacidadapp.DespachoPipelineLocal, fork map[string]any) bool {
	if despacho == nil || len(fork) == 0 {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(despacho.Fase), "implementacion") {
		return false
	}
	if strings.TrimSpace(despacho.TestsMinimos) == "" {
		return false
	}
	selected := stringSliceFromAny(fork["selected_models"])
	if len(selected) < 2 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(stringFromAny(fork["decision_mode"]))) {
	case "auto":
		return true
	default:
		return false
	}
}

func pipelineWriteSetSlices(writeSet []string, maxSubagentes int) [][]string {
	normalized := make([]string, 0, len(writeSet))
	seen := map[string]struct{}{}
	for _, item := range writeSet {
		ruta := strings.TrimSpace(item)
		if ruta == "" {
			continue
		}
		if _, ok := seen[ruta]; ok {
			continue
		}
		seen[ruta] = struct{}{}
		normalized = append(normalized, ruta)
	}
	sort.Strings(normalized)
	if len(normalized) < 2 || maxSubagentes <= 0 {
		return nil
	}
	numSlices := maxSubagentes
	if len(normalized) < numSlices {
		numSlices = len(normalized)
	}
	if numSlices <= 0 {
		return nil
	}
	slices := make([][]string, numSlices)
	for idx, ruta := range normalized {
		slot := idx % numSlices
		slices[slot] = append(slices[slot], ruta)
	}
	out := make([][]string, 0, len(slices))
	for _, slice := range slices {
		if len(slice) > 0 {
			out = append(out, slice)
		}
	}
	return out
}

func pipelineSubagentName(proyectoSlug string, despacho *capacidadapp.DespachoPipelineLocal, index int) string {
	slug := strings.TrimSpace(proyectoSlug)
	if slug == "" {
		slug = "pipeline"
	}
	fase := strings.TrimSpace(despacho.Fase)
	if fase == "" {
		fase = "implementacion"
	}
	return fmt.Sprintf("OpenClaw-%s-%s-slice-%d", slug, fase, index)
}

func pipelineForkSubagentName(proyectoSlug string, despacho *capacidadapp.DespachoPipelineLocal, index int, modelo string) string {
	slug := strings.TrimSpace(proyectoSlug)
	if slug == "" {
		slug = "pipeline"
	}
	fase := strings.TrimSpace(despacho.Fase)
	if fase == "" {
		fase = "implementacion"
	}
	modelo = strings.NewReplacer(":", "-", "/", "-", " ", "-").Replace(strings.TrimSpace(modelo))
	return fmt.Sprintf("OpenClaw-%s-%s-fork-%d-%s", slug, fase, index, modelo)
}

func pipelineSubagentDescription(despacho *capacidadapp.DespachoPipelineLocal, index, total int) string {
	base := firstNonEmpty(strings.TrimSpace(despacho.TareaObjetivo), strings.TrimSpace(despacho.Motivo), "pipeline_local")
	return fmt.Sprintf("Slice %d/%d de %s", index, total, base)
}

func pipelineForkSubagentDescription(despacho *capacidadapp.DespachoPipelineLocal, index, total int, modelo string) string {
	base := firstNonEmpty(strings.TrimSpace(despacho.TareaObjetivo), strings.TrimSpace(despacho.Motivo), "fork_funcion")
	return fmt.Sprintf("Fork %d/%d de %s con modelo %s", index, total, base, strings.TrimSpace(modelo))
}

func pipelineSubagentPrompt(proyectoSlug string, despacho *capacidadapp.DespachoPipelineLocal, writeSet []string, index, total int) string {
	lineas := []string{
		"TRABAJO DE PIPELINE LOCAL EN PARALELO.",
		fmt.Sprintf("Proyecto: %s", strings.TrimSpace(proyectoSlug)),
		fmt.Sprintf("Slice: %d/%d", index, total),
		fmt.Sprintf("Fase: %s", strings.TrimSpace(despacho.Fase)),
		fmt.Sprintf("Objetivo: %s", firstNonEmpty(strings.TrimSpace(despacho.TareaObjetivo), strings.TrimSpace(despacho.Motivo))),
		"Trabaja solo dentro de este write_set disjunto:",
		strings.Join(writeSet, ", "),
	}
	if foco := strings.TrimSpace(despacho.SimbolosFoco); foco != "" {
		lineas = append(lineas, "Simbolos foco compartidos: "+foco)
	}
	if tests := strings.TrimSpace(despacho.TestsMinimos); tests != "" {
		lineas = append(lineas, "Tests minimos del slice: "+tests)
	}
	lineas = append(lineas,
		"No toques archivos fuera del write_set asignado.",
		"Si el cambio exige salir del write_set, para y reporta BLOQUEO.",
		"Prioriza dejar el slice listo para integración posterior.",
	)
	return strings.Join(lineas, "\n")
}

func pipelineForkSubagentPrompt(proyectoSlug string, despacho *capacidadapp.DespachoPipelineLocal, fork map[string]any, variante map[string]any, writeSet []string, index, total int) string {
	modelo := strings.TrimSpace(stringFromAny(variante["modelo"]))
	funcionObjetivo := strings.TrimSpace(stringFromAny(fork["funcion_objetivo"]))
	lineas := []string{
		"TRABAJO DE FORK DE FUNCION EN PARALELO.",
		fmt.Sprintf("Proyecto: %s", strings.TrimSpace(proyectoSlug)),
		fmt.Sprintf("Fork: %d/%d", index, total),
		fmt.Sprintf("Modelo objetivo: %s", modelo),
		fmt.Sprintf("Fase: %s", strings.TrimSpace(despacho.Fase)),
		fmt.Sprintf("Funcion objetivo: %s", funcionObjetivo),
		fmt.Sprintf("Objetivo: %s", firstNonEmpty(strings.TrimSpace(despacho.TareaObjetivo), strings.TrimSpace(despacho.Motivo))),
		"Trabaja sobre la misma función y el mismo write_set, proponiendo una variante mejor sin romper la arquitectura existente.",
		"Write_set autorizado:",
		strings.Join(writeSet, ", "),
	}
	if materia := strings.TrimSpace(stringFromAny(fork["materia"])); materia != "" {
		lineas = append(lineas, "Materia dominante: "+materia)
	}
	if forkLines := intFromAny(fork["fork_lines"]); forkLines > 0 {
		lineas = append(lineas, fmt.Sprintf("Numero de lineas de fork decidido por Orquesta: %d", forkLines))
	}
	if boolFromAny(fork["preservar_arquitectura"]) {
		lineas = append(lineas, "Debes preservar arquitectura y estilo de la app. No pierdas hexagonalidad ni contratos existentes.")
	}
	if tests := strings.TrimSpace(despacho.TestsMinimos); tests != "" {
		lineas = append(lineas, "Tests minimos de la variante: "+tests)
	}
	lineas = append(lineas,
		"No toques archivos fuera del write_set.",
		"Si la mejora exige salir del write_set, para y reporta BLOQUEO.",
		"Entrega una variante clara y comparable frente a las otras, priorizando eficiencia sin perder seguridad ni arquitectura.",
	)
	return strings.Join(lineas, "\n")
}

func asegurarOwnershipTareaDespachoPremium(tareaID int64, agente string) error {
	agente = strings.TrimSpace(agente)
	if tareaID <= 0 || agente == "" {
		return nil
	}
	tarea, err := tareasService.Get(tareaID)
	if err != nil {
		if err == sql.ErrNoRows || strings.Contains(strings.ToLower(err.Error()), "no rows") {
			return nil
		}
		return err
	}
	if tarea == nil {
		return nil
	}
	actual := ""
	if tarea.Agente != nil {
		actual = strings.TrimSpace(*tarea.Agente)
	}
	estado := strings.ToLower(strings.TrimSpace(string(tarea.Estado)))
	if !strings.EqualFold(actual, agente) {
		switch estado {
		case string(db.TareaBacklog), string(db.TareaLibre):
			if err := tareasService.Take(tareaID, agente); err != nil {
				return err
			}
		default:
			if err := tareasService.Reassign(tareaID, agente); err != nil {
				return err
			}
		}
	}
	tarea, err = tareasService.Get(tareaID)
	if err != nil {
		if err == sql.ErrNoRows || strings.Contains(strings.ToLower(err.Error()), "no rows") {
			return nil
		}
		return err
	}
	if tarea == nil {
		return nil
	}
	if tarea.Estado != db.TareaEnProgreso {
		nombreAgente := agente
		if tarea.Agente != nil && strings.TrimSpace(*tarea.Agente) != "" {
			nombreAgente = strings.TrimSpace(*tarea.Agente)
		}
		if err := tareasService.Start(tareaID, nombreAgente); err != nil {
			return err
		}
	}
	return nil
}

func jsonMarshalPipelinePayload(payload map[string]any) (string, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func construirInstructionPipeline(despacho capacidadapp.DespachoPipelineLocal) string {
	lineas := []string{
		fmt.Sprintf("FASE: %s", strings.TrimSpace(despacho.Fase)),
		fmt.Sprintf("ACCION: %s", firstNonEmpty(strings.TrimSpace(despacho.AccionTarea), "continuar_trabajo")),
		fmt.Sprintf("TAREA: #%d %s", despacho.TareaObjetivoID, strings.TrimSpace(despacho.TareaObjetivo)),
		fmt.Sprintf("CARRIL: %s", strings.TrimSpace(despacho.Carril)),
		fmt.Sprintf("ENTREGA: %s", strings.TrimSpace(despacho.EntregaCanonica)),
	}
	if despacho.RequiereWorktree {
		lineas = append(lineas, "REGLA: trabaja solo en tu worktree activa")
	}
	if strings.TrimSpace(despacho.PerfilTarea) != "" {
		lineas = append(lineas, "PERFIL: "+strings.TrimSpace(despacho.PerfilTarea))
	}
	if len(despacho.WriteSet) > 0 {
		lineas = append(lineas, "WRITE_SET: "+strings.Join(despacho.WriteSet, ", "))
	}
	if strings.TrimSpace(despacho.SimbolosFoco) != "" {
		lineas = append(lineas, "Simbolos foco: "+strings.TrimSpace(despacho.SimbolosFoco))
	}
	if strings.TrimSpace(despacho.TestsMinimos) != "" {
		lineas = append(lineas, "Tests minimos: "+strings.TrimSpace(despacho.TestsMinimos))
	}
	if despacho.FinishApp {
		lineas = append(lineas, "MODO: finish_app")
		lineas = append(lineas, "REGLA: no pares al cerrar solo esta tarea; sigue enlazando el siguiente frente util hasta completar la app salvo bloqueo real")
	}
	if despacho.Paralelismo != nil {
		if despacho.Paralelismo.PuedeAbrirSubagentes {
			lineas = append(lineas, fmt.Sprintf("PARALELISMO: puedes abrir hasta %d subagentes solo si el trabajo se puede partir en slices disjuntos", despacho.Paralelismo.MaxSubagentes))
		} else {
			lineas = append(lineas, "PARALELISMO: no abras subagentes para este frente")
		}
		if motivo := strings.TrimSpace(despacho.Paralelismo.Motivo); motivo != "" {
			lineas = append(lineas, "Motivo paralelismo: "+motivo)
		}
	}
	lineas = append(lineas,
		"REGLA: no cambies nada fuera del alcance de la tarea",
		"REGLA: antes de terminar, valida tu entrega con los tests mínimos relevantes",
	)
	return strings.Join(lineas, "\n")
}

func razonamientoPorCarril(carril string) string {
	switch strings.ToLower(strings.TrimSpace(carril)) {
	case "revision_diff":
		return "high"
	case "premium_worktree":
		return "medium"
	default:
		return ""
	}
}

func resolverConectorYModeloStartPipeline(agente, modelo string) (string, string) {
	conector := strings.TrimSpace(runtimeagente.ConectorPorDefectoAgente(strings.TrimSpace(agente)))
	modelo = strings.TrimSpace(modelo)
	if conector == "" {
		return "", modelo
	}
	if !runtimeagente.ModeloCompatibleConConector(runtimeagente.ConnectorConfig{
		Slug:    conector,
		Comando: comandoCanonicoConectorPipeline(conector),
	}, modelo) {
		modelo = ""
	}
	return conector, modelo
}

func comandoCanonicoConectorPipeline(conector string) string {
	switch strings.ToLower(strings.TrimSpace(conector)) {
	case "gemini-cli":
		return "gemini"
	case "claude-code":
		return "claude-code"
	case "codex-cli":
		return "codex"
	case "ollama-cli", "ollama_pool_local", "ollama-pool-local":
		return "ollama"
	default:
		return strings.TrimSpace(conector)
	}
}

func existeRuntimeMailboxPipelinePendiente(agente string, proyectoID *int64) (bool, error) {
	estado := "pendiente"
	agente = strings.TrimSpace(agente)
	items, err := runtimesService.ListRuntimeMailbox(db.FiltroRuntimeMailbox{
		ToAgente:   &agente,
		ProyectoID: proyectoID,
		Estado:     &estado,
	})
	if err != nil {
		return false, err
	}
	for _, item := range items {
		if item == nil || !strings.EqualFold(strings.TrimSpace(item.Kind), "pipeline_local") {
			continue
		}
		return true, nil
	}
	return false, nil
}

func existeRuntimeMailboxPipelinePendienteEnOtroProyecto(agente string, proyectoID int64) (bool, string, error) {
	estado := "pendiente"
	agente = strings.TrimSpace(agente)
	items, err := runtimesService.ListRuntimeMailbox(db.FiltroRuntimeMailbox{
		ToAgente: &agente,
		Estado:   &estado,
	})
	if err != nil {
		return false, "", err
	}
	for _, item := range items {
		if item == nil || !strings.EqualFold(strings.TrimSpace(item.Kind), "pipeline_local") {
			continue
		}
		if item.ProyectoID == nil || *item.ProyectoID <= 0 || *item.ProyectoID == proyectoID {
			continue
		}
		return true, fmt.Sprintf("agente %s ya tiene pipeline_local pendiente en proyecto_id=%d", agente, *item.ProyectoID), nil
	}
	return false, "", nil
}

func agenteBloqueadoPorCuotaPipeline(agente string) (bool, string, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return false, "", nil
	}
	info, err := db.GetAgente(agente)
	if err != nil && err != sql.ErrNoRows {
		return false, "", err
	}
	if info != nil {
		if shouldPause, _ := agenteDebePausarPorPresupuestoVisible(info); shouldPause {
			estadoVisible := strings.TrimSpace(strings.ToLower(info.PresupuestoEstado))
			if estadoVisible == "" {
				estadoVisible = strings.TrimSpace(strings.ToLower(info.EstadoCuota))
			}
			return true, firstNonEmpty(estadoVisible, "cuota_visible"), nil
		}
		if info.PresupuestoCheckedAt != nil && !info.PresupuestoStale {
			if info.CuotaRestantePct != nil && *info.CuotaRestantePct > 0 {
				return false, strings.TrimSpace(strings.ToLower(info.EstadoCuota)), nil
			}
			if strings.EqualFold(strings.TrimSpace(info.PresupuestoEstado), "ok") {
				return false, strings.TrimSpace(strings.ToLower(info.EstadoCuota)), nil
			}
			if db.AgenteSinCuotaProveedorEfectivo(info) {
				return false, strings.TrimSpace(strings.ToLower(info.EstadoCuota)), nil
			}
		}
	}
	estado, err := db.GetPersistedAgentQuotaState(agente)
	if err != nil {
		return false, "", err
	}
	estado = strings.TrimSpace(strings.ToLower(estado))
	switch estado {
	case "enfriamiento", "agotado":
		return true, estado, nil
	default:
		return false, estado, nil
	}
}

func agenteOcupadoEnOtroProyectoPipeline(agente string, proyectoID int64) (bool, string, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" || proyectoID <= 0 {
		return false, "", nil
	}
	asignacion, err := db.GetAsignacionActivaAgente(agente)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, "", nil
		}
		return false, "", err
	}
	if asignacion == nil || asignacion.ProyectoID == proyectoID {
		return false, "", nil
	}
	return true, fmt.Sprintf("agente %s ya tiene asignacion activa en proyecto %s (%d)", agente, strings.TrimSpace(asignacion.ProyectoSlug), asignacion.ProyectoID), nil
}

func existeRuntimeHandleActivoPipeline(agente string, proyectoID *int64) (bool, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return false, nil
	}
	handle, err := runtimesService.GetOperationalRuntimeHandleAgentProject(agente, proyectoID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	if handle != nil && handle.ID > 0 {
		return true, nil
	}
	handle, err = runtimesService.GetActiveRuntimeHandleAgentProject(agente, proyectoID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return handle != nil && handle.ID > 0, nil
}

func agenteTieneRuntimeAmbiguoPipeline(agente string) (bool, string, error) {
	agente = strings.TrimSpace(agente)
	if agente == "" {
		return false, "", nil
	}
	handles, err := db.ListarRuntimeHandles(&agente)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, "", nil
		}
		return false, "", err
	}
	vivos := 0
	for _, handle := range handles {
		if handle == nil {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(handle.Estado)) {
		case "activo", "pausado":
			vivos++
		}
		if vivos > 1 {
			return true, fmt.Sprintf("agente %s tiene %d runtime handles vivos; requiere saneamiento antes de despachar", agente, vivos), nil
		}
	}
	return false, "", nil
}
