package capacidadapp

import (
	"fmt"
	"strings"

	"orquesta/db"
	"orquesta/reviewapp"
)

type ResultadoEjecucionPasoPipelineLocal struct {
	Paso             *PasoPipelineLocalDeterminista `json:"paso,omitempty"`
	FaseActivada     *string                        `json:"fase_activada,omitempty"`
	TareaActualizada *TareaPipelineLocal            `json:"tarea_actualizada,omitempty"`
	Despacho         *DespachoPipelineLocal         `json:"despacho,omitempty"`
	DispatchRuntime  *ResultadoDespachoPipeline     `json:"dispatch_runtime,omitempty"`
}

type DespachoPipelineLocal struct {
	ProyectoSlug     string               `json:"proyecto_slug,omitempty"`
	Fase             string               `json:"fase"`
	AccionTarea      string               `json:"accion_tarea,omitempty"`
	PerfilTarea      string               `json:"perfil_tarea,omitempty"`
	ModoEjecucion    string               `json:"modo_ejecucion,omitempty"`
	Carril           string               `json:"carril,omitempty"`
	EntregaCanonica  string               `json:"entrega_canonica,omitempty"`
	RequiereWorktree bool                 `json:"requiere_worktree"`
	UsaMicroprograma bool                 `json:"usa_microprogramacion"`
	RequiereModelo   bool                 `json:"requiere_modelo"`
	ObjetivoModelo   string               `json:"objetivo_modelo,omitempty"`
	ModeloFallback   string               `json:"modelo_fallback,omitempty"`
	ResolucionActual *db.ResolucionModelo `json:"resolucion_actual,omitempty"`
	TareaObjetivoID  int64                `json:"tarea_objetivo_id,omitempty"`
	TareaObjetivo    string               `json:"tarea_objetivo,omitempty"`
	AgenteTarea      string               `json:"agente_tarea,omitempty"`
	WriteSet         []string             `json:"write_set,omitempty"`
	AgenteSugerido   string               `json:"agente_sugerido,omitempty"`
	Motivo           string               `json:"motivo,omitempty"`
}

func (s *Service) EjecutarSiguientePasoPipelineLocalDeterminista(proyectoSlug string) (*ResultadoEjecucionPasoPipelineLocal, error) {
	paso, err := s.CalcularSiguientePasoPipelineLocalDeterminista(proyectoSlug)
	if err != nil {
		return nil, err
	}
	out := &ResultadoEjecucionPasoPipelineLocal{Paso: paso}
	if paso == nil {
		return out, nil
	}
	if fase := strings.TrimSpace(paso.FaseObjetivo); fase != "" {
		if activada, err := s.activarFasePipelineLocal(strings.TrimSpace(proyectoSlug), fase); err != nil {
			return nil, err
		} else if activada != "" {
			out.FaseActivada = &activada
		}
	}
	if tarea := paso.TareaObjetivo; tarea != nil && paso.AccionTarea != "" {
		actualizada, err := s.ejecutarAccionTareaPipelineLocal(tarea.ID, paso.AccionTarea)
		if err != nil {
			return nil, err
		}
		out.TareaActualizada = actualizada
	}
	out.Despacho = s.construirDespachoPipelineLocal(paso, out.TareaActualizada)
	if err := s.asegurarReviewGatePipelineLocal(strings.TrimSpace(proyectoSlug), out.Despacho); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Service) EjecutarYDespacharSiguientePasoPipelineLocalDeterminista(proyectoSlug string) (*ResultadoEjecucionPasoPipelineLocal, error) {
	out, err := s.EjecutarSiguientePasoPipelineLocalDeterminista(proyectoSlug)
	if err != nil {
		return nil, err
	}
	if out == nil || out.Despacho == nil {
		return out, nil
	}
	resultado, err := s.DespacharPipelineLocal(proyectoSlug, out.Despacho)
	if err != nil {
		return nil, err
	}
	out.DispatchRuntime = resultado
	return out, nil
}

func (s *Service) construirDespachoPipelineLocal(paso *PasoPipelineLocalDeterminista, tarea *TareaPipelineLocal) *DespachoPipelineLocal {
	if paso == nil || paso.EtapaObjetivo == nil {
		return nil
	}
	etapa := paso.EtapaObjetivo
	out := &DespachoPipelineLocal{
		ProyectoSlug:     strings.TrimSpace(paso.ProyectoSlug),
		Fase:             strings.TrimSpace(etapa.Fase),
		AccionTarea:      strings.TrimSpace(paso.AccionTarea),
		PerfilTarea:      strings.TrimSpace(etapa.PerfilTarea),
		ModoEjecucion:    strings.TrimSpace(etapa.ModoEjecucion),
		Carril:           strings.TrimSpace(etapa.Carril),
		EntregaCanonica:  strings.TrimSpace(etapa.EntregaCanonica),
		RequiereWorktree: etapa.RequiereWorktree,
		UsaMicroprograma: etapa.UsaMicroprograma,
		RequiereModelo:   etapa.RequiereModelo,
		ObjetivoModelo:   strings.TrimSpace(etapa.ObjetivoModelo),
		ModeloFallback:   strings.TrimSpace(etapa.ModeloFallback),
		ResolucionActual: etapa.ResolucionActual,
		Motivo:           strings.TrimSpace(paso.Motivo),
		AgenteSugerido:   s.resolverAgenteSugeridoPipeline(paso, etapa),
	}
	if tarea == nil {
		tarea = paso.TareaObjetivo
	}
	if tarea != nil {
		out.TareaObjetivoID = tarea.ID
		out.TareaObjetivo = strings.TrimSpace(tarea.Titulo)
		out.AgenteTarea = strings.TrimSpace(tarea.Agente)
		out.WriteSet = append([]string(nil), tarea.WriteSet...)
	}
	return out
}

func (s *Service) resolverAgenteSugeridoPipeline(paso *PasoPipelineLocalDeterminista, etapa *EtapaPipelineLocal) string {
	if s != nil && s.agentResolver != nil && paso != nil && etapa != nil {
		agente, err := s.agentResolver.ResolverAgentePipeline(EntradaResolverAgentePipeline{
			ProyectoSlug:     strings.TrimSpace(paso.ProyectoSlug),
			Fase:             strings.TrimSpace(etapa.Fase),
			AccionTarea:      strings.TrimSpace(paso.AccionTarea),
			PerfilTarea:      strings.TrimSpace(etapa.PerfilTarea),
			Carril:           strings.TrimSpace(etapa.Carril),
			ObjetivoModelo:   strings.TrimSpace(etapa.ObjetivoModelo),
			ModeloFallback:   strings.TrimSpace(etapa.ModeloFallback),
			RequiereWorktree: etapa.RequiereWorktree,
			UsaMicroprograma: etapa.UsaMicroprograma,
		})
		if err == nil {
			return strings.TrimSpace(agente)
		}
	}
	return agenteSugeridoParaCarril(strings.TrimSpace(etapa.Carril))
}

func agenteSugeridoParaCarril(carril string) string {
	switch strings.ToLower(strings.TrimSpace(carril)) {
	case "microprogramacion_local":
		return "worker_local_mini"
	case "premium_worktree":
		return "worker_premium"
	case "revision_diff":
		return "worker_revisor"
	case "determinista_app":
		return "orquesta"
	default:
		return ""
	}
}

func (s *Service) asegurarReviewGatePipelineLocal(proyectoSlug string, despacho *DespachoPipelineLocal) error {
	if s == nil || s.reviewGateProvider == nil || s.reviewGateManager == nil || despacho == nil {
		return nil
	}
	if !strings.EqualFold(strings.TrimSpace(despacho.Fase), "revision") || despacho.TareaObjetivoID <= 0 {
		return nil
	}
	items, err := s.reviewGateProvider.List(reviewapp.ListInput{
		ProyectoRef: strings.TrimSpace(proyectoSlug),
		Limit:       20,
	})
	if err != nil {
		return err
	}
	for _, item := range items {
		if item == nil || item.TareaID == nil || *item.TareaID != despacho.TareaObjetivoID {
			continue
		}
		return nil
	}
	_, err = s.reviewGateManager.Create(reviewapp.CreateGateInput{
		ProyectoRef:    strings.TrimSpace(proyectoSlug),
		TareaID:        &despacho.TareaObjetivoID,
		RequestedBy:    "orquesta",
		ReviewerAgente: strings.TrimSpace(despacho.AgenteSugerido),
		SeverityMax:    "medium",
	})
	return err
}

func (s *Service) activarFasePipelineLocal(proyectoSlug, faseObjetivo string) (string, error) {
	if s == nil || s.phaseControlProvider == nil || strings.TrimSpace(proyectoSlug) == "" || strings.TrimSpace(faseObjetivo) == "" {
		return "", nil
	}
	fases, err := s.phaseControlProvider.ListProjectPhases(strings.TrimSpace(proyectoSlug))
	if err != nil {
		return "", err
	}
	var objetivo *db.FaseProyecto
	for _, fase := range fases {
		if fase == nil {
			continue
		}
		nombre := strings.ToLower(strings.TrimSpace(fase.Nombre))
		if nombre == strings.ToLower(strings.TrimSpace(faseObjetivo)) {
			objetivo = fase
			continue
		}
		if strings.EqualFold(strings.TrimSpace(fase.Estado), "activa") {
			copia := *fase
			copia.Estado = "pendiente"
			if err := s.phaseControlProvider.UpdateProjectPhase(&copia); err != nil {
				return "", err
			}
		}
	}
	if objetivo == nil {
		id, err := s.phaseControlProvider.RegisterProjectPhase(&db.FaseProyecto{
			Proyecto: strings.TrimSpace(proyectoSlug),
			Nombre:   strings.TrimSpace(faseObjetivo),
			Orden:    100,
			Peso:     1,
			Estado:   "activa",
		})
		if err != nil {
			return "", err
		}
		objetivo, err = s.phaseControlProvider.GetProjectPhase(id)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(objetivo.Nombre), nil
	}
	if !strings.EqualFold(strings.TrimSpace(objetivo.Estado), "activa") {
		copia := *objetivo
		copia.Estado = "activa"
		if err := s.phaseControlProvider.UpdateProjectPhase(&copia); err != nil {
			return "", err
		}
		return strings.TrimSpace(objetivo.Nombre), nil
	}
	return "", nil
}

func (s *Service) ejecutarAccionTareaPipelineLocal(tareaID int64, accion string) (*TareaPipelineLocal, error) {
	if s == nil || s.taskActionProvider == nil || tareaID <= 0 {
		return nil, nil
	}
	actual, err := s.taskActionProvider.GetTask(tareaID)
	if err != nil {
		return nil, err
	}
	if actual == nil {
		return nil, fmt.Errorf("tarea no encontrada: %d", tareaID)
	}
	agente := "orquesta"
	cambioReal := false
	switch strings.ToLower(strings.TrimSpace(accion)) {
	case "especificar", "implementar", "corregir", "revisar":
		switch actual.Estado {
		case db.TareaLibre, db.TareaBacklog:
			if err := s.taskActionProvider.TakeTask(tareaID, agente); err != nil {
				return nil, err
			}
			cambioReal = true
		}
		actual, err = s.taskActionProvider.GetTask(tareaID)
		if err != nil {
			return nil, err
		}
		if actual != nil && actual.Estado != db.TareaEnProgreso {
			nombreAgente := agente
			if actual.Agente != nil && strings.TrimSpace(*actual.Agente) != "" {
				nombreAgente = strings.TrimSpace(*actual.Agente)
			}
			if err := s.taskActionProvider.StartTask(tareaID, nombreAgente); err != nil {
				return nil, err
			}
			cambioReal = true
		}
	}
	actual, err = s.taskActionProvider.GetTask(tareaID)
	if err != nil {
		return nil, err
	}
	if !cambioReal {
		return nil, nil
	}
	return tareaPipelineLocalDesdeDB(actual), nil
}
