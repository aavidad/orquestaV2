package orquestacoreworkflow

import orquestarails "orquesta/modulos/orquesta-rails"

const coreWorkflowDetailRailBoundaryV0 = "core_workflow"

func textValuesContainForbiddenOperationalSensitiveDetailV0(values []string) bool {
	return orquestarails.ValuesContainOperationalSensitiveDetailForFieldV0(
		coreWorkflowDetailRailBoundaryV0,
		"*",
		values,
	)
}

func textContainsForbiddenOperationalSensitiveDetailV0(value string) bool {
	return orquestarails.TextContainsOperationalSensitiveDetailForFieldV0(
		coreWorkflowDetailRailBoundaryV0,
		"*",
		value,
	)
}

func detailProhibitedRailsEnabledV0() bool {
	return orquestarails.DetailProhibitedRailsEnabledForFieldV0(coreWorkflowDetailRailBoundaryV0, "*")
}

func operationalSensitiveFragmentsForTestV0() []string {
	return orquestarails.OperationalSensitiveFragmentsV0
}
