package orquestacore

import (
	orquestafactory "orquesta/modulos/orquesta-factory"
)

const (
	RegistrarProyectoDesdeAppSpecVersionV0 = "v0"

	ErrAppSpecRequeridaV0                  = "app_spec_requerida"
	ErrAppSpecNoValidadaV0                 = "app_spec_no_validada"
	ErrBacklogRequeridoV0                  = "backlog_requerido"
	ErrBacklogVacioV0                      = "backlog_vacio"
	ErrBacklogIncompatibleConAppSpecV0     = "backlog_incompatible_con_app_spec"
	ErrIdempotencyKeyRequeridaV0           = "idempotency_key_requerida"
	ErrContratoFactoryNoSoportadoV0        = "contrato_factory_no_soportado"
	EstadoProyectoPlanBorradorV0           = "borrador"
	EstadoFasePlanificadaPendienteV0       = "pendiente"
	EventoProyectoRegistradoEnBorradorV0   = "ProyectoRegistradoEnBorradorV0"
	payloadVersionProyectoBorradorEventoV0 = "proyecto_borrador.v0"
)

type RegistrarProyectoDesdeAppSpecCommandV0 struct {
	IdempotencyKey string                                    `json:"idempotency_key"`
	AppSpec        orquestafactory.AppSpecV0                 `json:"app_spec"`
	AppSpecVersion string                                    `json:"app_spec_version"`
	Backlog        orquestafactory.BacklogInicialPropuestoV0 `json:"backlog"`
	BacklogVersion string                                    `json:"backlog_version"`
	Origen         string                                    `json:"origen,omitempty"`
	CorrelationID  string                                    `json:"correlation_id,omitempty"`
	RequestID      string                                    `json:"request_id,omitempty"`
	SolicitadoEn   string                                    `json:"solicitado_en,omitempty"`
}

type RegistroProyectoAceptadoV0 struct {
	RegistroID           string                 `json:"registro_id"`
	ProyectoPlanBorrador ProyectoPlanBorradorV0 `json:"proyecto_plan_borrador"`
	EventosDominio       []EventoDominioCoreV0  `json:"eventos_dominio"`
	Warnings             []string               `json:"warnings"`
}

type ProyectoPlanBorradorV0 struct {
	ProyectoIDPropuesto    string              `json:"proyecto_id_propuesto"`
	Nombre                 string              `json:"nombre"`
	Estado                 string              `json:"estado"`
	AppSpecRef             string              `json:"app_spec_ref"`
	RequestID              string              `json:"request_id,omitempty"`
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
	FaseID              string   `json:"fase_id,omitempty"`
	ModuloSugerido      string   `json:"modulo_sugerido"`
	Prioridad           string   `json:"prioridad"`
	Dependencias        []string `json:"dependencias"`
	CriteriosAceptacion []string `json:"criterios_aceptacion"`
	ContratoRequerido   string   `json:"contrato_requerido,omitempty"`
	WriteSetPrevisto    []string `json:"write_set_previsto,omitempty"`
	FuenteBacklogID     string   `json:"fuente_backlog_id"`
}

type EventoDominioCoreV0 struct {
	Tipo           string         `json:"tipo"`
	ProyectoID     string         `json:"proyecto_id"`
	OccurredAt     string         `json:"occurred_at"`
	PayloadVersion string         `json:"payload_version"`
	Payload        map[string]any `json:"payload"`
	CorrelationID  string         `json:"correlation_id,omitempty"`
}

type RegistrarProyectoDesdeAppSpecErrorV0 struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	Field         string `json:"field,omitempty"`
	Retryable     bool   `json:"retryable"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

func (err RegistrarProyectoDesdeAppSpecErrorV0) Error() string {
	return err.Code
}

func RegistrarProyectoDesdeAppSpecV0(cmd RegistrarProyectoDesdeAppSpecCommandV0) (RegistroProyectoAceptadoV0, error) {
	if err := validarRegistrarProyectoCommandV0(cmd); err != nil {
		return RegistroProyectoAceptadoV0{}, err
	}
	plan := construirProyectoPlanBorradorV0(cmd)
	return RegistroProyectoAceptadoV0{
		RegistroID:           stableIDV0("registro", cmd.IdempotencyKey),
		ProyectoPlanBorrador: plan,
		EventosDominio:       []EventoDominioCoreV0{proyectoRegistradoEventoV0(cmd, plan)},
		Warnings:             []string{},
	}, nil
}
