package orquestamcp

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	toolcapability "orquesta/modulos/orquesta-tool-capability"
)

const (
	MCPToolCapabilitiesListToolNameV0    = "orquesta.tool.capabilities.list.v0"
	MCPToolCapabilitiesListToolVersionV0 = "v0"
	MCPToolCapabilitiesListResourceURIV0 = "orquesta://contracts/tool-capabilities-list/v0"

	MCPToolCapabilitiesListEstadoOKV0    = "ok"
	MCPToolCapabilitiesListEstadoErrorV0 = "error"

	MCPToolCapabilitiesListErrCatalogUnavailableV0 = "tool_capability_catalog_unavailable"
	MCPToolCapabilitiesListErrCatalogErrorV0       = "tool_capability_catalog_error"
)

type MCPToolCapabilitiesListToolDescriptorV0 struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	InputSchema string   `json:"input_schema"`
	Output      string   `json:"output"`
	ResourceURI string   `json:"resource_uri"`
	Invariantes []string `json:"invariantes"`
}

type MCPToolCapabilitiesListToolInputV0 struct {
	SchemaVersion string `json:"schema_version,omitempty"`
	RequestRef    string `json:"request_ref,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
	CapabilityRef string `json:"capability_ref,omitempty"`
	ToolRef       string `json:"tool_ref,omitempty"`
	Locale        string `json:"locale,omitempty"`
}

type MCPToolCapabilitiesListToolResultV0 struct {
	Estado        string                                    `json:"estado"`
	RequestRef    string                                    `json:"request_ref,omitempty"`
	CorrelationID string                                    `json:"correlation_id,omitempty"`
	Filter        toolcapability.CapabilityManifestFilterV0 `json:"filter,omitempty"`
	Manifests     []toolcapability.CapabilityManifestV0     `json:"manifests,omitempty"`
	Errores       []MCPToolCapabilitiesListPublicErrorV0    `json:"errores_publicos,omitempty"`
}

type MCPToolCapabilitiesListPublicErrorV0 struct {
	Code  string `json:"code"`
	Field string `json:"field,omitempty"`
}

type MCPToolCapabilitiesListToolExecutorV0 struct {
	Catalog toolcapability.CapabilityCatalogPortV0
}

func MCPToolCapabilitiesListDescriptorV0() MCPToolCapabilitiesListToolDescriptorV0 {
	return MCPToolCapabilitiesListToolDescriptorV0{
		Name:        MCPToolCapabilitiesListToolNameV0,
		Version:     MCPToolCapabilitiesListToolVersionV0,
		InputSchema: "tool_capabilities_list:{schema_version?,request_ref?,correlation_id?,capability_ref?,tool_ref?,locale?}",
		Output:      "ok:{estado,request_ref?,correlation_id?,filter,manifests?[]{manifest_ref,capability_ref,tool_ref,version,locales,permissions?,integration_modes,test_refs}}|error:{estado,request_ref?,correlation_id?,errores_publicos[]{code,field?}}",
		ResourceURI: MCPToolCapabilitiesListResourceURIV0,
		Invariantes: []string{
			"read-only: lista manifests de capacidades sin materializar, instalar ni modificar estado",
			"delega siempre en CapabilityCatalogPortV0",
			"aplica CapabilityManifestFilterV0 y devuelve salida ordenada determinista",
			"binding nil produce error publico controlado",
		},
	}
}

func (executor MCPToolCapabilitiesListToolExecutorV0) Execute(
	ctx context.Context,
	input MCPToolCapabilitiesListToolInputV0,
) (MCPToolCapabilitiesListToolResultV0, error) {
	input = normalizeMCPToolCapabilitiesListInputV0(input)
	filter := toolcapability.CapabilityManifestFilterV0{
		CapabilityRef: input.CapabilityRef,
		ToolRef:       input.ToolRef,
		Locale:        input.Locale,
	}
	result := MCPToolCapabilitiesListToolResultV0{
		Estado:        MCPToolCapabilitiesListEstadoOKV0,
		RequestRef:    input.RequestRef,
		CorrelationID: input.CorrelationID,
		Filter:        filter,
	}
	if executor.Catalog == nil {
		return newMCPToolCapabilitiesListErrorV0(input, MCPToolCapabilitiesListErrCatalogUnavailableV0, "catalog"), nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	manifests, err := executor.Catalog.ListCapabilityManifestsV0(ctx, filter)
	if err != nil {
		return newMCPToolCapabilitiesListErrorV0(input, MCPToolCapabilitiesListErrCatalogErrorV0, "catalog"), nil
	}
	sortMCPToolCapabilityManifestsV0(manifests)
	result.Manifests = manifests
	return result, nil
}

func mcpToolCapabilitiesListTransportHandlerV0(
	executor MCPTransportToolCapabilitiesListExecutorV0,
) MCPTransportToolHandlerV0 {
	return func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
		var input MCPToolCapabilitiesListToolInputV0
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, err
		}
		if executor == nil {
			return mcpTransportToolErrorPayloadV0(MCPToolCapabilitiesListToolNameV0, MCPTransportToolUnboundV0)
		}
		result, err := executor.Execute(ctx, input)
		if err != nil {
			return nil, err
		}
		return json.Marshal(result)
	}
}

func normalizeMCPToolCapabilitiesListInputV0(input MCPToolCapabilitiesListToolInputV0) MCPToolCapabilitiesListToolInputV0 {
	input.SchemaVersion = strings.TrimSpace(input.SchemaVersion)
	input.RequestRef = strings.TrimSpace(input.RequestRef)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)
	input.CapabilityRef = strings.TrimSpace(input.CapabilityRef)
	input.ToolRef = strings.TrimSpace(input.ToolRef)
	input.Locale = strings.TrimSpace(input.Locale)
	return input
}

func newMCPToolCapabilitiesListErrorV0(
	input MCPToolCapabilitiesListToolInputV0,
	code string,
	field string,
) MCPToolCapabilitiesListToolResultV0 {
	return MCPToolCapabilitiesListToolResultV0{
		Estado:        MCPToolCapabilitiesListEstadoErrorV0,
		RequestRef:    input.RequestRef,
		CorrelationID: input.CorrelationID,
		Errores: []MCPToolCapabilitiesListPublicErrorV0{{
			Code:  code,
			Field: field,
		}},
	}
}

func sortMCPToolCapabilityManifestsV0(manifests []toolcapability.CapabilityManifestV0) {
	sort.SliceStable(manifests, func(left, right int) bool {
		a := manifests[left]
		b := manifests[right]
		if a.CapabilityRef != b.CapabilityRef {
			return a.CapabilityRef < b.CapabilityRef
		}
		if a.ToolRef != b.ToolRef {
			return a.ToolRef < b.ToolRef
		}
		if a.Version != b.Version {
			return a.Version < b.Version
		}
		return a.ManifestRef < b.ManifestRef
	})
}
