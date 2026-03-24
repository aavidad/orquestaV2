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

	"orquesta/internal/rpclocal"
)

func TestCommandNeedsDBPersistenciaInfo(t *testing.T) {
	t.Parallel()

	if got := commandNeedsDB([]string{"persistencia", "info"}); got {
		t.Fatalf("persistencia info no deberia requerir DB")
	}
	if got := skipRemoteDelegation([]string{"persistencia", "info"}); !got {
		t.Fatalf("persistencia info deberia saltarse la delegacion remota")
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
		"/tmp/orquesta.db",
		"http://127.0.0.1:17899",
		nil,
		errors.New("state ausente"),
		errors.New("connection refused"),
	)

	out := buf.String()
	for _, fragment := range []string{
		"Modo esperado: servidor local",
		"DB objetivo:   /tmp/orquesta.db",
		"Servidor:      sin state",
		"Health RPC:    KO",
		"Ruta activa:   sin servidor; local solo con --local",
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
		"/tmp/orquesta.db",
		"http://127.0.0.1:17899",
		&rpclocal.ServerInfo{
			Addr:      "127.0.0.1:17899",
			PID:       42,
			StartedAt: time.Date(2026, 3, 22, 18, 0, 0, 0, time.UTC),
			Version:   "dev",
		},
		nil,
		nil,
	)

	out := buf.String()
	for _, fragment := range []string{
		"Servidor:      pid=42 addr=127.0.0.1:17899",
		"Health RPC:    OK",
		"Ruta activa:   servidor local",
	} {
		if !strings.Contains(out, fragment) {
			t.Fatalf("salida sin fragmento %q: %s", fragment, out)
		}
	}
}
