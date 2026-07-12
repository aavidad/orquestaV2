package orquestamcp

import (
	"context"
	"encoding/json"
	"strings"
)

const (
	MCPDataProfileToolNameV0    = "orquesta.data.profile.v0"
	MCPDataProfileToolVersionV0 = "v0"
	MCPDataProfileResourceURIV0 = "orquesta://contracts/data-profile/v0"

	MCPDataProfileEstadoOKV0    = "ok"
	MCPDataProfileEstadoErrorV0 = "error"

	MCPDataProfileErrProfilerUnavailableV0 = "data_profile_port_unavailable"
	MCPDataProfileErrProfileFailedV0       = "data_profile_failed"
)

type MCPDataProfileToolInputV0 struct {
	SchemaVersion string `json:"schema_version,omitempty"`
	RequestRef    string `json:"request_ref,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
	// Vacio = lista los datasets catalogados en la raiz de ingesta. Con valor =
	// perfila ese dataset. Listar sin argumentos es lo que permite al director
	// descubrir que hay antes de pedir nada.
	DatasetRef string `json:"dataset_ref,omitempty"`
}

type MCPDataProfileColumnV0 struct {
	ColumnRef string `json:"column_ref"`
	Name      string `json:"name"`
	DataType  string `json:"data_type"`
	Nullable  bool   `json:"nullable"`
}

type MCPDataProfileDatasetV0 struct {
	DatasetRef string `json:"dataset_ref"`
	SourceKind string `json:"source_kind"`
}

type MCPDataProfileToolResultV0 struct {
	Estado          string                                 `json:"estado"`
	RequestRef      string                                 `json:"request_ref,omitempty"`
	CorrelationID   string                                 `json:"correlation_id,omitempty"`
	DatasetRef      string                                 `json:"dataset_ref,omitempty"`
	ProfileRef      string                                 `json:"profile_ref,omitempty"`
	SourceKind      string                                 `json:"source_kind,omitempty"`
	ContentHash     string                                 `json:"content_hash,omitempty"`
	AdapterRef      string                                 `json:"adapter_ref,omitempty"`
	RowCount        int64                                  `json:"row_count,omitempty"`
	Columns         []MCPDataProfileColumnV0               `json:"columns,omitempty"`
	Datasets        []MCPDataProfileDatasetV0              `json:"datasets,omitempty"`
	ErroresPublicos []MCPToolCapabilitiesListPublicErrorV0 `json:"errores_publicos,omitempty"`
}

// MCPDataProfileProfilerPortV0 lo implementa el stack sobre el adaptador real de
// ingesta. La tool no conoce CSV ni ficheros: pide catalogo o perfil.
type MCPDataProfileProfilerPortV0 interface {
	ProfileDataSourceV0(context.Context, MCPDataProfileToolInputV0) (MCPDataProfileToolResultV0, error)
}

type MCPDataProfileToolExecutorV0 struct {
	Profiler MCPDataProfileProfilerPortV0
}

func MCPDataProfileDescriptorV0() MCPToolCapabilitiesListToolDescriptorV0 {
	return MCPToolCapabilitiesListToolDescriptorV0{
		Name:        MCPDataProfileToolNameV0,
		Version:     MCPDataProfileToolVersionV0,
		InputSchema: "data_profile:{schema_version?,request_ref?,correlation_id?,dataset_ref?}",
		Output:      "ok:{estado,dataset_ref?,profile_ref?,source_kind?,content_hash?,adapter_ref?,row_count?,columns?[]{column_ref,name,data_type,nullable},datasets?[]{dataset_ref,source_kind}}|error:{estado,errores_publicos[]{code,field?}}",
		ResourceURI: MCPDataProfileResourceURIV0,
		Invariantes: []string{
			"read-only: perfila el dataset y no muta estado",
			"dataset_ref vacio lista el catalogo; con valor perfila ese dataset",
			"el dataset_ref es catalogado y relativo a la raiz de ingesta: rutas del host se rechazan",
			"puerto sin cablear se delata antes de validar la entrada",
		},
	}
}

func (executor MCPDataProfileToolExecutorV0) Execute(
	ctx context.Context,
	input MCPDataProfileToolInputV0,
) (MCPDataProfileToolResultV0, error) {
	input = normalizeMCPDataProfileInputV0(input)
	// Igual que en document.text.extract: el puerto muerto se delata ANTES de
	// mirar la entrada. Si no, una llamada con argumentos vacios responde algo
	// plausible y la tool sin cablear pasa el guard exhaustivo.
	if executor.Profiler == nil {
		return newMCPDataProfileErrorV0(input, MCPDataProfileErrProfilerUnavailableV0, "profiler"), nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := executor.Profiler.ProfileDataSourceV0(ctx, input)
	if err != nil {
		return newMCPDataProfileErrorV0(input, MCPDataProfileErrProfileFailedV0, "dataset_ref"), nil
	}
	result.Estado = MCPDataProfileEstadoOKV0
	result.RequestRef = input.RequestRef
	result.CorrelationID = input.CorrelationID
	return result, nil
}

func mcpDataProfileTransportHandlerV0(
	executor MCPTransportDataProfileExecutorV0,
) MCPTransportToolHandlerV0 {
	return func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
		var input MCPDataProfileToolInputV0
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, err
		}
		if executor == nil {
			return mcpTransportToolErrorPayloadV0(MCPDataProfileToolNameV0, MCPTransportToolUnboundV0)
		}
		result, err := executor.Execute(ctx, input)
		if err != nil {
			return nil, err
		}
		return json.Marshal(result)
	}
}

func normalizeMCPDataProfileInputV0(input MCPDataProfileToolInputV0) MCPDataProfileToolInputV0 {
	input.SchemaVersion = strings.TrimSpace(input.SchemaVersion)
	input.RequestRef = strings.TrimSpace(input.RequestRef)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)
	input.DatasetRef = strings.TrimSpace(input.DatasetRef)
	return input
}

func newMCPDataProfileErrorV0(
	input MCPDataProfileToolInputV0,
	code string,
	field string,
) MCPDataProfileToolResultV0 {
	return MCPDataProfileToolResultV0{
		Estado:          MCPDataProfileEstadoErrorV0,
		RequestRef:      input.RequestRef,
		CorrelationID:   input.CorrelationID,
		DatasetRef:      input.DatasetRef,
		ErroresPublicos: []MCPToolCapabilitiesListPublicErrorV0{{Code: code, Field: field}},
	}
}
