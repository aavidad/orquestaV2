package db

import "testing"

func TestSchemaBackendSpecForDriver(t *testing.T) {
	t.Parallel()

	cases := []struct {
		driver             string
		wantName           string
		wantBootstrap      bool
		wantDDLParts       bool
		wantPostMigrations bool
	}{
		{driver: "sqlite", wantName: "sqlite", wantBootstrap: true, wantDDLParts: true, wantPostMigrations: true},
		{driver: "sqlite3", wantName: "sqlite", wantBootstrap: true, wantDDLParts: true, wantPostMigrations: true},
		{driver: "postgres", wantName: "postgres", wantBootstrap: true, wantDDLParts: true, wantPostMigrations: false},
		{driver: "postgresql", wantName: "postgres", wantBootstrap: true, wantDDLParts: true, wantPostMigrations: false},
		{driver: "mysql", wantName: "mysql", wantBootstrap: false, wantDDLParts: false, wantPostMigrations: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.driver, func(t *testing.T) {
			t.Parallel()

			spec, ok := schemaBackendSpecForDriver(tc.driver)
			if !ok {
				t.Fatalf("schemaBackendSpecForDriver(%q) no encontrado", tc.driver)
			}
			if spec.name != tc.wantName {
				t.Fatalf("name=%q want %q", spec.name, tc.wantName)
			}
			if spec.supportsBootstrap != tc.wantBootstrap {
				t.Fatalf("supportsBootstrap=%v want %v", spec.supportsBootstrap, tc.wantBootstrap)
			}
			if got := len(spec.ddlParts()) > 0; got != tc.wantDDLParts {
				t.Fatalf("ddlParts disponibles=%v want %v", got, tc.wantDDLParts)
			}
			if got := len(spec.postMigrations()) > 0; got != tc.wantPostMigrations {
				t.Fatalf("postMigrations disponibles=%v want %v", got, tc.wantPostMigrations)
			}
		})
	}
}

func TestFallbackSchemaBackendSpecUsaSQLiteParaDriverDesconocido(t *testing.T) {
	t.Parallel()

	spec := fallbackSchemaBackendSpec("oracle")
	if spec.name != "sqlite" {
		t.Fatalf("fallback deberia usar sqlite; obtuvo %q", spec.name)
	}
	if !spec.supportsBootstrap {
		t.Fatalf("fallback sqlite deberia soportar bootstrap")
	}
	if len(spec.ddlParts()) == 0 {
		t.Fatalf("fallback sqlite deberia exponer DDL")
	}
}
