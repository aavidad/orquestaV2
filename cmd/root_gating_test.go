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
		{name: "status sin opt-in", args: []string{"status"}, want: true},
		{name: "tarea listar sin opt-in", args: []string{"tarea", "listar"}, want: true},
		{name: "status con flag", args: []string{"--use-localrpc", "status"}, want: true},
		{name: "flag local", args: []string{"--local", "status"}, want: false},
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

func TestShouldDelegateToLocalServerExcluyeComandosDeRecuperacion(t *testing.T) {
	t.Parallel()

	if !shouldDelegateToLocalServer([]string{"status"}) {
		t.Fatalf("deberia delegar los comandos con cobertura server-first")
	}
	if shouldDelegateToLocalServer([]string{"serve"}) {
		t.Fatalf("serve no deberia delegar aunque exista daemon")
	}
	if shouldDelegateToLocalServer([]string{"persistencia", "info"}) {
		t.Fatalf("persistencia info no deberia delegar porque es ruta de diagnostico local")
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

func TestLocalRPCEnabled(t *testing.T) {
	t.Parallel()

	prev := os.Getenv("ORQUESTA_USE_LOCALRPC")
	t.Cleanup(func() {
		if prev == "" {
			_ = os.Unsetenv("ORQUESTA_USE_LOCALRPC")
			return
		}
		_ = os.Setenv("ORQUESTA_USE_LOCALRPC", prev)
	})

	if localRPCEnabled([]string{"status"}) {
		t.Fatalf("no deberia activarse por defecto")
	}
	if !localRPCEnabled([]string{"--use-localrpc", "status"}) {
		t.Fatalf("deberia activarse por flag")
	}
	if err := os.Setenv("ORQUESTA_USE_LOCALRPC", "1"); err != nil {
		t.Fatalf("Setenv: %v", err)
	}
	if !localRPCEnabled([]string{"status"}) {
		t.Fatalf("deberia activarse por env")
	}
}
