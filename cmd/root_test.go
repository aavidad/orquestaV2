package cmd

import "testing"

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
