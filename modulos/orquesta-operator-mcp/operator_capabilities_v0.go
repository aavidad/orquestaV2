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
		Name:         OperatorMCPStatusToolNameV0,
		Version:      "v0",
		ConnectorRef: "status_connector_ref",
		InputRefs:    []string{"request_ref", "subject_ref"},
		OutputShape:  "compact_status_with_evidence_refs",
	}
}

func operatorBurstToolDescriptorV0() OperatorMCPToolDescriptorV0 {
	return OperatorMCPToolDescriptorV0{
		Name:         OperatorMCPBurstToolNameV0,
		Version:      "v0",
		ConnectorRef: "burst_connector_ref",
		InputRefs:    []string{"request_ref", "run_ref", "supervision_ref"},
		OutputShape:  "compact_burst_trace_refs",
	}
}

func operatorOutboxToolDescriptorV0() OperatorMCPToolDescriptorV0 {
	return OperatorMCPToolDescriptorV0{
		Name:         OperatorMCPOutboxToolNameV0,
		Version:      "v0",
		ConnectorRef: "outbox_connector_ref",
		InputRefs:    []string{"request_ref", "subject_ref"},
		OutputShape:  "pending_message_refs_with_counts",
	}
}

func operatorDirectedQueryToolDescriptorV0() OperatorMCPToolDescriptorV0 {
	return OperatorMCPToolDescriptorV0{
		Name:         OperatorMCPDirectedQueryToolV0,
		Version:      "v0",
		ConnectorRef: "query_connector_ref",
		InputRefs:    []string{"query_ref", "target_ref"},
		OutputShape:  "accepted_answer_ref_or_next_action",
	}
}
