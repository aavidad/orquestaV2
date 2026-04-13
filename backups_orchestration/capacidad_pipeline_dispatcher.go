package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"orquesta/capacidadapp"
	"orquesta/db"
	"orquesta/gitaplicacion"
	"orquesta/microprogramacionapp"
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
		payload := map[string]any{
			"from_agente": "server",
			"to_agente":   agente,
			"kind":        "pipeline_local",
			"accion":      firstNonEmpty(strings.TrimSpace(despacho.AccionTarea), "continuar_trabajo"),
			"texto":       strings.TrimSpace(despacho.Motivo),
			"instruction": construirInstructionPipeline(*despacho),
			"source":      "pipeline_local",
			"fase":        strings.TrimSpace(despacho.Fase),
			"perfil_tarea": strings.TrimSpace(despacho.PerfilTarea),
			"carril":           strings.TrimSpace(despacho.Carril),
			"entrega_canonica": strings.TrimSpace(despacho.EntregaCanonica),
			"tarea_objetivo_id": despacho.TareaObjetivoID,
			"tarea_objetivo":    strings.TrimSpace(despacho.TareaObjetivo),
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
		startID, _, err := runtimesService.EnqueueAgentControl(runtimesapp.AgentControlRequest{
			Agente:       agente,
			Proyecto:     strings.TrimSpace(entrada.ProyectoSlug),
			Accion:       "start",
			Modelo:       strings.TrimSpace(despacho.ObjetivoModelo),
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
		specs, err := microprogramacionService.List(microprogramacionapp.FiltroEspecificaciones{
			TareaID: &despacho.TareaObjetivoID,
		})
		if err != nil {
			return nil, err
		}
		var spec *microprogramacionapp.EspecificacionFuncion
		for _, s := range specs {
			if s != nil && strings.EqualFold(strings.TrimSpace(s.Estado), microprogramacionapp.EstadoEspecificacionActiva) {
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
		startID, _, err := runtimesService.EnqueueAgentControl(runtimesapp.AgentControlRequest{
			Agente:       agente,
			Proyecto:     strings.TrimSpace(entrada.ProyectoSlug),
			Accion:       "start",
			Modelo:       strings.TrimSpace(despacho.ObjetivoModelo),
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
