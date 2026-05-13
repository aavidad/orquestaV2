package orquestamcp

import (
	"context"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaexternalworkrun "orquesta/modulos/orquesta-external-work-run"
)

func TestMCPExternalWorkRunToolExecutorV0RechazaEntradaAmbigua(t *testing.T) {
	result, err := MCPExternalWorkRunToolExecutorV0{}.Execute(
		context.Background(),
		MCPExternalWorkRunToolInputV0{
			RequestID: "req-ambigua",
			ExternalWorkRunRequest: orquestaexternalworkrun.StartExternalWorkRunRequestV0{
				AppChangeRequest: orquestaappchange.AppChangeRequestV0{
					ChangeRef: "nested-change",
				},
			},
			AppChangeRequest: orquestaappchange.AppChangeRequestV0{
				ChangeRef: "top-change",
			},
		},
	)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Estado != MCPExternalWorkRunEstadoErrorV0 ||
		len(result.Errores) != 1 ||
		result.Errores[0].Code != MCPExternalWorkRunInputAmbiguousV0 {
		t.Fatalf("result=%+v", result)
	}
}
