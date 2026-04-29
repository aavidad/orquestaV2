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
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"orquesta/capacidadapp"
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

func statusRegisteredAgentCount(resumen *estadoResumen) int {
	if resumen == nil {
		return 0
	}
	seen := map[string]struct{}{}
	add := func(items []*db.Agente) {
		for _, agente := range items {
			if agente == nil {
				continue
			}
			nombre := strings.ToLower(strings.TrimSpace(agente.Nombre))
			if nombre != "" {
				seen[nombre] = struct{}{}
			}
		}
	}
	add(resumen.Agentes)
	add(resumen.AgentesActivos)
	add(resumen.AgentesTrabajando)
	add(resumen.AgentesSaturados)
	add(resumen.AgentesAtascados)
	add(resumen.AgentesAuthManual)
	add(resumen.AgentesQuotaBlocked)
	return len(seen)
}

func configuredServerURL() string {
	return strings.TrimRight(strings.TrimSpace(os.Getenv(serverURLVar)), "/")
}

func configuredServerAddrURL() string {
	if addr := strings.TrimSpace(os.Getenv("ORQUESTA_SERVER_ADDR")); addr != "" {
		return rpclocal.BaseURL(addr)
	}
	return ""
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
	add(configuredServerAddrURL())
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
	if resumen == nil {
		return
	}
	resumen.TareasActivas = filtrarTareasActivasVisibles(resumen.TareasActivas, resumen.Agentes)
	resolverAgentesActivosDesdeCompatibilidad(resumen)
	if len(resumen.TareasEnProgreso) == 0 {
		resumen.TareasEnProgreso = filtrarOpenClawTareasPorEstado(resumen.TareasActivas, db.TareaEnProgreso)
	} else {
		resumen.TareasEnProgreso = filtrarTareasActivasVisibles(resumen.TareasEnProgreso, resumen.Agentes)
	}
	if len(resumen.TareasReservadas) == 0 {
		resumen.TareasReservadas = filtrarOpenClawTareasPorEstado(resumen.TareasActivas, db.TareaAsignada)
	} else {
		resumen.TareasReservadas = filtrarTareasActivasVisibles(resumen.TareasReservadas, resumen.Agentes)
	}
	if resumen != nil && len(resumen.AgentesTrabajando) == 0 {
		resumen.AgentesTrabajando = derivarAgentesTrabajando(resumen.AgentesActivos, resumen.TareasActivas)
	}
	fmt.Printf("╔═══════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║           ORQUESTA — ESTADO DEL PROYECTO                 ║\n")
	fmt.Printf("╚═══════════════════════════════════════════════════════════╝\n\n")

	fmt.Printf("👥 Agentes: %d activos visibles / %d registrados", len(resumen.AgentesActivos), statusRegisteredAgentCount(resumen))
	agentesEnCuota := agentesBloqueadosPorCuotaVisibles(resumen)
	if len(resumen.AgentesTrabajando) > 0 {
		fmt.Printf(" · %d con trabajo activo", len(resumen.AgentesTrabajando))
	}
	if len(resumen.AgentesSaturados) > 0 {
		fmt.Printf(" · %d saturados", len(resumen.AgentesSaturados))
	}
	if len(resumen.AgentesAtascados) > 0 {
		fmt.Printf(" · %d atascados", len(resumen.AgentesAtascados))
	}
	if len(resumen.AgentesAuthManual) > 0 {
		fmt.Printf(" · %d requieren autenticacion", len(resumen.AgentesAuthManual))
	}
	if len(agentesEnCuota) > 0 {
		fmt.Printf(" · %d bloqueados por cuota", len(agentesEnCuota))
	}
	fmt.Println()
	workersConectados, workersTrabajando, supervisoresActivos := resumenVisibleWorkerCounters(resumen)
	if workersConectados > 0 || workersTrabajando > 0 || supervisoresActivos > 0 {
		fmt.Printf("   Workers conectados %d · workers trabajando %d · supervisores activos %d\n", workersConectados, workersTrabajando, supervisoresActivos)
	}
	if resumen.AutonomySurface != nil && resumen.AutonomySurface.Events > 0 {
		fmt.Printf("   Autonomía reciente %s\n", formatAutonomySurfaceSummary(resumen.AutonomySurface))
		for _, item := range resumen.AutonomySurface.Recent {
			fmt.Printf("   · [%s] %s", strings.TrimSpace(item.Project), strings.TrimSpace(item.Kind))
			if strings.TrimSpace(item.Agent) != "" {
				fmt.Printf(" agente=%s", strings.TrimSpace(item.Agent))
			}
			if strings.TrimSpace(item.TargetAgent) != "" {
				fmt.Printf(" destino=%s", strings.TrimSpace(item.TargetAgent))
			}
			if strings.TrimSpace(item.Reason) != "" {
				fmt.Printf(" · %s", strings.TrimSpace(item.Reason))
			}
			fmt.Println()
		}
	}
	if project, score, highlights, ok := autonomySurfaceCriticalProjectRisk(resumen.AutonomySurface); ok {
		fmt.Printf("   Proyecto crítico %s · integracion_bloqueada=%d", project, score)
		if len(highlights) > 0 {
			fmt.Printf(" · causas %s", strings.Join(highlights, " | "))
		}
		fmt.Println()
	}
	saturados := make(map[string]bool, len(resumen.AgentesSaturados))
	for _, a := range resumen.AgentesSaturados {
		if a != nil {
			saturados[strings.TrimSpace(a.Nombre)] = true
		}
	}
	atascados := make(map[string]bool, len(resumen.AgentesAtascados))
	for _, a := range resumen.AgentesAtascados {
		if a != nil {
			atascados[strings.TrimSpace(a.Nombre)] = true
		}
	}
	for _, a := range resumen.AgentesActivos {
		marker := "🟢"
		if saturados[strings.TrimSpace(a.Nombre)] {
			marker = "🟡"
		} else if atascados[strings.TrimSpace(a.Nombre)] {
			marker = "🟠"
		}
		fmt.Printf("   %s %-15s [%s]", marker, a.Nombre, a.Rol)
		if cuenta := resumenCuentaAgente(a); cuenta != "" {
			fmt.Printf(" — %s", cuenta)
		}
		if detalle := resumenCuotaAgente(a); detalle != "" {
			fmt.Printf(" — %s", detalle)
		}
		fmt.Println()
	}
	if len(resumen.AgentesActivos) == 0 {
		fmt.Printf("   — sin agentes activos visibles\n")
	}
	if len(resumen.AgentesAuthManual) > 0 {
		fmt.Printf("   🔐 Requieren autenticación manual:\n")
		for _, a := range resumen.AgentesAuthManual {
			if a == nil {
				continue
			}
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
		if nextQuotaReset := nextQuotaResetVisible(agentesEnCuota); nextQuotaReset != "" {
			if resetAt, err := time.Parse(time.RFC3339, nextQuotaReset); err == nil {
				fmt.Printf("      Próximo reset visible: %s\n", resetAt.Local().Format("2006-01-02 15:04"))
			}
		}
	}
	agentesEnPausa := agentesNoActivosEnPausaOperativaConResumen(resumen.Agentes, resumen)
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

	if len(resumen.PoolsLocales) > 0 {
		fmt.Printf("🧠 Pools locales compartidos:\n")
		for _, pool := range resumen.PoolsLocales {
			if pool == nil {
				continue
			}
			modelo := strings.TrimSpace(pool.ModeloPreferente)
			if modelo == "" {
				modelo = "—"
			}
			slotsActivos := 0
			ready := 0
			working := 0
			failed := 0
			if pool.Telemetria != nil {
				slotsActivos = pool.Telemetria.SlotsActivos
				ready = pool.Telemetria.SesionesReady
				working = pool.Telemetria.SesionesWorking
				failed = pool.Telemetria.SesionesFailed
			}
			fmt.Printf("   %-18s modelo %-18s slots %d/%d · ready %d · working %d", pool.PoolSlug, modelo, slotsActivos, pool.SlotsMaximos, ready, working)
			if failed > 0 {
				fmt.Printf(" · failed %d", failed)
			}
			fmt.Println()
		}
		fmt.Println()
	}

	if resumen.DeudaDispatch.Total > 0 {
		fmt.Printf("📮 Dispatch durable: %d total · pending %d · notified %d · failed %d · confirmed %d\n\n",
			resumen.DeudaDispatch.Total,
			resumen.DeudaDispatch.Pendientes,
			resumen.DeudaDispatch.Notificadas,
			resumen.DeudaDispatch.Fallidas,
			resumen.DeudaDispatch.WorkConfirmed,
		)
	}
	if resumen.Autonomia.Supervisando > 0 || resumen.Autonomia.Continuando > 0 || resumen.Autonomia.ContinuidadPendiente > 0 || resumen.Autonomia.WorkConfirmed > 0 || resumen.Autonomia.Handoffs > 0 {
		fmt.Printf("🤖 Autonomía: supervisando %d · continuando %d · continuidad %d · confirmados %d · handoffs %d\n\n",
			resumen.Autonomia.Supervisando,
			resumen.Autonomia.Continuando,
			resumen.Autonomia.ContinuidadPendiente,
			resumen.Autonomia.WorkConfirmed,
			resumen.Autonomia.Handoffs,
		)
	}

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

	retenidas := tareasRetenidasPorCuotaConResumen(resumen.TareasActivas, resumen.Agentes, resumen.AgentesQuotaBlocked, resumen)
	retenidasIDs := make(map[int64]struct{}, len(retenidas))
	for _, t := range retenidas {
		retenidasIDs[t.ID] = struct{}{}
	}
	tareasEnProgresoBase := resumen.TareasEnProgreso
	if len(tareasEnProgresoBase) == 0 {
		tareasEnProgresoBase = filtrarOpenClawTareasPorEstado(resumen.TareasActivas, db.TareaEnProgreso)
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
		agentePorNombre := make(map[string]*db.Agente, len(resumen.Agentes)+len(resumen.AgentesQuotaBlocked))
		for _, agente := range resumen.Agentes {
			if agente == nil {
				continue
			}
			agentePorNombre[strings.TrimSpace(agente.Nombre)] = agente
		}
		for _, agente := range resumen.AgentesQuotaBlocked {
			if agente == nil {
				continue
			}
			nombre := strings.TrimSpace(agente.Nombre)
			if nombre == "" {
				continue
			}
			if _, ok := agentePorNombre[nombre]; !ok {
				agentePorNombre[nombre] = agente
			}
		}
		fmt.Printf("⏸️  Retenidas por cuota:\n")
		for _, t := range retenidas {
			agente := "—"
			if t.Agente != "" {
				agente = t.Agente
			}
			fmt.Printf("   [%d] %-40s → %s", t.ID, truncar(t.Titulo, 38), agente)
			if info := agentePorNombre[agente]; info != nil {
				if resetAt := cooldownVisibleAgente(info); resetAt != nil {
					fmt.Printf(" · reset %s", resetAt.Local().Format("2006-01-02 15:04"))
				}
			}
			fmt.Println()
		}
		fmt.Println()
	}
}

func resumenVisibleWorkerCounters(resumen *estadoResumen) (int, int, int) {
	if resumen == nil {
		return 0, 0, 0
	}
	if resumen.WorkersConectados > 0 || resumen.WorkersTrabajando > 0 || resumen.SupervisoresActivos > 0 {
		return resumen.WorkersConectados, resumen.WorkersTrabajando, resumen.SupervisoresActivos
	}
	return statusVisibleWorkerCounters(resumen.AgentesActivos, resumen.AgentesTrabajando, resumen.Autonomia)
}

func resolverAgentesActivosDesdeCompatibilidad(resumen *estadoResumen) {
	if resumen == nil || len(resumen.AgentesActivos) > 0 {
		return
	}
	if len(resumen.Agentes) == 0 {
		return
	}
	if resumenTieneSemanticaOperativaModerna(resumen) {
		return
	}
	if !resumenRequiereFallbackCompatibilidadCuota(resumen) {
		return
	}
	agentesActivos := make([]*db.Agente, 0, len(resumen.Agentes))
	for _, agente := range resumen.Agentes {
		if agente == nil || agenteBloqueadoPorCuotaVisible(agente) {
			continue
		}
		agentesActivos = append(agentesActivos, agente)
	}
	resumen.AgentesActivos = agentesActivos
	if len(resumen.AgentesTrabajando) == 0 {
		resumen.AgentesTrabajando = derivarAgentesTrabajando(resumen.AgentesActivos, resumen.TareasActivas)
	}
}

func resumenRequiereFallbackCompatibilidadCuota(resumen *estadoResumen) bool {
	if resumen == nil {
		return false
	}
	for _, agente := range resumen.Agentes {
		if agente == nil {
			continue
		}
		if strings.TrimSpace(agente.EstadoCuota) != "" && !strings.EqualFold(strings.TrimSpace(agente.EstadoCuota), "activo") {
			return true
		}
		if agente.ReanimarAt != nil {
			return true
		}
		if agente.CuotaRestantePct != nil || agente.PresupuestoSesionPct != nil || agente.PresupuestoDiarioPct != nil || agente.PresupuestoSemanalPct != nil {
			return true
		}
		if agente.PresupuestoCheckedAt != nil || agente.PresupuestoResetAt != nil {
			return true
		}
		if strings.TrimSpace(agente.MotivoPausa) != "" {
			return true
		}
		if strings.TrimSpace(agente.PresupuestoFuente) != "" {
			return true
		}
	}
	return false
}

func resumenTieneSemanticaOperativaModerna(resumen *estadoResumen) bool {
	if resumen == nil {
		return false
	}
	if len(resumen.AgentesQuotaBlocked) > 0 || len(resumen.AgentesAtascados) > 0 || len(resumen.AgentesAuthManual) > 0 {
		return true
	}
	if resumen.Autonomia.Supervisando > 0 || resumen.Autonomia.Continuando > 0 ||
		resumen.Autonomia.ContinuidadPendiente > 0 || resumen.Autonomia.WorkConfirmed > 0 ||
		resumen.Autonomia.Handoffs > 0 {
		return true
	}
	if resumen.DeudaDispatch.Total > 0 || resumen.DeudaDispatch.Pendientes > 0 ||
		resumen.DeudaDispatch.Notificadas > 0 || resumen.DeudaDispatch.Fallidas > 0 ||
		resumen.DeudaDispatch.WorkConfirmed > 0 {
		return true
	}
	if len(resumen.TareasEnProgreso) > 0 || len(resumen.TareasReservadas) > 0 {
		return true
	}
	return false
}

func agenteResumenVisibleParaCompatibilidad(nombre string, resumen *estadoResumen) bool {
	nombre = strings.TrimSpace(strings.ToLower(nombre))
	if nombre == "" {
		return false
	}
	if resumen == nil {
		return agenteVisibleEnStatusFleet(nombre)
	}
	if len(resumen.Agentes) > 0 && len(resumen.AgentesActivos) == len(resumen.Agentes) {
		return true
	}
	return agenteVisibleEnStatusFleet(nombre)
}

func resumenCuotaAgente(a *db.Agente) string {
	if a == nil {
		return ""
	}
	if db.AgenteSinCuotaProveedorEfectivo(a) {
		return "cuota indefinida · local"
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
	if a.ObservedUsageTokens != nil {
		usage := fmt.Sprintf("uso_obs %d tok", *a.ObservedUsageTokens)
		if a.ObservedUsageCostUSD != nil {
			usage += fmt.Sprintf(" · $%.4f", *a.ObservedUsageCostUSD)
		}
		if a.ObservedUsageMessages != nil {
			usage += fmt.Sprintf(" · %d msg", *a.ObservedUsageMessages)
		}
		if a.ObservedUsageTurns != nil {
			usage += fmt.Sprintf(" · %d turns", *a.ObservedUsageTurns)
		}
		if path := strings.TrimSpace(a.ObservedSessionPath); path != "" {
			usage += " · sesion " + filepath.Base(path)
		}
		partes = append(partes, usage)
	}
	if strings.EqualFold(strings.TrimSpace(a.PresupuestoFuente), "claude_rust_session_observed") &&
		a.RemainingSeconds == nil && a.RemainingMessages == nil && a.RemainingTokens == nil && a.RemainingCredits == nil && a.CuotaRestantePct == nil {
		partes = append(partes, "sin cuota real del proveedor")
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
		} else if bloqueoCuotaEstimadoVisible(a) {
			partes = append(partes, "bloqueo_estimado:"+a.EstadoCuota)
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
	return resumenCuentaCanonica(a.CuentaID, a.CuentaEmail, a.CuentaUsuario)
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
	return agentesNoActivosConCuotaConResumen(agentes, nil)
}

func agentesNoActivosConCuotaConResumen(agentes []*db.Agente, resumen *estadoResumen) []*db.Agente {
	out := make([]*db.Agente, 0, len(agentes))
	snapshotConHabilitado := snapshotExponeHabilitado(agentes)
	for _, agente := range agentes {
		if agente == nil || agente.Activo || !agenteCuentaComoHabilitadoEnSnapshot(agente, snapshotConHabilitado) {
			continue
		}
		if !agenteBloqueadoPorCuotaVisible(agente) {
			continue
		}
		if !agenteResumenVisibleParaCompatibilidad(agente.Nombre, resumen) {
			continue
		}
		out = append(out, agente)
	}
	return out
}

func agentesBloqueadosPorCuotaVisibles(resumen *estadoResumen) []*db.Agente {
	if resumen == nil {
		return nil
	}
	out := make([]*db.Agente, 0, len(resumen.Agentes)+len(resumen.AgentesQuotaBlocked))
	seen := make(map[string]struct{}, len(resumen.Agentes)+len(resumen.AgentesQuotaBlocked))
	snapshotConHabilitado := snapshotExponeHabilitado(resumen.Agentes)
	agentesPorNombre := make(map[string]*db.Agente, len(resumen.Agentes))
	for _, agente := range resumen.Agentes {
		if agente == nil {
			continue
		}
		agentesPorNombre[strings.ToLower(strings.TrimSpace(agente.Nombre))] = agente
	}
	add := func(agentes []*db.Agente) {
		for _, agente := range agentes {
			if agente == nil {
				continue
			}
			nombre := strings.ToLower(strings.TrimSpace(agente.Nombre))
			if nombre == "" {
				continue
			}
			if !agenteResumenVisibleParaCompatibilidad(nombre, resumen) {
				continue
			}
			if base := agentesPorNombre[nombre]; base != nil && !agenteCuentaComoHabilitadoEnSnapshot(base, snapshotConHabilitado) {
				continue
			}
			if _, ok := seen[nombre]; ok {
				continue
			}
			seen[nombre] = struct{}{}
			out = append(out, agente)
		}
	}
	add(resumen.AgentesQuotaBlocked)
	add(agentesNoActivosConCuotaConResumen(resumen.Agentes, resumen))
	return out
}

func agentesNoActivosEnPausaOperativa(agentes []*db.Agente) []*db.Agente {
	return agentesNoActivosEnPausaOperativaConResumen(agentes, nil)
}

func agentesNoActivosEnPausaOperativaConResumen(agentes []*db.Agente, resumen *estadoResumen) []*db.Agente {
	out := make([]*db.Agente, 0, len(agentes))
	for _, agente := range agentes {
		if agente == nil || agente.Activo {
			continue
		}
		if !agenteResumenVisibleParaCompatibilidad(agente.Nombre, resumen) {
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

func tareasRetenidasPorCuota(tareas []tareaLite, agentes []*db.Agente, quotaBlocked []*db.Agente) []tareaLite {
	return tareasRetenidasPorCuotaConResumen(tareas, agentes, quotaBlocked, nil)
}

func tareasRetenidasPorCuotaConResumen(tareas []tareaLite, agentes []*db.Agente, quotaBlocked []*db.Agente, resumen *estadoResumen) []tareaLite {
	if len(tareas) == 0 || (len(agentes) == 0 && len(quotaBlocked) == 0) {
		return nil
	}
	porNombre := make(map[string]*db.Agente, len(agentes)+len(quotaBlocked))
	snapshotConHabilitado := snapshotExponeHabilitado(agentes)
	for _, agente := range agentes {
		if agente == nil {
			continue
		}
		if !agenteResumenVisibleParaCompatibilidad(agente.Nombre, resumen) {
			continue
		}
		porNombre[strings.TrimSpace(agente.Nombre)] = agente
	}
	bloqueadosPorRuntime := make(map[string]struct{}, len(quotaBlocked))
	for _, agente := range quotaBlocked {
		if agente == nil {
			continue
		}
		nombre := strings.TrimSpace(agente.Nombre)
		if nombre == "" {
			continue
		}
		if !agenteResumenVisibleParaCompatibilidad(nombre, resumen) {
			continue
		}
		bloqueadosPorRuntime[nombre] = struct{}{}
		if _, ok := porNombre[nombre]; !ok {
			porNombre[nombre] = agente
		}
	}
	out := make([]tareaLite, 0)
	for _, tarea := range tareas {
		agente := porNombre[strings.TrimSpace(tarea.Agente)]
		if agente == nil || agente.Activo {
			continue
		}
		if !agenteCuentaComoHabilitadoEnSnapshot(agente, snapshotConHabilitado) {
			continue
		}
		if _, ok := bloqueadosPorRuntime[strings.TrimSpace(tarea.Agente)]; ok {
			out = append(out, tarea)
			continue
		}
		if !agenteBloqueadoPorCuotaVisible(agente) {
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
	if db.AgenteSinCuotaProveedorEfectivo(a) {
		return false
	}
	if !agenteTieneSenalesCuotaVisible(a) {
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
	tieneSenalVentanaCorta := a.PresupuestoSesionPct != nil ||
		presupuestoVisibleEsVentanaCorta(strings.TrimSpace(a.PresupuestoVentana)) ||
		strings.EqualFold(strings.TrimSpace(a.PresupuestoFuente), "provider_backoff") ||
		(a.ReanimarAt != nil && !a.ReanimarAt.IsZero()) ||
		strings.Contains(strings.ToLower(strings.TrimSpace(a.MotivoPausa)), "diaria") ||
		strings.Contains(strings.ToLower(strings.TrimSpace(a.MotivoPausa)), "ventana corta")
	if a.PresupuestoDiarioPct != nil && *a.PresupuestoDiarioPct <= 0 && tieneSenalVentanaCorta {
		return true
	}
	tieneVentanaTemporalVisible := a.PresupuestoSesionPct != nil ||
		(a.PresupuestoDiarioPct != nil && tieneSenalVentanaCorta) ||
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

func agenteTieneSenalesCuotaVisible(a *db.Agente) bool {
	if a == nil {
		return false
	}
	if strings.TrimSpace(a.EstadoCuota) != "" {
		return true
	}
	if strings.TrimSpace(a.MotivoPausa) != "" ||
		strings.TrimSpace(a.PresupuestoEstado) != "" ||
		strings.TrimSpace(a.PresupuestoVentana) != "" ||
		strings.TrimSpace(a.PresupuestoFuente) != "" {
		return true
	}
	if a.ReanimarAt != nil || a.PresupuestoCheckedAt != nil || a.PresupuestoResetAt != nil ||
		a.PresupuestoSemanalResetAt != nil || a.PresupuestoDiarioResetAt != nil || a.PresupuestoSesionResetAt != nil {
		return true
	}
	return a.CuotaRestantePct != nil ||
		a.PresupuestoSesionPct != nil ||
		a.PresupuestoDiarioPct != nil ||
		a.PresupuestoSemanalPct != nil ||
		a.RemainingCredits != nil ||
		a.RemainingMessages != nil ||
		a.RemainingTokens != nil ||
		a.RemainingSeconds != nil
}

func bloqueoCuotaEstimadoVisible(a *db.Agente) bool {
	if !agenteBloqueadoPorCuotaVisible(a) {
		return false
	}
	if agenteMotivoPausaOperativa(a.MotivoPausa) {
		return false
	}
	estadoPresupuesto := strings.ToLower(strings.TrimSpace(a.PresupuestoEstado))
	if estadoPresupuesto != "" && estadoPresupuesto != "ok" && estadoPresupuesto != "observado_stale" {
		return false
	}
	if a.RemainingCredits != nil && *a.RemainingCredits <= 0 {
		return false
	}
	if a.PresupuestoSemanalPct != nil && *a.PresupuestoSemanalPct <= 0 {
		return false
	}
	if a.PresupuestoSesionPct != nil && *a.PresupuestoSesionPct <= 0 {
		return false
	}
	if a.PresupuestoDiarioPct != nil && *a.PresupuestoDiarioPct <= 0 {
		return false
	}
	motivo := strings.ToLower(strings.TrimSpace(a.MotivoPausa))
	if a.ReanimarAt != nil && !a.ReanimarAt.IsZero() &&
		(strings.Contains(motivo, "diaria") || strings.Contains(motivo, "ventana corta")) {
		return false
	}
	return a.PresupuestoCheckedAt == nil || a.PresupuestoCheckedAt.IsZero() || a.PresupuestoStale
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

func fetchServerOperational(baseURL string) (*serverOperationalInfo, error) {
	var payload serverOperationalInfo
	if err := fetchServerJSON(baseURL+"/api/server/operational", &payload); err != nil {
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
		Generado                 string                              `json:"generado"`
		TareasPorEstado          map[string]int                      `json:"tareasPorEstado"`
		AgentesActivos           []*db.Agente                        `json:"agentesActivos"`
		AgentesTrabajando        []*db.Agente                        `json:"agentesTrabajando"`
		AgentesSaturados         []*db.Agente                        `json:"agentesSaturados"`
		AgentesAtascados         []*db.Agente                        `json:"agentesAtascados"`
		AgentesAuthManual        []*db.Agente                        `json:"agentesAuthManual"`
		AgentesQuotaBlocked      []*db.Agente                        `json:"agentesQuotaBlocked"`
		PropuestasAbiertas       []propuestaLite                     `json:"propuestasAbiertas"`
		TareasActivas            []tareaLite                         `json:"tareasActivas"`
		TareasEnProgreso         []tareaLite                         `json:"tareasEnProgreso"`
		TareasReservadas         []tareaLite                         `json:"tareasReservadas"`
		PoolsLocales             []*capacidadapp.PoolLocalCompartido `json:"poolsLocales"`
		DeudaDispatch            deudaDispatchResumen                `json:"deudaDispatch"`
		Autonomia                autonomiaResumen                    `json:"autonomia"`
		AutonomySurface          *autonomySurface                    `json:"autonomySurface"`
		AutonomyHighlights       []string                            `json:"autonomyHighlights"`
		CriticalProjectRisk      *workspaceAutonomyProjectSummary    `json:"criticalProjectRisk"`
		WorkersConectados        int                                 `json:"workersConectados"`
		WorkersTrabajando        int                                 `json:"workersTrabajando"`
		SupervisoresActivos      int                                 `json:"supervisoresActivos"`
		AgentesCompat            []*db.Agente                        `json:"agentes"`
		ConteoTareasCompat       map[string]int                      `json:"conteo_tareas"`
		PropuestasCompatAbiertas []*db.Propuesta                     `json:"propuestas_abiertas"`
	}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		return nil, err
	}
	resumen := &estadoResumen{
		Generado:            payload.Generado,
		Agentes:             payload.AgentesCompat,
		TareasPorEstado:     payload.TareasPorEstado,
		AgentesActivos:      payload.AgentesActivos,
		AgentesTrabajando:   payload.AgentesTrabajando,
		AgentesSaturados:    payload.AgentesSaturados,
		AgentesAtascados:    payload.AgentesAtascados,
		AgentesAuthManual:   payload.AgentesAuthManual,
		AgentesQuotaBlocked: payload.AgentesQuotaBlocked,
		PropuestasAbiertas:  payload.PropuestasAbiertas,
		TareasActivas:       payload.TareasActivas,
		TareasEnProgreso:    payload.TareasEnProgreso,
		TareasReservadas:    payload.TareasReservadas,
		PoolsLocales:        payload.PoolsLocales,
		DeudaDispatch:       payload.DeudaDispatch,
		Autonomia:           payload.Autonomia,
		AutonomySurface:     payload.AutonomySurface,
		WorkersConectados:   payload.WorkersConectados,
		WorkersTrabajando:   payload.WorkersTrabajando,
		SupervisoresActivos: payload.SupervisoresActivos,
	}
	resumen.TareasActivas = filtrarTareasActivasVisibles(resumen.TareasActivas, resumen.Agentes)
	if len(resumen.TareasPorEstado) == 0 {
		resumen.TareasPorEstado = payload.ConteoTareasCompat
	}
	resumen.TareasPorEstado = reconciliarConteoTareasActivasVisible(resumen.TareasPorEstado, resumen.TareasActivas)
	if len(resumen.AgentesActivos) == 0 {
		if _, present := raw["agentesActivos"]; !present && !resumenTieneSemanticaOperativaModerna(resumen) {
			resumen.AgentesActivos = payload.AgentesCompat
		}
	}
	if len(resumen.AgentesTrabajando) == 0 {
		resumen.AgentesTrabajando = derivarAgentesTrabajando(resumen.AgentesActivos, resumen.TareasActivas)
	}
	if len(resumen.TareasEnProgreso) == 0 {
		resumen.TareasEnProgreso = filtrarOpenClawTareasPorEstado(resumen.TareasActivas, db.TareaEnProgreso)
	}
	if len(resumen.TareasReservadas) == 0 {
		resumen.TareasReservadas = filtrarOpenClawTareasPorEstado(resumen.TareasActivas, db.TareaAsignada)
	}
	resumen.WorkersConectados, resumen.WorkersTrabajando, resumen.SupervisoresActivos = reconciledVisibleWorkerCounters(
		resumen.WorkersConectados,
		resumen.WorkersTrabajando,
		resumen.SupervisoresActivos,
		resumen.AgentesActivos,
		resumen.AgentesTrabajando,
		resumen.Autonomia,
	)
	if resumen.AutonomySurface == nil {
		if surface, err := fetchServerProjectAutonomySurface(baseURL); err == nil && surface != nil {
			resumen.AutonomySurface = surface
		}
	}
	resumen.AutonomySurface = enrichStatusAutonomySurfaceWithRisk(resumen.AutonomySurface, statusWorkspaceRiskSummary{
		CriticalProjectRisk: payload.CriticalProjectRisk,
		Highlights:          payload.AutonomyHighlights,
	})
	if payload.CriticalProjectRisk != nil {
		resumen.AutonomySurface = applyStructuredCriticalProjectRisk(resumen.AutonomySurface, payload.CriticalProjectRisk)
	}
	if resumen.AutonomySurface != nil {
		if control, criticalProjectRisk, err := fetchServerWorkspaceControl(baseURL); err == nil && control != nil {
			resumen.AutonomySurface = enrichAutonomySurfaceWithWorkspaceRisk(resumen.AutonomySurface, control, criticalProjectRisk)
			switch {
			case criticalProjectRisk != nil:
				resumen.AutonomySurface = applyStructuredCriticalProjectRisk(resumen.AutonomySurface, criticalProjectRisk)
			case payload.CriticalProjectRisk != nil:
				resumen.AutonomySurface = applyStructuredCriticalProjectRisk(resumen.AutonomySurface, payload.CriticalProjectRisk)
			}
		}
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

type workspaceControlCompatPayload struct {
	workspaceControlReport
	CriticalProjectRisk *workspaceAutonomyProjectSummary `json:"critical_project_risk,omitempty"`
}

type apiWorkspaceControlCompatResponse struct {
	Control *workspaceControlCompatPayload `json:"control"`
}

func fetchServerWorkspaceControl(baseURL string) (*workspaceControlReport, *workspaceAutonomyProjectSummary, error) {
	var payload apiWorkspaceControlCompatResponse
	if err := fetchServerJSON(baseURL+"/api/workspace/control", &payload); err != nil {
		return nil, nil, err
	}
	if payload.Control == nil {
		return nil, nil, nil
	}
	return &payload.Control.workspaceControlReport, payload.Control.CriticalProjectRisk, nil
}

func enrichAutonomySurfaceWithWorkspaceRisk(surface *autonomySurface, report *workspaceControlReport, criticalProjectRisk *workspaceAutonomyProjectSummary) *autonomySurface {
	if surface == nil || report == nil {
		return surface
	}
	surface.Highlights = mergeStatusRiskHighlights(surface.Highlights, report.AutonomyHighlights)
	if len(surface.Projects) > 0 && len(report.AutonomyProjects) > 0 {
		byProject := make(map[string]workspaceAutonomyProjectSummary, len(report.AutonomyProjects))
		for _, item := range report.AutonomyProjects {
			project := strings.TrimSpace(item.Project)
			if project == "" {
				continue
			}
			byProject[strings.ToLower(project)] = item
		}
		for i := range surface.Projects {
			project := strings.ToLower(strings.TrimSpace(surface.Projects[i].Project))
			if project == "" {
				continue
			}
			if risk, ok := byProject[project]; ok {
				surface.Projects[i].Highlights = mergeStatusRiskHighlights(surface.Projects[i].Highlights, risk.Highlights)
			}
		}
	}
	surface = applyStructuredCriticalProjectRisk(surface, criticalProjectRisk)
	return surface
}

func applyStructuredCriticalProjectRisk(surface *autonomySurface, critical *workspaceAutonomyProjectSummary) *autonomySurface {
	if surface == nil || critical == nil {
		return surface
	}
	project := strings.TrimSpace(critical.Project)
	if project == "" || critical.Blocking <= 0 {
		return surface
	}
	baseHighlights := filterStatusRiskScalarHighlights(surface.Highlights)
	baseHighlights = mergeStatusRiskHighlights(baseHighlights, []string{
		fmt.Sprintf("integracion_bloqueada=%d", critical.Blocking),
		fmt.Sprintf("riesgo_top=%s(%d)", project, critical.Blocking),
	})
	surface.Highlights = baseHighlights

	for i := range surface.Projects {
		if !strings.EqualFold(strings.TrimSpace(surface.Projects[i].Project), project) {
			continue
		}
		surface.Projects[i].Highlights = mergeStatusRiskHighlights(surface.Projects[i].Highlights, critical.Highlights)
		if surface.Projects[i].Events == 0 && critical.Events > 0 {
			surface.Projects[i].Events = critical.Events
		}
		return surface
	}

	surface.Projects = append(surface.Projects, autonomyProjectSurface{
		Project:    project,
		Events:     critical.Events,
		Highlights: append([]string(nil), critical.Highlights...),
	})
	return surface
}

func filterStatusRiskScalarHighlights(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if strings.HasPrefix(value, "integracion_bloqueada=") || strings.HasPrefix(value, "riesgo_top=") {
			continue
		}
		out = append(out, value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func mergeStatusRiskHighlights(base []string, extra []string) []string {
	if len(base) == 0 && len(extra) == 0 {
		return nil
	}
	out := make([]string, 0, len(base)+len(extra))
	seen := map[string]struct{}{}
	for _, item := range append(append([]string(nil), base...), extra...) {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func autonomySurfaceCriticalProjectRisk(surface *autonomySurface) (string, int, []string, bool) {
	if surface == nil {
		return "", 0, nil, false
	}
	project, score, ok := autonomySurfaceCriticalProjectToken(surface.Highlights)
	if !ok {
		return "", 0, nil, false
	}
	for _, item := range surface.Projects {
		if !strings.EqualFold(strings.TrimSpace(item.Project), project) {
			continue
		}
		return project, score, compactProjectControlIntegrationHighlights(item.Highlights), true
	}
	return project, score, nil, true
}

func autonomySurfaceCriticalProjectToken(highlights []string) (string, int, bool) {
	for _, item := range highlights {
		value := strings.TrimSpace(item)
		if !strings.HasPrefix(value, "riesgo_top=") {
			continue
		}
		raw := strings.TrimPrefix(value, "riesgo_top=")
		open := strings.LastIndex(raw, "(")
		close := strings.LastIndex(raw, ")")
		if open <= 0 || close <= open+1 || close != len(raw)-1 {
			return "", 0, false
		}
		project := strings.TrimSpace(raw[:open])
		score, err := strconv.Atoi(strings.TrimSpace(raw[open+1 : close]))
		if err != nil || project == "" || score <= 0 {
			return "", 0, false
		}
		return project, score, true
	}
	return "", 0, false
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
		if agenteBloqueadoPorCuotaVisible(agente) {
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
