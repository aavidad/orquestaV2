package orquestastatefile

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	orquestacore "orquesta/modulos/orquesta-core"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

const functionContractIndexDefaultLimitV0 = 50
const functionContractIndexMaxLimitV0 = 200
const functionContractIndexCursorPrefixV0 = "function_contract_cursor_v0:"

type functionContractIndexRecordV0 struct {
	Ref           string
	RunRef        string
	Summary       string
	FunctionNames []string
	EvidenceRefs  []string
	OccurredAt    string
}

func (store *StoreV0) ListFunctionContractsV0(
	ctx context.Context,
	req orquestacore.ListFunctionContractsRequestV0,
) (orquestacore.ListFunctionContractsResultV0, error) {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return orquestacore.ListFunctionContractsResultV0{}, err
	}
	if unsupported := unsupportedFunctionContractFilterV0(req.Filtros); unsupported != "" {
		return orquestacore.ListFunctionContractsResultV0{}, orquestacore.NewFunctionContractQueryErrorV0(
			orquestacore.FunctionContractQueryErrFiltroNoSoportadoV0,
			unsupported,
			nil,
		)
	}
	offset, err := decodeFunctionContractCursorV0(req.Page.Cursor)
	if err != nil {
		return orquestacore.ListFunctionContractsResultV0{}, orquestacore.NewFunctionContractQueryErrorV0(
			orquestacore.FunctionContractQueryErrCursorInvalidoV0,
			"page.cursor",
			nil,
		)
	}
	records, err := store.loadFunctionContractIndexRecordsV0(ctx)
	if err != nil {
		return orquestacore.ListFunctionContractsResultV0{}, err
	}
	records = filterFunctionContractRecordsV0(records, req.Filtros)
	return pageFunctionContractRecordsV0(records, req.Page.Limit, offset), nil
}

func (store *StoreV0) ViewFunctionContractV0(
	ctx context.Context,
	req orquestacore.ViewFunctionContractRequestV0,
) (orquestacore.ViewFunctionContractResultV0, error) {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return orquestacore.ViewFunctionContractResultV0{}, err
	}
	ref := strings.TrimSpace(req.FunctionContractRef)
	if ref == "" {
		return orquestacore.ViewFunctionContractResultV0{}, orquestacore.NewFunctionContractQueryErrorV0(
			orquestacore.FunctionContractQueryErrIncompletoV0,
			"function_contract_ref",
			nil,
		)
	}
	records, err := store.loadFunctionContractIndexRecordsV0(ctx)
	if err != nil {
		return orquestacore.ViewFunctionContractResultV0{}, err
	}
	for _, record := range records {
		if record.Ref == ref {
			return orquestacore.ViewFunctionContractResultV0{}, orquestacore.NewFunctionContractQueryErrorV0(
				orquestacore.FunctionContractQueryErrEvidenciaInsuficienteV0,
				"function_contract_ref",
				record.EvidenceRefs,
			)
		}
	}
	return orquestacore.ViewFunctionContractResultV0{}, orquestacore.NewFunctionContractQueryErrorV0(
		orquestacore.FunctionContractQueryErrNoEncontradoV0,
		"function_contract_ref",
		nil,
	)
}

func (store *StoreV0) loadFunctionContractIndexRecordsV0(ctx context.Context) ([]functionContractIndexRecordV0, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	entries, err := os.ReadDir(filepath.Join(store.rootDir, eventsDirV0))
	if err != nil {
		return nil, err
	}
	records := map[string]functionContractIndexRecordV0{}
	for _, entry := range entries {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		if err := store.collectFunctionContractEventsV0(entry.Name(), records); err != nil {
			return nil, err
		}
	}
	return sortedFunctionContractIndexRecordsV0(records), nil
}

func (store *StoreV0) collectFunctionContractEventsV0(
	name string,
	records map[string]functionContractIndexRecordV0,
) error {
	document, ok, err := readJSONFileV0[eventsDocumentV0](filepath.Join(store.rootDir, eventsDirV0, name))
	if err != nil || !ok {
		return err
	}
	if err := validateEventsDocumentV0(document, document.RunRef); err != nil {
		return err
	}
	index, err := store.loadEventIndexV0(document.RunRef)
	if err != nil {
		return err
	}
	for _, recordRef := range sortedIndexRecordsV0(index) {
		event, err := loadEventRecordV0(store.eventRecordPathV0(document.RunRef, recordRef), document.RunRef, recordRef)
		if err != nil {
			return err
		}
		if event.EventType != orquestacoreworkflow.OrchestrationEventFunctionContractPublishedV0 {
			continue
		}
		record, ok := functionContractRecordFromEventV0(document.RunRef, event)
		if !ok {
			continue
		}
		records[record.Ref] = mergeFunctionContractIndexRecordV0(records[record.Ref], record)
	}
	return nil
}

func functionContractRecordFromEventV0(
	runRef string,
	event orquestacoreworkflow.OrchestrationEventV0,
) (functionContractIndexRecordV0, bool) {
	var payload orquestacoreworkflow.FunctionContractPublishedPayloadV0
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return functionContractIndexRecordV0{}, false
	}
	ref := strings.TrimSpace(payload.ContractRef)
	if ref == "" {
		return functionContractIndexRecordV0{}, false
	}
	return functionContractIndexRecordV0{
		Ref:           ref,
		RunRef:        strings.TrimSpace(runRef),
		Summary:       strings.TrimSpace(payload.Summary),
		FunctionNames: compactStringsV0(payload.FunctionNames),
		EvidenceRefs:  compactStringsV0(append(payload.EvidenceRefs, event.EventID)),
		OccurredAt:    strings.TrimSpace(event.OccurredAt),
	}, true
}

func mergeFunctionContractIndexRecordV0(
	current functionContractIndexRecordV0,
	next functionContractIndexRecordV0,
) functionContractIndexRecordV0 {
	if current.Ref == "" {
		return next
	}
	current.FunctionNames = compactStringsV0(append(current.FunctionNames, next.FunctionNames...))
	current.EvidenceRefs = compactStringsV0(append(current.EvidenceRefs, next.EvidenceRefs...))
	if current.Summary == "" {
		current.Summary = next.Summary
	}
	return current
}
