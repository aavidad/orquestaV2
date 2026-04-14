package gobernanzapolicy

import "testing"

func TestNormalizeOverrideSpec(t *testing.T) {
	spec, err := NormalizeOverrideSpec(" proyecto ", " regla ", " enable ")
	if err != nil {
		t.Fatalf("NormalizeOverrideSpec: %v", err)
	}
	if spec.ScopeType != ScopeProject || spec.Entity != EntityRule || spec.Action != ActionEnable {
		t.Fatalf("spec inesperado: %+v", spec)
	}
}

func TestNormalizeOverrideSpecRejectsInvalidValues(t *testing.T) {
	cases := []struct {
		name     string
		scope    string
		entity   string
		action   string
		wantText string
	}{
		{name: "scope", scope: "desconocido", entity: EntityRule, action: ActionEnable, wantText: "scope_tipo invalido"},
		{name: "entity", scope: ScopeProject, entity: "foo", action: ActionEnable, wantText: "entidad invalida"},
		{name: "action", scope: ScopeProject, entity: EntityRule, action: "noop", wantText: "accion invalida"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NormalizeOverrideSpec(tc.scope, tc.entity, tc.action)
			if err == nil {
				t.Fatal("se esperaba error")
			}
			if got := err.Error(); got == "" || got[:len(tc.wantText)] != tc.wantText {
				t.Fatalf("error inesperado: %v", err)
			}
		})
	}
}

func TestResolveOverrideLayers(t *testing.T) {
	layers, resolution := ResolveOverrideLayers(" demo ", " codex1 ")
	if resolution != "rol" {
		t.Fatalf("resolucion inesperada: %q", resolution)
	}
	if len(layers) != 2 {
		t.Fatalf("layers inesperadas: %+v", layers)
	}
	if layers[0].ScopeType != ScopeProject || layers[0].ScopeRef != "demo" {
		t.Fatalf("capa de proyecto inesperada: %+v", layers[0])
	}
	if layers[1].ScopeType != ScopeAgent || layers[1].ScopeRef != "codex1" {
		t.Fatalf("capa de agente inesperada: %+v", layers[1])
	}
}

func TestAppendResolutionScope(t *testing.T) {
	resolution := AppendResolutionScope("rol", ScopeProject)
	resolution = AppendResolutionScope(resolution, ScopeAgent)
	resolution = AppendResolutionScope(resolution, ScopeAgent)
	if resolution != "rol+proyecto+agente" {
		t.Fatalf("resolucion inesperada: %q", resolution)
	}
}
