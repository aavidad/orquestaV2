package orquestacore

import "context"

const (
	FunctionContractQueryErrIncompletoV0            = "function_contract_incompleto"
	FunctionContractQueryErrConsultaNoDisponibleV0  = "consulta_function_contract_no_disponible"
	FunctionContractQueryErrFiltroNoSoportadoV0     = "filtro_no_soportado"
	FunctionContractQueryErrCursorInvalidoV0        = "cursor_invalido"
	FunctionContractQueryErrEvidenciaInsuficienteV0 = "evidencia_insuficiente"
	FunctionContractQueryErrNoEncontradoV0          = "function_contract_no_encontrado"
)

type FunctionContractReadIndexPortV0 interface {
	ListFunctionContractsV0(context.Context, ListFunctionContractsRequestV0) (ListFunctionContractsResultV0, error)
	ViewFunctionContractV0(context.Context, ViewFunctionContractRequestV0) (ViewFunctionContractResultV0, error)
}

type ListFunctionContractsRequestV0 struct {
	RequestID     string                        `json:"request_id"`
	CorrelationID string                        `json:"correlation_id"`
	Filtros       FunctionContractFiltersV0     `json:"filtros"`
	Page          FunctionContractPageRequestV0 `json:"page"`
}

type FunctionContractFiltersV0 struct {
	Modulo          string `json:"modulo,omitempty"`
	Estado          string `json:"estado,omitempty"`
	ArchivoObjetivo string `json:"archivo_objetivo,omitempty"`
	SimboloObjetivo string `json:"simbolo_objetivo,omitempty"`
}

type FunctionContractPageRequestV0 struct {
	Limit  int    `json:"limit,omitempty"`
	Cursor string `json:"cursor,omitempty"`
}

type ListFunctionContractsResultV0 struct {
	Items      []FunctionContractSummaryV0 `json:"items"`
	NextCursor string                      `json:"next_cursor,omitempty"`
	Warnings   []string                    `json:"warnings"`
}

type FunctionContractSummaryV0 struct {
	FunctionContractRef string `json:"function_contract_ref"`
	Titulo              string `json:"titulo,omitempty"`
	Modulo              string `json:"modulo,omitempty"`
	ArchivoObjetivo     string `json:"archivo_objetivo,omitempty"`
	SimboloObjetivo     string `json:"simbolo_objetivo,omitempty"`
	Estado              string `json:"estado,omitempty"`
	Version             int    `json:"version,omitempty"`
}

type ViewFunctionContractRequestV0 struct {
	RequestID           string `json:"request_id"`
	CorrelationID       string `json:"correlation_id"`
	FunctionContractRef string `json:"function_contract_ref"`
	Version             int    `json:"version,omitempty"`
}

type ViewFunctionContractResultV0 struct {
	FunctionContract FunctionContractV0 `json:"function_contract"`
	Warnings         []string           `json:"warnings"`
}

type FunctionContractQueryErrorV0 struct {
	Errores []FunctionContractErrorV0 `json:"errores"`
}

func (err FunctionContractQueryErrorV0) Error() string {
	if len(err.Errores) == 0 {
		return FunctionContractQueryErrConsultaNoDisponibleV0
	}
	return err.Errores[0].Code
}

func (err FunctionContractQueryErrorV0) FunctionContractErrorsV0() []FunctionContractErrorV0 {
	return append([]FunctionContractErrorV0(nil), err.Errores...)
}

func NewFunctionContractQueryErrorV0(code string, field string, evidence []string) FunctionContractQueryErrorV0 {
	return FunctionContractQueryErrorV0{Errores: []FunctionContractErrorV0{{
		Code:     code,
		Field:    field,
		Message:  code,
		Evidence: append([]string(nil), evidence...),
	}}}
}
