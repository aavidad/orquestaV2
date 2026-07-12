package orquestamcp

import (
	"context"
	"encoding/json"
	"strings"
)

const (
	MCPDocumentTextExtractToolNameV0    = "orquesta.document.text.extract.v0"
	MCPDocumentTextExtractToolVersionV0 = "v0"
	MCPDocumentTextExtractResourceURIV0 = "orquesta://contracts/document-text-extract/v0"

	MCPDocumentTextExtractEstadoOKV0    = "ok"
	MCPDocumentTextExtractEstadoErrorV0 = "error"

	MCPDocumentTextExtractErrExtractorUnavailableV0 = "document_text_extract_port_unavailable"
	MCPDocumentTextExtractErrDocumentRefRequiredV0  = "document_text_extract_document_ref_required"
	MCPDocumentTextExtractErrExtractionFailedV0     = "document_text_extract_extraction_failed"

	// El transporte MCP corta en 64 KiB. Un PDF de 31 paginas no cabe entero, asi
	// que la tool pagina por defecto y declara el corte en vez de reventar el
	// presupuesto o mentir con una salida silenciosamente truncada.
	MCPDocumentTextExtractDefaultPageLimitV0 = 5
	MCPDocumentTextExtractMaxPageLimitV0     = 20
)

type MCPDocumentTextExtractToolInputV0 struct {
	SchemaVersion string `json:"schema_version,omitempty"`
	RequestRef    string `json:"request_ref,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
	DocumentRef   string `json:"document_ref"`
	PageFrom      int    `json:"page_from,omitempty"`
	PageLimit     int    `json:"page_limit,omitempty"`
}

type MCPDocumentTextExtractPageV0 struct {
	PageRef string   `json:"page_ref"`
	Index   int      `json:"index"`
	Width   float64  `json:"width"`
	Height  float64  `json:"height"`
	Lines   []string `json:"lines"`
}

type MCPDocumentTextExtractToolResultV0 struct {
	Estado          string                         `json:"estado"`
	RequestRef      string                         `json:"request_ref,omitempty"`
	CorrelationID   string                         `json:"correlation_id,omitempty"`
	DocumentRef     string                         `json:"document_ref,omitempty"`
	SourceRef       string                         `json:"source_ref,omitempty"`
	ContentHash     string                         `json:"content_hash,omitempty"`
	MediaKind       string                         `json:"media_kind,omitempty"`
	AdapterRef      string                         `json:"adapter_ref,omitempty"`
	PageCount       int                            `json:"page_count,omitempty"`
	PageFrom        int                            `json:"page_from,omitempty"`
	PageTo          int                            `json:"page_to,omitempty"`
	SpanCount       int                            `json:"span_count,omitempty"`
	HasMorePages    bool                           `json:"has_more_pages,omitempty"`
	Pages           []MCPDocumentTextExtractPageV0 `json:"pages,omitempty"`
	ErroresPublicos []MCPToolCapabilitiesListPublicErrorV0    `json:"errores_publicos,omitempty"`
}

// MCPDocumentTextExtractExtractorPortV0 lo implementa el stack sobre el
// adaptador real. La tool no conoce PDF ni poppler: solo pide texto anclado.
type MCPDocumentTextExtractExtractorPortV0 interface {
	ExtractDocumentTextV0(context.Context, MCPDocumentTextExtractToolInputV0) (MCPDocumentTextExtractToolResultV0, error)
}

type MCPDocumentTextExtractToolExecutorV0 struct {
	Extractor MCPDocumentTextExtractExtractorPortV0
}

func MCPDocumentTextExtractDescriptorV0() MCPToolCapabilitiesListToolDescriptorV0 {
	return MCPToolCapabilitiesListToolDescriptorV0{
		Name:        MCPDocumentTextExtractToolNameV0,
		Version:     MCPDocumentTextExtractToolVersionV0,
		InputSchema: "document_text_extract:{schema_version?,request_ref?,correlation_id?,document_ref,page_from?,page_limit?}",
		Output:      "ok:{estado,document_ref,source_ref,content_hash,media_kind,adapter_ref,page_count,page_from,page_to,span_count,has_more_pages,pages[]{page_ref,index,width,height,lines[]}}|error:{estado,errores_publicos[]{code,field?}}",
		ResourceURI: MCPDocumentTextExtractResourceURIV0,
		Invariantes: []string{
			"read-only: lee el documento y no muta estado",
			"document_ref es relativo a la raiz de ingesta: rutas absolutas y escapes se rechazan",
			"pagina la salida para no rebasar el presupuesto del transporte y declara has_more_pages",
			"binding nil produce error publico controlado, nunca salida vacia con estado ok",
		},
	}
}

func (executor MCPDocumentTextExtractToolExecutorV0) Execute(
	ctx context.Context,
	input MCPDocumentTextExtractToolInputV0,
) (MCPDocumentTextExtractToolResultV0, error) {
	input = normalizeMCPDocumentTextExtractInputV0(input)
	// El puerto sin cablear se delata ANTES de validar la entrada: si no, una
	// llamada con argumentos vacios responde "falta document_ref" y la tool
	// muerta pasa desapercibida. Registrado != cableado.
	if executor.Extractor == nil {
		return newMCPDocumentTextExtractErrorV0(input, MCPDocumentTextExtractErrExtractorUnavailableV0, "extractor"), nil
	}
	if input.DocumentRef == "" {
		return newMCPDocumentTextExtractErrorV0(input, MCPDocumentTextExtractErrDocumentRefRequiredV0, "document_ref"), nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := executor.Extractor.ExtractDocumentTextV0(ctx, input)
	if err != nil {
		return newMCPDocumentTextExtractErrorV0(input, MCPDocumentTextExtractErrExtractionFailedV0, "document_ref"), nil
	}
	result.Estado = MCPDocumentTextExtractEstadoOKV0
	result.RequestRef = input.RequestRef
	result.CorrelationID = input.CorrelationID
	return result, nil
}

func NormalizeMCPDocumentTextExtractPagingV0(input MCPDocumentTextExtractToolInputV0) (int, int) {
	pageFrom := input.PageFrom
	if pageFrom < 1 {
		pageFrom = 1
	}
	pageLimit := input.PageLimit
	if pageLimit <= 0 {
		pageLimit = MCPDocumentTextExtractDefaultPageLimitV0
	}
	if pageLimit > MCPDocumentTextExtractMaxPageLimitV0 {
		pageLimit = MCPDocumentTextExtractMaxPageLimitV0
	}
	return pageFrom, pageLimit
}

func mcpDocumentTextExtractTransportHandlerV0(
	executor MCPTransportDocumentTextExtractExecutorV0,
) MCPTransportToolHandlerV0 {
	return func(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
		var input MCPDocumentTextExtractToolInputV0
		if err := json.Unmarshal(raw, &input); err != nil {
			return nil, err
		}
		if executor == nil {
			return mcpTransportToolErrorPayloadV0(MCPDocumentTextExtractToolNameV0, MCPTransportToolUnboundV0)
		}
		result, err := executor.Execute(ctx, input)
		if err != nil {
			return nil, err
		}
		return json.Marshal(result)
	}
}

func normalizeMCPDocumentTextExtractInputV0(input MCPDocumentTextExtractToolInputV0) MCPDocumentTextExtractToolInputV0 {
	input.SchemaVersion = strings.TrimSpace(input.SchemaVersion)
	input.RequestRef = strings.TrimSpace(input.RequestRef)
	input.CorrelationID = strings.TrimSpace(input.CorrelationID)
	input.DocumentRef = strings.TrimSpace(input.DocumentRef)
	return input
}

func newMCPDocumentTextExtractErrorV0(
	input MCPDocumentTextExtractToolInputV0,
	code string,
	field string,
) MCPDocumentTextExtractToolResultV0 {
	return MCPDocumentTextExtractToolResultV0{
		Estado:          MCPDocumentTextExtractEstadoErrorV0,
		RequestRef:      input.RequestRef,
		CorrelationID:   input.CorrelationID,
		DocumentRef:     input.DocumentRef,
		ErroresPublicos: []MCPToolCapabilitiesListPublicErrorV0{{Code: code, Field: field}},
	}
}
