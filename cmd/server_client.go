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
	"net/url"
	"os"
	"strings"
	"time"

	"orquesta/db"
)

const defaultOrquestaServerURL = "http://127.0.0.1:8080"

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

var httpClientOrquesta = &http.Client{
	Timeout: 350 * time.Millisecond,
	Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 150 * time.Millisecond, KeepAlive: 30 * time.Second}).DialContext,
		ResponseHeaderTimeout: 200 * time.Millisecond,
		DisableKeepAlives:     true,
	},
}

func shouldPreferServerForCurrentCommand() bool {
	if strings.TrimSpace(os.Getenv("ORQUESTA_FORCE_LOCAL_DB")) == "1" {
		return false
	}
	if !commandSupportsServerMode(os.Args[1:]) {
		return false
	}
	return serverReachable()
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
		case "listar", "ver", "nueva", "tomar", "iniciar", "completar", "bloquear", "desbloquear", "nota", "reasignar", "backlog", "contrato":
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
		case "listar", "ver":
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
		case "listar", "crear", "cerrar":
			return true
		default:
			return false
		}
	case "agente":
		return len(tokens) > 1 && (tokens[1] == "preparar" || tokens[1] == "tick")
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
	return db.Open()
}

func apiGet(path string, dst any) (bool, error) {
	return apiGetQuery(path, nil, dst)
}

func apiGetQuery(path string, query url.Values, dst any) (bool, error) {
	base := serverBaseURL()
	if base == "" || !serverReachable() {
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
	if base == "" || !serverReachable() {
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
}
