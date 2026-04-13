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
