/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"time"

	"orquesta/capacidadapp"
	"orquesta/db"
	"orquesta/internal/rpclocal"
	"orquesta/sesionesapp"
)

type apiAgentesResponse struct {
	Agentes []*db.Agente `json:"agentes"`
}

type apiAgentesPresupuestoResponse struct {
	Generado string       `json:"generado"`
	Activos  bool         `json:"activos"`
	Agentes  []*db.Agente `json:"agentes"`
}

type apiAgenteCuentaItem struct {
	Nombre            string     `json:"nombre"`
	Rol               string     `json:"rol,omitempty"`
	Activo            bool       `json:"activo"`
	Habilitado        bool       `json:"habilitado"`
	CuentaID          string     `json:"cuenta_id,omitempty"`
	CuentaEmail       string     `json:"cuenta_email,omitempty"`
	CuentaUsuario     string     `json:"cuenta_usuario,omitempty"`
	CuentaFuente      string     `json:"cuenta_fuente,omitempty"`
	CuentaObservadaAt *time.Time `json:"cuenta_observada_at,omitempty"`
}

type apiAgentesCuentasResponse struct {
	Generado string                `json:"generado"`
	Activos  bool                  `json:"activos"`
	Agentes  []apiAgenteCuentaItem `json:"agentes"`
}

type apiCuentaPresupuestoItem struct {
	CuentaClave           string     `json:"cuenta_clave"`
	CuentaID              string     `json:"cuenta_id,omitempty"`
	CuentaEmail           string     `json:"cuenta_email,omitempty"`
	CuentaUsuario         string     `json:"cuenta_usuario,omitempty"`
	CuentaFuente          string     `json:"cuenta_fuente,omitempty"`
	CuentaObservadaAt     *time.Time `json:"cuenta_observada_at,omitempty"`
	Agentes               []string   `json:"agentes,omitempty"`
	Criterio              string     `json:"criterio"`
	CuotaRestantePct      *int       `json:"cuota_restante_pct,omitempty"`
	PresupuestoVentana    string     `json:"presupuesto_ventana,omitempty"`
	PresupuestoResetAt    *time.Time `json:"presupuesto_reset_at,omitempty"`
	PresupuestoStale      bool       `json:"presupuesto_stale,omitempty"`
	RemainingSeconds      *int64     `json:"remaining_seconds,omitempty"`
	RemainingMessages     *int64     `json:"remaining_messages,omitempty"`
	RemainingTokens       *int64     `json:"remaining_tokens,omitempty"`
	RemainingCredits      *float64   `json:"remaining_credits,omitempty"`
	ObservedUsageTokens   *int64     `json:"observed_usage_tokens,omitempty"`
	ObservedUsageCostUSD  *float64   `json:"observed_usage_cost_usd,omitempty"`
	ObservedUsageMessages *int       `json:"observed_usage_messages,omitempty"`
	ObservedUsageTurns    *int       `json:"observed_usage_turns,omitempty"`
	ObservedUsageAt       *time.Time `json:"observed_usage_at,omitempty"`
	ObservedSessionPath   string     `json:"observed_session_path,omitempty"`
	PresupuestoFuente     string     `json:"presupuesto_fuente,omitempty"`
	PresupuestoCheckedAt  *time.Time `json:"presupuesto_checked_at,omitempty"`
}

type apiAgentesRankingCuentasResponse struct {
	Generado string                     `json:"generado"`
	Activos  bool                       `json:"activos"`
	Cuentas  []apiCuentaPresupuestoItem `json:"cuentas"`
}

type apiProyectosResponse struct {
	Proyectos []*db.Proyecto `json:"proyectos"`
}

type apiProyectoFusionResponse struct {
	Resultado *db.FusionProyectosResultado `json:"resultado"`
}

type apiConectoresResponse struct {
	Conectores []*db.Conector `json:"conectores"`
}

type apiPoolsResponse struct {
	Pools []*db.PoolCapacidadResumen `json:"pools"`
}

type apiPoolResponse struct {
	Detalle *capacidadapp.PoolDetail `json:"detalle"`
}

type apiPoolModelosResponse struct {
	Modelos []*db.PoolModelo `json:"modelos"`
}

type apiPoolSaveResponse struct {
	ID int64 `json:"id"`
}

type apiPoolLocalCompartidoResponse struct {
	PoolLocal *capacidadapp.PoolLocalCompartido `json:"pool_local"`
}

type apiPoliticasModeloResponse struct {
	Politicas []*db.PoliticaModelo `json:"politicas"`
}

type apiPoliticaModeloSaveResponse struct {
	ID int64 `json:"id"`
}

type apiResolucionModeloResponse struct {
	Resolucion *db.ResolucionModelo `json:"resolucion"`
}

type apiPipelineLocalDeterministaResponse struct {
	Pipeline *capacidadapp.PipelineLocalDeterminista `json:"pipeline"`
}

type apiPasoPipelineLocalDeterministaResponse struct {
	Paso *capacidadapp.PasoPipelineLocalDeterminista `json:"paso"`
}

type apiEjecutarPasoPipelineLocalDeterministaResponse struct {
	Resultado *capacidadapp.ResultadoEjecucionPasoPipelineLocal `json:"resultado"`
}

type apiModelosRuntimeActivosResponse struct {
	Modelos []capacidadapp.ModeloRuntimeActivo `json:"modelos"`
}

type apiModelosRuntimeDescargarResponse struct {
	OK          bool     `json:"ok"`
	Descargados []string `json:"descargados"`
}

type apiReglasResponse struct {
	Reglas []*db.Regla `json:"reglas"`
}

type apiSkillsResponse struct {
	Skills []*db.Skill `json:"skills"`
}

type apiWorkflowsResponse struct {
	Workflows []*db.Workflow `json:"workflows"`
}

type apiReglaResponse struct {
	Regla *db.Regla `json:"regla"`
}

type apiSkillResponse struct {
	Skill *db.Skill `json:"skill"`
}

type apiWorkflowResponse struct {
	Workflow *db.Workflow `json:"workflow"`
}

type apiReglaVersionesResponse struct {
	Versiones []*db.ReglaVersion `json:"versiones"`
}

type apiSkillVersionesResponse struct {
	Versiones []*db.SkillVersion `json:"versiones"`
}

type apiWorkflowVersionesResponse struct {
	Versiones []*db.WorkflowVersion `json:"versiones"`
}

type apiPermisosCatalogoResponse struct {
	Permisos []*db.PermisoEdicionCatalogo `json:"permisos"`
}

type apiCatalogoMutationResponse struct {
	ID int64 `json:"id"`
}

type apiTareasResponse struct {
	Tareas []*db.Tarea `json:"tareas"`
}

type apiTareaResponse struct {
	Tarea *db.Tarea `json:"tarea"`
}

type apiProgresoResumenResponse struct {
	Resumen *db.ResumenProgresoProyecto `json:"resumen"`
}

type apiProgresoFasesResponse struct {
	Fases []*db.FaseProyecto `json:"fases"`
}

type apiProgresoFaseResponse struct {
	ID   int64            `json:"id"`
	Fase *db.FaseProyecto `json:"fase"`
}

type apiSesionResponse struct {
	Sesion *db.Sesion `json:"sesion"`
}

type apiSesionesInspeccionResponse struct {
	Sesiones []*db.Sesion `json:"sesiones"`
}

type apiSesionPresupuestoResponse struct {
	ID          int64                              `json:"id"`
	Sesion      *db.Sesion                         `json:"sesion,omitempty"`
	Presupuesto *sesionesapp.PresupuestoSesion     `json:"presupuesto"`
	Evaluacion  *sesionesapp.EvaluacionPresupuesto `json:"evaluacion"`
}

type apiSesionInicioResponse struct {
	Sesion               *db.Sesion               `json:"sesion"`
	SesionPrevia         *db.Sesion               `json:"sesion_previa"`
	Rol                  string                   `json:"rol"`
	PropuestasPendientes []*sesionesapp.Propuesta `json:"propuestas_pendientes"`
	Reglas               []*sesionesapp.Regla     `json:"reglas"`
	Skills               []*sesionesapp.Skill     `json:"skills"`
	Workflow             *sesionesapp.Workflow    `json:"workflow"`
}

type apiPropuestasResponse struct {
	Propuestas []*db.Propuesta `json:"propuestas"`
}

type apiPropuestaDetalleResponse struct {
	Propuesta   *db.Propuesta          `json:"propuesta"`
	ConteoVotos apiConteoVotosResponse `json:"conteo_votos"`
}

type apiAuditResponse struct {
	Audit []db.AuditEntry `json:"audit"`
}

type apiRespaldoBDResponse struct {
	OK   bool   `json:"ok"`
	Ruta string `json:"ruta"`
}

type apiPersistenciaVerificacionResponse struct {
	Informe *db.InformePersistencia `json:"informe"`
}

type apiDiagnosticoResponse struct {
	Diagnostico db.SnapshotDiagnostico `json:"diagnostico"`
}

type apiRuntimesResponse struct {
	Runtimes []*db.RuntimeInstance `json:"runtimes"`
}

type apiRuntimeTreeNode struct {
	Runtime *db.RuntimeInstance   `json:"runtime"`
	Hijos   []*apiRuntimeTreeNode `json:"hijos"`
}

type apiRuntimeTreeResponse struct {
	Runtimes []*apiRuntimeTreeNode `json:"runtimes"`
}

type apiRuntimeDetailResponse struct {
	Runtime *db.RuntimeInstance          `json:"runtime"`
	Samples []*db.RuntimeTelemetrySample `json:"samples"`
}

type apiRuntimeHandlesResponse struct {
	Handles []*db.RuntimeHandle `json:"handles"`
}

type apiRuntimeHandlesPurgeResponse struct {
	OK         bool     `json:"ok"`
	Deleted    int      `json:"deleted"`
	DeletedIDs []int64  `json:"deleted_ids"`
	Estados    []string `json:"estados"`
}

type apiRuntimeHandleResidualCloseResponse struct {
	OK     bool `json:"ok"`
	Closed bool `json:"closed"`
}

type apiRuntimeOrdersPurgeResponse struct {
	OK         bool     `json:"ok"`
	Deleted    int      `json:"deleted"`
	DeletedIDs []int64  `json:"deleted_ids"`
	Estados    []string `json:"estados"`
	Tipos      []string `json:"tipos,omitempty"`
}

type apiRuntimeResidualCloseResponse struct {
	OK        bool    `json:"ok"`
	Closed    int     `json:"closed"`
	ClosedIDs []int64 `json:"closed_ids"`
}

type apiRuntimeTranscriptResponse struct {
	Transcript []*db.RuntimeTranscriptEntry `json:"transcript"`
}

type apiRuntimeOrdersResponse struct {
	Orders []*db.RuntimeOrder `json:"orders"`
}

type apiRuntimeOrderResponse struct {
	Order *db.RuntimeOrder `json:"order"`
}

type apiRuntimeMailboxResponse struct {
	Mailbox []*db.RuntimeMailboxMessage `json:"mailbox"`
}

type apiRuntimeMailboxItemResponse struct {
	Message *db.RuntimeMailboxMessage `json:"message"`
}

type apiRuntimeCheckpointResponse struct {
	Checkpoint *db.RuntimeCheckpoint `json:"checkpoint"`
}

type apiRuntimeCheckpointsResponse struct {
	Checkpoints []*db.RuntimeCheckpoint `json:"checkpoints"`
}

type apiRuntimeCheckpointCreateResponse struct {
	OK bool  `json:"ok"`
	ID int64 `json:"id"`
}

type apiRuntimeOrderCreateResponse struct {
	OK bool  `json:"ok"`
	ID int64 `json:"id"`
}

type apiMemoriaEntidadesResponse struct {
	Entidades []*db.EntidadMemoria `json:"entidades"`
}

type apiMemoriaEntidadResponse struct {
	Entidad *db.EntidadMemoria `json:"entidad"`
}

var httpClientOrquesta = &http.Client{
	Timeout: 15 * time.Second,
	Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 250 * time.Millisecond, KeepAlive: 30 * time.Second}).DialContext,
		ResponseHeaderTimeout: 12 * time.Second,
		DisableKeepAlives:     true,
	},
}

var httpClientOrquestaRuntimeHeavy = &http.Client{
	Timeout: 45 * time.Second,
	Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 250 * time.Millisecond, KeepAlive: 30 * time.Second}).DialContext,
		ResponseHeaderTimeout: 30 * time.Second,
		DisableKeepAlives:     true,
	},
}

func apiHTTPClientForPath(method, path string) *http.Client {
	method = strings.ToUpper(strings.TrimSpace(method))
	path = strings.TrimSpace(path)
	if method == http.MethodPost {
		switch {
		case strings.HasPrefix(path, "/api/runtime-orders"),
			strings.HasPrefix(path, "/api/runtime/process-"),
			path == "/api/agente/tick":
			return httpClientOrquestaRuntimeHeavy
		}
	}
	return httpClientOrquesta
}

func shouldPreferAPIClient(args []string) bool {
	if forceLocalMode(args) {
		return false
	}
	if !commandSupportsServerMode(normalizedCommandArgs(args)) {
		return false
	}
	return true
}

func requireServerForCurrentCommand() bool {
	if strings.TrimSpace(os.Getenv("ORQUESTA_FORCE_LOCAL_DB")) == "1" {
		return false
	}
	requireServer := strings.TrimSpace(strings.ToLower(os.Getenv("ORQUESTA_REQUIRE_SERVER")))
	if requireServer != "1" && requireServer != "true" && requireServer != "yes" {
		return false
	}
	args := getCurrentExecArgs()
	if len(args) == 0 && len(os.Args) > 1 {
		args = os.Args[1:]
	}
	if forceLocalMode(args) {
		return false
	}
	args = normalizedCommandArgs(args)
	if len(args) == 0 {
		return false
	}
	return commandSupportsServerMode(args)
}

func commandSupportsServerMode(args []string) bool {
	tokens := commandPathTokens(args)
	if len(tokens) == 0 {
		return false
	}
	switch tokens[0] {
	case "status":
		return true
	case "tarea":
		if len(tokens) <= 1 {
			return false
		}
		switch tokens[1] {
		case "listar", "ver", "nueva", "tomar", "iniciar", "completar", "bloquear", "desbloquear", "nota", "notas", "reasignar", "backlog", "contrato", "cancelar", "refineria":
			return true
		default:
			return false
		}
	case "propuesta":
		if len(tokens) <= 1 {
			return false
		}
		switch tokens[1] {
		case "listar", "ver", "votos", "nueva", "actualizar", "cerrar", "reabrir", "reparar-votos":
			return true
		default:
			return false
		}
	case "memoria":
		if len(tokens) <= 1 {
			return false
		}
		switch tokens[1] {
		case "listar", "ver", "guardar":
			return true
		default:
			return false
		}
	case "exportar":
		return len(tokens) > 1 && (tokens[1] == "estado" || tokens[1] == "audit" || tokens[1] == "diagnostico")
	case "logs":
		return true
	case "auditoria":
		return len(tokens) > 1 && tokens[1] == "listar"
	case "lenguaje":
		if len(tokens) <= 2 {
			return false
		}
		switch tokens[1] {
		case "politica":
			return tokens[2] == "ver" || tokens[2] == "fijar"
		case "matriz":
			return tokens[2] == "listar" || tokens[2] == "fijar" || tokens[2] == "borrar"
		case "resolver":
			return true
		default:
			return false
		}
	case "respaldo":
		return len(tokens) > 1 && tokens[1] == "bd"
	case "importar":
		return len(tokens) > 1 && tokens[1] == "historial"
	case "votar":
		return true
	case "runtime":
		if len(tokens) <= 1 {
			return false
		}
		switch tokens[1] {
		case "listar", "ver", "handles", "purgar-handles", "purgar-ordenes", "despertar", "procesar-mailbox", "procesar-autonomia", "procesar-reanimaciones", "limpiar-pruebas", "traza", "transcript", "diagnostico", "ordenes", "orden-nueva", "orden-cancelar", "nudge", "discordia", "checkpoints", "checkpoint-nuevo", "checkpoint-ver", "mailbox", "mailbox-enviar", "mailbox-entregar", "mailbox-consumir", "mailbox-limpiar":
			return true
		default:
			return false
		}
	case "proyecto":
		if len(tokens) <= 1 {
			return false
		}
		switch tokens[1] {
		case "listar", "ver", "descubrir", "fusionar", "microciclo":
			return true
		default:
			return false
		}
	case "progreso":
		if len(tokens) <= 1 {
			return false
		}
		switch tokens[1] {
		case "ver":
			return true
		case "fase":
			return len(tokens) > 2 && (tokens[2] == "listar" || tokens[2] == "registrar" || tokens[2] == "actualizar")
		case "tarea":
			return len(tokens) > 2 && tokens[2] == "registrar"
		default:
			return false
		}
	case "pool":
		if len(tokens) <= 1 {
			return false
		}
		switch tokens[1] {
		case "listar", "ver", "guardar", "seed-inicial":
			return true
		case "modelo":
			return len(tokens) > 2 && (tokens[2] == "listar" || tokens[2] == "guardar" || tokens[2] == "seed-inicial")
		default:
			return false
		}
	case "politica-modelo":
		return len(tokens) > 1 && (tokens[1] == "listar" || tokens[1] == "guardar" || tokens[1] == "seed-inicial")
	case "modelo":
		return len(tokens) > 1 && (tokens[1] == "resolver" || tokens[1] == "pipeline-local" || tokens[1] == "pipeline-paso" || tokens[1] == "pipeline-ejecutar" || tokens[1] == "pipeline-despachar" || tokens[1] == "runtime-activos" || tokens[1] == "runtime-descargar")
	case "reglas":
		return len(tokens) > 1 && (tokens[1] == "listar" || tokens[1] == "crear" || tokens[1] == "editar" || tokens[1] == "activar" || tokens[1] == "desactivar" || tokens[1] == "versiones")
	case "skills":
		return len(tokens) > 1 && (tokens[1] == "listar" || tokens[1] == "crear" || tokens[1] == "editar" || tokens[1] == "activar" || tokens[1] == "desactivar" || tokens[1] == "versiones" || tokens[1] == "detectar-carencia" || tokens[1] == "importar" || tokens[1] == "remotas" || tokens[1] == "borrar")
	case "workflows":
		return len(tokens) > 1 && (tokens[1] == "listar" || tokens[1] == "crear" || tokens[1] == "ver" || tokens[1] == "editar" || tokens[1] == "activar" || tokens[1] == "desactivar" || tokens[1] == "versiones")
	case "permisos":
		return len(tokens) > 1 && (tokens[1] == "listar" || tokens[1] == "fijar")
	case "conector":
		if len(tokens) <= 1 {
			return false
		}
		switch tokens[1] {
		case "listar", "ver", "registrar":
			return true
		default:
			return false
		}
	case "microprogramacion":
		if len(tokens) <= 2 {
			return false
		}
		switch tokens[1] {
		case "especificacion":
			return tokens[2] == "listar" || tokens[2] == "ver" || tokens[2] == "crear" || tokens[2] == "emitir" || tokens[2] == "despachar" || tokens[2] == "validar-entrega"
		default:
			return false
		}
	case "config":
		if len(tokens) <= 1 {
			return false
		}
		switch tokens[1] {
		case "ver", "set", "agente-nuevo", "agente-retirar", "agente-rehabilitar":
			return true
		default:
			return false
		}
	case "asignacion":
		if len(tokens) <= 1 {
			return false
		}
		switch tokens[1] {
		case "listar", "activar":
			return true
		default:
			return false
		}
	case "lock":
		if len(tokens) <= 1 {
			return false
		}
		switch tokens[1] {
		case "listar", "tomar", "renovar", "liberar":
			return true
		default:
			return false
		}
	case "worktree":
		if len(tokens) <= 1 {
			return false
		}
		switch tokens[1] {
		case "listar", "resolver", "crear", "cerrar":
			return true
		default:
			return false
		}
	case "agente":
		return len(tokens) > 1 && (tokens[1] == "preparar" || tokens[1] == "tick" || tokens[1] == "overview" || tokens[1] == "reanimaciones" || tokens[1] == "investigar" || tokens[1] == "pausar" || tokens[1] == "control" || tokens[1] == "eliminar" || tokens[1] == "rehabilitar" || tokens[1] == "fusionar" || tokens[1] == "handoff" || tokens[1] == "reasignar-vivo" || tokens[1] == "lanzar-plan" || tokens[1] == "cuentas" || tokens[1] == "presupuesto" || tokens[1] == "ranking-cuentas" || tokens[1] == "observar-cuenta")
	case "sesion":
		if len(tokens) <= 1 {
			return false
		}
		switch tokens[1] {
		case "inicio", "guardar", "continuar", "fin", "listar", "historial", "ver", "nuevo-codex":
			return true
		case "presupuesto":
			return len(tokens) > 2 && (tokens[2] == "registrar" || tokens[2] == "ver")
		default:
			return false
		}
	default:
		return false
	}
}

func commandPathTokens(args []string) []string {
	var tokens []string
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			break
		}
		tokens = append(tokens, arg)
		if len(tokens) >= 3 {
			break
		}
	}
	return tokens
}

func serverBaseURL() string {
	if v := strings.TrimSpace(os.Getenv("ORQUESTA_SERVER_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	if strings.TrimSpace(os.Getenv("ORQUESTA_DISABLE_SERVER_CLIENT")) == "1" {
		return ""
	}
	if info, _, err := loadServerInfoWithHealthFallback(""); err == nil && info != nil && strings.TrimSpace(info.Addr) != "" {
		return rpclocal.BaseURL(info.Addr)
	}
	return rpclocal.ResolveServerAddr()
}

func serverURLConfiguredExplicitly() bool {
	return strings.TrimSpace(os.Getenv("ORQUESTA_SERVER_URL")) != ""
}

func serverReachable() bool {
	base := serverBaseURL()
	if base == "" {
		return false
	}

	if !serverURLConfiguredExplicitly() {
		ctx, cancel := context.WithTimeout(context.Background(), rpclocal.DefaultTimeout())
		err := rpclocal.Ping(ctx, rpclocal.ResolveServerAddr())
		cancel()
		if err == nil {
			return true
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/server", nil)
	if err != nil {
		return false
	}
	resp, err := httpClientOrquesta.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func ensureLocalDB() error {
	if db.DB != nil {
		return nil
	}
	if localRecoveryEnabled() {
		if !localRecoveryCommandAllowed(currentOrOSArgs()) {
			return localRecoveryUnsupportedError(currentOrOSArgs())
		}
		return db.OpenRecoveryReadOnly()
	}
	return db.Open()
}

func apiGet(path string, dst any) (bool, error) {
	return apiGetQuery(path, nil, dst)
}

func apiGetQuery(path string, query url.Values, dst any) (bool, error) {
	if strings.TrimSpace(os.Getenv("ORQUESTA_FORCE_LOCAL_DB")) == "1" {
		return apiInvokeLocal(http.MethodGet, path, query, nil, dst)
	}
	base := serverBaseURL()
	if base == "" {
		resetServerDiscovery()
		base = serverBaseURL()
	}
	if base == "" {
		if requireServerForCurrentCommand() {
			return true, fmt.Errorf("este comando requiere el servidor de Orquesta activo; arranca 'orquesta serve' o usa ORQUESTA_FORCE_LOCAL_DB=1 solo para recuperacion")
		}
		return false, nil
	}
	endpoint := base + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return true, err
	}
	resp, err := httpClientOrquesta.Do(req)
	if err != nil {
		if !serverURLConfiguredExplicitly() {
			resetServerDiscovery()
			if retryBase := serverBaseURL(); retryBase != "" && retryBase != base {
				retryEndpoint := retryBase + path
				if len(query) > 0 {
					retryEndpoint += "?" + query.Encode()
				}
				reqRetry, reqErr := http.NewRequest(http.MethodGet, retryEndpoint, nil)
				if reqErr == nil {
					if retryResp, retryErr := httpClientOrquesta.Do(reqRetry); retryErr == nil {
						defer retryResp.Body.Close()
						if retryResp.StatusCode == http.StatusOK {
							if decErr := json.NewDecoder(retryResp.Body).Decode(dst); decErr == nil {
								return true, nil
							}
						}
					}
				}
			}
		}
		if requireServerForCurrentCommand() || serverURLConfiguredExplicitly() {
			return true, fmt.Errorf("no se pudo contactar con el servidor de Orquesta: %w", err)
		}
		return false, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var apiErr apiErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil && strings.TrimSpace(apiErr.Error) != "" {
			return true, fmt.Errorf("%s", apiErr.Error)
		}
		return true, fmt.Errorf("respuesta HTTP inesperada: %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return true, err
	}
	return true, nil
}

func apiProjectSlugMap() (map[int64]string, bool, error) {
	var resp apiProyectosResponse
	ok, err := apiGet("/api/proyectos", &resp)
	if !ok || err != nil {
		return nil, ok, err
	}
	out := make(map[int64]string, len(resp.Proyectos))
	for _, proyecto := range resp.Proyectos {
		out[proyecto.ID] = proyecto.Slug
	}
	return out, true, nil
}

func apiPost(path string, payload any, dst any) (bool, error) {
	if strings.TrimSpace(os.Getenv("ORQUESTA_FORCE_LOCAL_DB")) == "1" {
		return apiInvokeLocal(http.MethodPost, path, nil, payload, dst)
	}
	base := serverBaseURL()
	if base == "" {
		resetServerDiscovery()
		base = serverBaseURL()
	}
	if base == "" {
		if requireServerForCurrentCommand() {
			return true, fmt.Errorf("este comando requiere el servidor de Orquesta activo; arranca 'orquesta serve' o usa ORQUESTA_FORCE_LOCAL_DB=1 solo para recuperacion")
		}
		return false, nil
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return true, err
	}
	req, err := http.NewRequest(http.MethodPost, base+path, bytes.NewReader(body))
	if err != nil {
		return true, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := apiHTTPClientForPath(http.MethodPost, path)
	resp, err := client.Do(req)
	if err != nil {
		if !serverURLConfiguredExplicitly() {
			resetServerDiscovery()
			if retryBase := serverBaseURL(); retryBase != "" && retryBase != base {
				reqRetry, reqErr := http.NewRequest(http.MethodPost, retryBase+path, bytes.NewReader(body))
				if reqErr == nil {
					reqRetry.Header.Set("Content-Type", "application/json")
					if retryResp, retryErr := client.Do(reqRetry); retryErr == nil {
						defer retryResp.Body.Close()
						if retryResp.StatusCode >= http.StatusOK && retryResp.StatusCode < http.StatusMultipleChoices {
							if dst == nil {
								return true, nil
							}
							if decErr := json.NewDecoder(retryResp.Body).Decode(dst); decErr == nil {
								return true, nil
							}
						}
					}
				}
			}
		}
		if requireServerForCurrentCommand() || serverURLConfiguredExplicitly() {
			return true, fmt.Errorf("no se pudo contactar con el servidor de Orquesta: %w", err)
		}
		return false, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		var apiErr apiErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil && strings.TrimSpace(apiErr.Error) != "" {
			return true, fmt.Errorf("%s", apiErr.Error)
		}
		return true, fmt.Errorf("respuesta HTTP inesperada: %d", resp.StatusCode)
	}
	if dst == nil {
		return true, nil
	}
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return true, err
	}
	return true, nil
}

func apiInvokeLocal(method, path string, query url.Values, payload any, dst any) (bool, error) {
	if err := ensureLocalDB(); err != nil {
		return true, err
	}

	target := path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}

	var body *bytes.Reader
	if payload == nil {
		body = bytes.NewReader(nil)
	} else {
		raw, err := json.Marshal(payload)
		if err != nil {
			return true, err
		}
		body = bytes.NewReader(raw)
	}

	req := httptest.NewRequest(method, target, body)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	registerAPIRoutes(mux)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		var apiErr apiErrorResponse
		if err := json.NewDecoder(bytes.NewReader(rec.Body.Bytes())).Decode(&apiErr); err == nil && strings.TrimSpace(apiErr.Error) != "" {
			return true, fmt.Errorf("%s", apiErr.Error)
		}
		return true, fmt.Errorf("respuesta HTTP inesperada: %d", rec.Code)
	}
	if dst == nil || rec.Body.Len() == 0 {
		return true, nil
	}
	if err := json.NewDecoder(bytes.NewReader(rec.Body.Bytes())).Decode(dst); err != nil {
		return true, err
	}
	return true, nil
}
