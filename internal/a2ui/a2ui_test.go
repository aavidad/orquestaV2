package a2ui

import (
	"encoding/json"
	"testing"
)

func TestDecodeAndValidateDataTable(t *testing.T) {
	payload := json.RawMessage(`{
		"type":"a2ui_render",
		"component":"DataTable",
		"props":{
			"title":"Analisis de riesgos",
			"columns":["ID","Descripcion","Probabilidad"],
			"data":[
				[1,"Inyeccion SQL","Baja"],
				[2,"Fuga de tokens","Media"]
			]
		}
	}`)

	req, err := DecodeRenderRequest(payload)
	if err != nil {
		t.Fatalf("DecodeRenderRequest: %v", err)
	}
	propsAny, err := ValidateRenderRequest(req)
	if err != nil {
		t.Fatalf("ValidateRenderRequest: %v", err)
	}
	props, ok := propsAny.(DataTableProps)
	if !ok {
		t.Fatalf("props con tipo inesperado: %T", propsAny)
	}
	if len(props.Columns) != 3 || len(props.Data) != 2 {
		t.Fatalf("DataTable inesperada: %+v", props)
	}
}

func TestValidateChartRejectsSeriesLengthMismatch(t *testing.T) {
	req := &RenderRequest{
		Type:      MessageRender,
		Component: ComponentChart,
		Props: json.RawMessage(`{
			"title":"Uso por agente",
			"chart_type":"bar",
			"labels":["Codex1","Codex2"],
			"series":[{"name":"cpu","values":[1]}]
		}`),
	}

	if _, err := ValidateRenderRequest(req); err == nil {
		t.Fatalf("se esperaba error por longitud invalida en la serie")
	}
}

func TestValidateApprovalFormDefaultsLabels(t *testing.T) {
	req := &RenderRequest{
		Type:      MessageRender,
		Component: ComponentApprovalForm,
		Props: json.RawMessage(`{
			"title":"Aplicar cambio",
			"message":"Deseas continuar",
			"action_id":"aplicar-cambio"
		}`),
	}

	propsAny, err := ValidateRenderRequest(req)
	if err != nil {
		t.Fatalf("ValidateRenderRequest: %v", err)
	}
	props, ok := propsAny.(ApprovalFormProps)
	if !ok {
		t.Fatalf("props con tipo inesperado: %T", propsAny)
	}
	if props.ConfirmLabel != "Confirmar" || props.CancelLabel != "Cancelar" {
		t.Fatalf("labels por defecto inesperadas: %+v", props)
	}
}

func TestValidateRejectsUnknownComponent(t *testing.T) {
	req := &RenderRequest{
		Type:      MessageRender,
		Component: ComponentKind("Mapa"),
		Props:     json.RawMessage(`{"title":"demo"}`),
	}

	if _, err := ValidateRenderRequest(req); err == nil {
		t.Fatalf("se esperaba error por componente desconocido")
	}
}

func TestValidateRejectsInvalidTopLevelType(t *testing.T) {
	req := &RenderRequest{
		Type:      MessageType("otro"),
		Component: ComponentMarkdown,
		Props:     json.RawMessage(`{"markdown":"hola"}`),
	}

	if _, err := ValidateRenderRequest(req); err == nil {
		t.Fatalf("se esperaba error por tipo invalido")
	}
}

func TestDecodeRejectsMissingProps(t *testing.T) {
	payload := json.RawMessage(`{
		"type":"a2ui_render",
		"component":"MarkdownBlock"
	}`)

	if _, err := DecodeRenderRequest(payload); err == nil {
		t.Fatalf("se esperaba error por props ausentes")
	}
}

func TestValidateRejectsNestedDataTableCell(t *testing.T) {
	req := &RenderRequest{
		Type:      MessageRender,
		Component: ComponentDataTable,
		Props: json.RawMessage(`{
			"title":"Analisis",
			"columns":["ID","Detalle"],
			"data":[[1,{"riesgo":"alto"}]]
		}`),
	}

	if _, err := ValidateRenderRequest(req); err == nil {
		t.Fatalf("se esperaba error por celda anidada")
	}
}

func TestEncodeAndDecodeMailboxPayload(t *testing.T) {
	req := &RenderRequest{
		Type:      MessageRender,
		Component: ComponentMarkdown,
		Props:     json.RawMessage(`{"title":"Nota","markdown":"hola"}`),
	}

	payload, err := EncodeMailboxPayload(req)
	if err != nil {
		t.Fatalf("EncodeMailboxPayload: %v", err)
	}
	envelope, propsAny, err := DecodeMailboxPayload(MailboxKindRender, payload)
	if err != nil {
		t.Fatalf("DecodeMailboxPayload: %v", err)
	}
	if envelope.Version != EnvelopeVersionV1 {
		t.Fatalf("version inesperada: %+v", envelope)
	}
	props, ok := propsAny.(MarkdownBlockProps)
	if !ok {
		t.Fatalf("props con tipo inesperado: %T", propsAny)
	}
	if props.Markdown != "hola" {
		t.Fatalf("markdown inesperado: %+v", props)
	}
}

func TestDecodeMailboxPayloadRejectsWrongKind(t *testing.T) {
	if _, _, err := DecodeMailboxPayload("handoff", `{}`); err == nil {
		t.Fatalf("se esperaba error por kind no soportado")
	}
}

func TestDecodeMailboxPayloadRejectsUnknownVersion(t *testing.T) {
	payload := `{
		"version":"a2ui.mailbox.v2",
		"request":{
			"type":"a2ui_render",
			"component":"MarkdownBlock",
			"props":{"markdown":"hola"}
		}
	}`

	if _, _, err := DecodeMailboxPayload(MailboxKindRender, payload); err == nil {
		t.Fatalf("se esperaba error por version no soportada")
	}
}

func TestBuildMailboxMessage(t *testing.T) {
	req := &RenderRequest{
		Type:      MessageRender,
		Component: ComponentDataTable,
		Props: json.RawMessage(`{
			"title":"Riesgos",
			"columns":["ID","Descripcion"],
			"data":[[1,"SQL"]]
		}`),
	}

	msg, err := BuildMailboxMessage("Codex3", "alberto", req)
	if err != nil {
		t.Fatalf("BuildMailboxMessage: %v", err)
	}
	if msg.Kind != MailboxKindRender || msg.FromAgente != "Codex3" || msg.ToAgente != "alberto" {
		t.Fatalf("mensaje inesperado: %+v", msg)
	}
	if _, _, err := DecodeMailboxPayload(msg.Kind, msg.Payload); err != nil {
		t.Fatalf("payload del mensaje no decodifica: %v", err)
	}
}

func TestBuildMailboxMessageRejectsMissingAgent(t *testing.T) {
	req := &RenderRequest{
		Type:      MessageRender,
		Component: ComponentMarkdown,
		Props:     json.RawMessage(`{"markdown":"hola"}`),
	}

	if _, err := BuildMailboxMessage("", "alberto", req); err == nil {
		t.Fatalf("se esperaba error por from_agente vacio")
	}
	if _, err := BuildMailboxMessage("Codex3", "", req); err == nil {
		t.Fatalf("se esperaba error por to_agente vacio")
	}
}
