package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"orquesta/capacidadapp"
	"orquesta/db"
	"orquesta/gitaplicacion"
	"orquesta/microprogramacionapp"
	"orquesta/runtimeagente"
	"orquesta/runtimesapp"
)

type despachadorPipelineOperativo struct{}

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
		if activo, err := existeRuntimeHandleActivoPipeline(agente, &proyecto.ID); err != nil {
			return nil, err
		} else if activo {
			return &capacidadapp.ResultadoDespachoPipeline{
				Estado: "mailbox_encolado_runtime_existente",
				Motivo: "pipeline_local encolado; ya existe un runtime activo para el agente y proyecto",
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
			Estado:       "encolado",
			Motivo:       "mailbox y start encolados por la vía canónica",
			StartOrderID: &startID,
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
