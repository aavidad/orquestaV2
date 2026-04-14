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
	ProyectoSlug   string                     `json:"proyecto_slug,omitempty"`
	Accion         string                     `json:"accion"`
	AccionTarea    string                     `json:"accion_tarea,omitempty"`
	Motivo         string                     `json:"motivo"`
	FaseActual     string                     `json:"fase_actual,omitempty"`
	FaseObjetivo   string                     `json:"fase_objetivo,omitempty"`
	EtapaObjetivo  *EtapaPipelineLocal        `json:"etapa_objetivo,omitempty"`
	GateBloqueante *reviewapp.Gate            `json:"gate_bloqueante,omitempty"`
	TareaObjetivo  *TareaPipelineLocal        `json:"tarea_objetivo,omitempty"`
	Pipeline       *PipelineLocalDeterminista `json:"pipeline,omitempty"`
}

type TareaPipelineLocal struct {
	ID           int64    `json:"id"`
	Titulo       string   `json:"titulo"`
	Descripcion  string   `json:"descripcion,omitempty"`
	Notas        string   `json:"notas,omitempty"`
	WriteSet     []string `json:"write_set,omitempty"`
	SimbolosFoco string   `json:"simbolos_foco,omitempty"`
	TestsMinimos string   `json:"tests_minimos,omitempty"`
	Estado       string   `json:"estado"`
	Agente       string   `json:"agente,omitempty"`
	Modulo       string   `json:"modulo,omitempty"`
	Prioridad    string   `json:"prioridad,omitempty"`
}

func (s *Service) SetReviewGateProvider(provider ReviewGateProvider) {
	s.reviewGateProvider = provider
}

func (s *Service) CalcularSiguientePasoPipelineLocalDeterminista(proyectoSlug string) (*PasoPipelineLocalDeterminista, error) {
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

	if gate, motivo, ok, err := s.resolverGateBloqueantePipelineLocal(strings.TrimSpace(proyectoSlug)); err != nil {
		return nil, err
	} else if ok {
		etapa := buscarEtapaPipelineLocal(pipeline, "correccion")
		return &PasoPipelineLocalDeterminista{
			ProyectoSlug:   strings.TrimSpace(proyectoSlug),
			Accion:         "abrir_correccion",
			AccionTarea:    "corregir",
			Motivo:         motivo,
			FaseActual:     faseActual,
			FaseObjetivo:   "correccion",
			EtapaObjetivo:  etapa,
			GateBloqueante: gate,
			TareaObjetivo:  s.seleccionarTareaPipelineLocal(strings.TrimSpace(proyectoSlug), "correccion"),
			Pipeline:       pipeline,
		}, nil
	}

	if gate, motivo, ok, err := s.resolverGateRevisionPendientePipelineLocal(strings.TrimSpace(proyectoSlug)); err != nil {
		return nil, err
	} else if ok {
		etapa := buscarEtapaPipelineLocal(pipeline, "revision")
		return &PasoPipelineLocalDeterminista{
			ProyectoSlug:   strings.TrimSpace(proyectoSlug),
			Accion:         "esperar_revision",
			AccionTarea:    "revisar",
			Motivo:         motivo,
			FaseActual:     faseActual,
			FaseObjetivo:   "revision",
			EtapaObjetivo:  etapa,
			GateBloqueante: gate,
			TareaObjetivo:  s.seleccionarTareaPipelineLocal(strings.TrimSpace(proyectoSlug), "revision"),
			Pipeline:       pipeline,
		}, nil
	}

	if faseActual == "" {
		etapa := primeraEtapaPipelineLocal(pipeline)
		if etapa == nil {
			return nil, fmt.Errorf("pipeline local sin fases")
		}
		return &PasoPipelineLocalDeterminista{
			ProyectoSlug:  strings.TrimSpace(proyectoSlug),
			Accion:        "arrancar_fase",
			AccionTarea:   accionTareaParaFase(etapa.Fase),
			Motivo:        "no hay fase activa registrada; se arranca la primera fase canónica",
			FaseObjetivo:  etapa.Fase,
			EtapaObjetivo: etapa,
			TareaObjetivo: s.seleccionarTareaPipelineLocal(strings.TrimSpace(proyectoSlug), etapa.Fase),
			Pipeline:      pipeline,
		}, nil
	}

	etapa := buscarEtapaPipelineLocal(pipeline, faseActual)
	if etapa == nil {
		etapa = primeraEtapaPipelineLocal(pipeline)
		return &PasoPipelineLocalDeterminista{
			ProyectoSlug:  strings.TrimSpace(proyectoSlug),
			Accion:        "reconciliar_fase",
			AccionTarea:   accionTareaParaFase(etapa.Fase),
			Motivo:        "la fase activa no coincide con el pipeline canónico; se propone reconciliación al inicio",
			FaseActual:    faseActual,
			FaseObjetivo:  etapa.Fase,
			EtapaObjetivo: etapa,
			TareaObjetivo: s.seleccionarTareaPipelineLocal(strings.TrimSpace(proyectoSlug), etapa.Fase),
			Pipeline:      pipeline,
		}, nil
	}
	tareaActual := s.seleccionarTareaPipelineLocal(strings.TrimSpace(proyectoSlug), etapa.Fase)
	if strings.EqualFold(etapa.Fase, "implementacion") || strings.EqualFold(etapa.Fase, "correccion") {
		if tareaActual != nil && strings.EqualFold(strings.TrimSpace(tareaActual.Estado), string(db.TareaCompletada)) {
			etapaRevision := buscarEtapaPipelineLocal(pipeline, "revision")
			return &PasoPipelineLocalDeterminista{
				ProyectoSlug:  strings.TrimSpace(proyectoSlug),
				Accion:        "avanzar_fase",
				AccionTarea:   accionTareaParaFase("revision"),
				Motivo:        "la tarea activa ya está completada; el pipeline debe avanzar a revisión",
				FaseActual:    faseActual,
				FaseObjetivo:  "revision",
				EtapaObjetivo: etapaRevision,
				TareaObjetivo: s.seleccionarTareaPipelineLocal(strings.TrimSpace(proyectoSlug), "revision"),
				Pipeline:      pipeline,
			}, nil
		}
	}
	if strings.EqualFold(etapa.Fase, "revision") {
		if tareaActual != nil && strings.EqualFold(strings.TrimSpace(tareaActual.Estado), string(db.TareaCompletada)) {
			if gate, ok, err := s.resolverGateAprobadoPipelineLocal(strings.TrimSpace(proyectoSlug), tareaActual.ID); err != nil {
				return nil, err
			} else if ok {
				etapaIntegracion := buscarEtapaPipelineLocal(pipeline, "integracion")
				return &PasoPipelineLocalDeterminista{
					ProyectoSlug:   strings.TrimSpace(proyectoSlug),
					Accion:         "avanzar_fase",
					AccionTarea:    accionTareaParaFase("integracion"),
					Motivo:         "la review gate de la tarea ya está aprobada; el pipeline debe avanzar a integración",
					FaseActual:     faseActual,
					FaseObjetivo:   "integracion",
					EtapaObjetivo:  etapaIntegracion,
					GateBloqueante: gate,
					TareaObjetivo:  s.seleccionarTareaPipelineLocal(strings.TrimSpace(proyectoSlug), "integracion"),
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
	return &PasoPipelineLocalDeterminista{
		ProyectoSlug:  strings.TrimSpace(proyectoSlug),
		Accion:        accion,
		AccionTarea:   accionTareaParaFase(etapa.Fase),
		Motivo:        motivo,
		FaseActual:    faseActual,
		FaseObjetivo:  etapa.Fase,
		EtapaObjetivo: etapa,
		TareaObjetivo: tareaActual,
		Pipeline:      pipeline,
	}, nil
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
		score := puntuarTareaPipelineLocal(tarea, fase)
		if score > mejor {
			mejor = score
			candidata = tarea
		}
	}
	if candidata == nil || mejor < 0 {
		return nil
	}
	return tareaPipelineLocalDesdeDB(candidata)
}

func puntuarTareaPipelineLocal(tarea *db.Tarea, fase string) int {
	if tarea == nil {
		return -1
	}
	estado := strings.ToLower(strings.TrimSpace(string(tarea.Estado)))
	switch strings.ToLower(strings.TrimSpace(fase)) {
	case "especificacion", "implementacion", "correccion":
		switch estado {
		case string(db.TareaCompletada):
			return 50
		case string(db.TareaEnProgreso):
			return 40
		case string(db.TareaAsignada):
			return 30
		case string(db.TareaLibre):
			return 20
		case string(db.TareaBacklog):
			return 10
		default:
			return -1
		}
	case "revision":
		switch estado {
		case string(db.TareaCompletada):
			return 30
		case string(db.TareaEnProgreso):
			return 20
		case string(db.TareaAsignada):
			return 10
		default:
			return -1
		}
	case "integracion":
		if estado == string(db.TareaCompletada) {
			return 10
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

func (s *Service) resolverGateBloqueantePipelineLocal(proyectoSlug string) (*reviewapp.Gate, string, bool, error) {
	if s.reviewGateProvider == nil || strings.TrimSpace(proyectoSlug) == "" {
		return nil, "", false, nil
	}
	items, err := s.reviewGateProvider.List(reviewapp.ListInput{
		ProyectoRef: strings.TrimSpace(proyectoSlug),
		Limit:       20,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(strings.ToLower(err.Error()), "no rows") {
			return nil, "", false, nil
		}
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
	items, err := s.reviewGateProvider.List(reviewapp.ListInput{
		ProyectoRef: strings.TrimSpace(proyectoSlug),
		Limit:       20,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(strings.ToLower(err.Error()), "no rows") {
			return nil, "", false, nil
		}
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
	items, err := s.reviewGateProvider.List(reviewapp.ListInput{
		ProyectoRef: strings.TrimSpace(proyectoSlug),
		Limit:       20,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(strings.ToLower(err.Error()), "no rows") {
			return nil, false, nil
		}
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
