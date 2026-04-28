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
	ProyectoSlug     string                            `json:"proyecto_slug,omitempty"`
	Fase             string                            `json:"fase"`
	AccionTarea      string                            `json:"accion_tarea,omitempty"`
	PerfilTarea      string                            `json:"perfil_tarea,omitempty"`
	ModoEjecucion    string                            `json:"modo_ejecucion,omitempty"`
	Carril           string                            `json:"carril,omitempty"`
	EntregaCanonica  string                            `json:"entrega_canonica,omitempty"`
	RequiereWorktree bool                              `json:"requiere_worktree"`
	UsaMicroprograma bool                              `json:"usa_microprogramacion"`
	RequiereModelo   bool                              `json:"requiere_modelo"`
	ObjetivoModelo   string                            `json:"objetivo_modelo,omitempty"`
	ModeloFallback   string                            `json:"modelo_fallback,omitempty"`
	ResolucionActual *db.ResolucionModelo              `json:"resolucion_actual,omitempty"`
	TareaObjetivoID  int64                             `json:"tarea_objetivo_id,omitempty"`
	TareaObjetivo    string                            `json:"tarea_objetivo,omitempty"`
	AgenteTarea      string                            `json:"agente_tarea,omitempty"`
	WriteSet         []string                          `json:"write_set,omitempty"`
	SimbolosFoco     string                            `json:"simbolos_foco,omitempty"`
	TestsMinimos     string                            `json:"tests_minimos,omitempty"`
	FinishApp        bool                              `json:"finish_app,omitempty"`
	AgenteSugerido   string                            `json:"agente_sugerido,omitempty"`
	SeleccionAgente  *SeleccionAgentePipeline          `json:"seleccion_agente,omitempty"`
	RevisorObjetivo  *RevisorEscalonado                `json:"revisor_objetivo,omitempty"`
	Especificacion   *EspecificacionFuncion            `json:"especificacion,omitempty"`
	Paralelismo      *PoliticaParalelismoPipelineLocal `json:"paralelismo,omitempty"`
	Motivo           string                            `json:"motivo,omitempty"`
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
	seleccionAgente := s.resolverSeleccionAgentePipeline(paso, paso.EtapaObjetivo)
	agenteSugerido := ""
	if seleccionAgente != nil {
		agenteSugerido = strings.TrimSpace(seleccionAgente.Agente)
	}
	if tarea := paso.TareaObjetivo; tarea != nil && paso.AccionTarea != "" {
		actualizada, err := s.ejecutarAccionTareaPipelineLocal(tarea.ID, paso.AccionTarea, agenteSugerido)
		if err != nil {
			return nil, err
		}
		out.TareaActualizada = actualizada
	}
	out.Despacho = s.construirDespachoPipelineLocalConSeleccion(paso, out.TareaActualizada, seleccionAgente)
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
	return s.construirDespachoPipelineLocalConSeleccion(paso, tarea, nil)
}

func (s *Service) construirDespachoPipelineLocalConAgente(paso *PasoPipelineLocalDeterminista, tarea *TareaPipelineLocal, agenteSugerido string) *DespachoPipelineLocal {
	return s.construirDespachoPipelineLocalConSeleccion(paso, tarea, &SeleccionAgentePipeline{Agente: strings.TrimSpace(agenteSugerido)})
}

func (s *Service) construirDespachoPipelineLocalConSeleccion(paso *PasoPipelineLocalDeterminista, tarea *TareaPipelineLocal, seleccion *SeleccionAgentePipeline) *DespachoPipelineLocal {
	if paso == nil || paso.EtapaObjetivo == nil {
		return nil
	}
	etapa := paso.EtapaObjetivo
	if seleccion == nil || strings.TrimSpace(seleccion.Agente) == "" {
		seleccion = s.resolverSeleccionAgentePipeline(paso, etapa)
	}
	agenteSugerido := ""
	if seleccion != nil {
		agenteSugerido = strings.TrimSpace(seleccion.Agente)
	}
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
		AgenteSugerido:   strings.TrimSpace(agenteSugerido),
		SeleccionAgente:  seleccion,
		Paralelismo:      paso.Paralelismo,
	}
	if tarea == nil {
		tarea = paso.TareaObjetivo
	}
	if tarea != nil {
		out.TareaObjetivoID = tarea.ID
		out.TareaObjetivo = strings.TrimSpace(tarea.Titulo)
		out.AgenteTarea = strings.TrimSpace(tarea.Agente)
		out.FinishApp = tarea.FinishApp
		out.WriteSet = append([]string(nil), tarea.WriteSet...)
		out.SimbolosFoco = strings.TrimSpace(tarea.SimbolosFoco)
		out.TestsMinimos = strings.TrimSpace(tarea.TestsMinimos)
		out.Especificacion = construirEspecificacionFuncion(tarea, etapa)
	}
	if paso.RevisorObjetivo != nil {
		out.RevisorObjetivo = paso.RevisorObjetivo
	}
	return out
}

func (s *Service) resolverAgenteSugeridoPipeline(paso *PasoPipelineLocalDeterminista, etapa *EtapaPipelineLocal) string {
	seleccion := s.resolverSeleccionAgentePipeline(paso, etapa)
	if seleccion == nil {
		return agenteSugeridoParaCarril(strings.TrimSpace(etapa.Carril))
	}
	return strings.TrimSpace(seleccion.Agente)
}

func (s *Service) resolverSeleccionAgentePipeline(paso *PasoPipelineLocalDeterminista, etapa *EtapaPipelineLocal) *SeleccionAgentePipeline {
	if s != nil && s.agentResolver != nil && paso != nil && etapa != nil {
		entrada := EntradaResolverAgentePipeline{
			ProyectoSlug:     strings.TrimSpace(paso.ProyectoSlug),
			Fase:             strings.TrimSpace(etapa.Fase),
			AccionTarea:      strings.TrimSpace(paso.AccionTarea),
			PerfilTarea:      strings.TrimSpace(etapa.PerfilTarea),
			Carril:           strings.TrimSpace(etapa.Carril),
			ObjetivoModelo:   strings.TrimSpace(etapa.ObjetivoModelo),
			ModeloFallback:   strings.TrimSpace(etapa.ModeloFallback),
			RequiereWorktree: etapa.RequiereWorktree,
			UsaMicroprograma: etapa.UsaMicroprograma,
		}
		if explicado, ok := s.agentResolver.(ResolvedorAgentePipelineExplicado); ok {
			seleccion, err := explicado.ResolverSeleccionAgentePipeline(entrada)
			if err == nil && seleccion != nil && strings.TrimSpace(seleccion.Agente) != "" {
				return seleccion
			}
		}
		agente, err := s.agentResolver.ResolverAgentePipeline(entrada)
		if err == nil && strings.TrimSpace(agente) != "" {
			return &SeleccionAgentePipeline{Agente: strings.TrimSpace(agente)}
		}
	}
	return nil
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
	items, err := s.listarGatesPipelineLocal(proyectoSlug)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item == nil || item.TareaID == nil || *item.TareaID != despacho.TareaObjetivoID {
			continue
		}
		return nil
	}
	reviewerAgente := strings.TrimSpace(despacho.AgenteSugerido)
	severityMax := "medium"
	if despacho.RevisorObjetivo != nil {
		if rol := strings.TrimSpace(despacho.RevisorObjetivo.NombreRol); rol != "" {
			reviewerAgente = rol
		}
		if despacho.RevisorObjetivo.Premium {
			severityMax = "high"
		}
	}
	_, err = s.reviewGateManager.Create(reviewapp.CreateGateInput{
		ProyectoRef:     strings.TrimSpace(proyectoSlug),
		TareaID:         &despacho.TareaObjetivoID,
		RequestedBy:     "orquesta",
		ReviewerAgente:  reviewerAgente,
		SeverityMax:     severityMax,
		InitialFindings: resumenEspecificacionGate(despacho.Especificacion),
	})
	return err
}

// resumenEspecificacionGate construye el texto de hallazgos iniciales para la gate de revisión
// a partir de la especificación de función. Permite al revisor validar la entrega contra el contrato.
func resumenEspecificacionGate(spec *EspecificacionFuncion) string {
	if spec == nil {
		return ""
	}
	var partes []string
	if v := strings.TrimSpace(spec.Encabezado); v != "" {
		partes = append(partes, "encabezado: "+v)
	}
	if v := strings.TrimSpace(spec.SalidaEsperada); v != "" {
		partes = append(partes, "salida_esperada: "+v)
	}
	if len(spec.WriteSet) > 0 {
		partes = append(partes, "write_set: "+strings.Join(spec.WriteSet, ", "))
	}
	if v := strings.TrimSpace(spec.TestsMinimos); v != "" {
		partes = append(partes, "tests_minimos: "+v)
	}
	return strings.Join(partes, "\n")
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

func (s *Service) ejecutarAccionTareaPipelineLocal(tareaID int64, accion, agenteSugerido string) (*TareaPipelineLocal, error) {
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
	agente := strings.TrimSpace(agenteSugerido)
	if agente == "" && actual.Agente != nil && strings.TrimSpace(*actual.Agente) != "" {
		agente = strings.TrimSpace(*actual.Agente)
	}
	if agente == "" {
		agente = "orquesta"
	}
	cambioReal := false
	switch strings.ToLower(strings.TrimSpace(accion)) {
	case "especificar", "implementar", "corregir", "revisar":
		if tareaPipelineLocalEsMicrociclo(actual) && actual.Agente != nil && strings.TrimSpace(*actual.Agente) != "" && !strings.EqualFold(strings.TrimSpace(*actual.Agente), agente) {
			if err := s.taskActionProvider.ReassignTask(tareaID, agente); err != nil {
				return nil, err
			}
			cambioReal = true
		}
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
