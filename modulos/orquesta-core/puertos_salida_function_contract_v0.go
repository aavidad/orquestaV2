package orquestacore

const (
	FunctionContractFormatoPatchEvidenciaV0    = "patch+evidencia"
	FunctionContractFormatoPatchUnificadoV0    = "patch_unificado"
	FunctionContractFormatoFicherosEvidenciaV0 = "ficheros+evidencia"
	FunctionContractEstadoBorradorV0           = "borrador"
	FunctionContractEstadoActivaV0             = "activa"
)

type FunctionContractV0 struct {
	Titulo                 string                    `json:"titulo"`
	Objetivo               string                    `json:"objetivo"`
	ArchivoObjetivo        string                    `json:"archivo_objetivo"`
	SimboloObjetivo        string                    `json:"simbolo_objetivo"`
	Descripcion            string                    `json:"descripcion,omitempty"`
	WriteSet               []string                  `json:"write_set"`
	DependenciasPermitidas []string                  `json:"dependencias_permitidas,omitempty"`
	DependenciasProhibidas []string                  `json:"dependencias_prohibidas,omitempty"`
	Precondiciones         []string                  `json:"precondiciones,omitempty"`
	Postcondiciones        []string                  `json:"postcondiciones,omitempty"`
	TestsObligatorios      []string                  `json:"tests_obligatorios"`
	FormatoEntrega         string                    `json:"formato_entrega"`
	CriterioCierre         []string                  `json:"criterio_cierre"`
	ErroresPublicos        []FunctionContractErrorV0 `json:"errores_publicos,omitempty"`
	PruebasContrato        []string                  `json:"pruebas_contrato,omitempty"`
	Estado                 string                    `json:"estado"`
	Version                int                       `json:"version"`
	OrigenEvidencia        string                    `json:"origen_evidencia,omitempty"`
}

type FunctionContractErrorV0 struct {
	Code      string   `json:"code"`
	Message   string   `json:"message"`
	Field     string   `json:"field,omitempty"`
	Retryable bool     `json:"retryable"`
	Evidence  []string `json:"evidence,omitempty"`
}
