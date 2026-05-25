package orquestastatefile

import (
	"context"
	"testing"

	orquestacore "orquesta/modulos/orquesta-core"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

func TestFunctionContractIndexV0ListaRefsPublicadasDesdeEventos(t *testing.T) {
	store := mustStoreV0(t, t.TempDir())
	event := mustStateFileFunctionContractEventV0(t, "contract:function:index:v0", 1)
	if err := store.AppendRunEventsV0(context.Background(), event.RunID, []orquestacoreworkflow.OrchestrationEventV0{event}); err != nil {
		t.Fatalf("AppendRunEventsV0: %v", err)
	}

	result, err := store.ListFunctionContractsV0(context.Background(), orquestacore.ListFunctionContractsRequestV0{
		Page: orquestacore.FunctionContractPageRequestV0{Limit: 10},
	})
	if err != nil {
		t.Fatalf("ListFunctionContractsV0: %v", err)
	}

	if len(result.Items) != 1 {
		t.Fatalf("items=%+v", result.Items)
	}
	item := result.Items[0]
	if item.FunctionContractRef != "contract:function:index:v0" ||
		item.Estado != orquestacore.FunctionContractEstadoEvidenciaInsuficienteV0 ||
		item.Version != 0 ||
		item.SimboloObjetivo != "BuildIndexV0" {
		t.Fatalf("item inesperado=%+v", item)
	}
	if len(result.Warnings) != 1 || result.Warnings[0] != "function_contract_payload_no_materializado" {
		t.Fatalf("warnings=%+v", result.Warnings)
	}
}

func TestFunctionContractIndexV0VerRefOnlyBloqueaConEvidencia(t *testing.T) {
	store := mustStoreV0(t, t.TempDir())
	event := mustStateFileFunctionContractEventV0(t, "contract:function:view:v0", 1)
	if err := store.AppendRunEventsV0(context.Background(), event.RunID, []orquestacoreworkflow.OrchestrationEventV0{event}); err != nil {
		t.Fatalf("AppendRunEventsV0: %v", err)
	}

	_, err := store.ViewFunctionContractV0(context.Background(), orquestacore.ViewFunctionContractRequestV0{
		FunctionContractRef: "contract:function:view:v0",
	})
	queryErr, ok := err.(orquestacore.FunctionContractQueryErrorV0)
	if !ok || len(queryErr.Errores) != 1 {
		t.Fatalf("error=%T %v", err, err)
	}
	if queryErr.Errores[0].Code != orquestacore.FunctionContractQueryErrEvidenciaInsuficienteV0 ||
		len(queryErr.Errores[0].Evidence) == 0 {
		t.Fatalf("error publico=%+v", queryErr.Errores[0])
	}
}

func TestFunctionContractIndexV0RechazaFiltroNoCausal(t *testing.T) {
	store := mustStoreV0(t, t.TempDir())
	_, err := store.ListFunctionContractsV0(context.Background(), orquestacore.ListFunctionContractsRequestV0{
		Filtros: orquestacore.FunctionContractFiltersV0{Modulo: "orquesta-core"},
	})
	queryErr, ok := err.(orquestacore.FunctionContractQueryErrorV0)
	if !ok || queryErr.Errores[0].Code != orquestacore.FunctionContractQueryErrFiltroNoSoportadoV0 {
		t.Fatalf("error=%T %+v", err, err)
	}
}

func mustStateFileFunctionContractEventV0(
	t *testing.T,
	ref string,
	sequence int64,
) orquestacoreworkflow.OrchestrationEventV0 {
	t.Helper()
	event, err := orquestacoreworkflow.NewFunctionContractPublishedEventV0(
		orquestacoreworkflow.OrchestrationEventMetaV0{
			EventID:        "evt-" + ref,
			RunID:          "run-function-contract-index",
			Sequence:       sequence,
			IdempotencyKey: "idem-" + ref,
			CorrelationID:  "corr-function-contract-index",
			CausationID:    "cmd-function-contract-index",
			OccurredAt:     "2026-05-25T10:00:00Z",
		},
		orquestacoreworkflow.FunctionContractPublishedPayloadV0{
			ContractRef:   ref,
			PhaseID:       string(orquestacoreworkflow.OrchestrationPhasePlanificacionMicrotareasV0),
			DecisionRef:   "decision-function-contract-index",
			Summary:       "Contrato publicado como ref causal.",
			FunctionNames: []string{"BuildIndexV0"},
			EvidenceRefs:  []string{"evidence-function-contract-index"},
		},
	)
	if err != nil {
		t.Fatalf("NewFunctionContractPublishedEventV0: %v", err)
	}
	return event
}
