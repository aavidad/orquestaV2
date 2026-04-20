/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"orquesta/db"
	"orquesta/internal/rpclocal"
	"orquesta/planocontrol"
)

var listenServerTCP = net.Listen

type unifiedServerRuntimeOptions struct {
	CoreOnly                 bool
	DisableAutobootstrap     bool
	DisableWarmWorker        bool
	DisableColdWorker        bool
	DisableNotificationRetry bool
}

type unifiedServerHTTPTimeouts struct {
	ReadHeader time.Duration
	Read       time.Duration
	Write      time.Duration
	Idle       time.Duration
}

func registrarRutasServe(mux *http.ServeMux) {
	mux.HandleFunc("/", webHandlerDash)
	mux.HandleFunc("/tareas", webHandlerTareas)
	mux.HandleFunc("/tareas/", webRouterTareas)
	mux.HandleFunc("/propuestas", webHandlerPropuestas)
	mux.HandleFunc("/propuestas/", webRouterPropuestas)
	mux.HandleFunc("/asignaciones", webHandlerAsignaciones)
	mux.HandleFunc("/sesiones", webHandlerSesiones)
	mux.HandleFunc("/sesiones/", webRouterSesiones)
	mux.HandleFunc("/proyectos", webHandlerProyectos)
	mux.HandleFunc("/proyectos/", webRouterProyectos)
	mux.HandleFunc("/nueva-app", webHandlerNuevaApp)
	mux.HandleFunc("/progreso", webHandlerProgreso)
	mux.HandleFunc("/progreso/", webRouterProgreso)
	mux.HandleFunc("/pools", webHandlerPools)
	mux.HandleFunc("/pools/", webRouterPools)
	mux.HandleFunc("/modelo", webHandlerModelo)
	mux.HandleFunc("/modelo/politicas/guardar", webHandlerModeloPoliticaGuardar)
	mux.HandleFunc("/memoria", webHandlerMemoria)
	mux.HandleFunc("/memoria/", webRouterMemoria)
	mux.HandleFunc("/deploy", webHandlerDeploy)
	mux.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			webHandlerConfigGuardar(w, r)
			return
		}
		webHandlerConfig(w, r)
	})
	mux.HandleFunc("/openclaw", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			webHandlerOpenClawAccion(w, r)
			return
		}
		webHandlerOpenClaw(w, r)
	})
	mux.HandleFunc("/conectores", webHandlerConectores)
	mux.HandleFunc("/conectores/", webRouterConectores)
	mux.HandleFunc("/diagnostico", webHandlerDiagnostico)
	mux.HandleFunc("/auditoria", webHandlerAuditoria)
	mux.HandleFunc("/refineria", webHandlerRefineria)
	mux.HandleFunc("/refineria/", webRouterRefineria)
	mux.HandleFunc("/respaldo", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			webHandlerRespaldoCrear(w, r)
			return
		}
		webHandlerRespaldo(w, r)
	})
	mux.HandleFunc("/gobernanza", webHandlerGobernanza)
	mux.HandleFunc("/gobernanza/", webRouterGobernanza)
	mux.HandleFunc("/git", webHandlerGitGov)
	mux.HandleFunc("/git/", webRouterGitGov)
	mux.HandleFunc("/agentes", webHandlerAgentes)
	mux.HandleFunc("/agentes/", webRouterAgentes)
	mux.HandleFunc("/lenguaje", webHandlerLenguaje)
	mux.HandleFunc("/lenguaje/", webRouterLenguaje)
	mux.HandleFunc("/runtimes", webHandlerRuntimes)
	mux.HandleFunc("/runtimes/", webRouterRuntimes)
	mux.HandleFunc("/time-travel", webHandlerTimeTravel)
	mux.HandleFunc("/time-travel/", webRouterTimeTravel)
	registerAPIRoutes(mux)
}

func montarRPCLocalEnMux(mux *http.ServeMux, kind, listenAddr string) (func(), error) {
	if mux == nil {
		return func() {}, fmt.Errorf("mux obligatorio")
	}
	info, err := newLocalRPCState(kind, normalizarAddrServidorLocal(listenAddr))
	if err != nil {
		return func() {}, err
	}
	if err := rpclocal.SaveServerInfo(info); err != nil {
		return func() {}, err
	}
	rpcMux := rpclocal.NewMux(&rpclocal.Server{State: info, Executor: executeRPCRequest})
	mux.Handle(rpclocal.HealthPath, rpcMux)
	mux.Handle(rpclocal.ExecPath, rpcMux)
	return func() {
		_ = rpclocal.RemoveState("")
	}, nil
}

func arrancarServidorUnificado(listenAddr, kind string, anunciar bool, debug serverDebugOptions, security serverSecurityOptions) error {
	mux := http.NewServeMux()
	registrarRutasServe(mux)

	listener, advertisedAddr, err := escucharServidorUnificado(listenAddr)
	if err != nil {
		return err
	}
	defer listener.Close()

	cleanupRPC := func() {}
	if shouldExposeLocalRPC(kind, advertisedAddr, security) {
		cleanupRPC, err = montarRPCLocalEnMux(mux, kind, advertisedAddr)
		if err != nil {
			return err
		}
	}
	defer cleanupRPC()

	if err := ensureServerDBOpen(); err != nil {
		return err
	}

	if anunciar {
		fmt.Printf("✓ Panel web en %s\n", publicServerURL(advertisedAddr, security))
		fmt.Println("  Ctrl+C para detener.")
	} else {
		fmt.Printf("Servidor local de Orquesta en %s\n", rpclocal.BaseURL(advertisedAddr))
	}
	_ = os.Setenv("ORQUESTA_SERVER_URL", rpclocal.BaseURL(advertisedAddr))

	var debugLogger *log.Logger
	if debug.Enabled {
		debugLogger = newServerDebugLogger(kind)
		debugLogger.Printf("startup addr=%s rpc=%s %s", listenAddr, rpclocal.BaseURL(normalizarAddrServidorLocal(listenAddr)), debugDurationsSummary(time.Minute, time.Minute, 30*time.Second, 15*time.Second))
	}
	prewarmUnifiedServerPrepareCaches(debugLogger)

	controlCtx, cancel := context.WithCancel(context.Background())
	runner := newControlPlaneRunner(debugLogger, debug.ControlPlane)
	runner.StartupGrace = 5 * time.Second
	runtimeOptions := loadUnifiedServerRuntimeOptions()
	unregisterRunner := registerActiveControlPlaneRunner(runner)
	defer runner.Wait()
	defer unregisterRunner()
	defer cancel()
	if runtimeOptions.CoreOnly {
		// Core-only mantiene apagados los carriles residentes pesados, pero el
		// núcleo runtime debe seguir vivo para consolidar start/resume vía
		// transcript, mailbox y observación de presupuesto.
		runner.StartRuntimeCore(controlCtx)
	} else {
		runner.StartResidentCore(controlCtx)
		runner.StartNonResidentWorkerWithOptions(controlCtx, planocontrol.NonResidentWorkerOptions{
			Warm:              !runtimeOptions.DisableWarmWorker,
			Cold:              !runtimeOptions.DisableColdWorker,
			NotificationRetry: !runtimeOptions.DisableNotificationRetry,
		})
	}
	if !runtimeOptions.DisableAutobootstrap {
		launchBootstrapServerAutonomy(controlCtx, debugLogger)
	} else if debugLogger != nil {
		debugLogger.Printf("server_autobootstrap skipped=disabled")
	}
	launchPrepareContextPrewarmLoop(controlCtx, debugLogger)
	launchStorageMaintenanceLoop(controlCtx, debugLogger)
	launchServerAutonomyMaintenanceLoop(controlCtx, debugLogger)

	timeouts := loadUnifiedServerHTTPTimeouts()
	server := &http.Server{
		Addr:              advertisedAddr,
		Handler:           wrapServeMuxWithDebug(mux, debug, debugLogger),
		ReadHeaderTimeout: timeouts.ReadHeader,
		ReadTimeout:       timeouts.Read,
		WriteTimeout:      timeouts.Write,
		IdleTimeout:       timeouts.Idle,
	}
	if strings.TrimSpace(security.TLSCert) == "" || strings.TrimSpace(security.TLSKey) == "" {
		return server.Serve(listener)
	}

	tlsConfig, err := buildServerTLSConfig(security)
	if err != nil {
		return err
	}
	server.TLSConfig = tlsConfig
	return server.ServeTLS(listener, security.TLSCert, security.TLSKey)
}

func loadUnifiedServerRuntimeOptions() unifiedServerRuntimeOptions {
	coreOnly := envBoolServidorUnificado("ORQUESTA_SERVER_CORE_ONLY") ||
		envBoolServidorUnificado("ORQUESTA_SERVER_DISABLE_NONRESIDENT_WORKER")
	disableAutobootstrap := coreOnly || envBoolServidorUnificado("ORQUESTA_SERVER_DISABLE_AUTOBOOTSTRAP")
	disableWarmWorker := coreOnly || envBoolServidorUnificado("ORQUESTA_SERVER_DISABLE_WARM_WORKER")
	disableColdWorker := coreOnly || envBoolServidorUnificado("ORQUESTA_SERVER_DISABLE_COLD_WORKER")
	disableNotificationRetry := coreOnly || envBoolServidorUnificado("ORQUESTA_SERVER_DISABLE_NOTIFICATION_RETRY")
	return unifiedServerRuntimeOptions{
		CoreOnly:                 coreOnly,
		DisableAutobootstrap:     disableAutobootstrap,
		DisableWarmWorker:        disableWarmWorker,
		DisableColdWorker:        disableColdWorker,
		DisableNotificationRetry: disableNotificationRetry,
	}
}

func prewarmUnifiedServerPrepareCaches(debugLogger *log.Logger) {
	cfg := loadServerAutobootstrapConfig()
	if !cfg.Enabled || strings.TrimSpace(cfg.ProjectSlug) == "" {
		return
	}
	agents := make([]string, 0, 1+len(cfg.WorkerAgents))
	if supervisor := strings.TrimSpace(cfg.SupervisorAgent); supervisor != "" {
		agents = append(agents, supervisor)
	}
	agents = append(agents, cfg.WorkerAgents...)
	seen := map[string]struct{}{}
	for _, agente := range agents {
		agente = strings.TrimSpace(agente)
		if agente == "" {
			continue
		}
		key := strings.ToLower(agente)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if _, _, _, err := db.ResolverPerfilEjecucionLanzamiento(&agente, cfg.ProjectSlug, "", "", ""); err != nil {
			if debugLogger != nil {
				debugLogger.Printf("model_policy_prewarm agente=%s proyecto=%s err=%v", agente, cfg.ProjectSlug, err)
			}
			continue
		}
		if err := agentesService.PrewarmPrepareContext(agente, cfg.ProjectSlug); err != nil {
			if debugLogger != nil {
				debugLogger.Printf("prepare_context_prewarm agente=%s proyecto=%s err=%v", agente, cfg.ProjectSlug, err)
			}
			continue
		}
		if debugLogger != nil {
			debugLogger.Printf("prepare_context_prewarm agente=%s proyecto=%s ok", agente, cfg.ProjectSlug)
		}
	}
}

func launchPrepareContextPrewarmLoop(ctx context.Context, debugLogger *log.Logger) {
	cfg := loadServerAutobootstrapConfig()
	if !cfg.Enabled || strings.TrimSpace(cfg.ProjectSlug) == "" {
		return
	}
	agents := make([]string, 0, 1+len(cfg.WorkerAgents))
	if supervisor := strings.TrimSpace(cfg.SupervisorAgent); supervisor != "" {
		agents = append(agents, supervisor)
	}
	agents = append(agents, cfg.WorkerAgents...)
	go func() {
		ticker := time.NewTicker(4 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			for _, agente := range agents {
				agente = strings.TrimSpace(agente)
				if agente == "" {
					continue
				}
				if err := agentesService.PrewarmPrepareContext(agente, cfg.ProjectSlug); err != nil && debugLogger != nil {
					debugLogger.Printf("prepare_context_prewarm_loop agente=%s proyecto=%s err=%v", agente, cfg.ProjectSlug, err)
				}
			}
		}
	}()
}

func loadUnifiedServerHTTPTimeouts() unifiedServerHTTPTimeouts {
	return unifiedServerHTTPTimeouts{
		ReadHeader: 5 * time.Second,
		Read:       15 * time.Second,
		Write:      30 * time.Second,
		Idle:       60 * time.Second,
	}
}

func envBoolServidorUnificado(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "si", "sí", "on":
		return true
	default:
		return false
	}
}

func launchBootstrapServerAutonomy(ctx context.Context, debugLogger *log.Logger) {
	go func() {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if err := bootstrapServerAutonomy(); err != nil {
			select {
			case <-ctx.Done():
				return
			default:
			}
			db.Audit("orquesta", "server_autobootstrap_error", "proyecto", 0, err.Error())
			if debugLogger != nil {
				debugLogger.Printf("server_autobootstrap error=%v", err)
			}
		}
	}()
}

func launchStorageMaintenanceLoop(ctx context.Context, debugLogger *log.Logger) {
	go func() {
		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				handle := db.CurrentHandle()
				backend, cfg, _ := db.CurrentPersistenceState()
				if handle == nil || backend == nil {
					continue
				}
				maintainer, ok := backend.(db.BackgroundMaintainer)
				if !ok {
					continue
				}
				outcome, err := maintainer.MaintenanceTick(handle.DB, cfg)
				if err != nil {
					if debugLogger != nil {
						debugLogger.Printf("storage_maintenance backend=%s error=%v", backend.Name(), err)
					}
					continue
				}
				if debugLogger != nil && len(outcome) > 0 {
					debugLogger.Printf("storage_maintenance backend=%s outcome=%v", backend.Name(), outcome)
				}
			}
		}
	}()
}

func serverAutonomyMaintenanceInterval() time.Duration {
	seconds := controlPlaneConfigIntOrDefault("server_autonomy_maintenance_interval_seconds", 30)
	if seconds < 30 {
		seconds = 30
	}
	return time.Duration(seconds) * time.Second
}

func launchServerAutonomyMaintenanceLoop(ctx context.Context, debugLogger *log.Logger) {
	cfg := loadServerAutobootstrapConfig()
	if !cfg.Enabled {
		return
	}
	runOnce := func() {
		started := runtimeProcessDegradadosEnCurso.CompareAndSwap(false, true)
		if !started {
			return
		}
		defer runtimeProcessDegradadosEnCurso.Store(false)
		summary, err := runtimeProcessDegradadosBatchDetailed()
		if err != nil {
			db.Audit("server", "runtime_process_degradados_loop_error", "runtime", 0, err.Error())
			if debugLogger != nil {
				debugLogger.Printf("runtime_process_degradados_loop error=%v", err)
			}
			return
		}
		if debugLogger != nil && summary.Count > 0 {
			debugLogger.Printf("runtime_process_degradados_loop count=%d ghost=%d reactivated=%d idle_autoassigned=%d",
				summary.Count, summary.GhostAssignmentsCompacted, summary.ReactivatedWithoutRuntime, summary.IdleAutoassigned)
		}
	}
	go func() {
		ticker := time.NewTicker(serverAutonomyMaintenanceInterval())
		defer ticker.Stop()
		runOnce()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			runOnce()
		}
	}()
}

func normalizarAddrServidorLocal(listenAddr string) string {
	addr := strings.TrimSpace(listenAddr)
	addr = strings.TrimPrefix(rpclocal.BaseURL(addr), "http://")
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		if strings.HasPrefix(addr, ":") {
			return rpclocal.DefaultHost + addr
		}
		return addr
	}
	if strings.TrimSpace(host) == "" || host == "0.0.0.0" || host == "::" {
		host = rpclocal.DefaultHost
	}
	return net.JoinHostPort(host, port)
}

func escucharServidorUnificado(listenAddr string) (net.Listener, string, error) {
	listener, err := listenServerTCP("tcp", strings.TrimSpace(listenAddr))
	if err != nil {
		return nil, "", err
	}
	return listener, normalizarAddrServidorLocal(listener.Addr().String()), nil
}

func buildServerTLSConfig(security serverSecurityOptions) (*tls.Config, error) {
	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	if strings.TrimSpace(security.TLSClientCA) == "" {
		return cfg, nil
	}

	raw, err := os.ReadFile(security.TLSClientCA)
	if err != nil {
		return nil, fmt.Errorf("leyendo tls-client-ca: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(raw) {
		return nil, fmt.Errorf("tls-client-ca inválida: %s", security.TLSClientCA)
	}
	cfg.ClientCAs = pool
	cfg.ClientAuth = tls.RequireAndVerifyClientCert
	return cfg, nil
}

func publicServerURL(listenAddr string, security serverSecurityOptions) string {
	scheme := "http"
	if strings.TrimSpace(security.TLSCert) != "" && strings.TrimSpace(security.TLSKey) != "" {
		scheme = "https"
	}
	return scheme + "://" + normalizarAddrServidorLocal(listenAddr)
}

func shouldExposeLocalRPC(kind, listenAddr string, security serverSecurityOptions) bool {
	if strings.TrimSpace(kind) == "server" {
		return true
	}
	if strings.TrimSpace(security.TLSCert) != "" || strings.TrimSpace(security.TLSKey) != "" || strings.TrimSpace(security.TLSClientCA) != "" {
		return false
	}
	addr := strings.TrimSpace(strings.TrimPrefix(rpclocal.BaseURL(listenAddr), "http://"))
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		if strings.HasPrefix(addr, ":") {
			host = ""
		} else {
			host = addr
		}
	}
	switch strings.TrimSpace(strings.ToLower(host)) {
	case "", "127.0.0.1", "localhost", "::1":
		return true
	case "0.0.0.0", "::":
		return false
	default:
		return false
	}
}
