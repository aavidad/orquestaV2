package orquestacoreworkflow

import "strings"

func ValidateWorkProfileV0(profile WorkProfileV0) error {
	profile = NormalizeWorkProfileV0(profile)
	if err := validateWorkProfileFieldsV0(profile); err != nil {
		return err
	}
	_, err := workflowTaskFromWorkProfileUncheckedV0(profile)
	return err
}

func validateWorkProfileFieldsV0(profile WorkProfileV0) error {
	definition, ok := LookupWorkProfileDefinitionV0(profile.ProfileKind)
	if profile.SchemaVersion != WorkProfileSchemaVersionV0 {
		return workProfileErrorV0("schema_version")
	}
	if !workProfileRefIsValidV0(profile.ProfileRef) {
		return workProfileErrorV0("profile_ref")
	}
	if !ok {
		return workProfileErrorV0("profile_kind")
	}
	if strings.TrimSpace(profile.TaskRef) == "" {
		return workProfileErrorV0("task_ref")
	}
	if strings.TrimSpace(profile.RunRef) == "" {
		return workProfileErrorV0("run_ref")
	}
	if strings.TrimSpace(profile.Title) == "" {
		return workProfileErrorV0("title")
	}
	if strings.TrimSpace(profile.Objective) == "" {
		return workProfileErrorV0("objective")
	}
	if len(profile.ScopeRefs) == 0 {
		return workProfileErrorV0("scope_refs")
	}
	if len(profile.FunctionContractRefs) == 0 {
		return workProfileErrorV0("function_contract_refs")
	}
	if definition.RequiredTestsRequired && len(profile.RequiredTests) == 0 {
		return workProfileErrorV0("required_tests")
	}
	return nil
}

func workProfileRefIsValidV0(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" &&
		workflowTaskOptionalRefIsCompactV0(value) &&
		!workflowTaskStringHasForbiddenDetailV0(value)
}

func workProfileErrorV0(field string) WorkProfileErrorV0 {
	return WorkProfileErrorV0{Code: ErrWorkProfileInvalidoV0, Field: field}
}

func workProfileTaskErrorV0(field string) WorkProfileErrorV0 {
	return WorkProfileErrorV0{Code: ErrWorkProfileTaskInvalidaV0, Field: field}
}
