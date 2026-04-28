/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"orquesta/capacidadapp"
	"orquesta/db"
	"orquesta/supervisionapp"
	"orquesta/tareasapp"
)

var (
	apiRepoAddServerFn  = repoAddServer
	apiRepoResolverFn   = resolverProyectoRepoServer
	apiRepoPlanFn       = capacidadService.CalcularSiguientePasoPipelineLocalDeterminista
	apiRepoDispatchFn   = capacidadService.EjecutarYDespacharSiguientePasoPipelineLocalDeterminista
	apiRepoTaskCreateFn = tareasService.Create
	apiRepoTaskGetFn    = tareasService.Get
)

type apiRepoMaterializarRequest struct {
	Proyecto string `json:"proyecto"`
	Path     string `json:"path"`
	Git      string `json:"git"`
	Branch   string `json:"branch"`
	Destino  string `json:"destino"`
}

type apiRepoRevisarRequest struct {
	Proyecto string `json:"proyecto"`
	Path     string `json:"path"`
	Git      string `json:"git"`
	Branch   string `json:"branch"`
	Destino  string `json:"destino"`
	Plan     bool   `json:"plan"`
}

type apiRepoMejorarRequest struct {
	Proyecto              string   `json:"proyecto"`
	Path                  string   `json:"path"`
	Git                   string   `json:"git"`
	Branch                string   `json:"branch"`
	Destino               string   `json:"destino"`
	Titulo                string   `json:"titulo"`
	Descripcion           string   `json:"descripcion"`
	Modulo                string   `json:"modulo"`
	Prioridad             string   `json:"prioridad"`
	CreadoPor             string   `json:"creado_por"`
	Notas                 string   `json:"notas"`
	FuncionObjetivo       string   `json:"funcion_objetivo"`
	WriteSet              []string `json:"write_set"`
	ModelosCandidatos     []string `json:"modelos_candidatos"`
	PreservarArquitectura bool     `json:"preservar_arquitectura"`
	FinishApp             bool     `json:"finish_app"`
	AutonomiaPersistente  bool     `json:"autonomia_persistente"`
	SupervisorAgente      string   `json:"supervisor_agente"`
	ReviewerAgente        string   `json:"reviewer_agente"`
	MaxWorkers            int      `json:"max_workers"`
	Despachar             *bool    `json:"despachar,omitempty"`
}

type apiRepoFunctionForkSpec struct {
	FuncionObjetivo       string   `json:"funcion_objetivo,omitempty"`
	WriteSet              []string `json:"write_set,omitempty"`
	ModelosCandidatos     []string `json:"modelos_candidatos,omitempty"`
	PreservarArquitectura bool     `json:"preservar_arquitectura,omitempty"`
	Materia               string   `json:"materia,omitempty"`
	ForkLines             int      `json:"fork_lines,omitempty"`
	SelectedModels        []string `json:"selected_models,omitempty"`
	DecisionMode          string   `json:"decision_mode,omitempty"`
	DecisionReason        string   `json:"decision_reason,omitempty"`
}

type apiRepoMaterializarResponse struct {
	OK            bool         `json:"ok"`
	Proyecto      *db.Proyecto `json:"proyecto,omitempty"`
	DiscoveryRoot string       `json:"discovery_root,omitempty"`
	RutaAbs       string       `json:"ruta_abs,omitempty"`
	RemoteURL     string       `json:"remote_url,omitempty"`
	BranchBase    string       `json:"branch_base,omitempty"`
}

type apiRepoRevisarResponse struct {
	OK        bool                                              `json:"ok"`
	Proyecto  *db.Proyecto                                      `json:"proyecto,omitempty"`
	Paso      *capacidadapp.PasoPipelineLocalDeterminista       `json:"paso,omitempty"`
	Resultado *capacidadapp.ResultadoEjecucionPasoPipelineLocal `json:"resultado,omitempty"`
}

type apiRepoMejorarResponse struct {
	OK        bool                                              `json:"ok"`
	Proyecto  *db.Proyecto                                      `json:"proyecto,omitempty"`
	Tarea     *db.Tarea                                         `json:"tarea,omitempty"`
	Fork      *apiRepoFunctionForkSpec                          `json:"fork,omitempty"`
	Policy    *supervisionapp.Policy                            `json:"policy,omitempty"`
	Resultado *capacidadapp.ResultadoEjecucionPasoPipelineLocal `json:"resultado,omitempty"`
}

func apiHandlerRepoMaterializar(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiRepoMaterializarRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	resultado, err := apiRepoAddServerFn(strings.TrimSpace(req.Path), strings.TrimSpace(req.Git), strings.TrimSpace(req.Branch), strings.TrimSpace(req.Destino))
	if err != nil {
		apiError(w, repoAPIStatusForError(err), err)
		return
	}
	if resultado == nil || resultado.Proyecto == nil {
		apiError(w, http.StatusInternalServerError, fmt.Errorf("materialización sin proyecto"))
		return
	}
	apiWriteJSON(w, http.StatusCreated, apiRepoMaterializarResponse{
		OK:            true,
		Proyecto:      resultado.Proyecto,
		DiscoveryRoot: resultado.DiscoveryRoot,
		RutaAbs:       resultado.RutaAbs,
		RemoteURL:     resultado.RemoteURL,
		BranchBase:    resultado.BranchBase,
	})
}

func apiHandlerRepoRevisar(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiRepoRevisarRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	proyecto, _, err := apiRepoResolverFn(req.Proyecto, req.Path, req.Git, req.Branch, req.Destino)
	if err != nil {
		apiError(w, repoAPIStatusForError(err), err)
		return
	}
	if proyecto == nil || strings.TrimSpace(proyecto.Slug) == "" {
		apiError(w, http.StatusInternalServerError, fmt.Errorf("proyecto materializado inválido"))
		return
	}
	if req.Plan {
		paso, err := apiRepoPlanFn(strings.TrimSpace(proyecto.Slug))
		if err != nil {
			apiError(w, repoAPIStatusForError(err), err)
			return
		}
		if paso == nil {
			apiError(w, http.StatusInternalServerError, fmt.Errorf("planificación sin paso"))
			return
		}
		apiWriteJSON(w, http.StatusOK, apiRepoRevisarResponse{OK: true, Proyecto: proyecto, Paso: paso})
		return
	}
	resultado, err := apiRepoDispatchFn(strings.TrimSpace(proyecto.Slug))
	if err != nil {
		apiError(w, repoAPIStatusForError(err), err)
		return
	}
	if resultado == nil {
		apiError(w, http.StatusInternalServerError, fmt.Errorf("ejecución sin resultado"))
		return
	}
	apiWriteJSON(w, http.StatusOK, apiRepoRevisarResponse{OK: true, Proyecto: proyecto, Resultado: resultado})
}

func apiHandlerRepoMejorar(w http.ResponseWriter, r *http.Request) {
	if !apiRequireMethod(w, r, http.MethodPost) {
		return
	}
	var req apiRepoMejorarRequest
	if err := apiDecodeJSON(r, &req); err != nil {
		apiError(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(req.Titulo) == "" {
		apiError(w, http.StatusBadRequest, fmt.Errorf("el título es obligatorio"))
		return
	}
	proyecto, _, err := apiRepoResolverFn(req.Proyecto, req.Path, req.Git, req.Branch, req.Destino)
	if err != nil {
		apiError(w, repoAPIStatusForError(err), err)
		return
	}
	if proyecto == nil || strings.TrimSpace(proyecto.Slug) == "" {
		apiError(w, http.StatusInternalServerError, fmt.Errorf("proyecto materializado inválido"))
		return
	}
	req = normalizeRepoMejorarRequest(req)
	req, err = resolveRepoPersistentAutonomyRequest(proyecto, req)
	if err != nil {
		apiError(w, repoAPIStatusForError(err), err)
		return
	}
	policy, err := upsertRepoPersistentAutonomy(proyecto, req)
	if err != nil {
		apiError(w, repoAPIStatusForError(err), err)
		return
	}
	id, err := apiRepoTaskCreateFn(tareasapp.CreateTaskInput{
		Titulo:      strings.TrimSpace(req.Titulo),
		Descripcion: strings.TrimSpace(req.Descripcion),
		Modulo:      strings.TrimSpace(req.Modulo),
		Prioridad:   prioridadRepoAPI(req.Prioridad),
		CreadoPor:   valorConFallback(strings.TrimSpace(req.CreadoPor), "repo_orchestrator"),
		Proyecto:    strings.TrimSpace(proyecto.Slug),
		Notas:       repoMejorarNotasEnriquecidas(req, planRepoFunctionForkSpec(req)),
	})
	if err != nil {
		apiError(w, repoAPIStatusForError(err), err)
		return
	}
	tarea, err := apiRepoTaskGetFn(id)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err)
		return
	}
	if tarea == nil {
		apiError(w, http.StatusInternalServerError, fmt.Errorf("creación sin tarea recargada"))
		return
	}
	despachar := true
	if req.Despachar != nil {
		despachar = *req.Despachar
	}
	fork := planRepoFunctionForkSpec(req)
	if fork == nil {
		fork = parseRepoFunctionForkSpecFromNotes(strings.TrimSpace(tarea.Notas))
	}
	if !despachar {
		apiWriteJSON(w, http.StatusCreated, apiRepoMejorarResponse{
			OK:       true,
			Proyecto: proyecto,
			Tarea:    tarea,
			Fork:     fork,
			Policy:   policy,
		})
		return
	}
	resultado, err := apiRepoDispatchFn(strings.TrimSpace(proyecto.Slug))
	if err != nil {
		apiError(w, repoAPIStatusForError(err), err)
		return
	}
	if resultado == nil {
		apiError(w, http.StatusInternalServerError, fmt.Errorf("ejecución sin resultado"))
		return
	}
	apiWriteJSON(w, http.StatusCreated, apiRepoMejorarResponse{
		OK:        true,
		Proyecto:  proyecto,
		Tarea:     tarea,
		Fork:      fork,
		Policy:    policy,
		Resultado: resultado,
	})
}

func repoAPIStatusForError(err error) int {
	if err == nil {
		return http.StatusInternalServerError
	}
	if errors.Is(err, errRepoBadRequest) {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

func repoMejorarNotasEnriquecidas(req apiRepoMejorarRequest, planned *apiRepoFunctionForkSpec) string {
	spec := planned
	if spec == nil {
		spec = normalizeRepoFunctionForkSpec(&apiRepoFunctionForkSpec{
			FuncionObjetivo:       req.FuncionObjetivo,
			WriteSet:              req.WriteSet,
			ModelosCandidatos:     req.ModelosCandidatos,
			PreservarArquitectura: req.PreservarArquitectura,
		})
	}
	notas := strings.TrimSpace(req.Notas)
	if spec == nil {
		return notas
	}
	var bloque []string
	bloque = append(bloque, "fork_funcion_v1:")
	if spec.FuncionObjetivo != "" {
		bloque = append(bloque, "  funcion_objetivo: "+spec.FuncionObjetivo)
	}
	if len(spec.WriteSet) > 0 {
		bloque = append(bloque, "  write_set: "+strings.Join(spec.WriteSet, ", "))
	}
	if len(spec.ModelosCandidatos) > 0 {
		bloque = append(bloque, "  modelos_candidatos: "+strings.Join(spec.ModelosCandidatos, ", "))
	}
	if spec.Materia != "" {
		bloque = append(bloque, "  materia: "+spec.Materia)
	}
	if spec.ForkLines > 0 {
		bloque = append(bloque, fmt.Sprintf("  fork_lines: %d", spec.ForkLines))
	}
	if len(spec.SelectedModels) > 0 {
		bloque = append(bloque, "  selected_models: "+strings.Join(spec.SelectedModels, ", "))
	}
	if spec.DecisionMode != "" {
		bloque = append(bloque, "  decision_mode: "+spec.DecisionMode)
	}
	if spec.DecisionReason != "" {
		bloque = append(bloque, "  decision_reason: "+spec.DecisionReason)
	}
	if spec.PreservarArquitectura {
		bloque = append(bloque, "  preservar_arquitectura: true")
	}
	if req.FinishApp {
		bloque = append(bloque, "autonomia:finish_app")
	}
	if req.AutonomiaPersistente {
		bloque = append(bloque, "autonomia_persistente_v1:")
		supervisor := valorConFallback(strings.TrimSpace(req.SupervisorAgente), "Codex1")
		reviewer := valorConFallback(strings.TrimSpace(req.ReviewerAgente), repoPersistentReviewerDefault(supervisor))
		maxWorkers := req.MaxWorkers
		if maxWorkers <= 0 {
			maxWorkers = 3
		}
		bloque = append(bloque, "  enabled: true")
		bloque = append(bloque, "  supervisor_agente: "+supervisor)
		bloque = append(bloque, "  reviewer_agente: "+reviewer)
		bloque = append(bloque, fmt.Sprintf("  max_workers: %d", maxWorkers))
		bloque = append(bloque, "  auto_create_tasks: true")
	}
	if notas == "" {
		return strings.Join(bloque, "\n")
	}
	return notas + "\n\n" + strings.Join(bloque, "\n")
}

func normalizeRepoMejorarRequest(req apiRepoMejorarRequest) apiRepoMejorarRequest {
	req.Proyecto = strings.TrimSpace(req.Proyecto)
	req.Path = strings.TrimSpace(req.Path)
	req.Git = strings.TrimSpace(req.Git)
	req.Branch = strings.TrimSpace(req.Branch)
	req.Destino = strings.TrimSpace(req.Destino)
	req.Titulo = strings.TrimSpace(req.Titulo)
	req.Descripcion = strings.TrimSpace(req.Descripcion)
	req.Modulo = strings.TrimSpace(req.Modulo)
	req.Prioridad = strings.TrimSpace(req.Prioridad)
	req.CreadoPor = strings.TrimSpace(req.CreadoPor)
	req.Notas = strings.TrimSpace(req.Notas)
	req.FuncionObjetivo = strings.TrimSpace(req.FuncionObjetivo)
	req.SupervisorAgente = canonicalAutonomyCodexName(strings.TrimSpace(req.SupervisorAgente))
	req.ReviewerAgente = canonicalAutonomyCodexName(strings.TrimSpace(req.ReviewerAgente))
	req.WriteSet = trimNonEmptyStrings(req.WriteSet)
	req.ModelosCandidatos = trimNonEmptyStrings(req.ModelosCandidatos)
	if req.AutonomiaPersistente {
		req.FinishApp = true
		if req.MaxWorkers <= 0 {
			req.MaxWorkers = 3
		}
		req.SupervisorAgente = valorConFallback(req.SupervisorAgente, "Codex1")
		req.ReviewerAgente = valorConFallback(req.ReviewerAgente, repoPersistentReviewerDefault(req.SupervisorAgente))
	}
	return req
}

func resolveRepoPersistentAutonomyRequest(proyecto *db.Proyecto, req apiRepoMejorarRequest) (apiRepoMejorarRequest, error) {
	if proyecto == nil || !req.AutonomiaPersistente {
		return req, nil
	}
	supervisor, err := repoPersistentResolveAgent(proyecto.ID, req.SupervisorAgente, nil, "repo_mejorar_finish_app_supervisor")
	if err != nil {
		return req, err
	}
	if supervisor == nil {
		return req, repoBadRequestf("no hay Codex supervisor operativo disponible para %s", strings.TrimSpace(proyecto.Slug))
	}
	req.SupervisorAgente = strings.TrimSpace(supervisor.Nombre)
	reviewer, err := repoPersistentResolveAgent(proyecto.ID, req.ReviewerAgente, []string{req.SupervisorAgente}, "repo_mejorar_finish_app_reviewer")
	if err != nil {
		return req, err
	}
	if reviewer == nil {
		reviewer, err = repoPersistentResolveAgent(proyecto.ID, repoPersistentReviewerDefault(req.SupervisorAgente), []string{req.SupervisorAgente}, "repo_mejorar_finish_app_reviewer")
		if err != nil {
			return req, err
		}
	}
	if reviewer != nil {
		req.ReviewerAgente = strings.TrimSpace(reviewer.Nombre)
	}
	if req.MaxWorkers <= 0 {
		req.MaxWorkers = 3
	}
	if _, err := repoPersistentEnsureWorkers(proyecto.ID, req.MaxWorkers, []string{req.SupervisorAgente, req.ReviewerAgente}); err != nil {
		return req, err
	}
	return req, nil
}

func repoPersistentReviewerDefault(supervisor string) string {
	supervisor = strings.TrimSpace(supervisor)
	supervisor = canonicalAutonomyCodexName(supervisor)
	if strings.EqualFold(supervisor, "Codex2") {
		return "Codex3"
	}
	return "Codex2"
}

func repoPersistentResolveAgent(proyectoID int64, preferred string, excludes []string, activationReason string) (*db.Agente, error) {
	for _, candidate := range repoPersistentCodexCandidates(preferred) {
		if repoPersistentExcluded(candidate, excludes) {
			continue
		}
		agente, _, err := asegurarAgenteAutonomiaOperativo(proyectoID, candidate, activationReason)
		if err != nil {
			return nil, err
		}
		if agente != nil {
			return agente, nil
		}
	}
	return nil, nil
}

func repoPersistentEnsureWorkers(proyectoID int64, maxWorkers int, excludes []string) ([]string, error) {
	if proyectoID <= 0 || maxWorkers <= 0 {
		return nil, nil
	}
	out := make([]string, 0, maxWorkers)
	for _, candidate := range repoPersistentCodexCandidates("") {
		if len(out) >= maxWorkers {
			break
		}
		if repoPersistentExcluded(candidate, excludes) || repoPersistentExcluded(candidate, out) {
			continue
		}
		agente, _, err := asegurarAgenteAutonomiaOperativo(proyectoID, candidate, "repo_mejorar_finish_app_worker")
		if err != nil {
			return nil, err
		}
		if agente == nil {
			continue
		}
		out = append(out, strings.TrimSpace(agente.Nombre))
	}
	return out, nil
}

func repoPersistentCodexCandidates(preferred string) []string {
	var out []string
	add := func(raw string) {
		raw = canonicalAutonomyCodexName(strings.TrimSpace(raw))
		if raw == "" {
			return
		}
		for _, existing := range out {
			if strings.EqualFold(existing, raw) {
				return
			}
		}
		if preferred == "" && !perteneceAFlotaOficialAutobootstrap(raw) {
			return
		}
		out = append(out, raw)
	}
	add(preferred)
	add(strings.TrimSpace(configOrDefault("server_autobootstrap_supervisor_agent", "Codex1")))
	for _, item := range filtrarFlotaOficialAutobootstrap(splitServerAutobootstrapAgents(configOrDefault("server_autobootstrap_worker_agents", "Codex2,Codex3,Codex4,Codex5"))) {
		add(item)
	}
	if agentes, err := db.ListarAgentes(); err == nil {
		sort.Slice(agentes, func(i, j int) bool {
			if agentes[i] == nil && agentes[j] == nil {
				return false
			}
			if agentes[i] == nil {
				return false
			}
			if agentes[j] == nil {
				return true
			}
			return strings.TrimSpace(agentes[i].Nombre) < strings.TrimSpace(agentes[j].Nombre)
		})
		for _, agente := range agentes {
			if agente == nil {
				continue
			}
			add(strings.TrimSpace(agente.Nombre))
		}
	}
	return out
}

func repoPersistentExcluded(candidate string, excludes []string) bool {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return true
	}
	for _, exclude := range excludes {
		if strings.EqualFold(candidate, strings.TrimSpace(exclude)) {
			return true
		}
	}
	return false
}

func buildRepoPersistentAutonomyPolicy(proyecto *db.Proyecto, req apiRepoMejorarRequest) *supervisionapp.PolicyInput {
	if proyecto == nil || !req.AutonomiaPersistente {
		return nil
	}
	definition := fmt.Sprintf(`{"modo":"finish_app_persistente","finish_app":true,"preservar_arquitectura":%t,"funcion_objetivo":%q}`, req.PreservarArquitectura, strings.TrimSpace(req.FuncionObjetivo))
	objetivo := strings.TrimSpace(req.Descripcion)
	if objetivo == "" {
		objetivo = "Completar la app autonomamente hasta definition of done o bloqueo real"
	}
	return &supervisionapp.PolicyInput{
		Enabled:              true,
		ObjetivoGeneral:      objetivo,
		DefinitionOfDoneJSON: definition,
		MaxWorkers:           req.MaxWorkers,
		SupervisorAgente:     canonicalAutonomyCodexName(req.SupervisorAgente),
		ReviewerAgente:       canonicalAutonomyCodexName(req.ReviewerAgente),
		ReserveReviewer:      true,
		ReserveSupervisor:    true,
		ReviewRequired:       true,
		AutoCreateTasks:      true,
		AutoCloseProject:     true,
		EstadoAutonomia:      supervisionapp.AutonomiaProyectoActiva,
	}
}

func upsertRepoPersistentAutonomy(proyecto *db.Proyecto, req apiRepoMejorarRequest) (*supervisionapp.Policy, error) {
	input := buildRepoPersistentAutonomyPolicy(proyecto, req)
	if input == nil {
		return nil, nil
	}
	policy, err := supervisionService.UpsertProjectPolicy(strings.TrimSpace(proyecto.Slug), *input)
	if err != nil {
		return nil, err
	}
	if err := bootstrapPersistentAutonomyProject(strings.TrimSpace(proyecto.Slug), policy, "repo_mejorar_persistente"); err != nil {
		return nil, err
	}
	policy, err = supervisionService.GetProjectPolicy(strings.TrimSpace(proyecto.Slug))
	if err != nil {
		return nil, err
	}
	return policy, nil
}

func normalizeRepoFunctionForkSpec(spec *apiRepoFunctionForkSpec) *apiRepoFunctionForkSpec {
	if spec == nil {
		return nil
	}
	out := &apiRepoFunctionForkSpec{
		FuncionObjetivo:       strings.TrimSpace(spec.FuncionObjetivo),
		WriteSet:              trimNonEmptyStrings(spec.WriteSet),
		ModelosCandidatos:     trimNonEmptyStrings(spec.ModelosCandidatos),
		PreservarArquitectura: spec.PreservarArquitectura,
		Materia:               db.NormalizarMateriaScoreAgente(spec.Materia),
		ForkLines:             max(0, spec.ForkLines),
		SelectedModels:        trimNonEmptyStrings(spec.SelectedModels),
		DecisionMode:          strings.TrimSpace(spec.DecisionMode),
		DecisionReason:        strings.TrimSpace(spec.DecisionReason),
	}
	if out.FuncionObjetivo == "" && len(out.WriteSet) == 0 && len(out.ModelosCandidatos) == 0 && !out.PreservarArquitectura && out.Materia == "" && out.ForkLines == 0 && len(out.SelectedModels) == 0 && out.DecisionMode == "" && out.DecisionReason == "" {
		return nil
	}
	return out
}

func parseRepoFunctionForkSpecFromNotes(notas string) *apiRepoFunctionForkSpec {
	notas = strings.TrimSpace(notas)
	if notas == "" || (!strings.Contains(notas, "fork_funcion_v1:") && !strings.Contains(notas, "fork_funcion:")) {
		return nil
	}
	spec := &apiRepoFunctionForkSpec{}
	lines := strings.Split(notas, "\n")
	inBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "fork_funcion_v1:" || trimmed == "fork_funcion:" {
			inBlock = true
			continue
		}
		if !inBlock {
			continue
		}
		if trimmed == "" {
			continue
		}
		switch {
		case strings.HasPrefix(trimmed, "funcion_objetivo:"):
			spec.FuncionObjetivo = strings.TrimSpace(strings.TrimPrefix(trimmed, "funcion_objetivo:"))
		case strings.HasPrefix(trimmed, "write_set:"):
			spec.WriteSet = splitCSV(strings.TrimSpace(strings.TrimPrefix(trimmed, "write_set:")))
		case strings.HasPrefix(trimmed, "modelos_candidatos:"):
			spec.ModelosCandidatos = splitCSV(strings.TrimSpace(strings.TrimPrefix(trimmed, "modelos_candidatos:")))
		case strings.HasPrefix(trimmed, "materia:"):
			spec.Materia = strings.TrimSpace(strings.TrimPrefix(trimmed, "materia:"))
		case strings.HasPrefix(trimmed, "fork_lines:"):
			fmt.Sscanf(strings.TrimSpace(strings.TrimPrefix(trimmed, "fork_lines:")), "%d", &spec.ForkLines)
		case strings.HasPrefix(trimmed, "selected_models:"):
			spec.SelectedModels = splitCSV(strings.TrimSpace(strings.TrimPrefix(trimmed, "selected_models:")))
		case strings.HasPrefix(trimmed, "decision_mode:"):
			spec.DecisionMode = strings.TrimSpace(strings.TrimPrefix(trimmed, "decision_mode:"))
		case strings.HasPrefix(trimmed, "decision_reason:"):
			spec.DecisionReason = strings.TrimSpace(strings.TrimPrefix(trimmed, "decision_reason:"))
		case strings.HasPrefix(trimmed, "preservar_arquitectura:"):
			spec.PreservarArquitectura = strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(trimmed, "preservar_arquitectura:")), "true")
		default:
			if strings.HasSuffix(trimmed, ":") {
				inBlock = false
			}
		}
	}
	return normalizeRepoFunctionForkSpec(spec)
}

func planRepoFunctionForkSpec(req apiRepoMejorarRequest) *apiRepoFunctionForkSpec {
	base := normalizeRepoFunctionForkSpec(&apiRepoFunctionForkSpec{
		FuncionObjetivo:       req.FuncionObjetivo,
		WriteSet:              req.WriteSet,
		ModelosCandidatos:     req.ModelosCandidatos,
		PreservarArquitectura: req.PreservarArquitectura,
	})
	if base == nil {
		return nil
	}
	base.Materia = inferRepoFunctionForkMateria(req, base)
	base.ForkLines = decideRepoFunctionForkLines(req, base)
	base.SelectedModels = selectRepoFunctionForkModels(base)
	base.DecisionMode = "auto"
	base.DecisionReason = describeRepoFunctionForkDecision(base)
	return normalizeRepoFunctionForkSpec(base)
}

func inferRepoFunctionForkMateria(req apiRepoMejorarRequest, spec *apiRepoFunctionForkSpec) string {
	spec = normalizeRepoFunctionForkSpec(spec)
	if spec == nil {
		return ""
	}
	hay := strings.ToLower(strings.Join([]string{
		strings.TrimSpace(req.Titulo),
		strings.TrimSpace(req.Descripcion),
		strings.TrimSpace(req.Modulo),
		spec.FuncionObjetivo,
		strings.Join(spec.WriteSet, " "),
	}, " "))
	hayPad := " " + strings.NewReplacer("/", " ", "\\", " ", "-", " ", "_", " ", ".", " ", ",", " ", ":", " ").Replace(hay) + " "
	switch {
	case strings.Contains(hay, "web/"), strings.Contains(hay, "/web"), strings.Contains(hay, "frontend"), strings.Contains(hay, "component"), strings.Contains(hay, "css"), strings.Contains(hayPad, " ui "), strings.Contains(hayPad, " ux "):
		return "frontend"
	case strings.Contains(hayPad, " doc "), strings.Contains(hayPad, " docs "), strings.Contains(hay, "readme"), strings.Contains(hay, "manual"), strings.Contains(hay, "guia"), strings.Contains(hay, "guía"):
		return "documentacion"
	case strings.Contains(hay, "infra"), strings.Contains(hay, "docker"), strings.Contains(hay, "k8s"), strings.Contains(hay, "kubernetes"), strings.Contains(hay, "terraform"), strings.Contains(hay, "ansible"), strings.Contains(hay, "helm"):
		return "infraestructura"
	case strings.Contains(hay, "review"), strings.Contains(hay, "revision"), strings.Contains(hay, "revisión"):
		return "revision"
	case strings.Contains(hay, "debug"), strings.Contains(hay, "depur"), strings.Contains(hay, "fix"), strings.Contains(hay, "bug"):
		return "depuracion"
	case strings.Contains(hay, "brainstorm"), strings.Contains(hay, "ideacion"), strings.Contains(hay, "ideación"), strings.Contains(hay, "alternativa"):
		return "brainstorming"
	case spec.PreservarArquitectura:
		return "arquitectura"
	default:
		return "codigo"
	}
}

func decideRepoFunctionForkLines(req apiRepoMejorarRequest, spec *apiRepoFunctionForkSpec) int {
	spec = normalizeRepoFunctionForkSpec(spec)
	if spec == nil {
		return 0
	}
	lines := 1
	if spec.FuncionObjetivo != "" {
		lines = 2
	}
	if len(spec.WriteSet) >= 3 {
		lines = max(lines, 2)
	}
	if spec.PreservarArquitectura && (spec.FuncionObjetivo != "" || len(spec.WriteSet) >= 2) {
		lines = max(lines, 3)
	}
	switch db.NormalizarMateriaScoreAgente(spec.Materia) {
	case "arquitectura", "infraestructura", "seguridad", "frontend":
		lines = max(lines, 2)
	}
	if len(trimNonEmptyStrings(req.ModelosCandidatos)) == 1 {
		lines = min(lines, 1)
	}
	return min(lines, 3)
}

func selectRepoFunctionForkModels(spec *apiRepoFunctionForkSpec) []string {
	spec = normalizeRepoFunctionForkSpec(spec)
	if spec == nil || spec.ForkLines <= 0 {
		return nil
	}
	selected := appendUniqueNonEmpty(nil, rankRepoFunctionForkLocalFamilies(spec.Materia)...)
	selected = appendUniqueNonEmpty(selected, spec.ModelosCandidatos...)
	selected = appendUniqueNonEmpty(selected, canonicalRepoFunctionForkFamilies()...)
	if len(selected) > spec.ForkLines {
		selected = selected[:spec.ForkLines]
	}
	return selected
}

func describeRepoFunctionForkDecision(spec *apiRepoFunctionForkSpec) string {
	spec = normalizeRepoFunctionForkSpec(spec)
	if spec == nil {
		return ""
	}
	partes := []string{"orquesta decide fork automático"}
	if spec.Materia != "" {
		partes = append(partes, "materia="+spec.Materia)
	}
	if spec.ForkLines > 0 {
		partes = append(partes, fmt.Sprintf("lineas=%d", spec.ForkLines))
	}
	if len(spec.SelectedModels) > 0 {
		partes = append(partes, "modelos="+strings.Join(spec.SelectedModels, ", "))
	}
	if spec.PreservarArquitectura {
		partes = append(partes, "preserva arquitectura")
	}
	return strings.Join(partes, " · ")
}

func rankRepoFunctionForkLocalFamilies(materia string) []string {
	items, err := db.ListarAgenteScoresLocales(nil, nil)
	if err != nil || len(items) == 0 {
		return nil
	}
	pesos := db.PesosScoreMateriaDesdePipeline("", materia, "")
	type familyMatter struct {
		score     float64
		confianza float64
		muestras  int
		updatedAt string
	}
	familias := map[string]map[string]familyMatter{}
	for _, item := range items {
		if item == nil {
			continue
		}
		family := repoAgentFamily(item.Agente)
		if family == "" {
			continue
		}
		materiaNorm := db.NormalizarMateriaScoreAgente(item.Materia)
		if familias[family] == nil {
			familias[family] = map[string]familyMatter{}
		}
		current, ok := familias[family][materiaNorm]
		if !ok || item.ScoreTotal > current.score || (item.ScoreTotal == current.score && item.Confianza > current.confianza) || (item.ScoreTotal == current.score && item.Confianza == current.confianza && item.Muestras > current.muestras) {
			updatedAt := ""
			if !item.UpdatedAt.IsZero() {
				updatedAt = item.UpdatedAt.UTC().Format("20060102150405")
			}
			familias[family][materiaNorm] = familyMatter{
				score:     item.ScoreTotal,
				confianza: item.Confianza,
				muestras:  item.Muestras,
				updatedAt: updatedAt,
			}
		}
	}
	type ranked struct {
		family    string
		score     float64
		confianza float64
		muestras  int
		updatedAt string
	}
	var ranking []ranked
	for family, materias := range familias {
		var totalPeso, totalScore, totalConfianza float64
		totalMuestras := 0
		updatedAt := ""
		for materiaKey, peso := range pesos {
			if peso <= 0 {
				continue
			}
			item, ok := materias[db.NormalizarMateriaScoreAgente(materiaKey)]
			if !ok {
				continue
			}
			totalPeso += peso
			totalScore += peso * item.score
			totalConfianza += peso * item.confianza
			totalMuestras += item.muestras
			if item.updatedAt > updatedAt {
				updatedAt = item.updatedAt
			}
		}
		if totalPeso <= 0 {
			continue
		}
		ranking = append(ranking, ranked{
			family:    family,
			score:     totalScore / totalPeso,
			confianza: totalConfianza / totalPeso,
			muestras:  totalMuestras,
			updatedAt: updatedAt,
		})
	}
	sort.SliceStable(ranking, func(i, j int) bool {
		if ranking[i].score != ranking[j].score {
			return ranking[i].score > ranking[j].score
		}
		if ranking[i].confianza != ranking[j].confianza {
			return ranking[i].confianza > ranking[j].confianza
		}
		if ranking[i].muestras != ranking[j].muestras {
			return ranking[i].muestras > ranking[j].muestras
		}
		return ranking[i].updatedAt > ranking[j].updatedAt
	})
	out := make([]string, 0, len(ranking))
	for _, item := range ranking {
		out = append(out, item.family)
	}
	return out
}

func repoAgentFamily(agente string) string {
	agente = strings.ToLower(strings.TrimSpace(agente))
	switch {
	case strings.HasPrefix(agente, "qwen"):
		return "qwen"
	case strings.HasPrefix(agente, "gemma"):
		return "gemma"
	case strings.HasPrefix(agente, "llama"):
		return "llama"
	case strings.HasPrefix(agente, "codex"):
		return "codex"
	case strings.HasPrefix(agente, "claude"):
		return "claude"
	case strings.HasPrefix(agente, "gemini"):
		return "gemini"
	default:
		return ""
	}
}

func canonicalRepoFunctionForkFamilies() []string {
	return []string{"qwen", "gemma", "llama", "codex", "claude", "gemini"}
}

func appendUniqueNonEmpty(dst []string, items ...string) []string {
	seen := make(map[string]struct{}, len(dst))
	out := make([]string, 0, len(dst)+len(items))
	for _, item := range dst {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		key := strings.ToLower(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		key := strings.ToLower(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}

func trimNonEmptyStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

func prioridadRepoAPI(raw string) db.PrioridadTarea {
	prioridad := db.PrioridadTarea(strings.TrimSpace(raw))
	if prioridad == "" {
		return db.PrioridadMedia
	}
	return prioridad
}
