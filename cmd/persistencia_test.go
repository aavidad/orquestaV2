/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"orquesta/db"
	"orquesta/internal/rpclocal"
)

func TestCommandNeedsDBPersistenciaInfo(t *testing.T) {
	t.Parallel()

	if got := commandNeedsDB([]string{"persistencia", "info"}); got {
		t.Fatalf("persistencia info no deberia requerir DB")
	}
	if got := commandNeedsDB([]string{"persistencia", "verificar"}); got {
		t.Fatalf("persistencia verificar no deberia requerir DB")
	}
	if got := skipRemoteDelegation([]string{"persistencia", "info"}); !got {
		t.Fatalf("persistencia info deberia saltarse la delegacion remota")
	}
	if got := skipRemoteDelegation([]string{"persistencia", "verificar"}); !got {
		t.Fatalf("persistencia verificar deberia saltarse la delegacion remota")
	}
}

func TestCommandNeedsDBPersistenciaHelp(t *testing.T) {
	t.Parallel()

	if got := commandNeedsDB([]string{"persistencia", "info", "--help"}); got {
		t.Fatalf("persistencia info --help no deberia requerir DB")
	}
}

func TestRenderPersistenciaInfoSinServidor(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	renderPersistenciaInfo(
		&buf,
		"/tmp/orquesta-localrpc.json",
		"sqlite",
		"/tmp/orquesta.db",
		"http://127.0.0.1:17899",
		nil,
		false,
		errors.New("state ausente"),
		errors.New("connection refused"),
	)

	out := buf.String()
	for _, fragment := range []string{
		"Modo esperado: servidor local",
		"Storage driver: sqlite",
		"Storage target: /tmp/orquesta.db",
		"Servidor:      sin state",
		"Health RPC:    KO",
		"Ruta activa:   sin servidor; local solo con",
	} {
		if !strings.Contains(out, fragment) {
			t.Fatalf("salida sin fragmento %q: %s", fragment, out)
		}
	}
}

func TestRenderPersistenciaInfoConServidor(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	renderPersistenciaInfo(
		&buf,
		"/tmp/orquesta-localrpc.json",
		"sqlite",
		"/tmp/orquesta.db",
		"http://127.0.0.1:17899",
		&rpclocal.ServerInfo{
			Addr:          "127.0.0.1:17899",
			PID:           42,
			StorageDriver: "sqlite",
			StorageTarget: "/tmp/orquesta.db",
			StartedAt:     time.Date(2026, 3, 22, 18, 0, 0, 0, time.UTC),
			Version:       "dev",
		},
		false,
		nil,
		nil,
	)

	out := buf.String()
	for _, fragment := range []string{
		"Servidor:      pid=42 addr=127.0.0.1:17899 storage=sqlite /tmp/orquesta.db",
		"Health RPC:    OK",
		"Ruta activa:   servidor local",
	} {
		if !strings.Contains(out, fragment) {
			t.Fatalf("salida sin fragmento %q: %s", fragment, out)
		}
	}
}

func TestRenderPersistenciaInfoConFallbackHealthz(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	renderPersistenciaInfo(
		&buf,
		"/tmp/orquesta-localrpc.json",
		"sqlite",
		"/tmp/orquesta.db",
		"http://127.0.0.1:17899",
		&rpclocal.ServerInfo{
			Addr:          "127.0.0.1:17899",
			PID:           42,
			StorageDriver: "sqlite",
			StorageTarget: "/tmp/orquesta.db",
			StartedAt:     time.Date(2026, 3, 22, 18, 0, 0, 0, time.UTC),
			Version:       "dev",
		},
		true,
		nil,
		nil,
	)

	out := buf.String()
	if !strings.Contains(out, "Descubrimiento: fallback a healthz por statefile ausente") {
		t.Fatalf("salida sin nota de fallback: %s", out)
	}
}

func TestRenderVerificacionPersistencia(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	renderVerificacionPersistencia(&buf, &db.InformePersistencia{
		Driver: "sqlite",
		Target: "/tmp/orquesta.db",
		Sano:   false,
		Comprobaciones: []db.ComprobacionPersistencia{
			{Nombre: "conexion", Estado: "ok", Detalle: "ping correcto"},
			{Nombre: "schema_core", Estado: "error", Detalle: "faltan tablas core: runtime_orders"},
		},
	})

	out := buf.String()
	for _, fragment := range []string{
		"Storage driver: sqlite",
		"Storage target: /tmp/orquesta.db",
		"Estado general: ERROR",
		"- [OK] conexion: ping correcto",
		"- [ERROR] schema_core: faltan tablas core: runtime_orders",
	} {
		if !strings.Contains(out, fragment) {
			t.Fatalf("salida sin fragmento %q: %s", fragment, out)
		}
	}
}
