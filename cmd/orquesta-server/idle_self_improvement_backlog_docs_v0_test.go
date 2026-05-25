package main

import (
	"os"
	"path/filepath"
	"testing"

	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestIdleSelfImprovementBacklogPlannerV0LeeIndiceYShardsV0(t *testing.T) {
	projectDir := t.TempDir()
	docsDir := filepath.Join(projectDir, "docs")
	if err := os.MkdirAll(docsDir, 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	mainDoc := `# Backlog

## Indice vivo de backlog

- Canonico actual: ` + "`docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`" + `
- Shard ejecutable: ` + "`docs/autoprogramacion_shard_2026-05-25.md`" + `

## T02 main-doc-task

Objetivo: mantener compatibilidad con documento historico.

Alcance:

- ` + "`cmd/orquesta-server`" + `
`
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(mainDoc), 0o600); err != nil {
		t.Fatalf("write main backlog: %v", err)
	}
	shardDoc := `# Shard

## T01 sharded-core-task

Objetivo: cerrar deuda de nucleo desde shard.

Alcance:

- ` + "`modulos/orquesta-server`" + `

Criterios:

- Tests: ` + "`go test -count=1 ./modulos/orquesta-server`" + `
`
	shardRel := "docs/autoprogramacion_shard_2026-05-25.md"
	if err := os.WriteFile(filepath.Join(projectDir, shardRel), []byte(shardDoc), 0o600); err != nil {
		t.Fatalf("write shard backlog: %v", err)
	}

	result, err := (idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}).PlanV0(
		orquestaserver.IdleSelfImprovementPlanRequestV0{
			MaxRequests: 3,
			BaseRequest: orquestaserver.IdleSelfImprovementRequestV0{
				RequestRef:    "request-ref-base",
				CorrelationID: "corr-request-ref-base",
				ProjectRef:    "project-ref-orquesta",
				WriteSet:      []string{"cmd/orquesta-server"},
				RequiredTests: []string{"go test -count=1 ./cmd/orquesta-server"},
			},
		},
	)
	if err != nil {
		t.Fatalf("PlanV0: %v", err)
	}
	sharded := requestByAreaForBacklogDocsTestV0(result.Requests, "t01-sharded-core-task")
	if sharded.RequestRef == "" {
		t.Fatalf("requests=%+v", result.Requests)
	}
	if !containsStringForTestV0(sharded.ContextRefs, "backlog_doc:"+shardRel) ||
		!containsStringForTestV0(sharded.ContextRefs, "backlog_section:t01-sharded-core-task") ||
		len(sharded.BacklogScanDocs) != 1 ||
		sharded.BacklogScanDocs[0].Path != shardRel ||
		sharded.BacklogScanDocs[0].StartLine != 3 ||
		sharded.BacklogScanDocs[0].SectionRef != "t01-sharded-core-task" ||
		sharded.BacklogScanDocs[0].SHA256 == "" {
		t.Fatalf("sharded=%+v", sharded)
	}
	if sharded.RequiredTests[0] != "go test -count=1 ./modulos/orquesta-server" {
		t.Fatalf("required_tests=%+v", sharded.RequiredTests)
	}
}

func requestByAreaForBacklogDocsTestV0(
	requests []orquestaserver.IdleSelfImprovementRequestV0,
	area string,
) orquestaserver.IdleSelfImprovementRequestV0 {
	for _, request := range requests {
		if request.SuggestedArea == area {
			return request
		}
	}
	return orquestaserver.IdleSelfImprovementRequestV0{}
}
