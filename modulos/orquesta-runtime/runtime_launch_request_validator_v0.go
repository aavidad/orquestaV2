package orquestaruntime

func ValidateRuntimeLaunchRequestV0(req RuntimeLaunchRequestV0) []RuntimeLaunchErrorV0 {
	v := runtimeLaunchRequestValidatorV0{correlationID: req.CorrelationID}

	v.requireConst("schema_version", req.SchemaVersion, RuntimeLaunchRequestSchemaVersionV0, RuntimeLaunchRequestInvalidaV0)
	v.requireOpaque("request_id", req.RequestID, RuntimeLaunchRequestInvalidaV0)
	v.requireOpaque("correlation_id", req.CorrelationID, RuntimeLaunchRequestInvalidaV0)
	v.requireOpaque("idempotency_key", req.IdempotencyKey, IdempotencyKeyRequeridaV0)
	v.requireUTCTime("requested_at", req.RequestedAt)
	v.validateSource(req.Source)
	v.requireLocale("locale", req.Locale)
	v.requireConst("launch_mode", req.LaunchMode, RuntimeLaunchModeNewSessionV0, RuntimeLaunchRequestInvalidaV0)
	v.validateTask(req.Task)
	v.validateFunctionContract(req.FunctionContract)
	v.validateCapacityDecision(req.CapacityDecision)
	v.validateRuntimeBinding(req.RuntimeBinding, req.CapacityDecision)
	v.validateEvidenceRefs(req.EvidenceRefs)
	v.validateContextBundle(req.ContextBundle, req.FunctionContract, req.CapacityDecision)
	v.validateDelivery(req.Delivery)
	v.validateSafety(req.Safety)

	return v.errors
}

func (req RuntimeLaunchRequestV0) Validate() []RuntimeLaunchErrorV0 {
	return ValidateRuntimeLaunchRequestV0(req)
}

func (req RuntimeLaunchRequestV0) Valid() bool {
	return len(ValidateRuntimeLaunchRequestV0(req)) == 0
}

type runtimeLaunchRequestValidatorV0 struct {
	correlationID string
	errors        []RuntimeLaunchErrorV0
}

func (v *runtimeLaunchRequestValidatorV0) validateSource(source *RuntimeLaunchSourceV0) {
	if source == nil {
		v.add(RuntimeLaunchRequestInvalidaV0, "source")
		return
	}
	v.requireConst("source.module", source.Module, RuntimeLaunchSourceModuleCoreV0, RuntimeLaunchRequestInvalidaV0)
	v.requireOpaque("source.adapter_ref", source.AdapterRef, RuntimeLaunchRequestInvalidaV0)
}

func (v *runtimeLaunchRequestValidatorV0) validateTask(task *RuntimeLaunchTaskV0) {
	if task == nil {
		v.add(RuntimeLaunchRequestInvalidaV0, "task")
		return
	}
	v.requireOpaque("task.task_ref", task.TaskRef, RuntimeLaunchRequestInvalidaV0)
	v.optionalOpaque("task.project_ref", task.ProjectRef)
	v.optionalOpaque("task.phase_ref", task.PhaseRef)
	if !isOneOf(task.Priority, "low", "normal", "high", "critical") {
		v.add(RuntimeLaunchRequestInvalidaV0, "task.priority")
	}
}
