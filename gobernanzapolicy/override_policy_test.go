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
