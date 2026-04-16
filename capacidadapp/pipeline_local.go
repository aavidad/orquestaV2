package capacidadapp

import (
	"strings"

	"orquesta/db"
)

type EtapaPipelineLocal struct {
	Fase             string               `json:"fase"`
	PerfilTarea      string               `json:"perfil_tarea"`
	ModoEjecucion    string               `json:"modo_ejecucion"`
	Carril           string               `json:"carril"`
	EntregaCanonica  string               `json:"entrega_canonica,omitempty"`
	RequiereWorktree bool                 `json:"requiere_worktree"`
	UsaMicroprograma bool                 `json:"usa_microprogramacion"`
	RequiereModelo   bool                 `json:"requiere_modelo"`
	ObjetivoModelo   string               `json:"objetivo_modelo,omitempty"`
	ModeloFallback   string               `json:"modelo_fallback,omitempty"`
	ResolucionActual *db.ResolucionModelo `json:"resolucion_actual,omitempty"`
}

type RevisorEscalonado struct {
	NombreRol        string               `json:"nombre_rol"`
	Clase            string               `json:"clase"`
	Premium          bool                 `json:"premium"`
	PerfilTarea      string               `json:"perfil_tarea"`
	Carril           string               `json:"carril"`
	EntregaCanonica  string               `json:"entrega_canonica,omitempty"`
	RequiereWorktree bool                 `json:"requiere_worktree"`
	ObjetivoModelo   string               `json:"objetivo_modelo"`
	ModeloFallback   string               `json:"modelo_fallback,omitempty"`
	ResolucionActual *db.ResolucionModelo `json:"resolucion_actual,omitempty"`
}

type RevisionEscalonada struct {
	AbrirSegundaOpinionCuando []string            `json:"abrir_segunda_opinion_cuando"`
	Revisores                 []RevisorEscalonado `json:"revisores"`
}

type PipelineLocalDeterminista struct {
	ProyectoSlug string               `json:"proyecto_slug,omitempty"`
	Fases        []EtapaPipelineLocal `json:"fases"`
	Revision     *RevisionEscalonada  `json:"revision,omitempty"`
}

func (s *Service) ConstruirPipelineLocalDeterminista(proyectoSlug string) (*PipelineLocalDeterminista, error) {
	proyectoSlug = strings.TrimSpace(proyectoSlug)
	fases := []EtapaPipelineLocal{
		s.construirEtapaPipelineLocal(proyectoSlug, "especificacion", "analisis", "worker_modelo", "premium_worktree", "git_worktree", true, false, true, "qwen3.5:27b-q4_K_M", "gemma4:26b"),
		s.construirEtapaPipelineLocal(proyectoSlug, "implementacion", "implementacion", "worker_modelo", "premium_worktree", "git_worktree", true, false, true, "qwen2.5-coder:32b", "gemma4:26b"),
		s.construirEtapaPipelineLocal(proyectoSlug, "revision", "revision", "worker_modelo", "revision_diff", "hallazgos_estructurados", true, false, true, "deepseek-coder-v2", "gemma4:26b"),
		s.construirEtapaPipelineLocal(proyectoSlug, "correccion", "implementacion", "worker_modelo", "premium_worktree", "git_worktree", true, false, true, "qwen2.5-coder:32b", "gemma4:26b"),
		s.construirEtapaPipelineLocal(proyectoSlug, "integracion", "analisis", "determinista_app", "determinista_app", "merge_controlado", false, false, false, "", ""),
	}
	return &PipelineLocalDeterminista{
		ProyectoSlug: proyectoSlug,
		Fases:        fases,
		Revision: &RevisionEscalonada{
			AbrirSegundaOpinionCuando: []string{
				"diff_grande",
				"zona_critica_control_plane",
				"seguridad_o_concurrencia",
				"hallazgos_contradictorios",
				"correcciones_repetidas",
				"riesgo_alto",
			},
			Revisores: []RevisorEscalonado{
				s.construirRevisorEscalonado(proyectoSlug, "revisor_base_local", "local", false, "revision", "revision_diff", "hallazgos_estructurados", true, "deepseek-coder-v2", "gemma4:26b"),
				s.construirRevisorEscalonado(proyectoSlug, "segunda_opinion_local", "local", false, "revision", "revision_diff", "hallazgos_estructurados", true, "qwen3.5:27b-q4_K_M", "gemma4:26b"),
				s.construirRevisorEscalonado(proyectoSlug, "segunda_opinion_premium", "premium", true, "revision", "revision_diff", "hallazgos_estructurados", true, "gpt-5.4", "claude"),
			},
		},
	}, nil
}

func (s *Service) construirEtapaPipelineLocal(proyectoSlug, fase, perfil, modo, carril, entregaCanonica string, requiereWorktree, usaMicroprograma, requiereModelo bool, objetivoModelo, modeloFallback string) EtapaPipelineLocal {
	etapa := EtapaPipelineLocal{
		Fase:             strings.TrimSpace(fase),
		PerfilTarea:      strings.TrimSpace(perfil),
		ModoEjecucion:    strings.TrimSpace(modo),
		Carril:           strings.TrimSpace(carril),
		EntregaCanonica:  strings.TrimSpace(entregaCanonica),
		RequiereWorktree: requiereWorktree,
		UsaMicroprograma: usaMicroprograma,
		RequiereModelo:   requiereModelo,
		ObjetivoModelo:   strings.TrimSpace(objetivoModelo),
		ModeloFallback:   strings.TrimSpace(modeloFallback),
	}
	if !requiereModelo {
		return etapa
	}
	resolucion, err := s.ResolveModelPolicy(db.ResolverPoliticaInput{
		ProyectoSlug: strings.TrimSpace(proyectoSlug),
		Fase:         strings.TrimSpace(fase),
		PerfilTarea:  strings.TrimSpace(perfil),
	})
	if err == nil {
		etapa.ResolucionActual = resolucion
	}
	return etapa
}

// EspecificacionFuncion consolida el contrato que el orquestador fija para cada entrega:
// encabezado, salida esperada, write_set y tests mínimos.
type EspecificacionFuncion struct {
	Encabezado     string   `json:"encabezado"`
	SalidaEsperada string   `json:"salida_esperada,omitempty"`
	WriteSet       []string `json:"write_set,omitempty"`
	SimbolosFoco   string   `json:"simbolos_foco,omitempty"`
	TestsMinimos   string   `json:"tests_minimos,omitempty"`
}

// extraerSeñalesTareaPipelineLocal deriva las señales de escalación de revisión
// a partir del contexto de la tarea (write_set, notas, descripción).
func extraerSeñalesTareaPipelineLocal(tarea *TareaPipelineLocal) []string {
	if tarea == nil {
		return nil
	}
	ctx := strings.ToLower(strings.Join([]string{
		strings.TrimSpace(tarea.Titulo),
		strings.TrimSpace(tarea.Descripcion),
		strings.TrimSpace(tarea.Notas),
	}, "\n"))
	var señales []string

	// zona_critica_control_plane: write_set toca controlplane, o el módulo es control_plane
	zonaCritica := false
	for _, ruta := range tarea.WriteSet {
		ruta = strings.ToLower(strings.TrimSpace(ruta))
		if strings.Contains(ruta, "controlplane") || strings.Contains(ruta, "control_plane") {
			zonaCritica = true
			break
		}
	}
	if !zonaCritica {
		mod := strings.ToLower(strings.TrimSpace(tarea.Modulo))
		if strings.Contains(mod, "controlplane") || strings.Contains(mod, "control_plane") {
			zonaCritica = true
		}
	}
	if zonaCritica {
		señales = append(señales, "zona_critica_control_plane")
	}

	if strings.Contains(ctx, "seguridad") || strings.Contains(ctx, "concurrencia") || strings.Contains(ctx, "seguridad_o_concurrencia") {
		señales = append(señales, "seguridad_o_concurrencia")
	}
	if strings.Contains(ctx, "riesgo_alto") || strings.Contains(ctx, "critico") || strings.EqualFold(strings.TrimSpace(tarea.Prioridad), "critica") {
		señales = append(señales, "riesgo_alto")
	}
	if len(tarea.WriteSet) > 5 {
		señales = append(señales, "diff_grande")
	}
	if strings.Contains(ctx, "correcciones_repetidas") || strings.Contains(ctx, "ciclo_correccion") {
		señales = append(señales, "correcciones_repetidas")
	}
	if strings.Contains(ctx, "hallazgos_contradictorios") || strings.Contains(ctx, "contradiccion") {
		señales = append(señales, "hallazgos_contradictorios")
	}
	return señales
}

func construirEspecificacionFuncion(tarea *TareaPipelineLocal, etapa *EtapaPipelineLocal) *EspecificacionFuncion {
	if tarea == nil || etapa == nil || !etapa.RequiereModelo {
		return nil
	}
	encabezado := strings.TrimSpace(tarea.Titulo)
	if encabezado == "" {
		return nil
	}
	return &EspecificacionFuncion{
		Encabezado:     encabezado,
		SalidaEsperada: strings.TrimSpace(etapa.EntregaCanonica),
		WriteSet:       append([]string(nil), tarea.WriteSet...),
		SimbolosFoco:   strings.TrimSpace(tarea.SimbolosFoco),
		TestsMinimos:   strings.TrimSpace(tarea.TestsMinimos),
	}
}

func seleccionarRevisorEscalonadoPipeline(revision *RevisionEscalonada, señales []string) *RevisorEscalonado {
	if revision == nil || len(revision.Revisores) == 0 {
		return nil
	}
	for _, señal := range señales {
		switch strings.ToLower(strings.TrimSpace(señal)) {
		case "seguridad_o_concurrencia", "zona_critica_control_plane", "riesgo_alto":
			for i := range revision.Revisores {
				if revision.Revisores[i].Premium {
					return &revision.Revisores[i]
				}
			}
		}
	}
	for _, señal := range señales {
		switch strings.ToLower(strings.TrimSpace(señal)) {
		case "hallazgos_contradictorios", "correcciones_repetidas", "diff_grande":
			for i := range revision.Revisores {
				if !revision.Revisores[i].Premium && strings.Contains(revision.Revisores[i].NombreRol, "segunda_opinion") {
					return &revision.Revisores[i]
				}
			}
		}
	}
	return &revision.Revisores[0]
}

func (s *Service) construirRevisorEscalonado(proyectoSlug, nombreRol, clase string, premium bool, perfil, carril, entregaCanonica string, requiereWorktree bool, objetivoModelo, modeloFallback string) RevisorEscalonado {
	revisor := RevisorEscalonado{
		NombreRol:        strings.TrimSpace(nombreRol),
		Clase:            strings.TrimSpace(clase),
		Premium:          premium,
		PerfilTarea:      strings.TrimSpace(perfil),
		Carril:           strings.TrimSpace(carril),
		EntregaCanonica:  strings.TrimSpace(entregaCanonica),
		RequiereWorktree: requiereWorktree,
		ObjetivoModelo:   strings.TrimSpace(objetivoModelo),
		ModeloFallback:   strings.TrimSpace(modeloFallback),
	}
	resolucion, err := s.ResolveModelPolicy(db.ResolverPoliticaInput{
		ProyectoSlug: strings.TrimSpace(proyectoSlug),
		Fase:         "revision",
		PerfilTarea:  strings.TrimSpace(perfil),
	})
	if err == nil {
		revisor.ResolucionActual = resolucion
	}
	return revisor
}
