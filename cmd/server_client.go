<<<<<<< HEAD
/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

=======
>>>>>>> origin/orq-orquestador-codex2
package cmd

import (
	"bytes"
<<<<<<< HEAD
	"context"
	"encoding/json"
	"fmt"
	"net"
=======
	"encoding/json"
	"fmt"
	"io"
>>>>>>> origin/orq-orquestador-codex2
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"orquesta/db"
<<<<<<< HEAD
)

const defaultOrquestaServerURL = "http://127.0.0.1:16543"

type apiAgentesResponse struct {
	Agentes []*db.Agente `json:"agentes"`
}

type apiProyectosResponse struct {
	Proyectos []*db.Proyecto `json:"proyectos"`
}

type apiConectoresResponse struct {
	Conectores []*db.Conector `json:"conectores"`
}

type apiTareasResponse struct {
	Tareas []*db.Tarea `json:"tareas"`
}

type apiTareaResponse struct {
	Tarea *db.Tarea `json:"tarea"`
}

type apiSesionResponse struct {
	Sesion *db.Sesion `json:"sesion"`
}

type apiSesionesInspeccionResponse struct {
	Sesiones []*db.Sesion `json:"sesiones"`
}

type apiSesionInicioResponse struct {
	Sesion               *db.Sesion      `json:"sesion"`
	SesionPrevia         *db.Sesion      `json:"sesion_previa"`
	Rol                  string          `json:"rol"`
	PropuestasPendientes []*db.Propuesta `json:"propuestas_pendientes"`
	Reglas               []*db.Regla     `json:"reglas"`
	Skills               []*db.Skill     `json:"skills"`
	Workflow             *db.Workflow    `json:"workflow"`
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

type apiRuntimeOrdersResponse struct {
	Orders []*db.RuntimeOrder `json:"orders"`
}

type apiRuntimeMailboxResponse struct {
	Mailbox []*db.RuntimeMailboxMessage `json:"mailbox"`
}

type apiRuntimeCheckpointResponse struct {
	Checkpoint *db.RuntimeCheckpoint `json:"checkpoint"`
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
	Timeout: 350 * time.Millisecond,
	Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 150 * time.Millisecond, KeepAlive: 30 * time.Second}).DialContext,
		ResponseHeaderTimeout: 200 * time.Millisecond,
		DisableKeepAlives:     true,
	},
}

func shouldPreferAPIClient(args []string) bool {
	if strings.TrimSpace(os.Getenv("ORQUESTA_FORCE_LOCAL_DB")) == "1" {
		return false
	}
	if !commandSupportsServerMode(normalizedCommandArgs(args)) {
		return false
	}
	return serverReachable()
}

func requireServerForCurrentCommand() bool {
	if strings.TrimSpace(os.Getenv("ORQUESTA_FORCE_LOCAL_DB")) == "1" {
		return false
	}
	requireServer := strings.TrimSpace(strings.ToLower(os.Getenv("ORQUESTA_REQUIRE_SERVER")))
	if requireServer != "1" && requireServer != "true" && requireServer != "yes" {
		return false
	}
	return commandSupportsServerMode(os.Args[1:])
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
		case "listar", "ver", "nueva", "tomar", "iniciar", "completar", "bloquear", "desbloquear", "nota", "notas", "reasignar", "backlog", "contrato", "cancelar":
			return true
		default:
			return false
		}
	case "propuesta":
		if len(tokens) <= 1 {
			return false
		}
		switch tokens[1] {
		case "listar", "ver", "votos", "nueva", "cerrar", "reabrir", "reparar-votos":
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
	case "respaldo":
		return len(tokens) > 1 && tokens[1] == "bd"
	case "votar":
		return true
	case "runtime":
		if len(tokens) <= 1 {
			return false
		}
		switch tokens[1] {
		case "listar", "ver", "handles", "ordenes", "orden-nueva", "nudge", "checkpoints", "checkpoint-nuevo", "mailbox", "mailbox-enviar", "mailbox-entregar", "mailbox-consumir":
			return true
		default:
			return false
		}
	case "proyecto":
		if len(tokens) <= 1 {
			return false
		}
		switch tokens[1] {
		case "listar", "ver", "descubrir":
			return true
		default:
			return false
		}
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
		return len(tokens) > 1 && (tokens[1] == "preparar" || tokens[1] == "tick" || tokens[1] == "pausar" || tokens[1] == "eliminar" || tokens[1] == "rehabilitar" || tokens[1] == "handoff" || tokens[1] == "reasignar-vivo")
	case "sesion":
		if len(tokens) <= 1 {
			return false
		}
		switch tokens[1] {
		case "inicio", "guardar", "continuar", "fin", "listar", "historial", "ver", "nuevo-codex":
			return true
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
	return defaultOrquestaServerURL
}

func serverReachable() bool {
	base := serverBaseURL()
	if base == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/status", nil)
	if err != nil {
		return false
	}
	resp, err := httpClientOrquesta.Do(req)
=======
	"orquesta/proposalapp"
)

const serverURLVar = "ORQUESTA_SERVER_URL"
const defaultServerURL = "http://127.0.0.1:8080"

var serverHTTPClient = &http.Client{Timeout: 3 * time.Second}
var (
	discoveredServerURL     string
	serverURLDiscoveryReady bool
)

type serverInfo struct {
	Name            string   `json:"name"`
	Version         string   `json:"version"`
	StorageMode     string   `json:"storageMode"`
	StorageDriver   string   `json:"storageDriver"`
	SQLPlaceholder  string   `json:"sqlPlaceholder"`
	BootstrapSchema bool     `json:"bootstrapSchema"`
	QueryRebinding  bool     `json:"queryRebinding"`
	Capabilities    []string `json:"capabilities"`
}

func configuredServerURL() string {
	return strings.TrimRight(strings.TrimSpace(os.Getenv(serverURLVar)), "/")
}

func activeServerURL() string {
	if serverURLDiscoveryReady {
		return discoveredServerURL
	}
	discoveredServerURL = discoverServerURL()
	serverURLDiscoveryReady = true
	return discoveredServerURL
}

func discoverServerURL() string {
	for _, candidate := range candidateServerURLs() {
		if pingServer(candidate) {
			return candidate
		}
	}
	return ""
}

func candidateServerURLs() []string {
	configured := configuredServerURL()
	if configured == "" {
		return []string{defaultServerURL}
	}
	if configured == defaultServerURL {
		return []string{configured}
	}
	return []string{configured, defaultServerURL}
}

func pingServer(baseURL string) bool {
	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/server", nil)
	if err != nil {
		return false
	}
	probeClient := *serverHTTPClient
	probeClient.Timeout = 250 * time.Millisecond
	resp, err := probeClient.Do(req)
>>>>>>> origin/orq-orquestador-codex2
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

<<<<<<< HEAD
func ensureLocalDB() error {
	if db.DB != nil {
		return nil
	}
	return db.Open()
}

func apiGet(path string, dst any) (bool, error) {
	return apiGetQuery(path, nil, dst)
}

func apiGetQuery(path string, query url.Values, dst any) (bool, error) {
	base := serverBaseURL()
	if base == "" {
		if requireServerForCurrentCommand() {
			return true, fmt.Errorf("este comando requiere el servidor de Orquesta activo; arranca 'orquesta serve' o desactiva ORQUESTA_REQUIRE_SERVER")
		}
		return false, nil
	}
	if !serverReachable() {
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
		if requireServerForCurrentCommand() {
			return true, fmt.Errorf("no se pudo contactar con el servidor de Orquesta: %w", err)
		}
		return false, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var apiErr apiErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil && strings.TrimSpace(apiErr.Error) != "" {
			return true, fmt.Errorf(apiErr.Error)
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
	base := serverBaseURL()
	if base == "" {
		if requireServerForCurrentCommand() {
			return true, fmt.Errorf("este comando requiere el servidor de Orquesta activo; arranca 'orquesta serve' o desactiva ORQUESTA_REQUIRE_SERVER")
		}
		return false, nil
	}
	if !serverReachable() {
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
	resp, err := httpClientOrquesta.Do(req)
	if err != nil {
		if requireServerForCurrentCommand() {
			return true, fmt.Errorf("no se pudo contactar con el servidor de Orquesta: %w", err)
		}
		return false, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		var apiErr apiErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil && strings.TrimSpace(apiErr.Error) != "" {
			return true, fmt.Errorf(apiErr.Error)
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
=======
func resetServerDiscovery() {
	discoveredServerURL = ""
	serverURLDiscoveryReady = false
}

func shouldBypassLocalDB(args []string) bool {
	if !isRemoteCapableCommand(args) {
		return false
	}
	return activeServerURL() != ""
}

func isRemoteCapableCommand(args []string) bool {
	if len(args) == 0 {
		return false
	}
	switch strings.TrimSpace(args[0]) {
	case "status":
		return true
	case "config":
		return len(args) > 1 && (strings.TrimSpace(args[1]) == "ver" ||
			strings.TrimSpace(args[1]) == "set" ||
			strings.TrimSpace(args[1]) == "agente-nuevo" ||
			strings.TrimSpace(args[1]) == "agente-retirar" ||
			strings.TrimSpace(args[1]) == "agente-rehabilitar")
	case "exportar":
		return len(args) > 1 && (strings.TrimSpace(args[1]) == "estado" || strings.TrimSpace(args[1]) == "audit")
	case "sesion":
		return len(args) > 1 && (strings.TrimSpace(args[1]) == "inicio" || strings.TrimSpace(args[1]) == "fin" || strings.TrimSpace(args[1]) == "listar" || strings.TrimSpace(args[1]) == "nuevo-codex")
	case "tarea":
		return len(args) > 1 && (strings.TrimSpace(args[1]) == "listar" || strings.TrimSpace(args[1]) == "ver" || strings.TrimSpace(args[1]) == "nueva" || strings.TrimSpace(args[1]) == "tomar" || strings.TrimSpace(args[1]) == "iniciar" || strings.TrimSpace(args[1]) == "completar" || strings.TrimSpace(args[1]) == "bloquear" || strings.TrimSpace(args[1]) == "desbloquear" || strings.TrimSpace(args[1]) == "nota")
	case "propuesta":
		return len(args) > 1 && (strings.TrimSpace(args[1]) == "listar" || strings.TrimSpace(args[1]) == "ver" || strings.TrimSpace(args[1]) == "nueva" || strings.TrimSpace(args[1]) == "cerrar")
	case "votar":
		return true
	default:
		return false
	}
}

func fetchServerInfo(baseURL string) (*serverInfo, error) {
	var payload serverInfo
	if err := fetchServerJSON(baseURL+"/api/server", &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

func fetchServerStatus(baseURL string) (*estadoResumen, error) {
	var payload estadoResumen
	if err := fetchServerJSON(baseURL+"/api/status", &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

func fetchServerConfigAll(baseURL string) (map[string]string, error) {
	var payload struct {
		Items map[string]string `json:"items"`
	}
	if err := fetchServerJSON(baseURL+"/api/config", &payload); err != nil {
		return nil, err
	}
	return payload.Items, nil
}

func fetchServerConfigValue(baseURL, key string) (string, error) {
	var payload struct {
		Clave string `json:"clave"`
		Valor string `json:"valor"`
	}
	if err := fetchServerJSON(baseURL+"/api/config?clave="+url.QueryEscape(key), &payload); err != nil {
		return "", err
	}
	return payload.Valor, nil
}

func submitServerConfigValue(baseURL, key, value string) error {
	payload := map[string]string{"clave": key, "valor": value}
	return postServerJSON(baseURL+"/api/config", payload, nil)
}

type serverVoteResult struct {
	Codigo            string          `json:"codigo"`
	Agente            string          `json:"agente"`
	Posicion          db.PosicionVoto `json:"posicion"`
	Comentario        string          `json:"comentario"`
	ConsensoAlcanzado bool            `json:"consensoAlcanzado"`
	Conteo            map[string]int  `json:"conteo"`
}

func submitServerVote(baseURL, codigo, agente string, posicion db.PosicionVoto, comentario string) (*serverVoteResult, error) {
	payload := map[string]any{
		"codigo":     codigo,
		"agente":     agente,
		"posicion":   string(posicion),
		"comentario": comentario,
	}
	var result serverVoteResult
	if err := postServerJSON(baseURL+"/api/votar", payload, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

type serverSessionStartResult struct {
	Agente               string          `json:"agente"`
	SesionID             int64           `json:"sesion_id"`
	Rol                  string          `json:"rol"`
	PropuestasPendientes []*db.Propuesta `json:"propuestas_pendientes"`
	Reglas               []*db.Regla     `json:"reglas"`
	Skills               []*db.Skill     `json:"skills"`
	WorkflowPasos        []string        `json:"workflow_pasos"`
}

func submitServerSessionStart(baseURL, agente string, nuevoCodex bool) (*serverSessionStartResult, error) {
	payload := map[string]any{
		"agente":      agente,
		"nuevo_codex": nuevoCodex,
	}
	var result serverSessionStartResult
	if err := postServerJSON(baseURL+"/api/sesiones/inicio", payload, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func submitServerSessionFinish(baseURL, agente string) error {
	return postServerJSON(baseURL+"/api/sesiones/fin", map[string]string{"agente": agente}, nil)
}

func fetchServerAgents(baseURL string) ([]*db.Agente, error) {
	var payload struct {
		Items []*db.Agente `json:"items"`
	}
	if err := fetchServerJSON(baseURL+"/api/agentes", &payload); err != nil {
		return nil, err
	}
	return payload.Items, nil
}

func submitServerCreateAgent(baseURL, nombre, rol string) error {
	payload := map[string]string{
		"nombre": nombre,
		"rol":    rol,
	}
	return postServerJSON(baseURL+"/api/agentes", payload, nil)
}

func submitServerAgentAction(baseURL, nombre, action string) error {
	return postServerJSON(baseURL+"/api/agentes/"+url.PathEscape(nombre)+"/"+action, map[string]string{"nombre": nombre}, nil)
}

func fetchServerTasks(baseURL string, query url.Values) ([]*db.Tarea, error) {
	var payload struct {
		Items []*db.Tarea `json:"items"`
	}
	u := baseURL + "/api/tareas"
	if encoded := query.Encode(); encoded != "" {
		u += "?" + encoded
	}
	if err := fetchServerJSON(u, &payload); err != nil {
		return nil, err
	}
	return payload.Items, nil
}

func fetchServerProposals(baseURL string, query url.Values) ([]*db.Propuesta, error) {
	var payload struct {
		Items []*db.Propuesta `json:"items"`
	}
	u := baseURL + "/api/propuestas"
	if encoded := query.Encode(); encoded != "" {
		u += "?" + encoded
	}
	if err := fetchServerJSON(u, &payload); err != nil {
		return nil, err
	}
	return payload.Items, nil
}

func fetchServerTaskDetail(baseURL string, id int64) (*db.Tarea, error) {
	var payload struct {
		Item *db.Tarea `json:"item"`
	}
	if err := fetchServerJSON(fmt.Sprintf("%s/api/tareas/%d", baseURL, id), &payload); err != nil {
		return nil, err
	}
	return payload.Item, nil
}

func fetchServerProposalDetail(baseURL, codigo string) (*proposalapp.ProposalDetail, error) {
	var payload proposalapp.ProposalDetail
	if err := fetchServerJSON(baseURL+"/api/propuestas/"+url.PathEscape(codigo), &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

func submitServerTaskAction(baseURL string, id int64, action string, payload map[string]any) error {
	if payload == nil {
		payload = map[string]any{}
	}
	return postServerJSON(fmt.Sprintf("%s/api/tareas/%d/%s", baseURL, id, action), payload, nil)
}

func submitServerCreateTask(baseURL string, payload map[string]any) (int64, error) {
	var result struct {
		ID int64 `json:"id"`
	}
	if err := postServerJSON(baseURL+"/api/tareas", payload, &result); err != nil {
		return 0, err
	}
	return result.ID, nil
}

func submitServerCreateProposal(baseURL string, payload map[string]any) (int64, string, error) {
	var result struct {
		ID     int64  `json:"id"`
		Codigo string `json:"codigo"`
	}
	if err := postServerJSON(baseURL+"/api/propuestas", payload, &result); err != nil {
		return 0, "", err
	}
	return result.ID, result.Codigo, nil
}

func submitServerCloseProposal(baseURL, codigo, estado, agente string) error {
	payload := map[string]string{
		"estado": estado,
		"agente": agente,
	}
	return postServerJSON(baseURL+"/api/propuestas/"+url.PathEscape(codigo)+"/cerrar", payload, nil)
}

func fetchServerJSON(url string, target any) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("construyendo request %s: %w", url, err)
	}
	resp, err := serverHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("consultando servidor %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("servidor devolvio %s en %s", resp.Status, url)
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decodificando respuesta %s: %w", url, err)
	}
	return nil
}

func fetchServerText(url string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("construyendo request %s: %w", url, err)
	}
	resp, err := serverHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("consultando servidor %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("servidor devolvio %s en %s", resp.Status, url)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("leyendo respuesta %s: %w", url, err)
	}
	return string(body), nil
}

func postServerJSON(url string, payload any, target any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("codificando request %s: %w", url, err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("construyendo request %s: %w", url, err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := serverHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("consultando servidor %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("servidor devolvio %s en %s", resp.Status, url)
	}
	if target == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decodificando respuesta %s: %w", url, err)
	}
	return nil
}

func newJSONResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
>>>>>>> origin/orq-orquestador-codex2
}
