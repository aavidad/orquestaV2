package orquestacli

import (
	"encoding/json"
	"testing"

	orquestagovernance "orquesta/modulos/orquesta-governance"
)

func TestDecodeGovernanceCatalogResultV0AceptaRespuestaTruncada(t *testing.T) {
	response := testGovernanceCatalogPublicResponseV0("req-truncated", "corr-truncated")
	response.Counters.Effective = 3
	response.OutputBudget = orquestagovernance.GovernanceCatalogPublicOutputBudgetV0{
		MaxEntries:       1,
		MaxBytes:         orquestagovernance.GovernanceCatalogPublicDefaultMaxBytesV0,
		MatchedEffective: 3,
		ReturnedEntries:  1,
		Status:           orquestagovernance.GovernanceCatalogPublicBudgetTruncatedV0,
		Reason:           "max_entries_exceeded",
	}

	raw, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}

	result, err := decodeGovernanceCatalogResultV0(raw)
	if err != nil {
		t.Fatalf("decodeGovernanceCatalogResultV0() error = %v", err)
	}
	if len(result.Effective) != 1 || result.Counters.Effective != 3 {
		t.Fatalf("result = %+v", result)
	}
}
