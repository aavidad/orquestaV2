package orquestamcp

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMCPOperationalStatusDescriptorV0Compacto(t *testing.T) {
	descriptor := MCPOperationalStatusDescriptorV0()
	if descriptor.Name != MCPOperationalStatusResourceNameV0 ||
		descriptor.Version != MCPOperationalStatusResourceVersionV0 ||
		descriptor.URI != MCPOperationalStatusResourceURIV0 {
		t.Fatalf("descriptor identidad: %+v", descriptor)
	}
	if descriptor.ContentType != MCPOperationalStatusContentTypeV0 {
		t.Fatalf("content type inesperado: %+v", descriptor)
	}
	if descriptor.SummaryKey == "" || strings.Contains(descriptor.SummaryKey, " ") {
		t.Fatalf("summary_key debe ser clave compacta: %+v", descriptor)
	}
}

func TestNewMCPOperationalStatusResourceV0CompactoYReadOnly(t *testing.T) {
	resource := NewMCPOperationalStatusResourceV0()
	if resource.URI != MCPOperationalStatusResourceURIV0 ||
		resource.Version != "v0" ||
		resource.ContractResource != "orquesta://contracts/operational-status-query/v0" {
		t.Fatalf("resource identidad: %+v", resource)
	}
	if resource.RecommendedEndpoint != MCPOperationalStatusEndpointV0 {
		t.Fatalf("endpoint recomendado inesperado: %+v", resource)
	}
	if len(resource.AllowedConsumers) != 4 || len(resource.AllowedScopes) != 7 || len(resource.AllowedSections) != 7 {
		t.Fatalf("listas compactas incompletas: %+v", resource)
	}
	if resource.Request.SchemaVersion != "operational_status_query.v0" ||
		resource.Response.SchemaVersion != "diagnostico_compacto.v0" {
		t.Fatalf("schemas inesperados: %+v", resource)
	}
	if len(resource.PublicErrors) != 10 || len(resource.Guardrails) == 0 || len(resource.CanonicalRefs) < 2 {
		t.Fatalf("metadata incompleta: %+v", resource)
	}

	payload, err := json.Marshal(resource)
	if err != nil {
		t.Fatalf("marshal resource: %v", err)
	}
	text := string(payload)
	for _, forbidden := range []string{
		"password",
		"bearer ",
		"select *",
		"insert into",
		"/home/",
		"filesystem productivo",
	} {
		if strings.Contains(strings.ToLower(text), strings.ToLower(forbidden)) {
			t.Fatalf("resource contiene detalle no compacto %q: %s", forbidden, text)
		}
	}
	if len(text) > 6000 {
		t.Fatalf("resource demasiado largo: %d bytes", len(text))
	}
}

func TestMCPOperationalStatusConsumerByKeyV0NormalizaEntradas(t *testing.T) {
	cases := map[string]string{
		"orquesta-mcp":     "mcp",
		"mcp":              "orquesta-mcp",
		"orquesta_cli/cli": "cli",
		"orquesta-web/web": "web",
	}

	for input, wantOneOf := range cases {
		got, ok := MCPOperationalStatusConsumerByKeyV0(input)
		if !ok {
			t.Fatalf("no encontro consumer para %q", input)
		}
		if got.Module != wantOneOf && got.Channel != wantOneOf {
			t.Fatalf("consumer para %q: %+v", input, got)
		}
	}

	if _, ok := MCPOperationalStatusConsumerByKeyV0(" M C P "); ok {
		t.Fatalf("entrada invalida no debe encontrarse")
	}
	if _, ok := MCPOperationalStatusConsumerByKeyV0("consumer-inexistente"); ok {
		t.Fatalf("consumer inexistente no debe encontrarse")
	}
}
