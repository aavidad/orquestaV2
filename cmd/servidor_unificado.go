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

	"orquesta/internal/rpclocal"
)

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
	if err := ensureServerDBOpen(); err != nil {
		return err
	}

	mux := http.NewServeMux()
	registrarRutasServe(mux)
	cleanupRPC := func() {}
	if shouldExposeLocalRPC(kind, listenAddr, security) {
		var err error
		cleanupRPC, err = montarRPCLocalEnMux(mux, kind, listenAddr)
		if err != nil {
			return err
		}
	}
	defer cleanupRPC()

	if anunciar {
		fmt.Printf("✓ Panel web en %s\n", publicServerURL(listenAddr, security))
		fmt.Println("  Ctrl+C para detener.")
	} else {
		fmt.Printf("Servidor local de Orquesta en %s\n", rpclocal.BaseURL(normalizarAddrServidorLocal(listenAddr)))
	}

	var debugLogger *log.Logger
	if debug.Enabled {
		debugLogger = newServerDebugLogger(kind)
		debugLogger.Printf("startup addr=%s rpc=%s %s", listenAddr, rpclocal.BaseURL(normalizarAddrServidorLocal(listenAddr)), debugDurationsSummary(time.Minute, time.Minute, 30*time.Second, 15*time.Second))
	}

	controlCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	newControlPlaneRunner(debugLogger, debug.ControlPlane).Start(controlCtx)

	server := &http.Server{
		Addr:    listenAddr,
		Handler: wrapServeMuxWithDebug(mux, debug, debugLogger),
	}
	if strings.TrimSpace(security.TLSCert) == "" || strings.TrimSpace(security.TLSKey) == "" {
		return server.ListenAndServe()
	}

	tlsConfig, err := buildServerTLSConfig(security)
	if err != nil {
		return err
	}
	server.TLSConfig = tlsConfig
	return server.ListenAndServeTLS(security.TLSCert, security.TLSKey)
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
