package orquestaautoprogramming

const (
	AutoprogrammingRequestContractMaxTaskRefsV0        = 40
	AutoprogrammingRequestContractMaxAreasV0           = 40
	AutoprogrammingRequestContractMaxWriteSetEntriesV0 = 40
)

func autoprogrammingRequestLimitContractIssuesV0(
	request AutoprogrammingRequestV0,
) []AutoprogrammingRequestIssueV0 {
	checks := []struct {
		field string
		value int
		max   int
	}{
		{
			field: "max_task_refs",
			value: request.MaxTaskRefs,
			max:   AutoprogrammingRequestContractMaxTaskRefsV0,
		},
		{
			field: "max_areas",
			value: request.MaxAreas,
			max:   AutoprogrammingRequestContractMaxAreasV0,
		},
		{
			field: "max_write_set_entries",
			value: request.MaxWriteSetEntries,
			max:   AutoprogrammingRequestContractMaxWriteSetEntriesV0,
		},
	}
	var issues []AutoprogrammingRequestIssueV0
	for _, check := range checks {
		if check.value < 0 || check.value > check.max {
			issues = append(issues, autoprogrammingRequestIssueV0(
				check.field+"_contract_invalid",
				check.field,
				check.field+" fuera de contrato",
			))
		}
	}
	return issues
}
