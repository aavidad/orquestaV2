package orquestadirector

import (
	orquestacontext "orquesta/modulos/orquesta-context"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
)

const (
	ErrDirectorContextBundleInvalidoV0 = "director_context_bundle_invalido"
	ErrDirectorContextBundleOutboxV0   = "director_context_bundle_outbox_invalido"
)

type LaunchContextBundleInputV0 struct {
	Message         orquestacoreworkflow.OutboxMessageV0 `json:"message"`
	TargetModule    string                               `json:"target_module"`
	TaskKind        string                               `json:"task_kind"`
	Objective       string                               `json:"objective"`
	CapacityLevel   string                               `json:"capacity_level"`
	ReadSet         []string                             `json:"read_set,omitempty"`
	WriteSet        []string                             `json:"write_set,omitempty"`
	ContractRefs    []string                             `json:"contract_refs,omitempty"`
	CrossModuleRefs []string                             `json:"cross_module_refs,omitempty"`
	MaxEntries      int                                  `json:"max_entries,omitempty"`
	MaxTotalBytes   int                                  `json:"max_total_bytes,omitempty"`
}

type LaunchContextBundleResultV0 struct {
	Bundle orquestacontext.ContextBundleV0 `json:"bundle"`
	Issues []LaunchContextBundleIssueV0    `json:"issues,omitempty"`
}

type LaunchContextBundleIssueV0 struct {
	Code    string `json:"code"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}

type LaunchContextBundleErrorV0 struct {
	Code   string                       `json:"code"`
	Field  string                       `json:"field,omitempty"`
	Issues []LaunchContextBundleIssueV0 `json:"issues,omitempty"`
}

func (err LaunchContextBundleErrorV0) Error() string {
	return err.Code
}
