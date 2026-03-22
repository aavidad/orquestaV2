/*
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
*/

package cmd

import (
	"os"
	"testing"
)

func TestShouldDelegateToLocalServer(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		args []string
		want bool
	}{
		{name: "sin args", args: nil, want: false},
		{name: "help", args: []string{"help"}, want: false},
		{name: "completion", args: []string{"completion", "bash"}, want: false},
		{name: "server status", args: []string{"server", "status"}, want: false},
		{name: "server stop", args: []string{"server", "stop"}, want: false},
		{name: "server doctor", args: []string{"server", "doctor"}, want: false},
		{name: "server run", args: []string{"server", "run"}, want: false},
		{name: "serve", args: []string{"serve"}, want: false},
		{name: "status", args: []string{"status"}, want: true},
		{name: "tarea listar", args: []string{"tarea", "listar"}, want: true},
		{name: "flag local", args: []string{"--local", "status"}, want: false},
		{name: "flag fallback", args: []string{"--allow-local-fallback", "status"}, want: true},
		{name: "ayuda corta", args: []string{"status", "-h"}, want: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := shouldDelegateToLocalServer(tc.args); got != tc.want {
				t.Fatalf("shouldDelegateToLocalServer(%v)=%v, want %v", tc.args, got, tc.want)
			}
		})
	}
}

func TestCommandNeedsDBWithDelegationCoverage(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		args []string
		want bool
	}{
		{name: "sin args", args: nil, want: false},
		{name: "help", args: []string{"help"}, want: false},
		{name: "completion", args: []string{"completion", "bash"}, want: false},
		{name: "server status", args: []string{"server", "status"}, want: false},
		{name: "server stop", args: []string{"server", "stop"}, want: false},
		{name: "server doctor", args: []string{"server", "doctor"}, want: false},
		{name: "server run", args: []string{"server", "run"}, want: true},
		{name: "serve", args: []string{"serve"}, want: true},
		{name: "status", args: []string{"status"}, want: true},
		{name: "tarea listar", args: []string{"tarea", "listar"}, want: true},
		{name: "flag local server stop", args: []string{"--local", "server", "stop"}, want: false},
		{name: "flag local status", args: []string{"--local", "status"}, want: true},
		{name: "flag fallback status", args: []string{"--allow-local-fallback", "status"}, want: true},
		{name: "help corto", args: []string{"status", "-h"}, want: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := commandNeedsDB(tc.args); got != tc.want {
				t.Fatalf("commandNeedsDB(%v)=%v, want %v", tc.args, got, tc.want)
			}
		})
	}
}

func TestAllowLocalFallback(t *testing.T) {
	t.Parallel()

	prev := os.Getenv("ORQUESTA_ALLOW_LOCAL_FALLBACK")
	t.Cleanup(func() {
		if prev == "" {
			_ = os.Unsetenv("ORQUESTA_ALLOW_LOCAL_FALLBACK")
			return
		}
		_ = os.Setenv("ORQUESTA_ALLOW_LOCAL_FALLBACK", prev)
	})

	if allowLocalFallback([]string{"status"}) {
		t.Fatalf("no deberia permitir fallback por defecto")
	}
	if !allowLocalFallback([]string{"--allow-local-fallback", "status"}) {
		t.Fatalf("deberia permitir fallback por flag")
	}
	if err := os.Setenv("ORQUESTA_ALLOW_LOCAL_FALLBACK", "1"); err != nil {
		t.Fatalf("Setenv: %v", err)
	}
	if !allowLocalFallback([]string{"status"}) {
		t.Fatalf("deberia permitir fallback por env")
	}
}
