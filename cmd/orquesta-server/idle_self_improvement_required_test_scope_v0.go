package main

import "strings"

const idleSelfImprovementRefOnlyRequiredTestV0 = "validar contexto required ref_only mediante lectura local, consulta al director o evidencia explicita"

const idleSelfImprovementBacklogDocumentalRequiredTestV0 = "validar scanner documental con busquedas focales sobre backlog, rail errors y duplicaciones"

func idleSelfImprovementBacklogScannerRequiredTestsV0(
	baseTests []string,
	writeSet []string,
) ([]string, []string) {
	return idleSelfImprovementScopedRequiredTestsV0(baseTests, writeSet, "scanner")
}

func idleSelfImprovementBacklogSectionRequiredTestsV0(
	baseTests []string,
	sectionTests []string,
	manualVerifications []string,
	writeSet []string,
) ([]string, []string) {
	if len(sectionTests) > 0 {
		return compactServerStackStringsV0(sectionTests), []string{"required_test_origin:section_declared"}
	}
	if len(manualVerifications) > 0 {
		return nil, []string{"required_test_origin:manual_verification"}
	}
	return idleSelfImprovementScopedRequiredTestsV0(baseTests, writeSet, "section")
}

func idleSelfImprovementScopedRequiredTestsV0(
	baseTests []string,
	writeSet []string,
	kind string,
) ([]string, []string) {
	docOnly := idleSelfImprovementWriteSetOnlyDocsV0(writeSet)
	out := []string{}
	contextRefs := []string{}
	omittedGlobal := false
	for _, test := range baseTests {
		test = strings.TrimSpace(test)
		if test == "" {
			continue
		}
		if docOnly && idleSelfImprovementGlobalGoTestAllV0(test) {
			omittedGlobal = true
			continue
		}
		out = append(out, test)
		contextRefs = append(contextRefs, idleSelfImprovementRequiredTestOriginRefV0(test))
	}
	if docOnly {
		out = idleSelfImprovementEnsureStringV0(out, idleSelfImprovementBacklogDocumentalRequiredTestV0)
		out = idleSelfImprovementEnsureStringV0(out, idleSelfImprovementRefOnlyRequiredTestV0)
		contextRefs = append(contextRefs,
			"required_test_origin:global_policy:documental_focal",
			"required_test_origin:ref_only_guard",
		)
	}
	if omittedGlobal {
		contextRefs = append(contextRefs, "required_test_scope_policy:global_go_test_all_omitted_for_doc_only_"+kind)
	}
	return compactServerStackStringsV0(out), compactServerStackStringsV0(contextRefs)
}

func idleSelfImprovementRequiredTestOriginRefV0(test string) string {
	if strings.TrimSpace(test) == idleSelfImprovementRefOnlyRequiredTestV0 {
		return "required_test_origin:ref_only_guard"
	}
	return "required_test_origin:global_policy"
}

func idleSelfImprovementWriteSetOnlyDocsV0(writeSet []string) bool {
	if len(writeSet) == 0 {
		return false
	}
	for _, item := range writeSet {
		item = strings.Trim(strings.TrimSpace(item), "`")
		if item == "" || !strings.HasPrefix(item, "docs/") {
			return false
		}
	}
	return true
}

func idleSelfImprovementGlobalGoTestAllV0(test string) bool {
	fields := strings.Fields(test)
	return len(fields) >= 3 &&
		fields[0] == "go" &&
		fields[1] == "test" &&
		fields[len(fields)-1] == "./..."
}

func idleSelfImprovementEnsureStringV0(values []string, value string) []string {
	for _, item := range values {
		if strings.TrimSpace(item) == value {
			return values
		}
	}
	return append(values, value)
}
