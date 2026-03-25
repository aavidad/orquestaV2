/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
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

func arrancarServidorUnificado(listenAddr, kind string, anunciar bool, debug serverDebugOptions) error {
	if err := ensureServerDBOpen(); err != nil {
		return err
	}

	mux := http.NewServeMux()
	registrarRutasServe(mux)
	cleanupRPC, err := montarRPCLocalEnMux(mux, kind, listenAddr)
	if err != nil {
		return err
	}
	defer cleanupRPC()

	if anunciar {
		fmt.Printf("✓ Panel web en http://localhost%s\n", listenAddr)
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

	return http.ListenAndServe(listenAddr, wrapServeMuxWithDebug(mux, debug, debugLogger))
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
