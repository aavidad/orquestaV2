/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"orquesta/db"
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

type statusContext struct {
	resumen *estadoResumen
	backend *serverInfo
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
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

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

func loadStatusSummary() (*statusContext, error) {
	if serverURL := activeServerURL(); serverURL != "" {
		resumen, err := fetchServerStatus(serverURL)
		if err != nil {
			return nil, err
		}
		backend, err := fetchServerInfo(serverURL)
		if err != nil {
			backend = nil
		}
		return &statusContext{resumen: resumen, backend: backend}, nil
	}
	resumen, err := buildEstadoResumen()
	if err != nil {
		return nil, err
	}
	return &statusContext{resumen: resumen}, nil
}

func renderStatusSummary(ctx *statusContext) {
	if ctx == nil {
		return
	}
	if lines := serverInfoLines(ctx.backend); len(lines) > 0 {
		for _, line := range lines {
			fmt.Println(line)
		}
		fmt.Println()
	}
	resumen := ctx.resumen
	fmt.Printf("╔═══════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║           ORQUESTA — ESTADO DEL PROYECTO                 ║\n")
	fmt.Printf("╚═══════════════════════════════════════════════════════════╝\n\n")

	fmt.Printf("👥 Agentes: %d activos ahora\n", len(resumen.AgentesActivos))
	for _, a := range resumen.AgentesActivos {
		fmt.Printf("   🟢 %-15s [%s]\n", a.Nombre, a.Rol)
	}
	if len(resumen.AgentesActivos) == 0 {
		fmt.Printf("   — sin agentes activos\n")
	}
	fmt.Println()

	totalTareas := 0
	completadasN := 0
	for estado, n := range resumen.TareasPorEstado {
		totalTareas += n
		if estado == string(db.EstadoCompletada) {
			completadasN = n
		}
	}
	pctStr := ""
	if totalTareas > 0 {
		pct := float64(completadasN) * 100.0 / float64(totalTareas)
		pctStr = fmt.Sprintf("  progreso: %d/%d completadas (%.0f%%)", completadasN, totalTareas, pct)
	}
	fmt.Printf("📋 Tareas —%s:\n", pctStr)
	order := []db.EstadoTarea{db.EstadoEnProgreso, db.EstadoAsignada, db.EstadoLibre, db.EstadoBacklog, db.EstadoCompletada, db.EstadoBloqueada, db.EstadoCancelada}
	for _, e := range order {
		if n := resumen.TareasPorEstado[string(e)]; n > 0 {
			fmt.Printf("   %-15s %d\n", e, n)
		}
	}
	fmt.Println()

	fmt.Printf("📣 Propuestas abiertas: %d\n", len(resumen.PropuestasAbiertas))
	for _, p := range resumen.PropuestasAbiertas {
		fmt.Printf("   %s %-40s  ✓%d ✗%d ⏳%d\n", p.Codigo, truncar(p.Titulo, 38), p.Acuerdo, p.Desacuerdo, p.Pendiente)
	}
	fmt.Println()

	if len(resumen.TareasActivas) > 0 {
		fmt.Printf("⚙️  En progreso ahora mismo:\n")
		for _, t := range resumen.TareasActivas {
			agente := "—"
			if t.Agente != "" {
				agente = t.Agente
			}
			fmt.Printf("   [%d] %-40s → %s\n", t.ID, truncar(t.Titulo, 38), agente)
		}
		fmt.Println()
	}
}

func serverInfoLines(info *serverInfo) []string {
	if info == nil {
		return nil
	}
	backend := "backend remoto"
	if info.Name != "" {
		backend = info.Name
		if info.Version != "" {
			backend += " " + info.Version
		}
	} else if info.Version != "" {
		backend = info.Version
	}
	details := make([]string, 0, 5)
	if info.StorageMode != "" {
		details = append(details, "modo "+info.StorageMode)
	}
	if info.StorageDriver != "" {
		details = append(details, "driver "+info.StorageDriver)
	}
	if info.SQLPlaceholder != "" {
		details = append(details, "placeholder "+info.SQLPlaceholder)
	}
	if info.BootstrapSchema {
		details = append(details, "bootstrap schema")
	}
	if info.QueryRebinding {
		details = append(details, "query rebinding")
	}
	lines := []string{fmt.Sprintf("🖥️  Backend activo: %s", backend)}
	if len(details) > 0 {
		lines = append(lines, fmt.Sprintf("   %s", strings.Join(details, " | ")))
	}
	if len(info.Capabilities) > 0 {
		lines = append(lines, fmt.Sprintf("   capacidades: %s", strings.Join(info.Capabilities, ", ")))
	}
	return lines
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
	return postServerJSON(baseURL+"/api/config", map[string]string{"clave": key, "valor": value}, nil)
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
	payload := map[string]any{"codigo": codigo, "agente": agente, "posicion": string(posicion), "comentario": comentario}
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
	payload := map[string]any{"agente": agente, "nuevo_codex": nuevoCodex}
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
	return postServerJSON(baseURL+"/api/agentes", map[string]string{"nombre": nombre, "rol": rol}, nil)
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
	return postServerJSON(baseURL+"/api/propuestas/"+url.PathEscape(codigo)+"/cerrar", map[string]string{"estado": estado, "agente": agente}, nil)
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
}
