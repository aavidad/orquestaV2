package orquestapersistence

const (
	PersistenceRepositoryContractVersionV0                  = "PersistenceRepositoryV0"
	PersistenceRepositoryOperationGuardarProyectoBorradorV0 = "guardar_proyecto_borrador"
	ProyectoPlanBorradorPayloadVersionV0                    = "ProyectoPlanBorradorV0"
	PersistenceEstadoPersistidoV0                           = "persistido"
	PersistenceTransaccionEstadoActivaV0                    = "activa"
	ProyectoPlanBorradorEstadoV0                            = "borrador"
	FasePlanificadaEstadoPendienteV0                        = "pendiente"

	ErrPersistenceJSONInvalidoV0         = "persistence_repository_json_invalido"
	ErrPayloadInvalidoV0                 = "payload_invalido"
	ErrPersistenceConnectorNoSoportadoV0 = "conector_persistencia_no_soportado"
	ErrTransaccionFallidaV0              = "transaccion_fallida"
	ErrReglaNegocioEnAdaptadorV0         = "regla_negocio_en_adaptador"
	ErrPersistenciaNoDisponibleV0        = "persistencia_no_disponible"
	ErrConflictoIdempotenciaV0           = "conflicto_idempotencia"

	capacidadTransaccionesV0     = "transacciones"
	capacidadIdempotenciaV0      = "idempotencia"
	capacidadPayloadVersionadoV0 = "payload_versionado"
)

// GuardarProyectoBorradorMaterialV0 mirrors the local contract fixture only.
// A future adapter must build the active unit of work and any global envelope.
type GuardarProyectoBorradorMaterialV0 struct {
	SchemaVersion   string                              `json:"schema_version"`
	Contrato        string                              `json:"contrato"`
	Operacion       string                              `json:"operacion"`
	AdapterMetadata PersistenceAdapterMetadataV0        `json:"adapter_metadata"`
	UnidadTrabajo   PersistenceUnidadTrabajoDeclaradaV0 `json:"unidad_trabajo"`
	Request         GuardarProyectoBorradorRequestV0    `json:"request"`
	Response        GuardarProyectoBorradorResponseV0   `json:"response"`
	ErroresPublicos []string                            `json:"errores_publicos"`
}

type PersistenceAdapterMetadataV0 struct {
	ConnectorProfileRefs []string `json:"connector_profile_refs"`
	Capacidades          []string `json:"capacidades"`
}

type PersistenceUnidadTrabajoDeclaradaV0 struct {
	TransaccionID string `json:"transaccion_id"`
	Estado        string `json:"estado"`
	ReadOnly      bool   `json:"read_only"`
	Aislamiento   string `json:"aislamiento"`
}

type GuardarProyectoBorradorRequestV0 struct {
	RequestID      string                 `json:"request_id"`
	CorrelationID  string                 `json:"correlation_id"`
	IdempotencyKey string                 `json:"idempotency_key"`
	PayloadVersion string                 `json:"payload_version"`
	Payload        ProyectoPlanBorradorV0 `json:"payload"`
}

type GuardarProyectoBorradorResponseV0 struct {
	Status         string `json:"status"`
	ProyectoID     string `json:"proyecto_id"`
	IdempotencyKey string `json:"idempotency_key"`
	PayloadVersion string `json:"payload_version"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

type ProyectoPlanBorradorV0 struct {
	ProyectoIDPropuesto    string              `json:"proyecto_id_propuesto"`
	Nombre                 string              `json:"nombre"`
	Estado                 string              `json:"estado"`
	AppSpecRef             string              `json:"app_spec_ref"`
	RequestID              string              `json:"request_id"`
	FasesIniciales         []FasePlanificadaV0 `json:"fases_iniciales"`
	Microtareas            []ItemBacklogCoreV0 `json:"microtareas"`
	BacklogNormalizado     []ItemBacklogCoreV0 `json:"backlog_normalizado"`
	ContratosRequeridos    []string            `json:"contratos_requeridos"`
	CriteriosCierre        []string            `json:"criterios_cierre"`
	DependenciasPendientes []string            `json:"dependencias_pendientes"`
}

type FasePlanificadaV0 struct {
	ID               string   `json:"id"`
	Nombre           string   `json:"nombre"`
	Orden            int      `json:"orden"`
	Estado           string   `json:"estado"`
	CriteriosEntrada []string `json:"criterios_entrada"`
	CriteriosSalida  []string `json:"criterios_salida"`
}

type ItemBacklogCoreV0 struct {
	ID                  string   `json:"id"`
	Titulo              string   `json:"titulo"`
	Descripcion         string   `json:"descripcion"`
	FaseID              string   `json:"fase_id"`
	ModuloSugerido      string   `json:"modulo_sugerido"`
	Prioridad           string   `json:"prioridad"`
	Dependencias        []string `json:"dependencias"`
	CriteriosAceptacion []string `json:"criterios_aceptacion"`
	ContratoRequerido   string   `json:"contrato_requerido"`
	WriteSetPrevisto    []string `json:"write_set_previsto"`
	FuenteBacklogID     string   `json:"fuente_backlog_id"`
}

type PersistenceRepositoryValidationIssueV0 struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}
