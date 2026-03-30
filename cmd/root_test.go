package cmd

import (
	"testing"

	"github.com/spf13/pflag"
)

func TestCommandNeedsDB(t *testing.T) {
	cases := []struct {
		args []string
		want bool
	}{
		{args: nil, want: false},
		{args: []string{"help"}, want: false},
		{args: []string{"completion"}, want: false},
		{args: []string{"server", "status"}, want: false},
		{args: []string{"server", "stop"}, want: false},
		{args: []string{"server", "doctor"}, want: false},
		{args: []string{"server", "run"}, want: true},
		{args: []string{"serve"}, want: true},
		{args: []string{"status"}, want: true},
		{args: []string{"tarea", "listar"}, want: true},
		{args: []string{"--local", "status"}, want: true},
		{args: []string{"server", "status", "--help"}, want: false},
	}

	for _, tc := range cases {
		if got := commandNeedsDB(tc.args); got != tc.want {
			t.Fatalf("commandNeedsDB(%v) = %v, want %v", tc.args, got, tc.want)
		}
	}
}

func TestNormalizedCommandArgs(t *testing.T) {
	got := normalizedCommandArgs([]string{"--local", "tarea", "listar"})
	if len(got) != 2 || got[0] != "tarea" || got[1] != "listar" {
		t.Fatalf("normalizedCommandArgs inesperado: %v", got)
	}
}

func TestNormalizedFlagResetValueStringSlice(t *testing.T) {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.StringSlice("estado", []string{"cerrado", "fallido"}, "demo")
	flag := flags.Lookup("estado")
	if flag == nil {
		t.Fatal("flag estado no encontrada")
	}
	if got := normalizedFlagResetValue(flag); got != "cerrado,fallido" {
		t.Fatalf("normalizedFlagResetValue(stringSlice)=%q, want %q", got, "cerrado,fallido")
	}
}
