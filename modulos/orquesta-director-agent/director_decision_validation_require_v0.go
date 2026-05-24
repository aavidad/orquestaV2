package orquestadirectoragent

func (v *directorAgentDecisionValidatorV0) require(field string, got string, want string) {
	if got != want {
		v.add("director_agent_valor_invalido", field)
	}
}

func (v *directorAgentDecisionValidatorV0) requireRef(field string, value string) {
	if !directorAgentRefCompactV0(value) || directorAgentDecisionFieldHasForbiddenDetailV0(field, value) {
		v.add("director_agent_ref_invalida", field)
	}
}

func (v *directorAgentDecisionValidatorV0) requireOptionalRef(field string, value string) {
	if value == "" {
		return
	}
	v.requireRef(field, value)
}

func (v *directorAgentDecisionValidatorV0) requireText(field string, value string) {
	if value == "" || len(value) > maxDirectorAgentStringV0 ||
		directorAgentDecisionFieldHasForbiddenDetailV0(field, value) {
		v.add("director_agent_texto_invalido", field)
	}
}

func (v *directorAgentDecisionValidatorV0) requireEvidence(field string, values []string) {
	if len(values) > maxDirectorAgentEvidenceRefsV0 {
		v.add("director_agent_evidence_invalida", field)
		return
	}
	for _, value := range values {
		v.requireRef(field, value)
	}
}

func (v *directorAgentDecisionValidatorV0) requireOptionalRefs(field string, values []string) {
	if len(values) > maxDirectorAgentEvidenceRefsV0 {
		v.add("director_agent_evidence_invalida", field)
		return
	}
	for _, value := range values {
		v.requireRef(field, value)
	}
}

func (v *directorAgentDecisionValidatorV0) requireOptionalWorkRefs(field string, values []string) {
	if len(values) > maxDirectorAgentRecursionLimitV0 {
		v.add("director_agent_lista_invalida", field)
		return
	}
	for _, value := range values {
		v.requireRef(field, value)
	}
}

func (v *directorAgentDecisionValidatorV0) requireOptionalContextRefs(field string, values []string) {
	if len(values) > maxDirectorAgentEvidenceRefsV0 {
		v.add("director_agent_evidence_invalida", field)
		return
	}
	for _, value := range values {
		if !directorAgentContextRefCompactV0(value) ||
			directorAgentDecisionFieldHasForbiddenDetailV0(field, value) {
			v.add("director_agent_ref_invalida", field)
		}
	}
}

func (v *directorAgentDecisionValidatorV0) requireTextList(field string, values []string) {
	if len(values) == 0 || len(values) > maxDirectorAgentTextListV0 {
		v.add("director_agent_lista_invalida", field)
		return
	}
	for _, value := range values {
		v.requireText(field, value)
	}
}

func (v *directorAgentDecisionValidatorV0) requireOperationalTextList(field string, values []string) {
	if len(values) == 0 || len(values) > maxDirectorAgentTextListV0 {
		v.add("director_agent_lista_invalida", field)
		return
	}
	for _, value := range values {
		v.requireOperationalText(field, value)
	}
}

func (v *directorAgentDecisionValidatorV0) requireOptionalOperationalTextList(field string, values []string) {
	if len(values) == 0 {
		return
	}
	v.requireOperationalTextList(field, values)
}

func (v *directorAgentDecisionValidatorV0) requireOperationalText(field string, value string) {
	if value == "" || len(value) > maxDirectorAgentStringV0 {
		v.add("director_agent_texto_invalido", field)
	}
}

func (v *directorAgentDecisionValidatorV0) validateFunctionRefs(
	field string,
	refs []DirectorAgentFunctionContractRefV0,
) {
	if len(refs) == 0 || len(refs) > maxDirectorAgentEvidenceRefsV0 {
		v.add("director_agent_lista_invalida", field)
		return
	}
	for _, ref := range refs {
		if ref.ContractRef == "" {
			v.add("director_agent_ref_invalida", field+".contract_ref")
			continue
		}
		v.requireRef(field+".contract_ref", ref.ContractRef)
		if ref.FunctionName != "" {
			v.requireText(field+".function_name", ref.FunctionName)
		}
	}
}

func (v *directorAgentDecisionValidatorV0) requireCapacity(field string, value string) {
	if value == "" {
		return
	}
	switch value {
	case DirectorAgentCapacityLowV0, DirectorAgentCapacityMediumV0,
		DirectorAgentCapacityHighV0, DirectorAgentCapacityXHighV0:
		return
	default:
		v.add("director_agent_capacidad_invalida", field)
	}
}

func (v *directorAgentDecisionValidatorV0) requireDirectorAnswer(field string, value string) {
	switch value {
	case DirectorAgentAnswerContinueV0, DirectorAgentAnswerReplanV0, DirectorAgentAnswerStopAgentV0:
		return
	default:
		v.add("director_agent_answer_invalida", field)
	}
}

func (v *directorAgentDecisionValidatorV0) add(code string, field string) {
	v.issues = append(v.issues, DirectorAgentDecisionIssueV0{Code: code, Field: field})
}
