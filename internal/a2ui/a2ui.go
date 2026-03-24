package a2ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

type MessageType string

const MessageRender MessageType = "a2ui_render"

type ComponentKind string

const (
	ComponentDataTable    ComponentKind = "DataTable"
	ComponentChart        ComponentKind = "Chart"
	ComponentApprovalForm ComponentKind = "ApprovalForm"
	ComponentMarkdown     ComponentKind = "MarkdownBlock"
)

const (
	MailboxKindRender = "a2ui_render"
	EnvelopeVersionV1 = "a2ui.mailbox.v1"
)

type RenderRequest struct {
	Type      MessageType     `json:"type"`
	Component ComponentKind   `json:"component"`
	Props     json.RawMessage `json:"props"`
}

type MailboxEnvelope struct {
	Version string        `json:"version"`
	Request RenderRequest `json:"request"`
}

type MailboxMessage struct {
	FromAgente string `json:"from_agente"`
	ToAgente   string `json:"to_agente"`
	Kind       string `json:"kind"`
	Payload    string `json:"payload"`
}

type DataTableProps struct {
	Title   string   `json:"title"`
	Columns []string `json:"columns"`
	Data    [][]any  `json:"data"`
}

type ChartProps struct {
	Title     string        `json:"title"`
	ChartType string        `json:"chart_type"`
	Labels    []string      `json:"labels"`
	Series    []ChartSeries `json:"series"`
}

type ChartSeries struct {
	Name   string    `json:"name"`
	Values []float64 `json:"values"`
}

type ApprovalFormProps struct {
	Title        string `json:"title"`
	Message      string `json:"message"`
	ActionID     string `json:"action_id"`
	ConfirmLabel string `json:"confirm_label"`
	CancelLabel  string `json:"cancel_label"`
}

type MarkdownBlockProps struct {
	Title    string `json:"title"`
	Markdown string `json:"markdown"`
}

func SupportedComponents() []ComponentKind {
	return []ComponentKind{
		ComponentApprovalForm,
		ComponentChart,
		ComponentDataTable,
		ComponentMarkdown,
	}
}

func DecodeRenderRequest(payload json.RawMessage) (*RenderRequest, error) {
	var req RenderRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("payload A2UI invalido: %w", err)
	}
	if len(bytes.TrimSpace(req.Props)) == 0 || bytes.Equal(bytes.TrimSpace(req.Props), []byte("null")) {
		return nil, fmt.Errorf("props A2UI obligatorias")
	}
	return &req, nil
}

func ValidateRenderRequest(req *RenderRequest) (any, error) {
	if req == nil {
		return nil, fmt.Errorf("request A2UI obligatoria")
	}
	if req.Type != MessageRender {
		return nil, fmt.Errorf("tipo A2UI no soportado: %s", strings.TrimSpace(string(req.Type)))
	}
	switch req.Component {
	case ComponentDataTable:
		var props DataTableProps
		if err := unmarshalProps(req.Props, &props); err != nil {
			return nil, err
		}
		if err := validateDataTableProps(&props); err != nil {
			return nil, err
		}
		return props, nil
	case ComponentChart:
		var props ChartProps
		if err := unmarshalProps(req.Props, &props); err != nil {
			return nil, err
		}
		if err := validateChartProps(&props); err != nil {
			return nil, err
		}
		return props, nil
	case ComponentApprovalForm:
		var props ApprovalFormProps
		if err := unmarshalProps(req.Props, &props); err != nil {
			return nil, err
		}
		if err := validateApprovalFormProps(&props); err != nil {
			return nil, err
		}
		return props, nil
	case ComponentMarkdown:
		var props MarkdownBlockProps
		if err := unmarshalProps(req.Props, &props); err != nil {
			return nil, err
		}
		if err := validateMarkdownBlockProps(&props); err != nil {
			return nil, err
		}
		return props, nil
	default:
		return nil, fmt.Errorf("componente A2UI no soportado: %s", strings.TrimSpace(string(req.Component)))
	}
}

func EncodeMailboxPayload(req *RenderRequest) (string, error) {
	if _, err := ValidateRenderRequest(req); err != nil {
		return "", err
	}
	payload, err := json.Marshal(MailboxEnvelope{
		Version: EnvelopeVersionV1,
		Request: *req,
	})
	if err != nil {
		return "", fmt.Errorf("serializando envelope A2UI: %w", err)
	}
	return string(payload), nil
}

func DecodeMailboxPayload(kind, payload string) (*MailboxEnvelope, any, error) {
	if strings.TrimSpace(kind) != MailboxKindRender {
		return nil, nil, fmt.Errorf("kind mailbox no soportado para A2UI: %s", strings.TrimSpace(kind))
	}
	var envelope MailboxEnvelope
	if err := json.Unmarshal([]byte(payload), &envelope); err != nil {
		return nil, nil, fmt.Errorf("payload mailbox A2UI invalido: %w", err)
	}
	if strings.TrimSpace(envelope.Version) != EnvelopeVersionV1 {
		return nil, nil, fmt.Errorf("version de envelope A2UI no soportada: %s", strings.TrimSpace(envelope.Version))
	}
	props, err := ValidateRenderRequest(&envelope.Request)
	if err != nil {
		return nil, nil, err
	}
	return &envelope, props, nil
}

func BuildMailboxMessage(fromAgente, toAgente string, req *RenderRequest) (*MailboxMessage, error) {
	fromAgente = strings.TrimSpace(fromAgente)
	toAgente = strings.TrimSpace(toAgente)
	if fromAgente == "" {
		return nil, fmt.Errorf("from_agente A2UI obligatorio")
	}
	if toAgente == "" {
		return nil, fmt.Errorf("to_agente A2UI obligatorio")
	}
	payload, err := EncodeMailboxPayload(req)
	if err != nil {
		return nil, err
	}
	return &MailboxMessage{
		FromAgente: fromAgente,
		ToAgente:   toAgente,
		Kind:       MailboxKindRender,
		Payload:    payload,
	}, nil
}

func unmarshalProps(raw json.RawMessage, dst any) error {
	if err := json.Unmarshal(raw, dst); err != nil {
		return fmt.Errorf("props A2UI invalidas: %w", err)
	}
	return nil
}

func validateDataTableProps(props *DataTableProps) error {
	if strings.TrimSpace(props.Title) == "" {
		return fmt.Errorf("DataTable requiere title")
	}
	if len(props.Columns) == 0 {
		return fmt.Errorf("DataTable requiere columns")
	}
	for i, col := range props.Columns {
		if strings.TrimSpace(col) == "" {
			return fmt.Errorf("DataTable column[%d] vacia", i)
		}
	}
	for i, row := range props.Data {
		if len(row) != len(props.Columns) {
			return fmt.Errorf("DataTable row[%d] tiene %d columnas, se esperaban %d", i, len(row), len(props.Columns))
		}
		for j, cell := range row {
			switch cell.(type) {
			case nil, string, float64, bool:
			default:
				return fmt.Errorf("DataTable cell[%d][%d] contiene tipo no soportado", i, j)
			}
		}
	}
	return nil
}

func validateChartProps(props *ChartProps) error {
	if strings.TrimSpace(props.Title) == "" {
		return fmt.Errorf("Chart requiere title")
	}
	switch strings.TrimSpace(props.ChartType) {
	case "bar", "line", "radar":
	default:
		return fmt.Errorf("Chart chart_type no soportado: %s", strings.TrimSpace(props.ChartType))
	}
	if len(props.Labels) == 0 {
		return fmt.Errorf("Chart requiere labels")
	}
	if len(props.Series) == 0 {
		return fmt.Errorf("Chart requiere series")
	}
	for i, label := range props.Labels {
		if strings.TrimSpace(label) == "" {
			return fmt.Errorf("Chart label[%d] vacia", i)
		}
	}
	for i, series := range props.Series {
		if strings.TrimSpace(series.Name) == "" {
			return fmt.Errorf("Chart series[%d] requiere name", i)
		}
		if len(series.Values) != len(props.Labels) {
			return fmt.Errorf("Chart series[%d] tiene %d valores, se esperaban %d", i, len(series.Values), len(props.Labels))
		}
	}
	return nil
}

func validateApprovalFormProps(props *ApprovalFormProps) error {
	if strings.TrimSpace(props.Title) == "" {
		return fmt.Errorf("ApprovalForm requiere title")
	}
	if strings.TrimSpace(props.Message) == "" {
		return fmt.Errorf("ApprovalForm requiere message")
	}
	if strings.TrimSpace(props.ActionID) == "" {
		return fmt.Errorf("ApprovalForm requiere action_id")
	}
	if strings.TrimSpace(props.ConfirmLabel) == "" {
		props.ConfirmLabel = "Confirmar"
	}
	if strings.TrimSpace(props.CancelLabel) == "" {
		props.CancelLabel = "Cancelar"
	}
	return nil
}

func validateMarkdownBlockProps(props *MarkdownBlockProps) error {
	if strings.TrimSpace(props.Markdown) == "" {
		return fmt.Errorf("MarkdownBlock requiere markdown")
	}
	return nil
}
