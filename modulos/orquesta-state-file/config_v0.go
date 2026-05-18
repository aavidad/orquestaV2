package orquestastatefile

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

const (
	runDocumentSchemaV0                  = "orquesta_state_file.run.v0"
	eventDocumentSchemaV0                = "orquesta_state_file.events.v0"
	workflowTaskDocumentSchemaV0         = "orquesta_state_file.workflow_task.v0"
	workflowWaitDocumentSchemaV0         = "orquesta_state_file.workflow_task_wait_state.v0"
	requiredTestEvidenceDocumentSchemaV0 = "orquesta_state_file.required_test_evidence.v0"
	operationalDirectorPlanStateSchemaV0 = "orquesta_state_file.operational_director_plan_state.v0"
	agentProcessDocumentSchemaV0         = "orquesta_state_file.agent_process.v0"
)

type ConfigV0 struct {
	RootDir string
}

type StoreV0 struct {
	mu      sync.Mutex
	rootDir string
}

func NewStoreV0(config ConfigV0) (*StoreV0, error) {
	rootDir := strings.TrimSpace(config.RootDir)
	if rootDir == "" {
		return nil, fmt.Errorf("root_dir requerido")
	}
	rootDir = filepath.Clean(rootDir)
	if err := ensureStoreDirsV0(rootDir); err != nil {
		return nil, err
	}
	return &StoreV0{rootDir: rootDir}, nil
}

func (store *StoreV0) RootDirV0() string {
	if store == nil {
		return ""
	}
	return store.rootDir
}
