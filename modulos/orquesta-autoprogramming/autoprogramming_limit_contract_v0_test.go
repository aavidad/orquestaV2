package orquestaautoprogramming

import (
	"fmt"
	"testing"
)

func TestValidateAutoprogrammingRequestV0AcceptsLargeExplicitLimitContract(t *testing.T) {
	request := validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = nil
		request.WriteSet = nil
		for i := 1; i <= 21; i++ {
			area := fmt.Sprintf("opes-large-%02d", i)
			request.Tasks = append(request.Tasks, AutoprogrammingTaskGroupCandidateV0{
				TaskRef: fmt.Sprintf("task-ref-opes-large-%02d", i),
				Area:    area,
			})
			request.WriteSet = append(request.WriteSet,
				"modulos/orquesta-autoprogramming/"+area+"/contract.go",
			)
		}
		request.MaxTaskRefs = len(request.Tasks)
		request.MaxAreas = len(request.Tasks)
		request.MaxWriteSetEntries = len(request.WriteSet)
	})

	result := ValidateAutoprogrammingRequestV0(request)

	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	if len(result.Groups) != 21 || len(result.WriteSet) != 21 {
		t.Fatalf("groups=%d write_set=%d result=%+v", len(result.Groups), len(result.WriteSet), result)
	}
}

func TestBuildAutoprogrammingProgrammableWorkV0UsesLargeExplicitLimitContract(t *testing.T) {
	request := validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = nil
		request.WriteSet = nil
		for i := 1; i <= 21; i++ {
			area := fmt.Sprintf("opes-large-%02d", i)
			request.Tasks = append(request.Tasks, AutoprogrammingTaskGroupCandidateV0{
				TaskRef: fmt.Sprintf("task-ref-opes-large-%02d", i),
				Area:    area,
			})
			request.WriteSet = append(request.WriteSet,
				"modulos/orquesta-autoprogramming/"+area+"/contract.go",
			)
		}
		request.MaxTaskRefs = len(request.Tasks)
		request.MaxAreas = len(request.Tasks)
		request.MaxWriteSetEntries = len(request.WriteSet)
	})

	result := BuildAutoprogrammingProgrammableWorkV0(request)

	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	if len(result.Work.Tasks) != 21 {
		t.Fatalf("tasks=%d", len(result.Work.Tasks))
	}
	for _, task := range result.Work.Tasks {
		if task.MaxRecursiveAgents != 21*AutoprogrammingDefaultMaxSubagentsPerAgentV0 {
			t.Fatalf("max_recursive_agents=%d", task.MaxRecursiveAgents)
		}
	}
}

func TestValidateAutoprogrammingRequestV0RejectsLimitContractOverHardCap(t *testing.T) {
	result := ValidateAutoprogrammingRequestV0(validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.MaxTaskRefs = AutoprogrammingRequestContractMaxTaskRefsV0 + 1
		request.MaxAreas = AutoprogrammingRequestContractMaxAreasV0 + 1
		request.MaxWriteSetEntries = AutoprogrammingRequestContractMaxWriteSetEntriesV0 + 1
	}))

	if result.Accepted {
		t.Fatalf("accepted=true")
	}
	assertAutoprogrammingRequestIssueV0(t, result, "max_task_refs_contract_invalid")
	assertAutoprogrammingRequestIssueV0(t, result, "max_areas_contract_invalid")
	assertAutoprogrammingRequestIssueV0(t, result, "max_write_set_entries_contract_invalid")
}
