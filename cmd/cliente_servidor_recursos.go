/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"orquesta/agentesapp"
	"orquesta/capacidadapp"
	"orquesta/coordinacion"
	"orquesta/db"
	"orquesta/fabricaapp"
	"orquesta/gitgobernanza"
	"orquesta/lenguajeapp"
	"orquesta/progresoapp"
)

type apiProyectoResponse struct {
	Proyecto *db.Proyecto `json:"proyecto"`
}

type apiProyectoControlResponse struct {
	Control *projectControlReport `json:"control"`
}

type apiConectorResponse struct {
	Conector *db.Conector `json:"conector"`
}

type apiConfigResponse struct {
	Config map[string]string `json:"config"`
	Clave  string            `json:"clave"`
	Valor  string            `json:"valor"`
}

type apiAsignacionesResponse struct {
	Asignaciones []*db.Asignacion `json:"asignaciones"`
}

type apiLockResponse struct {
	Lock *coordinacion.Lock `json:"lock"`
}

type apiLocksResponse struct {
	Locks []*coordinacion.Lock `json:"locks"`
}

type apiWorktreeResponse struct {
	Worktree *coordinacion.Worktree `json:"worktree"`
}

type apiWorktreesResponse struct {
	Worktrees []*coordinacion.Worktree `json:"worktrees"`
}

type apiAgentesPanelResponse struct {
	Rows []agentesapp.Row `json:"rows"`
}

type apiAgentesPanelCanonicalResponse struct {
	Agents []*agentesapp.PanelEntity `json:"agents"`
}

type apiAgenteOverviewResponse struct {
	Detail *agentesapp.Detail `json:"detail"`
}

type apiAgenteActividadResponse struct {
	Activity *agentActivityReport `json:"activity"`
}

type apiAgenteReanimationsResponse struct {
	Rows []agentesapp.ReanimationCandidate `json:"rows"`
}

type apiAgenteInvestigacionResponse struct {
	Investigacion *agentesapp.InvestigationReport `json:"investigacion"`
}

type apiGitMergesResponse struct {
	Merges []*gitgobernanza.GitMerge `json:"merges"`
}

type apiGitMergeSaveRequest struct {
	ID           int64  `json:"id"`
	ProyectoSlug string `json:"proyecto_slug"`
	SourceBranch string `json:"source_branch"`
	TargetBranch string `json:"target_branch"`
	RequestedBy  string `json:"requested_by"`
	Estado       string `json:"estado"`
	CommitOrigen string `json:"commit_origen"`
	CommitMerge  string `json:"commit_merge"`
	Notas        string `json:"notas"`
	MetadataJSON string `json:"metadata_json"`
}

type apiGitMergeSaveResponse struct {
	OK    bool                    `json:"ok"`
	ID    int64                   `json:"id"`
	Merge *gitgobernanza.GitMerge `json:"merge,omitempty"`
}

func (req apiGitMergeSaveRequest) intoInput() gitgobernanza.SaveMergeRequestInput {
	return gitgobernanza.SaveMergeRequestInput{
		ID:           req.ID,
		ProyectoSlug: strings.TrimSpace(req.ProyectoSlug),
		SourceBranch: strings.TrimSpace(req.SourceBranch),
		TargetBranch: strings.TrimSpace(req.TargetBranch),
		RequestedBy:  strings.TrimSpace(req.RequestedBy),
		Estado:       strings.TrimSpace(req.Estado),
		CommitOrigen: strings.TrimSpace(req.CommitOrigen),
		CommitMerge:  strings.TrimSpace(req.CommitMerge),
		Notas:        req.Notas,
		MetadataJSON: strings.TrimSpace(req.MetadataJSON),
	}
}

type apiAgenteHandoffResponse struct {
	ID      int64 `json:"id"`
	OrderID int64 `json:"order_id"`
}

type apiRuntimeMailboxCreateResponse struct {
	OK bool  `json:"ok"`
	ID int64 `json:"id"`
}

type apiProyectoDescubrirRequest struct {
	Ruta string `json:"ruta"`
}

type apiProyectoActualizarRequest struct {
	Slug       string `json:"slug"`
	Nombre     string `json:"nombre"`
	RutaAbs    string `json:"ruta_abs"`
	OrigenRepo string `json:"origen_repo"`
	RemoteURL  string `json:"remote_url"`
	BranchBase string `json:"branch_base"`
	Tipo       string `json:"tipo"`
	ParentID   *int64 `json:"parent_id"`
	Activo     *bool  `json:"activo"`
}

type apiProyectoFabricarAppRequest struct {
	Nombre           string   `json:"nombre"`
	Descripcion      string   `json:"descripcion"`
	ObjetivoNegocio  string   `json:"objetivo_negocio"`
	UsuariosObjetivo string   `json:"usuarios_objetivo"`
	Restricciones    string   `json:"restricciones"`
	Tipo             string   `json:"tipo"`
	Frontend         bool     `json:"frontend"`
	API              bool     `json:"api"`
	Auth             bool     `json:"auth"`
	Database         bool     `json:"db"`
	Docker           bool     `json:"docker"`
	I18n             bool     `json:"i18n"`
	Idiomas          []string `json:"idiomas"`
	Por              string   `json:"por"`

	// Plataformas
	PlatWeb      bool `json:"plat_web"`
	PlatDesktop  bool `json:"plat_desktop"`
	PlatMobile   bool `json:"plat_mobile"`
	PlatCLI      bool `json:"plat_cli"`
	PlatEmbedded bool `json:"plat_embedded"`

	// Sistemas operativos
	SOLinux   bool `json:"so_linux"`
	SOWindows bool `json:"so_windows"`
	SOmacOS   bool `json:"so_macos"`
	SOAndroid bool `json:"so_android"`
	SOiOS     bool `json:"so_ios"`

	// Compliance legal
	ComplianceRGPD          bool `json:"compliance_rgpd"`
	ComplianceENS           bool `json:"compliance_ens"`
	ComplianceLSSI          bool `json:"compliance_lssi"`
	ComplianceWCAG          bool `json:"compliance_wcag"`
	ComplianceFacturaElec   bool `json:"compliance_factura_elec"`
	ComplianceReutilizacion bool `json:"compliance_reutilizacion"`

	// Infraestructura
	CI         bool `json:"ci"`
	Kubernetes bool `json:"kubernetes"`
	Terraform  bool `json:"terraform"`
	Monitoring bool `json:"monitoring"`

	// Decisiones guiadas
	Arquitectura       string   `json:"arquitectura"`
	APIStyle           string   `json:"api_style"`
	FrontendStack      string   `json:"frontend_stack"`
	DatabaseEngine     string   `json:"db_engine"`
	AuthMode           string   `json:"auth_mode"`
	IdentityProvider   string   `json:"identity_provider"`
	TestingLevel       string   `json:"testing_level"`
	ObservabilityLevel string   `json:"observability_level"`
	DeploymentTarget   string   `json:"deployment_target"`
	ArtifactType       string   `json:"artifact_type"`
	BackgroundJobs     bool     `json:"background_jobs"`
	Notifications      bool     `json:"notifications"`
	MultiTenant        bool     `json:"multi_tenant"`
	RBAC               bool     `json:"rbac"`
	ThemeSupport       bool     `json:"theme_support"`
	BrandingProfiles   bool     `json:"branding_profiles"`
	OfflineMode        bool     `json:"offline_mode"`
	ImportExport       bool     `json:"import_export"`
	Webhooks           bool     `json:"webhooks"`
	FileUploads        bool     `json:"file_uploads"`
	Reporting          bool     `json:"reporting"`
	ServicioResidente  bool     `json:"servicio_residente"`
	Cache              bool     `json:"cache"`
	Queue              bool     `json:"queue"`
	Scheduler          bool     `json:"scheduler"`
	ObjectStorage      bool     `json:"object_storage"`
	Search             bool     `json:"search"`
	RateLimiting       bool     `json:"rate_limiting"`
	FeatureFlags       bool     `json:"feature_flags"`
	AuditTrail         bool     `json:"audit_trail"`
	Backups            bool     `json:"backups"`
	DisasterRecovery   bool     `json:"disaster_recovery"`
	Integraciones      []string `json:"integraciones"`
}

type apiProyectoFabricarAppResponse struct {
	OK      bool   `json:"ok"`
	Slug    string `json:"slug"`
	Tipo    string `json:"tipo"`
	Created int    `json:"created"`
	Backlog int    `json:"backlog"`
}

type apiProyectoFabricarAppPreviewResponse struct {
	OK    bool                       `json:"ok"`
	Tasks []fabricaapp.BlueprintTask `json:"tasks"`
}

type apiProyectoIdiomasRequest struct {
	Idiomas []string `json:"idiomas"`
	Por     string   `json:"por"`
}

type apiProyectoIdiomasResponse struct {
	OK      bool     `json:"ok"`
	Slug    string   `json:"slug"`
	Idiomas []string `json:"idiomas"`
	Created int      `json:"created"`
	Backlog int      `json:"backlog"`
}

type apiProyectoSharedContextCreateRequest struct {
	Agente      string  `json:"agente"`
	Tipo        string  `json:"tipo"`
	Titulo      string  `json:"titulo"`
	Detalle     string  `json:"detalle"`
	PayloadJSON string  `json:"payload_json"`
	Peso        float64 `json:"peso"`
	Origen      string  `json:"origen"`
	ExpiresAt   string  `json:"expires_at"`
}

type apiProyectoSharedContextResponse struct {
	Items   []*db.SharedContextItem `json:"items"`
	Summary string                  `json:"summary"`
}

type apiProyectoSharedContextMutationResponse struct {
	OK bool  `json:"ok"`
	ID int64 `json:"id"`
}

type apiProyectoOperacionResponse struct {
	Operacion *db.ProyectoOperacion `json:"operacion"`
}

type apiProyectoOperacionSetRequest struct {
	EstadoOperativo  string `json:"estado_operativo"`
	Motivo           string `json:"motivo"`
	ObjetivoPct      int    `json:"objetivo_pct"`
	MinAgentes       int    `json:"min_agentes"`
	MaxAgentes       int    `json:"max_agentes"`
	Prioridad        int    `json:"prioridad"`
	ResumeAutomatico bool   `json:"resume_automatico"`
}

type apiLenguajePoliticaResponse struct {
	Politica *lenguajeapp.LanguagePolicy `json:"politica"`
}

type apiLenguajeMatrizResponse struct {
	Matriz []*lenguajeapp.LanguageMatrixEntry `json:"matriz"`
}

type apiLenguajeResolucionResponse struct {
	Resolucion *lenguajeapp.LanguageResolution `json:"resolucion"`
}

type apiLenguajePoliticaSetRequest struct {
	Politica *lenguajeapp.LanguagePolicy `json:"politica"`
	Por      string                      `json:"por"`
}

type apiLenguajeMatrizSetRequest struct {
	Scope    string `json:"scope"`
	Selector string `json:"selector"`
	Contexto string `json:"contexto"`
	Idioma   string `json:"idioma"`
	Razon    string `json:"razon"`
	Por      string `json:"por"`
}

type apiLenguajeMatrizDeleteRequest struct {
	Scope    string `json:"scope"`
	Selector string `json:"selector"`
	Contexto string `json:"contexto"`
}

type apiAgenteRequest struct {
	Nombre    string `json:"nombre"`
	Rol       string `json:"rol"`
	Proveedor string `json:"proveedor"`
}

type apiAgenteObservarCuentaRequest struct {
	Email      string     `json:"email"`
	Usuario    string     `json:"usuario"`
	Fuente     string     `json:"fuente"`
	ObservedAt *time.Time `json:"observed_at,omitempty"`
}

type apiConfigSetRequest struct {
	Clave string `json:"clave"`
	Valor string `json:"valor"`
}

type apiConectorUpsertRequest struct {
	Slug         string `json:"slug"`
	Nombre       string `json:"nombre"`
	Transporte   string `json:"transporte"`
	Comando      string `json:"comando"`
	ArgsJSON     string `json:"args_json"`
	EnvJSON      string `json:"env_json"`
	MetadataJSON string `json:"metadata_json"`
	Activo       bool   `json:"activo"`
}

func cargarPoolsDesdeAPI(activo *bool) ([]*db.PoolCapacidadResumen, bool, error) {
	var resp apiPoolsResponse
	query := url.Values{}
	if activo != nil {
		query.Set("activo", strconv.FormatBool(*activo))
	}
	ok, err := apiGetQuery("/api/pools", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Pools, true, nil
}

func cargarPoolDetalleDesdeAPI(slug string) (*capacidadapp.PoolDetail, bool, error) {
	var resp apiPoolResponse
	ok, err := apiGet(fmt.Sprintf("/api/pools/%s", url.PathEscape(strings.TrimSpace(slug))), &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Detalle, true, nil
}

func guardarPoolPorAPI(pool *db.PoolCapacidad) (int64, bool, error) {
	if pool == nil {
		return 0, false, fmt.Errorf("pool obligatorio")
	}
	var resp apiPoolSaveResponse
	ok, err := apiPost("/api/pools", apiPoolSaveRequest{
		Slug:                strings.TrimSpace(pool.Slug),
		Proveedor:           strings.TrimSpace(pool.Proveedor),
		Runtime:             strings.TrimSpace(pool.Runtime),
		Plan:                strings.TrimSpace(pool.Plan),
		EsDePago:            pool.EsDePago,
		CapacidadTotal:      pool.CapacidadTotal,
		CapacidadReservada:  pool.CapacidadReservada,
		PermiteHijos:        pool.PermiteHijos,
		PermiteModelosMulti: pool.PermiteModelosMulti,
		PermiteSobrecoste:   pool.PermiteSobrecoste,
		PoliticaHandoff:     strings.TrimSpace(pool.PoliticaHandoff),
		FuenteTelemetria:    strings.TrimSpace(pool.FuenteTelemetria),
		MetadataJSON:        strings.TrimSpace(pool.MetadataJSON),
		Activo:              pool.Activo,
	}, &resp)
	if !ok || err != nil {
		return 0, ok, err
	}
	return resp.ID, true, nil
}

func seedInicialPoolsPorAPI() (bool, error) {
	return apiPost("/api/pools/seed-inicial", map[string]any{}, nil)
}

func cargarModelosPoolDesdeAPI(slug string) ([]*db.PoolModelo, bool, error) {
	var resp apiPoolModelosResponse
	ok, err := apiGet(fmt.Sprintf("/api/pools/%s/modelos", url.PathEscape(strings.TrimSpace(slug))), &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Modelos, true, nil
}

func guardarModeloPoolPorAPI(poolSlug string, modelo *db.PoolModelo) (int64, bool, error) {
	if modelo == nil {
		return 0, false, fmt.Errorf("modelo obligatorio")
	}
	var resp apiPoolSaveResponse
	ok, err := apiPost(fmt.Sprintf("/api/pools/%s/modelos", url.PathEscape(strings.TrimSpace(poolSlug))), apiPoolModeloSaveRequest{
		ModelSlug:          strings.TrimSpace(modelo.ModelSlug),
		Activo:             modelo.Activo,
		Prioridad:          modelo.Prioridad,
		CosteRelativo:      modelo.CosteRelativo,
		LimiteConocidoJSON: strings.TrimSpace(modelo.LimiteConocidoJSON),
	}, &resp)
	if !ok || err != nil {
		return 0, ok, err
	}
	return resp.ID, true, nil
}

func seedInicialModelosPoolPorAPI() (bool, error) {
	return apiPost("/api/pools/modelos/seed-inicial", map[string]any{}, nil)
}

func cargarPoolLocalCompartidoDesdeAPI(slug string) (*capacidadapp.PoolLocalCompartido, bool, error) {
	var resp apiPoolLocalCompartidoResponse
	ok, err := apiGet(fmt.Sprintf("/api/pools/%s/local", url.PathEscape(strings.TrimSpace(slug))), &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.PoolLocal, true, nil
}

func asegurarPoolLocalCompartidoPorAPI(entrada capacidadapp.EntradaAsegurarPoolLocalCompartido) (*capacidadapp.PoolLocalCompartido, bool, error) {
	var resp apiPoolLocalCompartidoResponse
	ok, err := apiPost("/api/pools/local-compartido", apiPoolLocalCompartidoSaveRequest{
		PoolSlug:               strings.TrimSpace(entrada.PoolSlug),
		Proveedor:              strings.TrimSpace(entrada.Proveedor),
		Runtime:                strings.TrimSpace(entrada.Runtime),
		ModeloPreferente:       strings.TrimSpace(entrada.ModeloPreferente),
		SlotsMaximos:           entrada.SlotsMaximos,
		ConectorCanonico:       strings.TrimSpace(entrada.ConectorCanonico),
		ConectorCompatibilidad: strings.TrimSpace(entrada.ConectorCompatibilidad),
		ExperimentalCompat:     entrada.ExperimentalCompat,
		Perfiles:               entrada.Perfiles,
	}, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.PoolLocal, true, nil
}

func cargarPoliticasModeloDesdeAPI(scopeTipo, scopeRef string, activa *bool) ([]*db.PoliticaModelo, bool, error) {
	var resp apiPoliticasModeloResponse
	query := url.Values{}
	if strings.TrimSpace(scopeTipo) != "" {
		query.Set("scope_tipo", strings.TrimSpace(scopeTipo))
	}
	if strings.TrimSpace(scopeRef) != "" {
		query.Set("scope_ref", strings.TrimSpace(scopeRef))
	}
	if activa != nil {
		query.Set("activa", strconv.FormatBool(*activa))
	}
	ok, err := apiGetQuery("/api/politicas-modelo", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Politicas, true, nil
}

func guardarPoliticaModeloPorAPI(policy *db.PoliticaModelo) (int64, bool, error) {
	if policy == nil {
		return 0, false, fmt.Errorf("politica obligatoria")
	}
	var resp apiPoliticaModeloSaveResponse
	ok, err := apiPost("/api/politicas-modelo", apiPoliticaModeloSaveRequest{
		ScopeTipo:       strings.TrimSpace(policy.ScopeTipo),
		ScopeRef:        strings.TrimSpace(policy.ScopeRef),
		PerfilTarea:     strings.TrimSpace(policy.PerfilTarea),
		PoolSlug:        strings.TrimSpace(policy.PoolSlug),
		ModelSlug:       strings.TrimSpace(policy.ModelSlug),
		ReasoningEffort: strings.TrimSpace(policy.ReasoningEffort),
		Prioridad:       policy.Prioridad,
		Activa:          policy.Activa,
		MetadataJSON:    strings.TrimSpace(policy.MetadataJSON),
	}, &resp)
	if !ok || err != nil {
		return 0, ok, err
	}
	return resp.ID, true, nil
}

func seedInicialPoliticasModeloPorAPI() (bool, error) {
	return apiPost("/api/politicas-modelo/seed-inicial", map[string]any{}, nil)
}

func resolverModeloPorAPI(input db.ResolverPoliticaInput) (*db.ResolucionModelo, bool, error) {
	var resp apiResolucionModeloResponse
	query := url.Values{}
	if input.TareaID != nil && *input.TareaID > 0 {
		query.Set("tarea_id", strconv.FormatInt(*input.TareaID, 10))
	}
	if strings.TrimSpace(input.ProyectoSlug) != "" {
		query.Set("proyecto", strings.TrimSpace(input.ProyectoSlug))
	}
	if strings.TrimSpace(input.Fase) != "" {
		query.Set("fase", strings.TrimSpace(input.Fase))
	}
	if strings.TrimSpace(input.PerfilTarea) != "" {
		query.Set("perfil", strings.TrimSpace(input.PerfilTarea))
	}
	ok, err := apiGetQuery("/api/modelo/resolver", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Resolucion, true, nil
}

func cargarPipelineLocalDeterministaDesdeAPI(proyecto string) (*capacidadapp.PipelineLocalDeterminista, bool, error) {
	var resp apiPipelineLocalDeterministaResponse
	query := url.Values{}
	if strings.TrimSpace(proyecto) != "" {
		query.Set("proyecto", strings.TrimSpace(proyecto))
	}
	ok, err := apiGetQuery("/api/modelo/pipeline-local", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Pipeline, true, nil
}

func calcularSiguientePasoPipelineLocalDeterministaDesdeAPI(proyecto string) (*capacidadapp.PasoPipelineLocalDeterminista, bool, error) {
	var resp apiPasoPipelineLocalDeterministaResponse
	query := url.Values{}
	if strings.TrimSpace(proyecto) != "" {
		query.Set("proyecto", strings.TrimSpace(proyecto))
	}
	ok, err := apiGetQuery("/api/modelo/pipeline-local/paso", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Paso, true, nil
}

func ejecutarSiguientePasoPipelineLocalDeterministaDesdeAPI(proyecto string) (*capacidadapp.ResultadoEjecucionPasoPipelineLocal, bool, error) {
	var resp apiEjecutarPasoPipelineLocalDeterministaResponse
	query := url.Values{}
	if strings.TrimSpace(proyecto) != "" {
		query.Set("proyecto", strings.TrimSpace(proyecto))
	}
	path := "/api/modelo/pipeline-local/ejecutar"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	ok, err := apiPost(path, map[string]any{}, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Resultado, true, nil
}

func despacharSiguientePasoPipelineLocalDeterministaDesdeAPI(proyecto string) (*capacidadapp.ResultadoEjecucionPasoPipelineLocal, bool, error) {
	var resp apiEjecutarPasoPipelineLocalDeterministaResponse
	query := url.Values{}
	if strings.TrimSpace(proyecto) != "" {
		query.Set("proyecto", strings.TrimSpace(proyecto))
	}
	path := "/api/modelo/pipeline-local/despachar"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	ok, err := apiPost(path, map[string]any{}, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Resultado, true, nil
}

func cargarModelosRuntimeActivosDesdeAPI() ([]capacidadapp.ModeloRuntimeActivo, bool, error) {
	var resp apiModelosRuntimeActivosResponse
	ok, err := apiGet("/api/modelo/runtime-activos", &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Modelos, true, nil
}

func descargarModelosRuntimeActivosPorAPI() ([]string, bool, error) {
	var resp apiModelosRuntimeDescargarResponse
	ok, err := apiPost("/api/modelo/runtime-descargar", map[string]any{}, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Descargados, true, nil
}

func cargarResumenProgresoDesdeAPI(proyecto string) (*progresoapp.ResumenProgresoProyecto, bool, error) {
	var resp apiProgresoResumenResponse
	query := url.Values{}
	query.Set("proyecto", strings.TrimSpace(proyecto))
	ok, err := apiGetQuery("/api/progreso", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Resumen, true, nil
}

func cargarFasesProgresoDesdeAPI(proyecto string) ([]*progresoapp.FaseProyecto, bool, error) {
	var resp apiProgresoFasesResponse
	query := url.Values{}
	query.Set("proyecto", strings.TrimSpace(proyecto))
	ok, err := apiGetQuery("/api/progreso/fases", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Fases, true, nil
}

func registrarFaseProgresoPorAPI(proyecto string, req apiProgresoFaseRegistrarRequest) (*apiProgresoFaseResponse, bool, error) {
	req.Proyecto = strings.TrimSpace(proyecto)
	var resp apiProgresoFaseResponse
	ok, err := apiPost("/api/progreso/fases", req, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}

func actualizarFaseProgresoPorAPI(id int64, req apiProgresoFaseActualizarRequest) (*apiProgresoFaseResponse, bool, error) {
	var resp apiProgresoFaseResponse
	ok, err := apiPost(fmt.Sprintf("/api/progreso/fases/%d", id), req, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}

func registrarAvanceProgresoPorAPI(tareaID int64, req apiProgresoTareaRegistrarRequest) (bool, error) {
	ok, err := apiPost(fmt.Sprintf("/api/progreso/tareas/%d", tareaID), req, nil)
	return ok, err
}

func cargarPresupuestoSesionDesdeAPI(sesionID int64, agente string) (*apiSesionPresupuestoResponse, bool, error) {
	var resp apiSesionPresupuestoResponse
	query := url.Values{}
	if sesionID > 0 {
		query.Set("sesion", strconv.FormatInt(sesionID, 10))
	}
	if strings.TrimSpace(agente) != "" {
		query.Set("agente", strings.TrimSpace(agente))
	}
	ok, err := apiGetQuery("/api/sesiones/presupuesto", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}

func registrarPresupuestoSesionPorAPI(req apiSesionPresupuestoRequest) (*apiSesionPresupuestoResponse, bool, error) {
	var resp apiSesionPresupuestoResponse
	ok, err := apiPost("/api/sesiones/presupuesto", req, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}

func cargarConfigDesdeAPI(clave string) (*apiConfigResponse, bool, error) {
	var resp apiConfigResponse
	if clave == "" {
		ok, err := apiGet("/api/config", &resp)
		if !ok || err != nil {
			return nil, ok, err
		}
		return &resp, true, nil
	}
	query := url.Values{}
	query.Set("clave", clave)
	ok, err := apiGetQuery("/api/config", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}

func configurarValorPorAPI(clave, valor string) (bool, error) {
	ok, err := apiPost("/api/config", apiConfigSetRequest{Clave: clave, Valor: valor}, nil)
	return ok, err
}

func cargarProyectosDesdeAPI(lite bool) ([]*db.Proyecto, bool, error) {
	var resp apiProyectosResponse
	query := url.Values{}
	if lite {
		query.Set("lite", "1")
	}
	ok, err := apiGetQuery("/api/proyectos", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Proyectos, true, nil
}

func descubrirProyectosPorAPI(ruta string) ([]*db.Proyecto, bool, error) {
	var resp apiProyectosResponse
	ok, err := apiPost("/api/proyectos/descubrir", apiProyectoDescubrirRequest{Ruta: ruta}, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Proyectos, true, nil
}

func cargarProyectoDesdeAPI(ref string) (*db.Proyecto, bool, error) {
	var resp apiProyectoResponse
	ok, err := apiGet(fmt.Sprintf("/api/proyectos/%s", url.PathEscape(ref)), &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Proyecto, true, nil
}

func cargarProyectoControlDesdeAPI(ref string, since time.Time) (*projectControlReport, bool, error) {
	var resp apiProyectoControlResponse
	query := url.Values{}
	if !since.IsZero() {
		query.Set("desde", since.UTC().Format(time.RFC3339))
	}
	ok, err := apiGetQuery(fmt.Sprintf("/api/proyectos/%s/control", url.PathEscape(strings.TrimSpace(ref))), query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Control, true, nil
}

func cargarWorkspaceControlDesdeAPI(since time.Time) (*workspaceControlReport, bool, error) {
	var resp apiWorkspaceControlResponse
	query := url.Values{}
	if !since.IsZero() {
		query.Set("desde", since.UTC().Format(time.RFC3339))
	}
	ok, err := apiGetQuery("/api/workspace/control", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Control, true, nil
}

func cargarAgenteActividadDesdeAPI(agent, project string, since time.Time) (*agentActivityReport, bool, error) {
	var resp apiAgenteActividadResponse
	query := url.Values{}
	if !since.IsZero() {
		query.Set("desde", since.UTC().Format(time.RFC3339))
	}
	if strings.TrimSpace(project) != "" {
		query.Set("proyecto", strings.TrimSpace(project))
	}
	ok, err := apiGetQuery(fmt.Sprintf("/api/agentes/%s/actividad", url.PathEscape(strings.TrimSpace(agent))), query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Activity, true, nil
}

func fusionarProyectoPorAPI(origen, destino string, archivarOrigen bool) (*db.FusionProyectosResultado, bool, error) {
	var resp apiProyectoFusionResponse
	ok, err := apiPost(
		fmt.Sprintf("/api/proyectos/%s/fusionar", url.PathEscape(strings.TrimSpace(destino))),
		apiProyectoFusionRequest{
			Origen:         strings.TrimSpace(origen),
			ArchivarOrigen: archivarOrigen,
		},
		&resp,
	)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Resultado, true, nil
}

func actualizarProyectoPorAPI(ref string, req apiProyectoActualizarRequest) (*db.Proyecto, bool, error) {
	var resp apiProyectoResponse
	ok, err := apiPost(fmt.Sprintf("/api/proyectos/%s", url.PathEscape(strings.TrimSpace(ref))), req, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Proyecto, true, nil
}

func activarMicrocicloProyectoPorAPI(ref string, req apiProyectoMicrocicloRequest) (*proyectoMicrocicloResult, bool, error) {
	var resp apiProyectoMicrocicloResponse
	ok, err := apiPost(fmt.Sprintf("/api/proyectos/%s/autonomia/microciclo", url.PathEscape(strings.TrimSpace(ref))), req, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Resultado, true, nil
}

func fabricarAppProyectoPorAPI(ref string, req apiProyectoFabricarAppRequest) (*apiProyectoFabricarAppResponse, bool, error) {
	var resp apiProyectoFabricarAppResponse
	ok, err := apiPost(fmt.Sprintf("/api/proyectos/%s/fabricar-app", url.PathEscape(strings.TrimSpace(ref))), req, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}

func listarContextoCompartidoProyectoPorAPI(ref, agente, tipo string, limit int) (*apiProyectoSharedContextResponse, bool, error) {
	var resp apiProyectoSharedContextResponse
	params := url.Values{}
	if strings.TrimSpace(agente) != "" {
		params.Set("agente", strings.TrimSpace(agente))
	}
	if strings.TrimSpace(tipo) != "" {
		params.Set("tipo", strings.TrimSpace(tipo))
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	ok, err := apiGetQuery(fmt.Sprintf("/api/proyectos/%s/contexto-compartido", url.PathEscape(strings.TrimSpace(ref))), params, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}

func anotarContextoCompartidoProyectoPorAPI(ref string, req apiProyectoSharedContextCreateRequest) (*apiProyectoSharedContextMutationResponse, bool, error) {
	var resp apiProyectoSharedContextMutationResponse
	ok, err := apiPost(fmt.Sprintf("/api/proyectos/%s/contexto-compartido", url.PathEscape(strings.TrimSpace(ref))), req, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}

func cargarReglasDesdeAPI(rol string, activa *bool) ([]*db.Regla, bool, error) {
	var resp apiReglasResponse
	query := url.Values{}
	if strings.TrimSpace(rol) != "" {
		query.Set("rol", strings.TrimSpace(rol))
	}
	if activa != nil {
		query.Set("activa", fmt.Sprintf("%t", *activa))
	}
	ok, err := apiGetQuery("/api/reglas", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Reglas, true, nil
}

func crearReglaPorAPI(req apiReglaCrearRequest) (int64, bool, error) {
	var resp apiCatalogoMutationResponse
	ok, err := apiPost("/api/reglas", req, &resp)
	if !ok || err != nil {
		return 0, ok, err
	}
	return resp.ID, true, nil
}

func cargarReglaDesdeAPI(id int64) (*db.Regla, bool, error) {
	var resp apiReglaResponse
	ok, err := apiGet(fmt.Sprintf("/api/reglas/%d", id), &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Regla, true, nil
}

func actualizarReglaPorAPI(id int64, req apiReglaActualizarRequest) (bool, error) {
	return apiPost(fmt.Sprintf("/api/reglas/%d", id), req, nil)
}

func setReglaActivaPorAPI(id int64, actor string, activa bool) (bool, error) {
	return apiPost(fmt.Sprintf("/api/reglas/%d/activa", id), apiCatalogoActivacionRequest{
		Actor:  strings.TrimSpace(actor),
		Activa: activa,
	}, nil)
}

func cargarVersionesReglaDesdeAPI(id int64) ([]*db.ReglaVersion, bool, error) {
	var resp apiReglaVersionesResponse
	ok, err := apiGet(fmt.Sprintf("/api/reglas/%d/versiones", id), &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Versiones, true, nil
}

func cargarSkillsDesdeAPI(rol string, activa *bool) ([]*db.Skill, bool, error) {
	var resp apiSkillsResponse
	query := url.Values{}
	if strings.TrimSpace(rol) != "" {
		query.Set("rol", strings.TrimSpace(rol))
	}
	if activa != nil {
		query.Set("activa", fmt.Sprintf("%t", *activa))
	}
	ok, err := apiGetQuery("/api/skills", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Skills, true, nil
}

func crearSkillPorAPI(req apiSkillCrearRequest) (int64, bool, error) {
	var resp apiCatalogoMutationResponse
	ok, err := apiPost("/api/skills", req, &resp)
	if !ok || err != nil {
		return 0, ok, err
	}
	return resp.ID, true, nil
}

func cargarSkillDesdeAPI(id int64) (*db.Skill, bool, error) {
	var resp apiSkillResponse
	ok, err := apiGet(fmt.Sprintf("/api/skills/%d", id), &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Skill, true, nil
}

func actualizarSkillPorAPI(id int64, req apiSkillActualizarRequest) (bool, error) {
	return apiPost(fmt.Sprintf("/api/skills/%d", id), req, nil)
}

func setSkillActivoPorAPI(id int64, actor string, activa bool) (bool, error) {
	return apiPost(fmt.Sprintf("/api/skills/%d/activa", id), apiCatalogoActivacionRequest{
		Actor:  strings.TrimSpace(actor),
		Activa: activa,
	}, nil)
}

func cargarVersionesSkillDesdeAPI(id int64) ([]*db.SkillVersion, bool, error) {
	var resp apiSkillVersionesResponse
	ok, err := apiGet(fmt.Sprintf("/api/skills/%d/versiones", id), &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Versiones, true, nil
}

func detectarCarenciaSkillPorAPI(req apiSkillDeteccionRequest) (*apiSkillDeteccionResponse, bool, error) {
	var resp apiSkillDeteccionResponse
	ok, err := apiPost("/api/skills/detectar-carencia", req, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}

func cargarWorkflowsDesdeAPI(rol string, activo *bool) ([]*db.Workflow, bool, error) {
	var resp apiWorkflowsResponse
	query := url.Values{}
	if strings.TrimSpace(rol) != "" {
		query.Set("rol", strings.TrimSpace(rol))
	}
	if activo != nil {
		query.Set("activa", fmt.Sprintf("%t", *activo))
	}
	ok, err := apiGetQuery("/api/workflows", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Workflows, true, nil
}

func crearWorkflowPorAPI(req apiWorkflowCrearRequest) (int64, bool, error) {
	var resp apiCatalogoMutationResponse
	ok, err := apiPost("/api/workflows", req, &resp)
	if !ok || err != nil {
		return 0, ok, err
	}
	return resp.ID, true, nil
}

func cargarWorkflowDesdeAPI(id int64) (*db.Workflow, bool, error) {
	var resp apiWorkflowResponse
	ok, err := apiGet(fmt.Sprintf("/api/workflows/%d", id), &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Workflow, true, nil
}

func actualizarWorkflowPorAPI(id int64, req apiWorkflowActualizarRequest) (bool, error) {
	return apiPost(fmt.Sprintf("/api/workflows/%d", id), req, nil)
}

func setWorkflowActivoPorAPI(id int64, actor string, activa bool) (bool, error) {
	return apiPost(fmt.Sprintf("/api/workflows/%d/activa", id), apiCatalogoActivacionRequest{
		Actor:  strings.TrimSpace(actor),
		Activa: activa,
	}, nil)
}

func cargarVersionesWorkflowDesdeAPI(id int64) ([]*db.WorkflowVersion, bool, error) {
	var resp apiWorkflowVersionesResponse
	ok, err := apiGet(fmt.Sprintf("/api/workflows/%d/versiones", id), &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Versiones, true, nil
}

func cargarPermisosCatalogoDesdeAPI(entidad string) ([]*db.PermisoEdicionCatalogo, bool, error) {
	var resp apiPermisosCatalogoResponse
	query := url.Values{}
	if strings.TrimSpace(entidad) != "" {
		query.Set("entidad", strings.TrimSpace(entidad))
	}
	ok, err := apiGetQuery("/api/permisos-catalogo", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Permisos, true, nil
}

func cargarGobernanzaCatalogoDesdeAPI(tipoAgente, proyecto, agente string) (*db.GovernanceCatalog, bool, error) {
	var resp apiGovernanceCatalogResponse
	query := url.Values{}
	if strings.TrimSpace(tipoAgente) != "" {
		query.Set("tipo_agente", strings.TrimSpace(tipoAgente))
	}
	if strings.TrimSpace(proyecto) != "" {
		query.Set("proyecto", strings.TrimSpace(proyecto))
	}
	if strings.TrimSpace(agente) != "" {
		query.Set("agente", strings.TrimSpace(agente))
	}
	ok, err := apiGetQuery("/api/gobernanza/catalogo", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Catalogo, true, nil
}

func cargarGovernanceOverridesDesdeAPI(scopeTipo, scopeRef, tipoAgente, agente, entidad string) ([]*db.GovernanceOverride, bool, error) {
	var resp apiGovernanceOverridesResponse
	query := url.Values{}
	if strings.TrimSpace(scopeTipo) != "" {
		query.Set("scope_tipo", strings.TrimSpace(scopeTipo))
	}
	if strings.TrimSpace(scopeRef) != "" {
		query.Set("scope_ref", strings.TrimSpace(scopeRef))
	}
	if strings.TrimSpace(tipoAgente) != "" {
		query.Set("tipo_agente", strings.TrimSpace(tipoAgente))
	}
	if strings.TrimSpace(agente) != "" {
		query.Set("agente", strings.TrimSpace(agente))
	}
	if strings.TrimSpace(entidad) != "" {
		query.Set("entidad", strings.TrimSpace(entidad))
	}
	ok, err := apiGetQuery("/api/gobernanza/overrides", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Overrides, true, nil
}

func guardarGovernanceOverridePorAPI(req apiGovernanceOverrideSaveRequest) (int64, bool, error) {
	var resp apiCatalogoMutationResponse
	ok, err := apiPost("/api/gobernanza/overrides", req, &resp)
	if !ok || err != nil {
		return 0, ok, err
	}
	return resp.ID, true, nil
}

func guardarPermisoCatalogoPorAPI(req apiPermisoCatalogoSetRequest) (bool, error) {
	return apiPost("/api/permisos-catalogo", req, nil)
}

func cargarPoliticaLenguajeDesdeAPI() (*lenguajeapp.LanguagePolicy, bool, error) {
	var resp apiLenguajePoliticaResponse
	ok, err := apiGet("/api/lenguaje/politica", &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Politica, true, nil
}

func cargarMatrizLenguajeDesdeAPI() ([]*lenguajeapp.LanguageMatrixEntry, bool, error) {
	var resp apiLenguajeMatrizResponse
	ok, err := apiGet("/api/lenguaje/matriz", &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Matriz, true, nil
}

func resolverLenguajePorAPI(proyecto string, tareaID *int64, contexto string) (*lenguajeapp.LanguageResolution, bool, error) {
	query := url.Values{}
	if strings.TrimSpace(proyecto) != "" {
		query.Set("proyecto", strings.TrimSpace(proyecto))
	}
	if tareaID != nil && *tareaID > 0 {
		query.Set("tarea", fmt.Sprintf("%d", *tareaID))
	}
	if strings.TrimSpace(contexto) != "" {
		query.Set("contexto", strings.TrimSpace(contexto))
	}
	var resp apiLenguajeResolucionResponse
	ok, err := apiGetQuery("/api/lenguaje/resolver", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Resolucion, true, nil
}

func fijarPoliticaLenguajePorAPI(policy *lenguajeapp.LanguagePolicy, por string) (bool, error) {
	ok, err := apiPost("/api/lenguaje/politica", apiLenguajePoliticaSetRequest{
		Politica: policy,
		Por:      strings.TrimSpace(por),
	}, nil)
	return ok, err
}

func fijarEntradaMatrizLenguajePorAPI(scope, selector, contexto, idioma, razon, por string) (bool, error) {
	ok, err := apiPost("/api/lenguaje/matriz", apiLenguajeMatrizSetRequest{
		Scope:    strings.TrimSpace(scope),
		Selector: strings.TrimSpace(selector),
		Contexto: strings.TrimSpace(contexto),
		Idioma:   strings.TrimSpace(idioma),
		Razon:    strings.TrimSpace(razon),
		Por:      strings.TrimSpace(por),
	}, nil)
	return ok, err
}

func borrarEntradaMatrizLenguajePorAPI(scope, selector, contexto string) (bool, error) {
	ok, err := apiPost("/api/lenguaje/matriz/borrar", apiLenguajeMatrizDeleteRequest{
		Scope:    strings.TrimSpace(scope),
		Selector: strings.TrimSpace(selector),
		Contexto: strings.TrimSpace(contexto),
	}, nil)
	return ok, err
}

func cargarConectoresDesdeAPI() ([]*db.Conector, bool, error) {
	var resp apiConectoresResponse
	ok, err := apiGet("/api/conectores", &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Conectores, true, nil
}

func cargarConectorDesdeAPI(ref string) (*db.Conector, bool, error) {
	var resp apiConectorResponse
	ok, err := apiGet(fmt.Sprintf("/api/conectores/%s", url.PathEscape(ref)), &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Conector, true, nil
}

func registrarConectorPorAPI(req apiConectorUpsertRequest) (int64, bool, error) {
	var resp apiConectorResponse
	ok, err := apiPost("/api/conectores", req, &resp)
	if !ok || err != nil {
		return 0, ok, err
	}
	if resp.Conector == nil {
		return 0, true, fmt.Errorf("respuesta sin conector")
	}
	return resp.Conector.ID, true, nil
}

func registrarAgentePorAPI(nombre, rol string) (bool, error) {
	ok, err := apiPost("/api/agentes", apiAgenteRequest{Nombre: nombre, Rol: rol}, nil)
	return ok, err
}

func registrarAgenteAutoPorAPI(proveedor, rol string) (string, bool, error) {
	var resp map[string]any
	ok, err := apiPost("/api/agentes", apiAgenteRequest{
		Proveedor: strings.TrimSpace(proveedor),
		Rol:       strings.TrimSpace(rol),
	}, &resp)
	if !ok || err != nil {
		return "", ok, err
	}
	nombre, _ := resp["nombre"].(string)
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return "", true, fmt.Errorf("respuesta sin nombre de agente")
	}
	return nombre, true, nil
}

func listarAgentesPresupuestoPorAPI(activos bool, agente string) (*apiAgentesPresupuestoResponse, bool, error) {
	var resp apiAgentesPresupuestoResponse
	path := "/api/agentes/presupuesto"
	query := url.Values{}
	if activos {
		query.Set("activos", "true")
	}
	if strings.TrimSpace(agente) != "" {
		query.Set("agente", strings.TrimSpace(agente))
	}
	ok, err := apiGetQuery(path, query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}

func refrescarAgentesPresupuestoPorAPI(agente string) (map[string]any, bool, error) {
	var resp map[string]any
	ok, err := apiPost("/api/agentes/presupuesto/refrescar", map[string]any{
		"agente": strings.TrimSpace(agente),
	}, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp, true, nil
}

func listarAgentesCuentasPorAPI(activos bool) (*apiAgentesCuentasResponse, bool, error) {
	var resp apiAgentesCuentasResponse
	path := "/api/agentes/cuentas"
	if activos {
		path += "?activos=true"
	}
	ok, err := apiGet(path, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}

func listarAgentesRankingCuentasPorAPI(activos bool) (*apiAgentesRankingCuentasResponse, bool, error) {
	var resp apiAgentesRankingCuentasResponse
	path := "/api/agentes/ranking-cuentas"
	if activos {
		path += "?activos=true"
	}
	ok, err := apiGet(path, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}

func observarCuentaAgentePorAPI(nombre, email, usuario, fuente string, observedAt *time.Time) (bool, error) {
	ok, err := apiPost(
		fmt.Sprintf("/api/agentes/%s/observar-cuenta", url.PathEscape(strings.TrimSpace(nombre))),
		apiAgenteObservarCuentaRequest{
			Email:      strings.TrimSpace(email),
			Usuario:    strings.TrimSpace(usuario),
			Fuente:     strings.TrimSpace(fuente),
			ObservedAt: observedAt,
		},
		nil,
	)
	return ok, err
}

func retirarAgentePorAPI(nombre string) (bool, error) {
	ok, err := apiPost(fmt.Sprintf("/api/agentes/%s/retirar", url.PathEscape(nombre)), map[string]any{}, nil)
	return ok, err
}

func rehabilitarAgentePorAPI(nombre string) (bool, error) {
	ok, err := apiPost(fmt.Sprintf("/api/agentes/%s/rehabilitar", url.PathEscape(nombre)), map[string]any{}, nil)
	return ok, err
}

func eliminarAgentePorAPI(nombre string) (bool, error) {
	ok, err := apiPost(fmt.Sprintf("/api/agentes/%s/eliminar", url.PathEscape(nombre)), map[string]any{}, nil)
	return ok, err
}

func fusionarAgentePorAPI(origen, destino, destinoRespaldo, etiqueta string, retener int) (*apiAgenteFusionResponse, bool, error) {
	var resp apiAgenteFusionResponse
	ok, err := apiPost(fmt.Sprintf("/api/agentes/%s/fusionar", url.PathEscape(origen)), apiAgenteFusionRequest{
		Destino:          strings.TrimSpace(destino),
		DestinoRespaldo:  strings.TrimSpace(destinoRespaldo),
		EtiquetaRespaldo: strings.TrimSpace(etiqueta),
		Retener:          retener,
	}, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}

func resetReanimacionAgentePorAPI(nombre string) (bool, error) {
	ok, err := apiPost(fmt.Sprintf("/api/agentes/%s/reset-reanimacion", url.PathEscape(nombre)), map[string]any{}, nil)
	return ok, err
}

func pausarAgentePorAPI(nombre string, minutos int, motivo string) (bool, error) {
	return registrarPausaAgentePorAPI(nombre, minutos, motivo, "", "", "")
}

func registrarPausaAgentePorAPI(nombre string, minutos int, motivo, accion, entidad, detalle string) (bool, error) {
	ok, err := apiPost("/api/agente/pausar", apiAgentePausarRequest{
		Agente:  nombre,
		Minutos: minutos,
		Motivo:  motivo,
		Accion:  accion,
		Entidad: entidad,
		Detalle: detalle,
	}, nil)
	return ok, err
}

func crearHandoffAgentePorAPI(origen, destino string, tareaID *int64, motivo, resumen, externalSessionID string) (int64, bool, error) {
	req := apiRuntimeHandoffRequest{
		AgenteOrigen:      origen,
		AgenteDestino:     destino,
		Motivo:            motivo,
		Resumen:           resumen,
		ExternalSessionID: externalSessionID,
	}
	if tareaID != nil {
		req.TareaID = *tareaID
	}
	var resp apiAgenteHandoffResponse
	ok, err := apiPost("/api/agente/handoff", req, &resp)
	if !ok || err != nil {
		return 0, ok, err
	}
	if resp.OrderID == 0 {
		resp.OrderID = resp.ID
	}
	return resp.OrderID, true, nil
}

func adoptarContextoAgentePorAPI(req apiAgenteAdoptarContextoRequest) (*apiAgenteAdoptarContextoResponse, bool, error) {
	var resp apiAgenteAdoptarContextoResponse
	ok, err := apiPost("/api/agente/adoptar-contexto", req, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}

func investigarAgentesPorAPI(texto, proyecto string, limit int) (*agentesapp.InvestigationReport, bool, error) {
	query := url.Values{}
	query.Set("q", strings.TrimSpace(texto))
	if strings.TrimSpace(proyecto) != "" {
		query.Set("proyecto", strings.TrimSpace(proyecto))
	}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	var resp apiAgenteInvestigacionResponse
	ok, err := apiGetQuery("/api/agente/investigar", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Investigacion, true, nil
}

func cargarRuntimeMailboxDesdeAPI(query url.Values) ([]*db.RuntimeMailboxMessage, bool, error) {
	var resp apiRuntimeMailboxResponse
	ok, err := apiGetQuery("/api/runtime-mailbox", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Mailbox, true, nil
}

func crearRuntimeMailboxDesdeAPI(fromAgente, toAgente, proyecto string, runtimeOrderID int64, kind, payload string) (int64, bool, error) {
	var resp apiRuntimeMailboxCreateResponse
	ok, err := apiPost("/api/runtime-mailbox", apiRuntimeMailboxCreateRequest{
		FromAgente:     fromAgente,
		ToAgente:       toAgente,
		Proyecto:       proyecto,
		RuntimeOrderID: runtimeOrderID,
		Kind:           kind,
		Payload:        payload,
	}, &resp)
	if !ok || err != nil {
		return 0, ok, err
	}
	return resp.ID, true, nil
}

func crearRuntimeCheckpointDesdeAPI(agente, proyecto string, sesionID, runtimeID int64, checkpointKind, resumen, branch, cwd, payload, resumeStrategy, source string) (int64, bool, error) {
	var resp apiRuntimeCheckpointCreateResponse
	ok, err := apiPost("/api/runtime-checkpoints", apiRuntimeCheckpointCreateRequest{
		Agente:         agente,
		Proyecto:       proyecto,
		SesionID:       sesionID,
		RuntimeID:      runtimeID,
		CheckpointKind: checkpointKind,
		Resumen:        resumen,
		Branch:         branch,
		CWD:            cwd,
		Payload:        payload,
		ResumeStrategy: resumeStrategy,
		Source:         source,
	}, &resp)
	if !ok || err != nil {
		return 0, ok, err
	}
	return resp.ID, true, nil
}

func marcarRuntimeMailboxEntregadoPorAPI(id int64) (bool, error) {
	ok, err := apiPost(fmt.Sprintf("/api/runtime-mailbox/%d/entregar", id), map[string]any{}, nil)
	return ok, err
}

func marcarRuntimeMailboxConsumidoPorAPI(id int64) (bool, error) {
	ok, err := apiPost(fmt.Sprintf("/api/runtime-mailbox/%d/consumir", id), map[string]any{}, nil)
	return ok, err
}

func limpiarRuntimeMailboxPorAPI(toAgente, fromAgente, proyecto string, estados, kinds []string) (*apiRuntimeMailboxClearResponse, bool, error) {
	var resp apiRuntimeMailboxClearResponse
	ok, err := apiPost("/api/runtime-mailbox/limpiar", apiRuntimeMailboxClearRequest{
		ToAgente:   strings.TrimSpace(toAgente),
		FromAgente: strings.TrimSpace(fromAgente),
		Proyecto:   strings.TrimSpace(proyecto),
		Estados:    estados,
		Kinds:      kinds,
	}, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}

func despertarRuntimePorAPI(orders, mailbox, warm bool) (*apiRuntimeWakeResponse, bool, error) {
	var resp apiRuntimeWakeResponse
	ok, err := apiPost("/api/runtime/wake", apiRuntimeWakeRequest{
		Orders:  orders,
		Mailbox: mailbox,
		Warm:    warm,
	}, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}

func procesarRuntimeMailboxPorAPI(toAgente, proyecto string) (*apiRuntimeProcessMailboxResponse, bool, error) {
	var resp apiRuntimeProcessMailboxResponse
	ok, err := apiPost("/api/runtime/process-mailbox", apiRuntimeProcessMailboxRequest{
		ToAgente: strings.TrimSpace(toAgente),
		Proyecto: strings.TrimSpace(proyecto),
	}, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}

func procesarRuntimeOrdersPorAPI() (*apiRuntimeProcessOrdersResponse, bool, error) {
	var resp apiRuntimeProcessOrdersResponse
	ok, err := apiPost("/api/runtime/process-orders", map[string]any{}, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}

func procesarAutonomiaPorAPI(wait bool) (*apiRuntimeProcessAutonomiaResponse, bool, error) {
	var resp apiRuntimeProcessAutonomiaResponse
	ok, err := apiPost("/api/runtime/process-autonomia", map[string]any{"wait": wait}, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}

func procesarTranscriptPorAPI(wait bool, handleID int64, agente, proyecto string) (*apiRuntimeProcessAutonomiaResponse, bool, error) {
	var resp apiRuntimeProcessAutonomiaResponse
	ok, err := apiPost("/api/runtime/process-transcript", apiRuntimeProcessTranscriptRequest{
		Wait:     wait,
		HandleID: handleID,
		Agente:   strings.TrimSpace(agente),
		Proyecto: strings.TrimSpace(proyecto),
	}, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}

func procesarDegradadosPorAPI(wait bool) (*apiRuntimeProcessAutonomiaResponse, bool, error) {
	var resp apiRuntimeProcessAutonomiaResponse
	ok, err := apiPost("/api/runtime/process-degradados", map[string]any{"wait": wait}, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}

func purgarTranscriptRuidoPorAPI(all bool) (*apiRuntimeProcessAutonomiaResponse, bool, error) {
	var resp apiRuntimeProcessAutonomiaResponse
	ok, err := apiPost("/api/runtime/purge-transcript-noise", apiRuntimePurgeTranscriptNoiseRequest{All: all}, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}

func procesarHigienePorAPI(wait bool) (*apiRuntimeProcessAutonomiaResponse, bool, error) {
	var resp apiRuntimeProcessAutonomiaResponse
	ok, err := apiPost("/api/runtime/process-hygiene", map[string]any{"wait": wait}, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}

func procesarReanimacionesPorAPI(wait bool) (*apiRuntimeProcessReanimationsResponse, bool, error) {
	var resp apiRuntimeProcessReanimationsResponse
	ok, err := apiPost("/api/runtime/process-reanimations", map[string]any{"wait": wait}, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return &resp, true, nil
}

func cargarAsignacionesDesdeAPI(query url.Values) ([]*db.Asignacion, bool, error) {
	var resp apiAsignacionesResponse
	ok, err := apiGetQuery("/api/asignaciones", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Asignaciones, true, nil
}

func activarAsignacionPorAPI(req apiAsignacionActivarRequest) (bool, error) {
	ok, err := apiPost("/api/asignaciones/activar", req, nil)
	return ok, err
}

func cargarLocksDesdeAPI(query url.Values) ([]*coordinacion.Lock, bool, error) {
	var resp apiLocksResponse
	ok, err := apiGetQuery("/api/locks", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Locks, true, nil
}

func tomarLockPorAPI(req apiLockRequest) (*coordinacion.Lock, bool, error) {
	var resp apiLockResponse
	ok, err := apiPost("/api/locks", req, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Lock, true, nil
}

func renovarLockPorAPI(id int64, req apiLockRequest) (*coordinacion.Lock, bool, error) {
	var resp apiLockResponse
	ok, err := apiPost(fmt.Sprintf("/api/locks/%d/renovar", id), req, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Lock, true, nil
}

func liberarLockPorAPI(id int64, req apiLockRequest) (*coordinacion.Lock, bool, error) {
	var resp apiLockResponse
	ok, err := apiPost(fmt.Sprintf("/api/locks/%d/liberar", id), req, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Lock, true, nil
}

func cargarWorktreesDesdeAPI(query url.Values) ([]*coordinacion.Worktree, bool, error) {
	var resp apiWorktreesResponse
	ok, err := apiGetQuery("/api/worktrees", query, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Worktrees, true, nil
}

func crearWorktreePorAPI(req apiWorktreeRequest) (*coordinacion.Worktree, bool, error) {
	var resp apiWorktreeResponse
	ok, err := apiPost("/api/worktrees", req, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Worktree, true, nil
}

func cerrarWorktreePorAPI(id int64, req apiWorktreeRequest) (*coordinacion.Worktree, bool, error) {
	var resp apiWorktreeResponse
	ok, err := apiPost(fmt.Sprintf("/api/worktrees/%d/cerrar", id), req, &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	return resp.Worktree, true, nil
}
