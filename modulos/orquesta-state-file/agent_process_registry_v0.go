package orquestastatefile

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

type agentProcessDocumentV0 struct {
	SchemaVersion   string                   `json:"schema_version"`
	RunRef          string                   `json:"run_ref"`
	AgentRequestRef string                   `json:"agent_request_ref"`
	Record          agentProcessRecordJSONV0 `json:"record"`
}

type agentProcessRecordJSONV0 struct {
	RunID          string   `json:"run_id"`
	AgentRequestID string   `json:"agent_request_id"`
	ProcessRef     string   `json:"process_ref"`
	SessionRef     string   `json:"session_ref"`
	LaunchRef      string   `json:"launch_ref"`
	PID            int      `json:"pid,omitempty"`
	ReadinessRef   string   `json:"readiness_ref"`
	EvidenceRefs   []string `json:"evidence_refs,omitempty"`
}

func (store *StoreV0) RecordAgentProcessV0(
	ctx context.Context,
	record orquestacionnucleoapp.AgentProcessRecordV0,
) error {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	record = orquestacionnucleoapp.NormalizeAgentProcessRegistryRecordV0(record)
	if err := orquestacionnucleoapp.ValidateAgentProcessRegistryRecordV0(record); err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.recordAgentProcessLockedV0(record)
}

func (store *StoreV0) ResolveAgentProcessV0(
	ctx context.Context,
	runID string,
	agentRequestID string,
) (orquestacionnucleoapp.AgentProcessRecordV0, error) {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return orquestacionnucleoapp.AgentProcessRecordV0{}, err
	}
	runID, agentRequestID = orquestacionnucleoapp.NormalizeAgentProcessRegistryLookupV0(
		runID,
		agentRequestID,
	)
	if err := orquestacionnucleoapp.ValidateAgentProcessRegistryLookupV0(runID, agentRequestID); err != nil {
		return orquestacionnucleoapp.AgentProcessRecordV0{}, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.resolveAgentProcessLockedV0(runID, agentRequestID)
}

func (store *StoreV0) ListAgentProcessesV0(
	ctx context.Context,
	filter orquestacionnucleoapp.AgentProcessRegistryListFilterV0,
) ([]orquestacionnucleoapp.AgentProcessRecordV0, error) {
	ctx = contextOrBackgroundV0(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	filter = orquestacionnucleoapp.NormalizeAgentProcessRegistryListFilterV0(filter)
	if err := orquestacionnucleoapp.ValidateAgentProcessRegistryListFilterV0(filter); err != nil {
		return nil, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.listAgentProcessesLockedV0(filter)
}

func (store *StoreV0) recordAgentProcessLockedV0(
	record orquestacionnucleoapp.AgentProcessRecordV0,
) error {
	path := store.agentProcessPathV0(record.RunID, record.AgentRequestID)
	existing, ok, err := readJSONFileV0[agentProcessDocumentV0](path)
	if err != nil {
		return err
	}
	if ok {
		if err := validateAgentProcessDocumentV0(existing, record.RunID, record.AgentRequestID); err != nil {
			return err
		}
		if !reflect.DeepEqual(agentProcessRecordFromJSONV0(existing.Record), record) {
			return invalidErrorV0("agent_process_registry", "agent_process conflict")
		}
		return nil
	}
	return writeJSONAtomicV0(path, agentProcessDocumentFromRecordV0(record))
}

func (store *StoreV0) resolveAgentProcessLockedV0(
	runID string,
	agentRequestID string,
) (orquestacionnucleoapp.AgentProcessRecordV0, error) {
	document, ok, err := readJSONFileV0[agentProcessDocumentV0](
		store.agentProcessPathV0(runID, agentRequestID),
	)
	if err != nil {
		return orquestacionnucleoapp.AgentProcessRecordV0{}, err
	}
	if !ok {
		return orquestacionnucleoapp.AgentProcessRecordV0{}, storeErrorV0(
			"agent_process_registry",
			"agent_process no encontrado",
		)
	}
	if err := validateAgentProcessDocumentV0(document, runID, agentRequestID); err != nil {
		return orquestacionnucleoapp.AgentProcessRecordV0{}, err
	}
	record := agentProcessRecordFromJSONV0(document.Record)
	return record, orquestacionnucleoapp.ValidateAgentProcessRegistryRecordV0(record)
}

func (store *StoreV0) listAgentProcessesLockedV0(
	filter orquestacionnucleoapp.AgentProcessRegistryListFilterV0,
) ([]orquestacionnucleoapp.AgentProcessRecordV0, error) {
	root := filepath.Join(store.rootDir, agentProcessesDirV0)
	var records []orquestacionnucleoapp.AgentProcessRecordV0
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			return nil
		}
		document, ok, err := readJSONFileV0[agentProcessDocumentV0](path)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		record := agentProcessRecordFromJSONV0(document.Record)
		if filter.RunID != "" && record.RunID != filter.RunID {
			return nil
		}
		if err := validateAgentProcessDocumentV0(document, record.RunID, record.AgentRequestID); err != nil {
			return err
		}
		if err := orquestacionnucleoapp.ValidateAgentProcessRegistryRecordV0(record); err != nil {
			return err
		}
		records = append(records, record)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(records, func(i int, j int) bool {
		if records[i].RunID != records[j].RunID {
			return records[i].RunID < records[j].RunID
		}
		return records[i].AgentRequestID < records[j].AgentRequestID
	})
	return records, nil
}

func agentProcessDocumentFromRecordV0(
	record orquestacionnucleoapp.AgentProcessRecordV0,
) agentProcessDocumentV0 {
	return agentProcessDocumentV0{
		SchemaVersion:   agentProcessDocumentSchemaV0,
		RunRef:          record.RunID,
		AgentRequestRef: record.AgentRequestID,
		Record:          agentProcessRecordToJSONV0(record),
	}
}

func validateAgentProcessDocumentV0(
	document agentProcessDocumentV0,
	expectedRunRef string,
	expectedAgentRequestRef string,
) error {
	if document.SchemaVersion != agentProcessDocumentSchemaV0 {
		return storeErrorV0("agent_process.schema_version", "schema_version invalida")
	}
	if document.RunRef != expectedRunRef || document.AgentRequestRef != expectedAgentRequestRef {
		return storeErrorV0("agent_process.ref", "ref inconsistente")
	}
	return nil
}

func agentProcessRecordToJSONV0(
	record orquestacionnucleoapp.AgentProcessRecordV0,
) agentProcessRecordJSONV0 {
	return agentProcessRecordJSONV0{
		RunID:          record.RunID,
		AgentRequestID: record.AgentRequestID,
		ProcessRef:     record.ProcessRef,
		SessionRef:     record.SessionRef,
		LaunchRef:      record.LaunchRef,
		PID:            record.PID,
		ReadinessRef:   record.ReadinessRef,
		EvidenceRefs:   append(record.EvidenceRefs[:0:0], record.EvidenceRefs...),
	}
}

func agentProcessRecordFromJSONV0(
	record agentProcessRecordJSONV0,
) orquestacionnucleoapp.AgentProcessRecordV0 {
	return orquestacionnucleoapp.AgentProcessRecordV0{
		RunID:          record.RunID,
		AgentRequestID: record.AgentRequestID,
		ProcessRef:     record.ProcessRef,
		SessionRef:     record.SessionRef,
		LaunchRef:      record.LaunchRef,
		PID:            record.PID,
		ReadinessRef:   record.ReadinessRef,
		EvidenceRefs:   append(record.EvidenceRefs[:0:0], record.EvidenceRefs...),
	}
}
