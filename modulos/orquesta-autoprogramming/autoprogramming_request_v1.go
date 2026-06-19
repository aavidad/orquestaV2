package orquestaautoprogramming

import (
	"fmt"
	"strings"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

const (
	AutoprogrammingRequestSchemaVersionV1          = "autoprogramming_request.v1"
	AutoprogrammingProgrammableWorkSchemaVersionV1 = "autoprogramming_programmable_work.v1"
)

type AutoprogrammingRequestV1 struct {
	AutoprogrammingRequestV0
	SchemaVersion string                         `json:"schema_version,omitempty"`
	AppKind       string                         `json:"app_kind,omitempty"`
	WorkProfiles  []AutoprogrammingWorkProfileV1 `json:"work_profiles,omitempty"`
}

type AutoprogrammingWorkProfileV1 struct {
	AppKind              string                                               `json:"app_kind,omitempty"`
	Area                 string                                               `json:"area,omitempty"`
	TaskRef              string                                               `json:"task_ref,omitempty"`
	ProfileKind          orquestacoreworkflow.WorkProfileKindV0               `json:"profile_kind"`
	FunctionContractRefs []orquestacoreworkflow.WorkflowFunctionContractRefV0 `json:"function_contract_refs,omitempty"`
	SkillRefs            []string                                             `json:"skill_refs,omitempty"`
}

type AutoprogrammingResolvedWorkProfileV1 struct {
	Area        string                                 `json:"area"`
	TaskRefs    []string                               `json:"task_refs"`
	ProfileKind orquestacoreworkflow.WorkProfileKindV0 `json:"profile_kind"`
	Source      string                                 `json:"source"`
}

func ValidateAutoprogrammingRequestV1(
	request AutoprogrammingRequestV1,
) AutoprogrammingRequestValidationResultV0 {
	base := ValidateAutoprogrammingRequestV0(request.AutoprogrammingRequestV0)
	issues := append([]AutoprogrammingRequestIssueV0(nil), base.Issues...)
	issues = append(issues, autoprogrammingRequestProfileIssuesV1(request)...)
	return AutoprogrammingRequestValidationResultV0{
		Accepted:      len(issues) == 0,
		Groups:        base.Groups,
		WriteSet:      base.WriteSet,
		RequiredTests: base.RequiredTests,
		Issues:        issues,
	}
}

func autoprogrammingRequestProfileIssuesV1(
	request AutoprogrammingRequestV1,
) []AutoprogrammingRequestIssueV0 {
	var issues []AutoprogrammingRequestIssueV0
	if schema := strings.TrimSpace(request.SchemaVersion); schema != "" &&
		schema != AutoprogrammingRequestSchemaVersionV1 {
		issues = append(issues, autoprogrammingRequestIssueV0(
			"schema_version_invalid",
			"schema_version",
			"schema_version v1 no reconocido",
		))
	}
	if len(request.WorkProfiles) > 0 && normalizeAutoprogrammingAppKindV1(request.AppKind) == "" {
		issues = append(issues, autoprogrammingRequestIssueV0(
			"app_kind_missing",
			"app_kind",
			"app_kind requerido para perfiles de trabajo v1",
		))
	}
	for i, profile := range request.WorkProfiles {
		field := fmt.Sprintf("work_profiles[%d]", i)
		kind := orquestacoreworkflow.NormalizeWorkProfileKindV0(profile.ProfileKind)
		if strings.TrimSpace(string(kind)) == "" {
			issues = append(issues, autoprogrammingRequestIssueV0(
				"work_profile_kind_missing",
				field+".profile_kind",
				"profile_kind requerido",
			))
			continue
		}
		if _, ok := orquestacoreworkflow.LookupWorkProfileDefinitionV0(kind); !ok {
			issues = append(issues, autoprogrammingRequestIssueV0(
				"work_profile_kind_unknown",
				field+".profile_kind",
				"profile_kind no reconocido",
			))
		}
		if strings.TrimSpace(profile.Area) != "" && normalizeAutoprogrammingTaskAreaV0(profile.Area) == "" {
			issues = append(issues, autoprogrammingRequestIssueV0(
				"work_profile_area_invalid",
				field+".area",
				"area de perfil no normalizable",
			))
		}
	}
	return issues
}

func normalizeAutoprogrammingAppKindV1(value string) string {
	normalized := normalizeAutoprogrammingTaskAreaV0(value)
	if strings.HasSuffix(normalized, "-app") {
		return strings.TrimSuffix(normalized, "-app") + "-application"
	}
	return normalized
}
