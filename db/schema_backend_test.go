package db

import "testing"

func TestSchemaBackendSpecForDriver(t *testing.T) {
	t.Parallel()

	cases := []struct {
		driver        string
		wantName      string
		wantBootstrap bool
		wantDDL       bool
		wantSeed      bool
	}{
		{driver: "sqlite", wantName: "sqlite", wantBootstrap: true, wantDDL: true, wantSeed: true},
		{driver: "sqlite3", wantName: "sqlite", wantBootstrap: true, wantDDL: true, wantSeed: true},
		{driver: "mysql", wantName: "mysql", wantBootstrap: false, wantDDL: false, wantSeed: true},
		{driver: "postgres", wantName: "postgres", wantBootstrap: false, wantDDL: false, wantSeed: false},
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
			if got := spec.ddl() != ""; got != tc.wantDDL {
				t.Fatalf("ddl disponible=%v want %v", got, tc.wantDDL)
			}
			if got := spec.seedSQL() != ""; got != tc.wantSeed {
				t.Fatalf("seed disponible=%v want %v", got, tc.wantSeed)
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
	if spec.ddl() == "" {
		t.Fatalf("fallback sqlite deberia exponer ddl")
	}
	if spec.seedSQL() == "" {
		t.Fatalf("fallback sqlite deberia exponer seed")
	}
}
