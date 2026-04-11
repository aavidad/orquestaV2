package cmd

import "testing"

func TestCommandSupportsServerModeMicroprogramacion(t *testing.T) {
	t.Parallel()

	casos := []struct {
		args []string
		want bool
	}{
		{args: []string{"microprogramacion", "especificacion", "listar"}, want: true},
		{args: []string{"microprogramacion", "especificacion", "ver"}, want: true},
		{args: []string{"microprogramacion", "especificacion", "crear"}, want: true},
		{args: []string{"microprogramacion", "especificacion", "emitir"}, want: true},
		{args: []string{"microprogramacion", "especificacion", "despachar"}, want: true},
		{args: []string{"microprogramacion", "especificacion", "validar-entrega"}, want: true},
		{args: []string{"microprogramacion"}, want: false},
		{args: []string{"microprogramacion", "otra", "crear"}, want: false},
	}
	for _, tc := range casos {
		if got := commandSupportsServerMode(tc.args); got != tc.want {
			t.Fatalf("commandSupportsServerMode(%v)=%v want=%v", tc.args, got, tc.want)
		}
	}
}
