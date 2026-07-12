package orquestamcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	toolcapability "orquesta/modulos/orquesta-tool-capability"
)

func TestMCPToolCapabilitiesListTransportV0RegistradoYDelegado(t *testing.T) {
	catalog := &fakeMCPToolCapabilityCatalogV0{
		manifests: []toolcapability.CapabilityManifestV0{
			mcpToolCapabilityManifestTestV0("manifest-z", "cap-z", "tool-z", "en"),
			mcpToolCapabilityManifestTestV0("manifest-a", "cap-a", "tool-a", "es"),
		},
	}
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{
		ToolCapabilities: MCPToolCapabilitiesListToolExecutorV0{Catalog: catalog},
	}); err != nil {
		t.Fatalf("register: %v", err)
	}
	raw, err := transport.CallToolV0(context.Background(), MCPToolCapabilitiesListToolNameV0, MCPToolCapabilitiesListToolInputV0{
		RequestRef:    "request-ref-tool-capabilities-list-test",
		CapabilityRef: "cap-a",
		Locale:        "es",
	})
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	var result MCPToolCapabilitiesListToolResultV0
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("decode: %v payload=%s", err, raw)
	}
	if result.Estado != MCPToolCapabilitiesListEstadoOKV0 || len(result.Manifests) != 1 || result.Manifests[0].ManifestRef != "manifest-a" {
		t.Fatalf("result=%+v", result)
	}
	if catalog.filter.CapabilityRef != "cap-a" || catalog.filter.Locale != "es" {
		t.Fatalf("filter=%+v", catalog.filter)
	}
}

func TestMCPToolCapabilitiesListV0ToolsListIncluyeDescriptorYSchema(t *testing.T) {
	tools := MCPTransportToolsV0(MCPTransportBindingsV0{})
	found := false
	for _, tool := range tools {
		if tool.Name == MCPToolCapabilitiesListToolNameV0 {
			found = true
			if tool.ResourceURI != MCPToolCapabilitiesListResourceURIV0 ||
				!strings.Contains(tool.InputShape, "capability_ref?") ||
				!strings.Contains(tool.OutputShape, "manifests?[]") {
				t.Fatalf("tool envelope=%+v", tool)
			}
		}
	}
	if !found {
		t.Fatalf("%s no registrado", MCPToolCapabilitiesListToolNameV0)
	}
	fields, ok := MCPTransportToolInputFieldsV0(MCPToolCapabilitiesListToolNameV0)
	if !ok {
		t.Fatalf("schema no encontrado")
	}
	seen := map[string]bool{}
	for _, field := range fields {
		seen[field.Name] = true
	}
	for _, want := range []string{"request_ref", "correlation_id", "capability_ref", "tool_ref", "locale"} {
		if !seen[want] {
			t.Fatalf("campo %q no encontrado en %+v", want, fields)
		}
	}
}

func TestMCPToolCapabilitiesListV0BindingNilControlado(t *testing.T) {
	transport := newFakeMCPTransportV0()
	if err := RegisterMCPTransportV0(transport, MCPTransportBindingsV0{}); err != nil {
		t.Fatalf("register: %v", err)
	}
	raw, err := transport.CallToolV0(context.Background(), MCPToolCapabilitiesListToolNameV0, MCPToolCapabilitiesListToolInputV0{})
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	var result MCPTransportToolErrorV0
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("decode: %v payload=%s", err, raw)
	}
	if result.ErrorCode != MCPTransportToolUnboundV0 || result.Tool != MCPToolCapabilitiesListToolNameV0 {
		t.Fatalf("result=%+v", result)
	}

	executed, err := (MCPToolCapabilitiesListToolExecutorV0{}).Execute(context.Background(), MCPToolCapabilitiesListToolInputV0{})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if executed.Estado != MCPToolCapabilitiesListEstadoErrorV0 ||
		len(executed.Errores) != 1 ||
		executed.Errores[0].Code != MCPToolCapabilitiesListErrCatalogUnavailableV0 {
		t.Fatalf("executed=%+v", executed)
	}
}

func TestMCPToolCapabilitiesListExecutorV0OrdenaYOcultaErrorInterno(t *testing.T) {
	result, err := (MCPToolCapabilitiesListToolExecutorV0{Catalog: fakeMCPToolCapabilityCatalogErrorV0{}}).Execute(
		context.Background(),
		MCPToolCapabilitiesListToolInputV0{RequestRef: "request-ref-tool-capabilities-error-test"},
	)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.Estado != MCPToolCapabilitiesListEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != MCPToolCapabilitiesListErrCatalogErrorV0 {
		t.Fatalf("result=%+v", result)
	}
	payload, _ := json.Marshal(result)
	if strings.Contains(string(payload), "internal failure detail") {
		t.Fatalf("leaked internal error: %s", payload)
	}
}

type fakeMCPToolCapabilityCatalogV0 struct {
	filter    toolcapability.CapabilityManifestFilterV0
	manifests []toolcapability.CapabilityManifestV0
}

func (fake *fakeMCPToolCapabilityCatalogV0) ListCapabilityManifestsV0(
	_ context.Context,
	filter toolcapability.CapabilityManifestFilterV0,
) ([]toolcapability.CapabilityManifestV0, error) {
	fake.filter = filter
	out := make([]toolcapability.CapabilityManifestV0, 0, len(fake.manifests))
	for _, manifest := range fake.manifests {
		if filter.CapabilityRef != "" && manifest.CapabilityRef != filter.CapabilityRef {
			continue
		}
		if filter.ToolRef != "" && manifest.ToolRef != filter.ToolRef {
			continue
		}
		if filter.Locale != "" && !mcpToolCapabilityLocalesContainTestV0(manifest.Locales, filter.Locale) {
			continue
		}
		out = append(out, manifest)
	}
	return out, nil
}

type fakeMCPToolCapabilityCatalogErrorV0 struct{}

func (fakeMCPToolCapabilityCatalogErrorV0) ListCapabilityManifestsV0(
	context.Context,
	toolcapability.CapabilityManifestFilterV0,
) ([]toolcapability.CapabilityManifestV0, error) {
	return nil, errors.New("internal failure detail")
}

func mcpToolCapabilityManifestTestV0(manifestRef, capabilityRef, toolRef string, locales ...string) toolcapability.CapabilityManifestV0 {
	return toolcapability.CapabilityManifestV0{
		SchemaVersion:    toolcapability.CapabilityManifestSchemaV0,
		ManifestRef:      manifestRef,
		CapabilityRef:    capabilityRef,
		ToolRef:          toolRef,
		Version:          "1.0.0",
		InputSchemaRef:   "input-schema-ref",
		OutputSchemaRef:  "output-schema-ref",
		Locales:          locales,
		EffectProfile:    toolcapability.CapabilityEffectProfileV0{ProfileRef: "effect-profile-ref"},
		Idempotency:      toolcapability.CapabilityIdempotencyV0{KeySchemaRef: "idempotency-key-schema-ref", ScopeRef: "idempotency-scope-ref"},
		ConfigSchemaRef:  "config-schema-ref",
		TestRefs:         []string{"test-ref"},
		ReceiptSchemaRef: "receipt-schema-ref",
		IntegrationModes: []string{toolcapability.IntegrationModeEmbeddedModuleV0},
	}
}

func mcpToolCapabilityLocalesContainTestV0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
