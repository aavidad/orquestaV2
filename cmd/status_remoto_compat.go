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
	"orquesta/internal/rpclocal"
	"orquesta/propuestasapp"
)

const serverURLVar = "ORQUESTA_SERVER_URL"

var serverHTTPClient = httpClientOrquesta

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
	if strings.TrimSpace(os.Getenv("ORQUESTA_DISABLE_SERVER_CLIENT")) == "1" {
		return ""
	}
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
	candidates := make([]string, 0, 3)
	seen := map[string]struct{}{}
	add := func(raw string) {
		raw = strings.TrimRight(strings.TrimSpace(raw), "/")
		if raw == "" {
			return
		}
		if _, ok := seen[raw]; ok {
			return
		}
		seen[raw] = struct{}{}
		candidates = append(candidates, raw)
	}

	add(configuredServerURL())
	add(rpclocal.ResolveServerAddr())
	add(defaultServerURL)
	return candidates
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
	if !commandSupportsServerMode(normalizedCommandArgs(args)) {
		return false
	}
	return activeServerURL() != ""
}

func loadStatusSummary() (*statusContext, error) {
	serverURL := activeServerURL()
	if serverURL == "" {
		resetServerDiscovery()
		serverURL = activeServerURL()
	}
	if serverURL != "" {
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
	if !localRecoveryEnabled() {
		return nil, serverFirstCommandError("status")
	}
	if err := openDBForCommand(currentOrOSArgs()); err != nil {
		return nil, err
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
	if resumen != nil && len(resumen.AgentesTrabajando) == 0 {
		resumen.AgentesTrabajando = derivarAgentesTrabajando(resumen.AgentesActivos, resumen.TareasActivas)
	}
	fmt.Printf("╔═══════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║           ORQUESTA — ESTADO DEL PROYECTO                 ║\n")
	fmt.Printf("╚═══════════════════════════════════════════════════════════╝\n\n")

	fmt.Printf("👥 Agentes: %d conectados", len(resumen.AgentesActivos))
	if len(resumen.AgentesTrabajando) > 0 {
		fmt.Printf(" · %d con trabajo activo", len(resumen.AgentesTrabajando))
	}
	fmt.Println()
	for _, a := range resumen.AgentesActivos {
		fmt.Printf("   🟢 %-15s [%s]", a.Nombre, a.Rol)
		if cuenta := resumenCuentaAgente(a); cuenta != "" {
			fmt.Printf(" — %s", cuenta)
		}
		if detalle := resumenCuotaAgente(a); detalle != "" {
			fmt.Printf(" — %s", detalle)
		}
		fmt.Println()
	}
	if len(resumen.AgentesActivos) == 0 {
		fmt.Printf("   — sin agentes conectados\n")
	}
	agentesEnCuota := agentesNoActivosConCuota(resumen.Agentes)
	if len(agentesEnCuota) > 0 {
		fmt.Printf("   ⏸️  En enfriamiento/cuota:\n")
		for _, a := range agentesEnCuota {
			fmt.Printf("      %-15s [%s]", a.Nombre, a.Rol)
			if cuenta := resumenCuentaAgente(a); cuenta != "" {
				fmt.Printf(" — %s", cuenta)
			}
			if detalle := resumenCuotaAgente(a); detalle != "" {
				fmt.Printf(" — %s", detalle)
			}
			fmt.Println()
		}
	}
	agentesEnPausa := agentesNoActivosEnPausaOperativa(resumen.Agentes)
	if len(agentesEnPausa) > 0 {
		fmt.Printf("   ⏸️  En pausa operativa:\n")
		for _, a := range agentesEnPausa {
			fmt.Printf("      %-15s [%s]", a.Nombre, a.Rol)
			if cuenta := resumenCuentaAgente(a); cuenta != "" {
				fmt.Printf(" — %s", cuenta)
			}
			if detalle := resumenCuotaAgente(a); detalle != "" {
				fmt.Printf(" — %s", detalle)
			}
			fmt.Println()
		}
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

	retenidas := tareasRetenidasPorCuota(resumen.TareasActivas, resumen.Agentes)
	retenidasIDs := make(map[int64]struct{}, len(retenidas))
	for _, t := range retenidas {
		retenidasIDs[t.ID] = struct{}{}
	}
	tareasEnProgresoBase := resumen.TareasEnProgreso
	if len(tareasEnProgresoBase) == 0 {
		tareasEnProgresoBase = filtrarOpenClawTareasPorEstado(resumen.TareasActivas, db.TareaEnProgreso, db.TareaBloqueada)
	}
	tareasEnProgresoVisibles := make([]tareaLite, 0, len(tareasEnProgresoBase))
	for _, t := range tareasEnProgresoBase {
		if _, blocked := retenidasIDs[t.ID]; blocked {
			continue
		}
		tareasEnProgresoVisibles = append(tareasEnProgresoVisibles, t)
	}
	if len(tareasEnProgresoVisibles) > 0 {
		fmt.Printf("⚙️  En progreso ahora mismo:\n")
		for _, t := range tareasEnProgresoVisibles {
			agente := "—"
			if t.Agente != "" {
				agente = t.Agente
			}
			fmt.Printf("   [%d] %-40s → %s\n", t.ID, truncar(t.Titulo, 38), agente)
		}
		fmt.Println()
	}
	tareasReservadasBase := resumen.TareasReservadas
	if len(tareasReservadasBase) == 0 {
		tareasReservadasBase = filtrarOpenClawTareasPorEstado(resumen.TareasActivas, db.TareaAsignada)
	}
	tareasReservadasVisibles := make([]tareaLite, 0, len(tareasReservadasBase))
	for _, t := range tareasReservadasBase {
		if _, blocked := retenidasIDs[t.ID]; blocked {
			continue
		}
		tareasReservadasVisibles = append(tareasReservadasVisibles, t)
	}
	if len(tareasReservadasVisibles) > 0 {
		fmt.Printf("📦 Reservadas ahora mismo:\n")
		for _, t := range tareasReservadasVisibles {
			agente := "—"
			if t.Agente != "" {
				agente = t.Agente
			}
			fmt.Printf("   [%d] %-40s → %s\n", t.ID, truncar(t.Titulo, 38), agente)
		}
		fmt.Println()
	}
	if len(retenidas) > 0 {
		fmt.Printf("⏸️  Retenidas por cuota:\n")
		for _, t := range retenidas {
			agente := "—"
			if t.Agente != "" {
				agente = t.Agente
			}
			fmt.Printf("   [%d] %-40s → %s\n", t.ID, truncar(t.Titulo, 38), agente)
		}
		fmt.Println()
	}
}

func resumenCuotaAgente(a *db.Agente) string {
	if a == nil {
		return ""
	}
	partes := make([]string, 0, 4)
	bloqueadoPorCuota := agenteBloqueadoPorCuotaVisible(a)
	if a.EstadoCuota != "" && a.EstadoCuota != "activo" {
		if reanimarAt := cooldownVisibleAgente(a); reanimarAt != nil {
			partes = append(partes, "cooldown hasta "+reanimarAt.Local().Format("2006-01-02 15:04"))
		}
	}
	if a.CuotaRestantePct != nil {
		partes = append(partes, fmt.Sprintf("%s %d%%", etiquetaCuotaVisible(a), *a.CuotaRestantePct))
	}
	if a.PresupuestoVentana != "" {
		partes = append(partes, "ventana "+a.PresupuestoVentana)
	}
	if a.PresupuestoResetAt != nil && !a.PresupuestoResetAt.IsZero() {
		partes = append(partes, "reset "+a.PresupuestoResetAt.Local().Format("2006-01-02 15:04"))
	}
	if a.PresupuestoStale {
		if edad := edadPresupuestoObservado(a.PresupuestoCheckedAt); edad != "" {
			partes = append(partes, "telemetría observada stale ("+edad+")")
		} else {
			partes = append(partes, "telemetría observada stale")
		}
	}
	if extra := resumenDesgloseCuotaAgente(a); extra != "" {
		partes = append(partes, extra)
	}
	if a.RemainingCredits != nil {
		partes = append(partes, fmt.Sprintf("cred %.2f", *a.RemainingCredits))
	}
	if a.PresupuestoEstado != "" && a.PresupuestoEstado != "ok" {
		partes = append(partes, a.PresupuestoEstado)
	}
	if a.EstadoCuota != "" && a.EstadoCuota != "activo" {
		if agenteMotivoPausaOperativa(a.MotivoPausa) {
			partes = append(partes, "pausa:runtime")
		} else if !bloqueadoPorCuota {
			partes = append(partes, "pausa:operativa")
		} else {
			partes = append(partes, "cuota:"+a.EstadoCuota)
		}
	}
	return strings.Join(partes, " · ")
}

func cooldownVisibleAgente(a *db.Agente) *time.Time {
	if a == nil {
		return nil
	}
	now := time.Now().UTC()
	if a.ReanimarAt != nil && !a.ReanimarAt.IsZero() && a.ReanimarAt.After(now) {
		return a.ReanimarAt
	}
	for _, candidate := range []*time.Time{a.PresupuestoResetAt, a.PresupuestoSemanalResetAt, a.PresupuestoDiarioResetAt, a.PresupuestoSesionResetAt} {
		if candidate != nil && !candidate.IsZero() && candidate.After(now) {
			return candidate
		}
	}
	return nil
}

func resumenCuentaAgente(a *db.Agente) string {
	if a == nil {
		return ""
	}
	partes := make([]string, 0, 2)
	if email := strings.TrimSpace(a.CuentaEmail); email != "" {
		partes = append(partes, "cuenta "+email)
	}
	if usuario := strings.TrimSpace(a.CuentaUsuario); usuario != "" && !strings.EqualFold(usuario, strings.TrimSpace(a.CuentaEmail)) {
		partes = append(partes, "usuario "+usuario)
	}
	return strings.Join(partes, " · ")
}

func resumenDesgloseCuotaAgente(a *db.Agente) string {
	if a == nil {
		return ""
	}
	partes := []string{}
	if a.PresupuestoSesionPct != nil {
		partes = append(partes, renderVentanaPresupuesto("sesión", a.PresupuestoSesionPct, a.PresupuestoSesionResetAt))
	}
	if a.PresupuestoDiarioPct != nil {
		partes = append(partes, renderVentanaPresupuesto("diario", a.PresupuestoDiarioPct, a.PresupuestoDiarioResetAt))
	}
	if a.PresupuestoSemanalPct != nil {
		partes = append(partes, renderVentanaPresupuesto("semanal", a.PresupuestoSemanalPct, a.PresupuestoSemanalResetAt))
	}
	if len(partes) == 0 {
		return ""
	}
	texto := strings.Join(partes, " / ")
	if presupuestoVisibleObservadoStale(a) {
		return "estimado " + texto
	}
	return texto
}

func agentesNoActivosConCuota(agentes []*db.Agente) []*db.Agente {
	out := make([]*db.Agente, 0, len(agentes))
	for _, agente := range agentes {
		if agente == nil || agente.Activo {
			continue
		}
		if !agenteBloqueadoPorCuotaVisible(agente) {
			continue
		}
		out = append(out, agente)
	}
	return out
}

func agentesNoActivosEnPausaOperativa(agentes []*db.Agente) []*db.Agente {
	out := make([]*db.Agente, 0, len(agentes))
	for _, agente := range agentes {
		if agente == nil || agente.Activo {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(agente.EstadoCuota), "activo") {
			continue
		}
		if agenteBloqueadoPorCuotaVisible(agente) {
			continue
		}
		out = append(out, agente)
	}
	return out
}

func tareasRetenidasPorCuota(tareas []tareaLite, agentes []*db.Agente) []tareaLite {
	if len(tareas) == 0 || len(agentes) == 0 {
		return nil
	}
	porNombre := make(map[string]*db.Agente, len(agentes))
	for _, agente := range agentes {
		if agente == nil {
			continue
		}
		porNombre[strings.TrimSpace(agente.Nombre)] = agente
	}
	out := make([]tareaLite, 0)
	for _, tarea := range tareas {
		agente := porNombre[strings.TrimSpace(tarea.Agente)]
		if agente == nil || agente.Activo || !agenteBloqueadoPorCuotaVisible(agente) {
			continue
		}
		out = append(out, tarea)
	}
	return out
}

func agenteBloqueadoPorCuotaVisible(a *db.Agente) bool {
	if a == nil {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(a.EstadoCuota), "activo") {
		return false
	}
	if agenteMotivoPausaOperativa(a.MotivoPausa) {
		return false
	}
	if a.PresupuestoCheckedAt != nil && !a.PresupuestoCheckedAt.IsZero() {
		return true
	}
	if strings.TrimSpace(a.PresupuestoEstado) != "" && !strings.EqualFold(strings.TrimSpace(a.PresupuestoEstado), "ok") {
		return true
	}
	if a.RemainingCredits != nil && *a.RemainingCredits <= 0 {
		return true
	}
	if a.PresupuestoSemanalPct != nil && *a.PresupuestoSemanalPct <= 0 {
		return true
	}
	tieneVentanaTemporalVisible := a.PresupuestoSesionPct != nil ||
		presupuestoVisibleEsVentanaCorta(strings.TrimSpace(a.PresupuestoVentana)) ||
		strings.EqualFold(strings.TrimSpace(a.PresupuestoFuente), "provider_backoff")
	if a.PresupuestoSemanalPct != nil && *a.PresupuestoSemanalPct > 0 && !tieneVentanaTemporalVisible {
		return false
	}
	if a.PresupuestoSesionPct != nil {
		return *a.PresupuestoSesionPct <= 0
	}
	if tieneVentanaTemporalVisible && a.CuotaRestantePct != nil {
		return *a.CuotaRestantePct <= 0
	}
	if (a.CuotaRestantePct != nil && *a.CuotaRestantePct > 0) ||
		(a.PresupuestoSemanalPct != nil && *a.PresupuestoSemanalPct > 0) ||
		(a.PresupuestoSesionPct != nil && *a.PresupuestoSesionPct > 0) {
		return false
	}
	return true
}

func presupuestoVisibleEsVentanaCorta(windowKind string) bool {
	windowKind = strings.ToLower(strings.TrimSpace(windowKind))
	switch windowKind {
	case "5h", "session", "provider":
		return true
	}
	return strings.HasSuffix(windowKind, "m")
}

func renderVentanaPresupuesto(nombre string, pct *int, resetAt *time.Time) string {
	if pct == nil {
		return ""
	}
	parte := fmt.Sprintf("%s %d%%", nombre, *pct)
	if resetAt != nil && !resetAt.IsZero() {
		parte += " reset " + resetAt.Local().Format("2006-01-02 15:04")
	}
	return parte
}

func etiquetaCuotaVisible(a *db.Agente) string {
	if presupuestoVisibleObservadoStaleAgente(a) {
		return "observado"
	}
	return "efectivo"
}

func presupuestoVisibleObservadoStale(a *db.Agente) bool {
	return presupuestoVisibleObservadoStaleAgente(a)
}

func presupuestoVisibleObservadoStaleAgente(a *db.Agente) bool {
	if a == nil || !a.PresupuestoStale {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(a.PresupuestoEstado), "observado_stale")
}

func etiquetaCuotaVisibleCuenta(stale bool, estado string) string {
	if stale {
		return "observado"
	}
	return "efectivo"
}

func edadPresupuestoObservado(checkedAt *time.Time) string {
	if checkedAt == nil || checkedAt.IsZero() {
		return ""
	}
	d := time.Since(checkedAt.UTC())
	if d < 0 {
		d = 0
	}
	if d < time.Minute {
		return "hace <1m"
	}
	if d < time.Hour {
		return fmt.Sprintf("hace %dm", int(d.Round(time.Minute)/time.Minute))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("hace %dh%02dm", int(d/time.Hour), int((d%time.Hour)/time.Minute))
	}
	return fmt.Sprintf("hace %dd%02dh", int(d/(24*time.Hour)), int((d%(24*time.Hour))/time.Hour))
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
	body, err := fetchServerText(baseURL + "/api/status")
	if err != nil {
		return nil, err
	}
	var payload struct {
		Generado                 string          `json:"generado"`
		TareasPorEstado          map[string]int  `json:"tareasPorEstado"`
		AgentesActivos           []*db.Agente    `json:"agentesActivos"`
		AgentesTrabajando        []*db.Agente    `json:"agentesTrabajando"`
		PropuestasAbiertas       []propuestaLite `json:"propuestasAbiertas"`
		TareasActivas            []tareaLite     `json:"tareasActivas"`
		TareasEnProgreso         []tareaLite     `json:"tareasEnProgreso"`
		TareasReservadas         []tareaLite     `json:"tareasReservadas"`
		AgentesCompat            []*db.Agente    `json:"agentes"`
		ConteoTareasCompat       map[string]int  `json:"conteo_tareas"`
		PropuestasCompatAbiertas []*db.Propuesta `json:"propuestas_abiertas"`
	}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		return nil, err
	}
	resumen := &estadoResumen{
		Generado:           payload.Generado,
		Agentes:            payload.AgentesCompat,
		TareasPorEstado:    payload.TareasPorEstado,
		AgentesActivos:     payload.AgentesActivos,
		AgentesTrabajando:  payload.AgentesTrabajando,
		PropuestasAbiertas: payload.PropuestasAbiertas,
		TareasActivas:      payload.TareasActivas,
		TareasEnProgreso:   payload.TareasEnProgreso,
		TareasReservadas:   payload.TareasReservadas,
	}
	if len(resumen.TareasPorEstado) == 0 {
		resumen.TareasPorEstado = payload.ConteoTareasCompat
	}
	if len(resumen.AgentesActivos) == 0 {
		if _, present := raw["agentesActivos"]; !present {
		resumen.AgentesActivos = payload.AgentesCompat
		}
	}
	if len(resumen.AgentesTrabajando) == 0 {
		resumen.AgentesTrabajando = derivarAgentesTrabajando(resumen.AgentesActivos, resumen.TareasActivas)
	}
	if len(resumen.TareasEnProgreso) == 0 {
		resumen.TareasEnProgreso = filtrarOpenClawTareasPorEstado(resumen.TareasActivas, db.TareaEnProgreso, db.TareaBloqueada)
	}
	if len(resumen.TareasReservadas) == 0 {
		resumen.TareasReservadas = filtrarOpenClawTareasPorEstado(resumen.TareasActivas, db.TareaAsignada)
	}
	if len(resumen.PropuestasAbiertas) == 0 && len(payload.PropuestasCompatAbiertas) > 0 {
		resumen.PropuestasAbiertas = make([]propuestaLite, 0, len(payload.PropuestasCompatAbiertas))
		for _, propuesta := range payload.PropuestasCompatAbiertas {
			if propuesta == nil {
				continue
			}
			lite := propuestaLite{
				ID:           propuesta.ID,
				Codigo:       propuesta.Codigo,
				Titulo:       propuesta.Titulo,
				Estado:       propuesta.Estado,
				PropuestoPor: propuesta.PropuestoPor,
			}
			for _, voto := range propuesta.Votos {
				if voto == nil {
					continue
				}
				switch voto.Posicion {
				case db.VotoAcuerdo:
					lite.Acuerdo++
				case db.VotoDesacuerdo:
					lite.Desacuerdo++
				case db.VotoAbstencion:
					lite.Abstencion++
				default:
					lite.Pendiente++
				}
			}
			resumen.PropuestasAbiertas = append(resumen.PropuestasAbiertas, lite)
		}
	}
	return resumen, nil
}

func derivarAgentesTrabajando(agentesActivos []*db.Agente, tareasActivas []tareaLite) []*db.Agente {
	if len(agentesActivos) == 0 || len(tareasActivas) == 0 {
		return nil
	}
	trabajando := make(map[string]bool, len(tareasActivas))
	for _, tarea := range tareasActivas {
		if tarea.Estado != db.TareaEnProgreso || strings.TrimSpace(tarea.Agente) == "" {
			continue
		}
		trabajando[strings.TrimSpace(tarea.Agente)] = true
	}
	if len(trabajando) == 0 {
		return nil
	}
	out := make([]*db.Agente, 0, len(trabajando))
	for _, agente := range agentesActivos {
		if agente == nil {
			continue
		}
		if trabajando[strings.TrimSpace(agente.Nombre)] {
			out = append(out, agente)
		}
	}
	return out
}

func fetchServerConfigAll(baseURL string) (map[string]string, error) {
	var payload struct {
		Items  map[string]string `json:"items"`
		Config map[string]string `json:"config"`
	}
	if err := fetchServerJSON(baseURL+"/api/config", &payload); err != nil {
		return nil, err
	}
	if len(payload.Config) > 0 {
		return payload.Config, nil
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
		Items   []*db.Agente `json:"items"`
		Agentes []*db.Agente `json:"agentes"`
	}
	if err := fetchServerJSON(baseURL+"/api/agentes", &payload); err != nil {
		return nil, err
	}
	if payload.Agentes != nil {
		return payload.Agentes, nil
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
		Items  []*db.Tarea `json:"items"`
		Tareas []*db.Tarea `json:"tareas"`
	}
	u := baseURL + "/api/tareas"
	if encoded := query.Encode(); encoded != "" {
		u += "?" + encoded
	}
	if err := fetchServerJSON(u, &payload); err != nil {
		return nil, err
	}
	if payload.Tareas != nil {
		return payload.Tareas, nil
	}
	return payload.Items, nil
}

func fetchServerProposals(baseURL string, query url.Values) ([]*db.Propuesta, error) {
	var payload struct {
		Items      []*db.Propuesta `json:"items"`
		Propuestas []*db.Propuesta `json:"propuestas"`
	}
	u := baseURL + "/api/propuestas"
	if encoded := query.Encode(); encoded != "" {
		u += "?" + encoded
	}
	if err := fetchServerJSON(u, &payload); err != nil {
		return nil, err
	}
	if payload.Propuestas != nil {
		return payload.Propuestas, nil
	}
	return payload.Items, nil
}

func fetchServerTaskDetail(baseURL string, id int64) (*db.Tarea, error) {
	var payload struct {
		Item  *db.Tarea `json:"item"`
		Tarea *db.Tarea `json:"tarea"`
	}
	if err := fetchServerJSON(fmt.Sprintf("%s/api/tareas/%d", baseURL, id), &payload); err != nil {
		return nil, err
	}
	if payload.Tarea != nil {
		return payload.Tarea, nil
	}
	return payload.Item, nil
}

func fetchServerProposalDetail(baseURL, codigo string) (*propuestasapp.ProposalDetail, error) {
	var payload struct {
		Proposal  *db.Propuesta `json:"Proposal"`
		Propuesta *db.Propuesta `json:"propuesta"`
		Votes     []*db.Voto    `json:"Votes"`
	}
	if err := fetchServerJSON(baseURL+"/api/propuestas/"+url.PathEscape(codigo), &payload); err != nil {
		return nil, err
	}
	propuesta := payload.Proposal
	if propuesta == nil {
		propuesta = payload.Propuesta
	}
	votos := payload.Votes
	if len(votos) == 0 && propuesta != nil && len(propuesta.Votos) > 0 {
		votos = propuesta.Votos
	}
	return &propuestasapp.ProposalDetail{Proposal: propuesta, Votes: votos}, nil
}

func submitServerTaskAction(baseURL string, id int64, action string, payload map[string]any) error {
	if payload == nil {
		payload = map[string]any{}
	}
	return postServerJSON(fmt.Sprintf("%s/api/tareas/%d/%s", baseURL, id, action), payload, nil)
}

func submitServerCreateTask(baseURL string, payload map[string]any) (int64, error) {
	var result struct {
		ID    int64     `json:"id"`
		Tarea *db.Tarea `json:"tarea"`
	}
	if err := postServerJSON(baseURL+"/api/tareas", payload, &result); err != nil {
		return 0, err
	}
	if result.ID == 0 && result.Tarea != nil {
		result.ID = result.Tarea.ID
	}
	return result.ID, nil
}

func submitServerCreateProposal(baseURL string, payload map[string]any) (int64, string, error) {
	var result struct {
		ID        int64         `json:"id"`
		Codigo    string        `json:"codigo"`
		Propuesta *db.Propuesta `json:"propuesta"`
	}
	if err := postServerJSON(baseURL+"/api/propuestas", payload, &result); err != nil {
		return 0, "", err
	}
	if result.Codigo == "" && result.Propuesta != nil {
		result.Codigo = result.Propuesta.Codigo
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
