package gobernanzapolicy

import "testing"

func TestHashCatalogStableAndSensitiveToChanges(t *testing.T) {
	projectID := int64(7)
	base := &CatalogSnapshot{
		AgentType:  "programador",
		ProjectID:  &projectID,
		Resolution: "rol+proyecto",
		Rules: []RuleSnapshot{
			{ID: 1, Category: "arquitectura", Title: "server-first", Description: "usa API"},
		},
		Skills: []SkillSnapshot{
			{ID: 2, Name: "rg", Description: "buscar", WhenToUse: "explorar", Priority: 10},
		},
		Workflows: []WorkflowSnapshot{
			{ID: 3, Name: "inicio", Description: "arranque", Steps: `["leer","programar"]`},
		},
	}

	hash1 := HashCatalog(base)
	hash2 := HashCatalog(base)
	if hash1 == "" || hash1 != hash2 {
		t.Fatalf("hash inestable: %q vs %q", hash1, hash2)
	}

	changed := *base
	changed.Rules = []RuleSnapshot{
		{ID: 1, Category: "arquitectura", Title: "server-first", Description: "usa API y no DB local"},
	}
	hash3 := HashCatalog(&changed)
	if hash3 == hash1 {
		t.Fatalf("el hash deberia cambiar tras modificar el catalogo: %q", hash3)
	}
}

func TestBuildContextSummaryFromCatalog(t *testing.T) {
	projectID := int64(9)
	catalog := &CatalogSnapshot{
		AgentType:  "programador",
		ProjectID:  &projectID,
		Resolution: "rol+proyecto",
		Hash:       "abc123",
		Rules:      []RuleSnapshot{{ID: 1}},
		Skills:     []SkillSnapshot{{ID: 2}, {ID: 3}},
		Workflows:  []WorkflowSnapshot{{ID: 4}},
	}

	context, summary := BuildContextSummaryFromCatalog(catalog)
	if context["tipo_agente"] != "programador" || context["hash"] != "abc123" {
		t.Fatalf("contexto inesperado: %+v", context)
	}
	if context["reglas"] != 1 || context["skills"] != 2 || context["workflows"] != 1 {
		t.Fatalf("conteos inesperados: %+v", context)
	}
	if context["proyecto_id"] != projectID {
		t.Fatalf("proyecto_id inesperado: %+v", context)
	}
	if summary == "" {
		t.Fatal("resumen vacio")
	}
}
