package orquestaoperatormcp

func NewOperatorMCPCapabilitiesV0() OperatorMCPCapabilitiesV0 {
	return OperatorMCPCapabilitiesV0{
		SchemaVersion: OperatorMCPSchemaVersionV0,
		ResourceURI:   OperatorMCPCapabilitiesURIV0,
		Tools: []OperatorMCPToolDescriptorV0{
			operatorStatusToolDescriptorV0(),
			operatorBurstToolDescriptorV0(),
			operatorOutboxToolDescriptorV0(),
			operatorDirectedQueryToolDescriptorV0(),
		},
		OpaqueRefs: []string{"request_ref", "subject_ref", "run_ref", "supervision_ref", "evidence_ref", "message_ref", "answer_ref"},
		Guardrails: []string{
			"delegate operational work to connectors",
			"return compact public errors and evidence refs",
			"do not expose storage, accounts or implementation names",
		},
	}
}

func operatorStatusToolDescriptorV0() OperatorMCPToolDescriptorV0 {
	return OperatorMCPToolDescriptorV0{
		Name:              OperatorMCPStatusToolNameV0,
		Version:           "v0",
		ConnectorRef:      "status_connector_ref",
		InputShape:        "OperatorStatusQueryV0",
		InputRefs:         []string{"request_ref", "subject_ref", "status_connector_ref", "include_sections"},
		RequiredInputRefs: []string{"request_ref", "subject_ref", "status_connector_ref"},
		OutputShape:       "compact_status_with_evidence_refs",
		PublicErrors:      operatorToolPublicErrorsV0(),
	}
}

func operatorBurstToolDescriptorV0() OperatorMCPToolDescriptorV0 {
	return OperatorMCPToolDescriptorV0{
		Name:              OperatorMCPBurstToolNameV0,
		Version:           "v0",
		ConnectorRef:      "burst_connector_ref",
		InputShape:        "OperatorSupervisedBurstRequestV0",
		InputRefs:         []string{"request_ref", "run_ref", "burst_connector_ref", "supervision_ref", "max_steps", "evidence_refs"},
		RequiredInputRefs: []string{"request_ref", "run_ref", "burst_connector_ref", "supervision_ref", "max_steps"},
		OutputShape:       "compact_burst_trace_refs",
		PublicErrors:      operatorToolPublicErrorsV0(),
	}
}

func operatorOutboxToolDescriptorV0() OperatorMCPToolDescriptorV0 {
	return OperatorMCPToolDescriptorV0{
		Name:              OperatorMCPOutboxToolNameV0,
		Version:           "v0",
		ConnectorRef:      "outbox_connector_ref",
		InputShape:        "OperatorPendingOutboxQueryV0",
		InputRefs:         []string{"request_ref", "subject_ref", "outbox_connector_ref", "limit", "include_kinds"},
		RequiredInputRefs: []string{"request_ref", "subject_ref", "outbox_connector_ref", "limit"},
		OutputShape:       "pending_message_refs_with_counts",
		PublicErrors:      operatorToolPublicErrorsV0(),
	}
}

func operatorDirectedQueryToolDescriptorV0() OperatorMCPToolDescriptorV0 {
	return OperatorMCPToolDescriptorV0{
		Name:              OperatorMCPDirectedQueryToolV0,
		Version:           "v0",
		ConnectorRef:      "query_connector_ref",
		InputShape:        "OperatorDirectedQueryV0",
		InputRefs:         []string{"query_ref", "target_ref", "query_connector_ref", "question", "evidence_refs"},
		RequiredInputRefs: []string{"query_ref", "target_ref", "query_connector_ref", "question"},
		OutputShape:       "accepted_answer_ref_or_next_action",
		PublicErrors:      operatorToolPublicErrorsV0(),
	}
}

func operatorToolPublicErrorsV0() []string {
	return []string{
		ErrOperatorMCPRequiredFieldV0,
		ErrOperatorMCPOpaqueRefV0,
		ErrOperatorMCPBudgetInvalidV0,
		ErrOperatorMCPLimitInvalidV0,
		ErrOperatorMCPQuestionInvalidV0,
		ErrOperatorMCPSectionInvalidV0,
		ErrOperatorMCPPortUnavailableV0,
		ErrOperatorMCPPortErrorV0,
		ErrOperatorMCPConnectorUnavailableV0,
	}
}
