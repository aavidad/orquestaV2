package capacidadapp

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"orquesta/db"
	"orquesta/reviewapp"
)

type ReviewGateProvider interface {
	List(input reviewapp.ListInput) ([]*reviewapp.Gate, error)
}

type PasoPipelineLocalDeterminista struct {
	ProyectoSlug    string                            `json:"proyecto_slug,omitempty"`
	Accion          string                            `json:"accion"`
	AccionTarea     string                            `json:"accion_tarea,omitempty"`
	Motivo          string                            `json:"motivo"`
	Paralelismo     *PoliticaParalelismoPipelineLocal `json:"paralelismo,omitempty"`
	FaseActual      string                            `json:"fase_actual,omitempty"`
	FaseObjetivo    string                            `json:"fase_objetivo,omitempty"`
	EtapaObjetivo   *EtapaPipelineLocal               `json:"etapa_objetivo,omitempty"`
	GateBloqueante  *reviewapp.Gate                   `json:"gate_bloqueante,omitempty"`
	TareaObjetivo   *TareaPipelineLocal               `json:"tarea_objetivo,omitempty"`
	RevisorObjetivo *RevisorEscalonado                `json:"revisor_objetivo,omitempty"`
	Especificacion  *EspecificacionFuncion            `json:"especificacion,omitempty"`
	Pipeline        *PipelineLocalDeterminista        `json:"pipeline,omitempty"`
}

type TareaPipelineLocal struct {
	ID           int64    `json:"id"`
	Titulo       string   `json:"titulo"`
	Descripcion  string   `json:"descripcion,omitempty"`
	Notas        string   `json:"notas,omitempty"`
	FinishApp    bool     `json:"finish_app,omitempty"`
	WriteSet     []string `json:"write_set,omitempty"`
	SimbolosFoco string   `json:"simbolos_foco,omitempty"`
	TestsMinimos string   `json:"tests_minimos,omitempty"`
	Estado       string   `json:"estado"`
	Agente       string   `json:"agente,omitempty"`
	Modulo       string   `json:"modulo,omitempty"`
	Prioridad    string   `json:"prioridad,omitempty"`
}

type PoliticaParalelismoPipelineLocal struct {
	PuedeAbrirSubagentes bool   `json:"puede_abrir_subagentes"`
	MaxSubagentes        int    `json:"max_subagentes,omitempty"`
	Motivo               string `json:"motivo,omitempty"`
}

type pipelineLocalPlanRuntime struct {
	service      *Service
	proyectoSlug string
	tasks        []*db.Tarea
	tasksLoaded  bool
	tasksErr     error
	gates        []*reviewapp.Gate
	gatesLoaded  bool
	gatesErr     error
}

func (s *Service) SetReviewGateProvider(provider ReviewGateProvider) {
	s.reviewGateProvider = provider
}

func (s *Service) CalcularSiguientePasoPipelineLocalDeterminista(proyectoSlug string) (*PasoPipelineLocalDeterminista, error) {
	runtime := &pipelineLocalPlanRuntime{
		service:      s,
		proyectoSlug: strings.TrimSpace(proyectoSlug),
	}
	pipeline, err := s.ConstruirPipelineLocalDeterminista(proyectoSlug)
	if err != nil {
		return nil, err
	}
	faseActual := ""
	if s.phaseProvider != nil {
		faseActual, err = s.phaseProvider.GetActivePhase(strings.TrimSpace(proyectoSlug))
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) && !strings.Contains(strings.ToLower(err.Error()), "no rows") {
				return nil, err
			}
		}
		faseActual = strings.TrimSpace(faseActual)
	}

	if gate, motivo, ok, err := runtime.resolverGateBloqueante(); err != nil {
		return nil, err
	} else if ok {
		etapa := buscarEtapaPipelineLocal(pipeline, "correccion")
		tareaCorreccion := runtime.seleccionarTarea("correccion")
		return &PasoPipelineLocalDeterminista{
			ProyectoSlug:   strings.TrimSpace(proyectoSlug),
			Accion:         "abrir_correccion",
			AccionTarea:    "corregir",
			Motivo:         motivo,
			Paralelismo:    resolverPoliticaParalelismoPipelineLocal(tareaCorreccion, etapa),
			FaseActual:     faseActual,
			FaseObjetivo:   "correccion",
			EtapaObjetivo:  etapa,
			GateBloqueante: gate,
			TareaObjetivo:  tareaCorreccion,
			Especificacion: especificacionParaPaso(tareaCorreccion, etapa),
			Pipeline:       pipeline,
		}, nil
	}

	if gate, motivo, ok, err := runtime.resolverGateRevisionPendiente(); err != nil {
		return nil, err
	} else if ok {
		etapaRevision := buscarEtapaPipelineLocal(pipeline, "revision")
		tareaRevision := runtime.seleccionarTarea("revision")
		return &PasoPipelineLocalDeterminista{
			ProyectoSlug:    strings.TrimSpace(proyectoSlug),
			Accion:          "esperar_revision",
			AccionTarea:     "revisar",
			Motivo:          motivo,
			Paralelismo:     resolverPoliticaParalelismoPipelineLocal(tareaRevision, etapaRevision),
			FaseActual:      faseActual,
			FaseObjetivo:    "revision",
			EtapaObjetivo:   etapaRevision,
			GateBloqueante:  gate,
			TareaObjetivo:   tareaRevision,
			RevisorObjetivo: seleccionarRevisorEscalonadoPipeline(pipeline.Revision, extraerSeñalesTareaPipelineLocal(tareaRevision)),
			Especificacion:  especificacionParaPaso(tareaRevision, etapaRevision),
			Pipeline:        pipeline,
		}, nil
	}

	if faseActual == "" {
		etapa := primeraEtapaPipelineLocal(pipeline)
		if etapa == nil {
			return nil, fmt.Errorf("pipeline local sin fases")
		}
		tareaInicial := runtime.seleccionarTarea(etapa.Fase)
		return &PasoPipelineLocalDeterminista{
			ProyectoSlug:   strings.TrimSpace(proyectoSlug),
			Accion:         "arrancar_fase",
			AccionTarea:    accionTareaParaFase(etapa.Fase),
			Motivo:         "no hay fase activa registrada; se arranca la primera fase canónica",
			Paralelismo:    resolverPoliticaParalelismoPipelineLocal(tareaInicial, etapa),
			FaseObjetivo:   etapa.Fase,
			EtapaObjetivo:  etapa,
			TareaObjetivo:  tareaInicial,
			Especificacion: especificacionParaPaso(tareaInicial, etapa),
			Pipeline:       pipeline,
		}, nil
	}

	etapa := buscarEtapaPipelineLocal(pipeline, faseActual)
	if etapa == nil {
		etapa = primeraEtapaPipelineLocal(pipeline)
		tareaReconciliar := runtime.seleccionarTarea(etapa.Fase)
		return &PasoPipelineLocalDeterminista{
			ProyectoSlug:   strings.TrimSpace(proyectoSlug),
			Accion:         "reconciliar_fase",
			AccionTarea:    accionTareaParaFase(etapa.Fase),
			Motivo:         "la fase activa no coincide con el pipeline canónico; se propone reconciliación al inicio",
			Paralelismo:    resolverPoliticaParalelismoPipelineLocal(tareaReconciliar, etapa),
			FaseActual:     faseActual,
			FaseObjetivo:   etapa.Fase,
			EtapaObjetivo:  etapa,
			TareaObjetivo:  tareaReconciliar,
			Especificacion: especificacionParaPaso(tareaReconciliar, etapa),
			Pipeline:       pipeline,
		}, nil
	}
	tareaActual := runtime.seleccionarTarea(etapa.Fase)
	tareaAbiertaImplementacion := runtime.seleccionarTarea("implementacion")
	if (strings.EqualFold(etapa.Fase, "revision") || strings.EqualFold(etapa.Fase, "integracion")) && tareaPipelineLocalAbierta(tareaAbiertaImplementacion) {
		etapaImplementacion := buscarEtapaPipelineLocal(pipeline, "implementacion")
		if etapaImplementacion != nil {
			return &PasoPipelineLocalDeterminista{
				ProyectoSlug:   strings.TrimSpace(proyectoSlug),
				Accion:         "reconciliar_fase",
				AccionTarea:    accionTareaParaFase("implementacion"),
				Motivo:         "hay trabajo abierto pendiente; el pipeline debe volver a implementación antes de seguir revisión o integración histórica",
				Paralelismo:    resolverPoliticaParalelismoPipelineLocal(tareaAbiertaImplementacion, etapaImplementacion),
				FaseActual:     faseActual,
				FaseObjetivo:   "implementacion",
				EtapaObjetivo:  etapaImplementacion,
				TareaObjetivo:  tareaAbiertaImplementacion,
				Especificacion: especificacionParaPaso(tareaAbiertaImplementacion, etapaImplementacion),
				Pipeline:       pipeline,
			}, nil
		}
	}
	if strings.EqualFold(etapa.Fase, "especificacion") {
		if tareaActual != nil && strings.EqualFold(strings.TrimSpace(tareaActual.Estado), string(db.TareaCompletada)) {
			etapaImplementacion := buscarEtapaPipelineLocal(pipeline, "implementacion")
			tareaImplementacion := runtime.seleccionarTarea("implementacion")
			return &PasoPipelineLocalDeterminista{
				ProyectoSlug:   strings.TrimSpace(proyectoSlug),
				Accion:         "avanzar_fase",
				AccionTarea:    accionTareaParaFase("implementacion"),
				Motivo:         "la especificación está completada; el pipeline debe avanzar a implementación",
				Paralelismo:    resolverPoliticaParalelismoPipelineLocal(tareaImplementacion, etapaImplementacion),
				FaseActual:     faseActual,
				FaseObjetivo:   "implementacion",
				EtapaObjetivo:  etapaImplementacion,
				TareaObjetivo:  tareaImplementacion,
				Especificacion: especificacionParaPaso(tareaImplementacion, etapaImplementacion),
				Pipeline:       pipeline,
			}, nil
		}
	}
	if strings.EqualFold(etapa.Fase, "implementacion") || strings.EqualFold(etapa.Fase, "correccion") {
		if tareaActual != nil && strings.EqualFold(strings.TrimSpace(tareaActual.Estado), string(db.TareaCompletada)) {
			etapaRevision := buscarEtapaPipelineLocal(pipeline, "revision")
			tareaRevision := runtime.seleccionarTarea("revision")
			return &PasoPipelineLocalDeterminista{
				ProyectoSlug:    strings.TrimSpace(proyectoSlug),
				Accion:          "avanzar_fase",
				AccionTarea:     accionTareaParaFase("revision"),
				Motivo:          "la tarea activa ya está completada; el pipeline debe avanzar a revisión",
				Paralelismo:     resolverPoliticaParalelismoPipelineLocal(tareaRevision, etapaRevision),
				FaseActual:      faseActual,
				FaseObjetivo:    "revision",
				EtapaObjetivo:   etapaRevision,
				TareaObjetivo:   tareaRevision,
				RevisorObjetivo: seleccionarRevisorEscalonadoPipeline(pipeline.Revision, extraerSeñalesTareaPipelineLocal(tareaRevision)),
				Especificacion:  especificacionParaPaso(tareaRevision, etapaRevision),
				Pipeline:        pipeline,
			}, nil
		}
	}
	if strings.EqualFold(etapa.Fase, "revision") {
		if tareaActual != nil && strings.EqualFold(strings.TrimSpace(tareaActual.Estado), string(db.TareaCompletada)) {
			if gate, ok, err := runtime.resolverGateAprobado(tareaActual.ID); err != nil {
				return nil, err
			} else if ok {
				etapaIntegracion := buscarEtapaPipelineLocal(pipeline, "integracion")
				return &PasoPipelineLocalDeterminista{
					ProyectoSlug:   strings.TrimSpace(proyectoSlug),
					Accion:         "avanzar_fase",
					AccionTarea:    accionTareaParaFase("integracion"),
					Motivo:         "la review gate de la tarea ya está aprobada; el pipeline debe avanzar a integración",
					Paralelismo:    resolverPoliticaParalelismoPipelineLocal(runtime.seleccionarTarea("integracion"), etapaIntegracion),
					FaseActual:     faseActual,
					FaseObjetivo:   "integracion",
					EtapaObjetivo:  etapaIntegracion,
					GateBloqueante: gate,
					TareaObjetivo:  runtime.seleccionarTarea("integracion"),
					Pipeline:       pipeline,
				}, nil
			}
		}
	}
	accion := "ejecutar_fase"
	motivo := "la fase activa coincide con el pipeline canónico y no hay gates abiertos que cambien el flujo"
	if strings.EqualFold(etapa.Fase, "integracion") {
		accion = "integrar"
		motivo = "la fase de integración es determinista y debe resolverla la app sin delegar en modelos"
	}
	var revisorObjetivo *RevisorEscalonado
	if strings.EqualFold(etapa.Fase, "revision") {
		revisorObjetivo = seleccionarRevisorEscalonadoPipeline(pipeline.Revision, extraerSeñalesTareaPipelineLocal(tareaActual))
	}
	return &PasoPipelineLocalDeterminista{
		ProyectoSlug:    strings.TrimSpace(proyectoSlug),
		Accion:          accion,
		AccionTarea:     accionTareaParaFase(etapa.Fase),
		Motivo:          motivo,
		Paralelismo:     resolverPoliticaParalelismoPipelineLocal(tareaActual, etapa),
		FaseActual:      faseActual,
		FaseObjetivo:    etapa.Fase,
		EtapaObjetivo:   etapa,
		TareaObjetivo:   tareaActual,
		RevisorObjetivo: revisorObjetivo,
		Especificacion:  especificacionParaPaso(tareaActual, etapa),
		Pipeline:        pipeline,
	}, nil
}

func resolverPoliticaParalelismoPipelineLocal(tarea *TareaPipelineLocal, etapa *EtapaPipelineLocal) *PoliticaParalelismoPipelineLocal {
	if etapa == nil {
		return &PoliticaParalelismoPipelineLocal{
			PuedeAbrirSubagentes: false,
			Motivo:               "sin etapa objetivo; no conviene abrir subagentes",
		}
	}
	fase := strings.ToLower(strings.TrimSpace(etapa.Fase))
	if strings.EqualFold(strings.TrimSpace(etapa.ModoEjecucion), "determinista_app") || strings.EqualFold(fase, "integracion") {
		return &PoliticaParalelismoPipelineLocal{
			PuedeAbrirSubagentes: false,
			Motivo:               "fase determinista o de integración; la app debe resolverla sin abrir carriles paralelos",
		}
	}
	if strings.EqualFold(fase, "revision") || strings.EqualFold(fase, "correccion") {
		return &PoliticaParalelismoPipelineLocal{
			PuedeAbrirSubagentes: false,
			Motivo:               "fase de revisión/corrección; conviene evitar abrir más agentes sobre el mismo frente",
		}
	}
	if tarea == nil {
		return &PoliticaParalelismoPipelineLocal{
			PuedeAbrirSubagentes: false,
			Motivo:               "sin tarea concreta; no hay base suficiente para paralelizar",
		}
	}
	señales := extraerSeñalesTareaPipelineLocal(tarea)
	for _, señal := range señales {
		switch strings.ToLower(strings.TrimSpace(señal)) {
		case "zona_critica_control_plane", "seguridad_o_concurrencia", "riesgo_alto", "hallazgos_contradictorios", "correcciones_repetidas":
			return &PoliticaParalelismoPipelineLocal{
				PuedeAbrirSubagentes: false,
				Motivo:               "frente crítico o inestable; evita abrir subagentes concurrentes sobre este trabajo",
			}
		}
	}
	ctx := strings.ToLower(strings.TrimSpace(strings.Join([]string{tarea.Descripcion, tarea.Notas}, "\n")))
	if strings.Contains(ctx, "write-set exclusivo") || strings.Contains(ctx, "write_set exclusivo") || strings.Contains(ctx, "trabaja solo dentro de ese write_set") {
		return &PoliticaParalelismoPipelineLocal{
			PuedeAbrirSubagentes: false,
			Motivo:               "el contrato del frente declara write_set exclusivo; no conviene abrir más agentes de escritura",
		}
	}
	if len(tarea.WriteSet) >= 4 && strings.TrimSpace(tarea.TestsMinimos) != "" && strings.EqualFold(fase, "implementacion") {
		return &PoliticaParalelismoPipelineLocal{
			PuedeAbrirSubagentes: true,
			MaxSubagentes:        2,
			Motivo:               "frente amplio de implementación con contrato explícito; se puede dividir en slices disjuntos",
		}
	}
	return &PoliticaParalelismoPipelineLocal{
		PuedeAbrirSubagentes: false,
		Motivo:               "frente compacto o sin slices claros; el coste de coordinación supera la ganancia",
	}
}

func (r *pipelineLocalPlanRuntime) loadTasks() ([]*db.Tarea, error) {
	if r == nil || r.service == nil || r.service.taskProvider == nil || strings.TrimSpace(r.proyectoSlug) == "" {
		return nil, nil
	}
	if r.tasksLoaded {
		return r.tasks, r.tasksErr
	}
	r.tasksLoaded = true
	r.tasks, r.tasksErr = r.service.taskProvider.ListProjectTasks(strings.TrimSpace(r.proyectoSlug))
	return r.tasks, r.tasksErr
}

func (r *pipelineLocalPlanRuntime) loadGates() ([]*reviewapp.Gate, error) {
	if r == nil || r.service == nil || r.service.reviewGateProvider == nil || strings.TrimSpace(r.proyectoSlug) == "" {
		return nil, nil
	}
	if r.gatesLoaded {
		return r.gates, r.gatesErr
	}
	r.gatesLoaded = true
	r.gates, r.gatesErr = r.service.listarGatesPipelineLocal(strings.TrimSpace(r.proyectoSlug))
	return r.gates, r.gatesErr
}

func (r *pipelineLocalPlanRuntime) seleccionarTarea(fase string) *TareaPipelineLocal {
	tareas, err := r.loadTasks()
	if err != nil || len(tareas) == 0 {
		return nil
	}
	var candidata *db.Tarea
	mejor := -1
	for _, tarea := range tareas {
		candidata, mejor = elegirNuevaCandidataTareaPipelineLocal(candidata, tarea, fase, mejor)
	}
	if candidata == nil || mejor < 0 {
		return nil
	}
	return tareaPipelineLocalDesdeDB(candidata)
}

func (r *pipelineLocalPlanRuntime) resolverGateBloqueante() (*reviewapp.Gate, string, bool, error) {
	items, err := r.loadGates()
	if err != nil {
		return nil, "", false, err
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		if reviewapp.IsBlockingGateState(item.Estado) {
			return item, "hay una review gate abierta con cambios o bloqueo; la siguiente acción debe ser corrección", true, nil
		}
	}
	return nil, "", false, nil
}

func (r *pipelineLocalPlanRuntime) resolverGateRevisionPendiente() (*reviewapp.Gate, string, bool, error) {
	items, err := r.loadGates()
	if err != nil {
		return nil, "", false, err
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(item.Estado), reviewapp.GateStatePending) || strings.EqualFold(strings.TrimSpace(item.Estado), reviewapp.GateStateInReview) {
			return item, "hay una review gate pendiente o en revisión; el pipeline debe esperar o continuar revisión", true, nil
		}
	}
	return nil, "", false, nil
}

func (r *pipelineLocalPlanRuntime) resolverGateAprobado(tareaID int64) (*reviewapp.Gate, bool, error) {
	if tareaID <= 0 {
		return nil, false, nil
	}
	items, err := r.loadGates()
	if err != nil {
		return nil, false, err
	}
	for _, item := range items {
		if item == nil || item.TareaID == nil || *item.TareaID != tareaID {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(item.Estado), reviewapp.GateStateApproved) {
			return item, true, nil
		}
	}
	return nil, false, nil
}

// especificacionParaPaso construye la EspecificacionFuncion para incluirla en el paso del pipeline.
// Solo se genera cuando la etapa objetivo requiere un modelo (carril real, no determinista).
func especificacionParaPaso(tarea *TareaPipelineLocal, etapa *EtapaPipelineLocal) *EspecificacionFuncion {
	return construirEspecificacionFuncion(tarea, etapa)
}

func accionTareaParaFase(fase string) string {
	switch strings.ToLower(strings.TrimSpace(fase)) {
	case "especificacion":
		return "especificar"
	case "implementacion":
		return "implementar"
	case "revision":
		return "revisar"
	case "correccion":
		return "corregir"
	case "integracion":
		return "integrar"
	default:
		return ""
	}
}

func tareaPipelineLocalAbierta(tarea *TareaPipelineLocal) bool {
	if tarea == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(tarea.Estado)) {
	case string(db.TareaEnProgreso), string(db.TareaAsignada), string(db.TareaLibre), string(db.TareaBacklog):
		return true
	default:
		return false
	}
}

func tareaPipelineLocalEsMicrociclo(tarea *db.Tarea) bool {
	if tarea == nil {
		return false
	}
	return strings.Contains(strings.TrimSpace(tarea.Notas), "autonomia:microrefactor_loop")
}

func tareaPipelineLocalEsFinishAppDB(tarea *db.Tarea) bool {
	if tarea == nil {
		return false
	}
	return strings.Contains(strings.TrimSpace(tarea.Notas), "autonomia:finish_app")
}

func tareaPipelineLocalMicrocicloPrioritario(tarea *db.Tarea) bool {
	if !tareaPipelineLocalEsMicrociclo(tarea) {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(string(tarea.Estado))) {
	case string(db.TareaEnProgreso), string(db.TareaBloqueada):
		return true
	default:
		return false
	}
}

func tareaPipelineLocalEsFrentePremiumSemilla(tarea *db.Tarea) bool {
	if tarea == nil {
		return false
	}
	if strings.Contains(strings.TrimSpace(tarea.Notas), "autonomia:premium_frontier") {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(tarea.Titulo), "Autonomía premium: abrir siguiente frente mayor útil")
}

func tareaPipelineLocalTieneContratoPremiumReal(tarea *db.Tarea) bool {
	if tarea == nil {
		return false
	}
	writeSet := extraerWriteSetTareaPipelineLocal(tarea.Descripcion, tarea.Notas)
	if len(writeSet) == 0 {
		return false
	}
	return strings.TrimSpace(extraerTestsMinimosTareaPipelineLocal(tarea.Descripcion, tarea.Notas)) != ""
}

func (s *Service) seleccionarTareaPipelineLocal(proyectoSlug, fase string) *TareaPipelineLocal {
	if s == nil || s.taskProvider == nil || strings.TrimSpace(proyectoSlug) == "" {
		return nil
	}
	tareas, err := s.taskProvider.ListProjectTasks(strings.TrimSpace(proyectoSlug))
	if err != nil || len(tareas) == 0 {
		return nil
	}
	var candidata *db.Tarea
	mejor := -1
	for _, tarea := range tareas {
		candidata, mejor = elegirNuevaCandidataTareaPipelineLocal(candidata, tarea, fase, mejor)
	}
	if candidata == nil || mejor < 0 {
		return nil
	}
	return tareaPipelineLocalDesdeDB(candidata)
}

// elegirNuevaCandidataTareaPipelineLocal aplica las reglas de prioridad entre la candidata
// actual y una nueva tarea, devolviendo cuál debe ser la nueva candidata y su puntuación.
// Reglas (en orden de prioridad):
//  1. Semilla premium + nueva con contrato real → nueva gana
//  2. Nueva semilla + candidata con contrato real → candidata se mantiene
//  3. Nueva con microciclo activo + candidata sin él → nueva gana
//  4. Candidata con microciclo activo + nueva sin él → candidata se mantiene
//  5. Puntuación por fase (mayor score gana; empate se rompe por ID más alto)
func elegirNuevaCandidataTareaPipelineLocal(candidata, nueva *db.Tarea, fase string, mejorScore int) (*db.Tarea, int) {
	if candidata != nil {
		actualFinish := tareaPipelineLocalEsFinishAppDB(candidata)
		nuevaFinish := tareaPipelineLocalEsFinishAppDB(nueva)
		if nuevaFinish != actualFinish {
			if nuevaFinish {
				return nueva, puntuarTareaPipelineLocal(nueva, fase)
			}
			return candidata, mejorScore
		}
		candidataSemilla := tareaPipelineLocalEsFrentePremiumSemilla(candidata)
		nuevaSemilla := tareaPipelineLocalEsFrentePremiumSemilla(nueva)
		candidataContratoReal := tareaPipelineLocalTieneContratoPremiumReal(candidata)
		nuevaContratoReal := tareaPipelineLocalTieneContratoPremiumReal(nueva)
		if candidataSemilla != nuevaSemilla && candidataContratoReal != nuevaContratoReal {
			if candidataSemilla && nuevaContratoReal {
				return nueva, puntuarTareaPipelineLocal(nueva, fase)
			}
			if nuevaSemilla && candidataContratoReal {
				return candidata, mejorScore
			}
		}
		actualMicrociclo := tareaPipelineLocalMicrocicloPrioritario(candidata)
		nuevaMicrociclo := tareaPipelineLocalMicrocicloPrioritario(nueva)
		if nuevaMicrociclo != actualMicrociclo {
			if nuevaMicrociclo {
				return nueva, puntuarTareaPipelineLocal(nueva, fase)
			}
			return candidata, mejorScore
		}
	}
	score := puntuarTareaPipelineLocal(nueva, fase)
	if score > mejorScore || (score == mejorScore && candidata != nil && nueva.ID > candidata.ID) {
		return nueva, score
	}
	return candidata, mejorScore
}

func puntuarTareaPipelineLocal(tarea *db.Tarea, fase string) int {
	if tarea == nil {
		return -1
	}
	bonus := 0
	if tareaPipelineLocalEsFinishAppDB(tarea) {
		bonus += 200
	}
	estado := strings.ToLower(strings.TrimSpace(string(tarea.Estado)))
	switch strings.ToLower(strings.TrimSpace(fase)) {
	case "especificacion", "implementacion", "correccion":
		switch estado {
		case string(db.TareaEnProgreso):
			return 50 + bonus
		case string(db.TareaLibre):
			return 40 + bonus
		case string(db.TareaBacklog):
			return 30 + bonus
		case string(db.TareaAsignada):
			return 20 + bonus
		case string(db.TareaCompletada):
			return 10 + bonus
		default:
			return -1
		}
	case "revision":
		switch estado {
		case string(db.TareaCompletada):
			return 30 + bonus
		case string(db.TareaEnProgreso):
			return 20 + bonus
		case string(db.TareaAsignada):
			return 10 + bonus
		default:
			return -1
		}
	case "integracion":
		if estado == string(db.TareaCompletada) {
			return 10 + bonus
		}
		return -1
	default:
		return -1
	}
}

func tareaPipelineLocalDesdeDB(tarea *db.Tarea) *TareaPipelineLocal {
	if tarea == nil {
		return nil
	}
	out := &TareaPipelineLocal{
		ID:           tarea.ID,
		Titulo:       strings.TrimSpace(tarea.Titulo),
		Descripcion:  strings.TrimSpace(tarea.Descripcion),
		Notas:        strings.TrimSpace(tarea.Notas),
		FinishApp:    tareaPipelineLocalEsFinishAppDB(tarea),
		WriteSet:     extraerWriteSetTareaPipelineLocal(tarea.Descripcion, tarea.Notas),
		SimbolosFoco: extraerSimbolosFocoTareaPipelineLocal(tarea.Descripcion, tarea.Notas),
		TestsMinimos: extraerTestsMinimosTareaPipelineLocal(tarea.Descripcion, tarea.Notas),
		Estado:       strings.TrimSpace(string(tarea.Estado)),
		Modulo:       strings.TrimSpace(tarea.Modulo),
		Prioridad:    strings.TrimSpace(string(tarea.Prioridad)),
	}
	if tarea.Agente != nil {
		out.Agente = strings.TrimSpace(*tarea.Agente)
	}
	return out
}

func extraerWriteSetTareaPipelineLocal(descripcion, notas string) []string {
	for _, raw := range []string{descripcion, notas} {
		writeSet := ExtraerWriteSetTextoPipelineLocal(raw)
		if len(writeSet) > 0 {
			return writeSet
		}
	}
	return nil
}

func extraerSimbolosFocoTareaPipelineLocal(descripcion, notas string) string {
	for _, raw := range []string{descripcion, notas} {
		if simbolos := ExtraerSimbolosFocoTextoPipelineLocal(raw); simbolos != "" {
			return simbolos
		}
	}
	return ""
}

func extraerTestsMinimosTareaPipelineLocal(descripcion, notas string) string {
	for _, raw := range []string{descripcion, notas} {
		if tests := ExtraerTestsMinimosTextoPipelineLocal(raw); tests != "" {
			return tests
		}
	}
	return ""
}

func ExtraerWriteSetTextoPipelineLocal(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	lineas := strings.FieldsFunc(raw, func(r rune) bool { return r == '\n' || r == '\r' })
	for _, linea := range lineas {
		linea = strings.TrimSpace(linea)
		if linea == "" {
			continue
		}
		lower := strings.ToLower(linea)
		idx := strings.Index(lower, "write-set")
		if idx < 0 {
			idx = strings.Index(lower, "write_set")
		}
		if idx < 0 {
			continue
		}
		if colon := strings.Index(linea[idx:], ":"); colon >= 0 {
			linea = strings.TrimSpace(linea[idx+colon+1:])
		} else {
			continue
		}
		if end := strings.Index(linea, ". "); end >= 0 {
			linea = strings.TrimSpace(linea[:end])
		}
		linea = strings.TrimSuffix(linea, ".")
		linea = strings.ReplaceAll(linea, " y ", ", ")
		linea = strings.ReplaceAll(linea, ";", ",")
		partes := strings.Split(linea, ",")
		writeSet := make([]string, 0, len(partes))
		seen := map[string]struct{}{}
		for _, parte := range partes {
			ruta := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(parte), "."))
			if ruta == "" {
				continue
			}
			if _, ok := seen[ruta]; ok {
				continue
			}
			seen[ruta] = struct{}{}
			writeSet = append(writeSet, ruta)
		}
		if len(writeSet) > 0 {
			return writeSet
		}
	}
	return nil
}

func ExtraerSimbolosFocoTextoPipelineLocal(raw string) string {
	return extraerClauseTextoPipelineLocal(raw, "simbolos foco")
}

func ExtraerTestsMinimosTextoPipelineLocal(raw string) string {
	return extraerClauseTextoPipelineLocal(raw, "tests minimos", "tests mínimos")
}

func extraerClauseTextoPipelineLocal(raw string, marcadores ...string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	lineas := strings.FieldsFunc(raw, func(r rune) bool { return r == '\n' || r == '\r' })
	for _, linea := range lineas {
		linea = strings.TrimSpace(linea)
		if linea == "" {
			continue
		}
		lower := strings.ToLower(linea)
		for _, marcador := range marcadores {
			marcador = strings.ToLower(strings.TrimSpace(marcador))
			if marcador == "" {
				continue
			}
			idx := strings.Index(lower, marcador)
			if idx < 0 {
				continue
			}
			segmento := linea[idx:]
			if colon := strings.Index(segmento, ":"); colon >= 0 {
				segmento = strings.TrimSpace(segmento[colon+1:])
			} else {
				continue
			}
			if end := strings.Index(segmento, ". "); end >= 0 {
				segmento = strings.TrimSpace(segmento[:end])
			}
			segmento = strings.TrimSpace(strings.TrimSuffix(segmento, "."))
			if segmento == "" {
				continue
			}
			if strings.Contains(marcador, "tests minimos") || strings.Contains(marcador, "tests mínimos") {
				segmento = strings.ReplaceAll(segmento, " -count=1", "")
				segmento = strings.ReplaceAll(segmento, " y go test ", " ; go test ")
			}
			return strings.Join(strings.Fields(segmento), " ")
		}
	}
	return ""
}

func (s *Service) listarGatesPipelineLocal(proyectoSlug string) ([]*reviewapp.Gate, error) {
	items, err := s.reviewGateProvider.List(reviewapp.ListInput{
		ProyectoRef: strings.TrimSpace(proyectoSlug),
		Limit:       20,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(strings.ToLower(err.Error()), "no rows") {
			return nil, nil
		}
		return nil, err
	}
	return items, nil
}

func (s *Service) resolverGateBloqueantePipelineLocal(proyectoSlug string) (*reviewapp.Gate, string, bool, error) {
	if s.reviewGateProvider == nil || strings.TrimSpace(proyectoSlug) == "" {
		return nil, "", false, nil
	}
	items, err := s.listarGatesPipelineLocal(proyectoSlug)
	if err != nil {
		return nil, "", false, err
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		if reviewapp.IsBlockingGateState(item.Estado) {
			return item, "hay una review gate abierta con cambios o bloqueo; la siguiente acción debe ser corrección", true, nil
		}
	}
	return nil, "", false, nil
}

func (s *Service) resolverGateRevisionPendientePipelineLocal(proyectoSlug string) (*reviewapp.Gate, string, bool, error) {
	if s.reviewGateProvider == nil || strings.TrimSpace(proyectoSlug) == "" {
		return nil, "", false, nil
	}
	items, err := s.listarGatesPipelineLocal(proyectoSlug)
	if err != nil {
		return nil, "", false, err
	}
	for _, item := range items {
		if item == nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(item.Estado), reviewapp.GateStatePending) || strings.EqualFold(strings.TrimSpace(item.Estado), reviewapp.GateStateInReview) {
			return item, "hay una review gate pendiente o en revisión; el pipeline debe esperar o continuar revisión", true, nil
		}
	}
	return nil, "", false, nil
}

func (s *Service) resolverGateAprobadoPipelineLocal(proyectoSlug string, tareaID int64) (*reviewapp.Gate, bool, error) {
	if s.reviewGateProvider == nil || strings.TrimSpace(proyectoSlug) == "" || tareaID <= 0 {
		return nil, false, nil
	}
	items, err := s.listarGatesPipelineLocal(proyectoSlug)
	if err != nil {
		return nil, false, err
	}
	for _, item := range items {
		if item == nil || item.TareaID == nil || *item.TareaID != tareaID {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(item.Estado), reviewapp.GateStateApproved) {
			return item, true, nil
		}
	}
	return nil, false, nil
}

func primeraEtapaPipelineLocal(pipeline *PipelineLocalDeterminista) *EtapaPipelineLocal {
	if pipeline == nil || len(pipeline.Fases) == 0 {
		return nil
	}
	return &pipeline.Fases[0]
}

func buscarEtapaPipelineLocal(pipeline *PipelineLocalDeterminista, fase string) *EtapaPipelineLocal {
	if pipeline == nil {
		return nil
	}
	for i := range pipeline.Fases {
		if strings.EqualFold(strings.TrimSpace(pipeline.Fases[i].Fase), strings.TrimSpace(fase)) {
			return &pipeline.Fases[i]
		}
	}
	return nil
}
