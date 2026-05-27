package orquestaautoprogramming

import (
	"fmt"
	"testing"
)

func TestValidateAutoprogrammingRequestV0DoesNotExpandLargeAppLimitsFromText(t *testing.T) {
	request := largeAppAutoprogrammingRequestV0(11, func(request *AutoprogrammingRequestV0) {
		for i := range request.Tasks {
			request.Tasks[i].Objective = "OPES app grande puede declarar max_task_refs por contrato"
			request.Tasks[i].AcceptanceCriteria = []string{
				"max_task_refs=40 max_areas=40 max_write_set_entries=40 en texto no amplian limites",
			}
		}
	})

	result := ValidateAutoprogrammingRequestV0(request)

	if result.Accepted {
		t.Fatalf("accepted=true without structured limits")
	}
	assertAutoprogrammingRequestIssueV0(t, result, "scope_tasks_too_large")
	assertAutoprogrammingRequestIssueV0(t, result, "scope_areas_too_large")
	assertAutoprogrammingRequestIssueV0(t, result, "scope_write_set_too_large")
}

func TestValidateAutoprogrammingRequestV0ExpandsLargeAppLimitsOnlyByContract(t *testing.T) {
	request := largeAppAutoprogrammingRequestV0(40, func(request *AutoprogrammingRequestV0) {
		request.MaxTaskRefs = 40
		request.MaxAreas = 40
		request.MaxWriteSetEntries = 40
	})

	result := ValidateAutoprogrammingRequestV0(request)

	if !result.Accepted {
		t.Fatalf("accepted=false issues=%+v", result.Issues)
	}
	if len(result.Groups) != 40 || len(result.WriteSet) != 40 {
		t.Fatalf("groups=%d write_set=%d", len(result.Groups), len(result.WriteSet))
	}
}

func largeAppAutoprogrammingRequestV0(
	count int,
	mutate func(*AutoprogrammingRequestV0),
) AutoprogrammingRequestV0 {
	request := validAutoprogrammingRequestV0(func(request *AutoprogrammingRequestV0) {
		request.Tasks = nil
		request.WriteSet = nil
		for i := 1; i <= count; i++ {
			area := fmt.Sprintf("opes-large-contract-%02d", i)
			request.Tasks = append(request.Tasks, AutoprogrammingTaskGroupCandidateV0{
				TaskRef: fmt.Sprintf("task-ref-opes-contract-%02d", i),
				Area:    area,
			})
			request.WriteSet = append(request.WriteSet,
				"modulos/orquesta-autoprogramming/"+area+"/contract.go",
			)
		}
	})
	if mutate != nil {
		mutate(&request)
	}
	return request
}
