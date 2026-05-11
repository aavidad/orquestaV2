package orquestapersistence

func ValidateGuardarProyectoBorradorMaterialV0(material GuardarProyectoBorradorMaterialV0) []PersistenceRepositoryValidationIssueV0 {
	var issues []PersistenceRepositoryValidationIssueV0
	if trimV0(material.SchemaVersion) != PersistenceRepositoryContractVersionV0 {
		issues = append(issues, issueV0(ErrPayloadInvalidoV0, "schema_version", "schema_version no soportada"))
	}
	if trimV0(material.Contrato) != PersistenceRepositoryContractVersionV0 {
		issues = append(issues, issueV0(ErrPayloadInvalidoV0, "contrato", "contrato no soportado"))
	}
	if trimV0(material.Operacion) != PersistenceRepositoryOperationGuardarProyectoBorradorV0 {
		issues = append(issues, issueV0(ErrPayloadInvalidoV0, "operacion", "operacion no soportada"))
	}
	issues = append(issues, validateAdapterMetadataV0(material.AdapterMetadata)...)
	issues = append(issues, validateUnidadTrabajoDeclaradaV0(material.UnidadTrabajo)...)
	issues = append(issues, prefixIssuesV0("request.", ValidateGuardarProyectoBorradorRequestV0(material.Request))...)
	issues = append(issues, validateResponseV0(material.Response, material.Request)...)
	issues = append(issues, validateErroresPublicosV0(material.ErroresPublicos)...)
	return issues
}

func ValidateGuardarProyectoBorradorRequestV0(request GuardarProyectoBorradorRequestV0) []PersistenceRepositoryValidationIssueV0 {
	var issues []PersistenceRepositoryValidationIssueV0
	issues = append(issues, validateRequestEnvelopeNoForbiddenPersistenceTermsV0(request)...)
	addRequiredV0(&issues, "request_id", request.RequestID)
	addRequiredV0(&issues, "correlation_id", request.CorrelationID)
	addRequiredV0(&issues, "idempotency_key", request.IdempotencyKey)
	if trimV0(request.PayloadVersion) != ProyectoPlanBorradorPayloadVersionV0 {
		issues = append(issues, issueV0(ErrPayloadInvalidoV0, "payload_version", "payload_version debe ser ProyectoPlanBorradorV0"))
	}
	issues = append(issues, validateProyectoPlanBorradorV0(request.Payload, "payload.")...)
	return issues
}

func validateRequestEnvelopeNoForbiddenPersistenceTermsV0(request GuardarProyectoBorradorRequestV0) []PersistenceRepositoryValidationIssueV0 {
	values := map[string]string{
		"request_id":      request.RequestID,
		"correlation_id":  request.CorrelationID,
		"idempotency_key": request.IdempotencyKey,
		"payload_version": request.PayloadVersion,
	}
	var issues []PersistenceRepositoryValidationIssueV0
	for field, value := range values {
		if containsForbiddenPersistenceTermV0(value) {
			issues = append(issues, issueV0(
				ErrPersistenceConnectorNoSoportadoV0,
				field,
				"detalle concreto de persistencia no pertenece al contrato",
			))
		}
	}
	return issues
}

func validateAdapterMetadataV0(metadata PersistenceAdapterMetadataV0) []PersistenceRepositoryValidationIssueV0 {
	var issues []PersistenceRepositoryValidationIssueV0
	issues = append(issues, validateNoForbiddenPersistenceTermsV0(metadata, "adapter_metadata.")...)
	if len(metadata.ConnectorProfileRefs) == 0 {
		issues = append(issues, issueV0(
			ErrPersistenceConnectorNoSoportadoV0,
			"adapter_metadata.connector_profile_refs",
			"connector_profile_refs requerido",
		))
	}
	for _, ref := range metadata.ConnectorProfileRefs {
		if !validPersistenceConnectorRefV0(ref) {
			issues = append(issues, issueV0(
				ErrPersistenceConnectorNoSoportadoV0,
				"adapter_metadata.connector_profile_refs",
				"connector_profile_ref no opaco",
			))
		}
	}
	capabilities := stringSetV0(metadata.Capacidades)
	for _, required := range []string{capacidadTransaccionesV0, capacidadIdempotenciaV0, capacidadPayloadVersionadoV0} {
		if !capabilities[required] {
			issues = append(issues, issueV0(ErrPayloadInvalidoV0, "adapter_metadata.capacidades", "capacidad requerida ausente"))
		}
	}
	return issues
}

func validateUnidadTrabajoDeclaradaV0(unit PersistenceUnidadTrabajoDeclaradaV0) []PersistenceRepositoryValidationIssueV0 {
	var issues []PersistenceRepositoryValidationIssueV0
	if trimV0(unit.TransaccionID) == "" {
		issues = append(issues, issueV0(ErrTransaccionFallidaV0, "unidad_trabajo.transaccion_id", "transaccion_id requerido"))
	}
	if trimV0(unit.Estado) != PersistenceTransaccionEstadoActivaV0 {
		issues = append(issues, issueV0(ErrTransaccionFallidaV0, "unidad_trabajo.estado", "unidad de trabajo no activa"))
	}
	if unit.ReadOnly {
		issues = append(issues, issueV0(ErrTransaccionFallidaV0, "unidad_trabajo.read_only", "guardar requiere unidad de trabajo de escritura"))
	}
	if trimV0(unit.Aislamiento) == "" {
		issues = append(issues, issueV0(ErrTransaccionFallidaV0, "unidad_trabajo.aislamiento", "aislamiento requerido"))
	}
	return issues
}

func validateResponseV0(response GuardarProyectoBorradorResponseV0, request GuardarProyectoBorradorRequestV0) []PersistenceRepositoryValidationIssueV0 {
	var issues []PersistenceRepositoryValidationIssueV0
	if trimV0(response.Status) != PersistenceEstadoPersistidoV0 {
		issues = append(issues, issueV0(ErrPayloadInvalidoV0, "response.status", "status debe ser persistido"))
	}
	addRequiredV0(&issues, "response.proyecto_id", response.ProyectoID)
	if trimV0(response.IdempotencyKey) == "" {
		issues = append(issues, issueV0(ErrPayloadInvalidoV0, "response.idempotency_key", "idempotency_key requerido"))
	} else if trimV0(request.IdempotencyKey) != "" && trimV0(response.IdempotencyKey) != trimV0(request.IdempotencyKey) {
		issues = append(issues, issueV0(ErrConflictoIdempotenciaV0, "response.idempotency_key", "idempotency_key no coincide"))
	}
	if trimV0(response.PayloadVersion) != ProyectoPlanBorradorPayloadVersionV0 {
		issues = append(issues, issueV0(ErrPayloadInvalidoV0, "response.payload_version", "payload_version debe ser ProyectoPlanBorradorV0"))
	}
	addRequiredV0(&issues, "response.created_at", response.CreatedAt)
	addRequiredV0(&issues, "response.updated_at", response.UpdatedAt)
	return issues
}

func validateErroresPublicosV0(values []string) []PersistenceRepositoryValidationIssueV0 {
	required := []string{
		ErrPersistenciaNoDisponibleV0,
		ErrPayloadInvalidoV0,
		ErrPersistenceConnectorNoSoportadoV0,
		ErrConflictoIdempotenciaV0,
		ErrTransaccionFallidaV0,
		ErrReglaNegocioEnAdaptadorV0,
	}
	seen := stringSetV0(values)
	var issues []PersistenceRepositoryValidationIssueV0
	for _, value := range required {
		if !seen[value] {
			issues = append(issues, issueV0(ErrPayloadInvalidoV0, "errores_publicos", "error publico requerido ausente"))
		}
	}
	return issues
}

func validateProyectoPlanBorradorV0(payload ProyectoPlanBorradorV0, prefix string) []PersistenceRepositoryValidationIssueV0 {
	var issues []PersistenceRepositoryValidationIssueV0
	addRequiredV0(&issues, prefix+"proyecto_id_propuesto", payload.ProyectoIDPropuesto)
	addRequiredV0(&issues, prefix+"nombre", payload.Nombre)
	if trimV0(payload.Estado) != ProyectoPlanBorradorEstadoV0 {
		issues = append(issues, issueV0(ErrPayloadInvalidoV0, prefix+"estado", "estado debe ser borrador"))
	}
	addRequiredV0(&issues, prefix+"app_spec_ref", payload.AppSpecRef)
	addRequiredV0(&issues, prefix+"request_id", payload.RequestID)
	if len(payload.FasesIniciales) == 0 {
		issues = append(issues, issueV0(ErrPayloadInvalidoV0, prefix+"fases_iniciales", "fases_iniciales requerido"))
	}
	for index, fase := range payload.FasesIniciales {
		issues = append(issues, validateFasePlanificadaV0(fase, indexedPrefixV0(prefix+"fases_iniciales", index))...)
	}
	if len(payload.Microtareas) == 0 {
		issues = append(issues, issueV0(ErrPayloadInvalidoV0, prefix+"microtareas", "microtareas requerido"))
	}
	for index, item := range payload.Microtareas {
		issues = append(issues, validateItemBacklogCoreV0(item, indexedPrefixV0(prefix+"microtareas", index))...)
	}
	if len(payload.BacklogNormalizado) == 0 {
		issues = append(issues, issueV0(ErrPayloadInvalidoV0, prefix+"backlog_normalizado", "backlog_normalizado requerido"))
	}
	for index, item := range payload.BacklogNormalizado {
		issues = append(issues, validateItemBacklogCoreV0(item, indexedPrefixV0(prefix+"backlog_normalizado", index))...)
	}
	issues = append(issues, validateStringListV0(payload.ContratosRequeridos, prefix+"contratos_requeridos")...)
	issues = append(issues, validateStringListV0(payload.CriteriosCierre, prefix+"criterios_cierre")...)
	issues = append(issues, validateStringListV0(payload.DependenciasPendientes, prefix+"dependencias_pendientes")...)
	return issues
}

func validateFasePlanificadaV0(fase FasePlanificadaV0, prefix string) []PersistenceRepositoryValidationIssueV0 {
	var issues []PersistenceRepositoryValidationIssueV0
	addRequiredV0(&issues, prefix+"id", fase.ID)
	addRequiredV0(&issues, prefix+"nombre", fase.Nombre)
	if fase.Orden < 1 {
		issues = append(issues, issueV0(ErrPayloadInvalidoV0, prefix+"orden", "orden debe ser mayor o igual a 1"))
	}
	if trimV0(fase.Estado) != FasePlanificadaEstadoPendienteV0 {
		issues = append(issues, issueV0(ErrPayloadInvalidoV0, prefix+"estado", "estado de fase debe ser pendiente"))
	}
	issues = append(issues, validateStringListV0(fase.CriteriosEntrada, prefix+"criterios_entrada")...)
	issues = append(issues, validateStringListV0(fase.CriteriosSalida, prefix+"criterios_salida")...)
	return issues
}

func validateItemBacklogCoreV0(item ItemBacklogCoreV0, prefix string) []PersistenceRepositoryValidationIssueV0 {
	var issues []PersistenceRepositoryValidationIssueV0
	addRequiredV0(&issues, prefix+"id", item.ID)
	addRequiredV0(&issues, prefix+"titulo", item.Titulo)
	addRequiredV0(&issues, prefix+"descripcion", item.Descripcion)
	addRequiredV0(&issues, prefix+"fase_id", item.FaseID)
	addRequiredV0(&issues, prefix+"modulo_sugerido", item.ModuloSugerido)
	addRequiredV0(&issues, prefix+"prioridad", item.Prioridad)
	addRequiredV0(&issues, prefix+"contrato_requerido", item.ContratoRequerido)
	addRequiredV0(&issues, prefix+"fuente_backlog_id", item.FuenteBacklogID)
	issues = append(issues, validateStringListV0(item.Dependencias, prefix+"dependencias")...)
	issues = append(issues, validateStringListV0(item.CriteriosAceptacion, prefix+"criterios_aceptacion")...)
	issues = append(issues, validateStringListV0(item.WriteSetPrevisto, prefix+"write_set_previsto")...)
	return issues
}

func validateStringListV0(values []string, field string) []PersistenceRepositoryValidationIssueV0 {
	if values == nil {
		return []PersistenceRepositoryValidationIssueV0{issueV0(ErrPayloadInvalidoV0, field, "lista requerida")}
	}
	var issues []PersistenceRepositoryValidationIssueV0
	for _, value := range values {
		if trimV0(value) == "" {
			issues = append(issues, issueV0(ErrPayloadInvalidoV0, field, "lista no admite valores vacios"))
		}
	}
	return issues
}
