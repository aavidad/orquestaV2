package db

import (
	"errors"
	"testing"
)

func TestEsErrorMigracionIgnorable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "sqlite duplicate column", err: errors.New("duplicate column name: proyecto_id"), want: true},
		{name: "sqlite already exists", err: errors.New("table proyectos already exists"), want: true},
		{name: "mysql duplicate key", err: errors.New("Error 1061: duplicate key name 'idx_runtime_handles_agente_activo'"), want: true},
		{name: "real failure", err: errors.New("syntax error near FROM"), want: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := esErrorMigracionIgnorable(tc.err)
			if got != tc.want {
				t.Fatalf("esErrorMigracionIgnorable(%v)=%v want %v", tc.err, got, tc.want)
			}
		})
	}
}
