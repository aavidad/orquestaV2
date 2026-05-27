package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
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

func TestIdleSelfImprovementBacklogPlannerV0AcotaHashDocsASegurasV0(t *testing.T) {
	projectDir := t.TempDir()
	docsDir := filepath.Join(projectDir, "docs")
	if err := os.MkdirAll(docsDir, 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	mainPath := filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0)
	if err := os.WriteFile(mainPath, []byte("## T01 safe\n\nObjetivo: seguro.\n"), 0o600); err != nil {
		t.Fatalf("write main: %v", err)
	}
	planner := idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}
	if hash, missing := planner.backlogScanDocumentHashV0(idleSelfImprovementBacklogDocRelV0); missing || hash == "" {
		t.Fatalf("hash=%s missing=%v", hash, missing)
	}
	for _, rel := range []string{
		"../outside.md",
		"/tmp/outside.md",
		".orquesta-runtime/control.md",
		"docs/../docs/autoprogramacion_orquesta_pendientes_2026-05-23.md",
	} {
		if _, missing := planner.backlogScanDocumentHashV0(rel); !missing {
			t.Fatalf("rel=%s accepted", rel)
		}
	}
	if err := os.Symlink(mainPath, filepath.Join(docsDir, "link.md")); err == nil {
		if _, missing := planner.backlogScanDocumentHashV0("docs/link.md"); !missing {
			t.Fatalf("symlink accepted")
		}
	}
	if err := os.WriteFile(filepath.Join(docsDir, "huge.md"),
		[]byte(strings.Repeat("x", idleSelfImprovementBacklogScanMaxDocumentBytesV0+1)), 0o600); err != nil {
		t.Fatalf("write huge: %v", err)
	}
	if _, missing := planner.backlogScanDocumentHashV0("docs/huge.md"); !missing {
		t.Fatalf("huge doc accepted")
	}
}

func TestIdleSelfImprovementBacklogPlannerV0RechazaBacklogScanDocNoCanonicoDelPacketV0(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	content := "## T01 safe\n\nObjetivo: seguro.\n"
	if err := os.WriteFile(filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0), []byte(content), 0o600); err != nil {
		t.Fatalf("write main: %v", err)
	}
	planner := idleSelfImprovementBacklogPlannerV0{ProjectWorkDir: projectDir}
	docs := planner.backlogScanDocumentsV0("t01-safe", 1, []string{idleSelfImprovementBacklogDocRelV0})
	packet := orquestaruntime.AgentStartPacketV0{
		Task: orquestaruntime.AgentStartTaskV0{DoneCriteria: []string{backlogScanDocTokenV0(docs[0])}},
	}
	if !planner.packetBacklogDocsCurrentV0(packet) {
		t.Fatalf("canonical packet rejected")
	}
	packet.Task.DoneCriteria = []string{"backlog_scan_doc:../outside.md:line:1:sha256:" + docs[0].SHA256}
	if planner.packetBacklogDocsCurrentV0(packet) {
		t.Fatalf("non canonical packet accepted")
	}
	packet.Task.DoneCriteria = []string{"backlog_scan_doc:docs/no-index.md:line:1:sha256:" + docs[0].SHA256}
	if planner.packetBacklogDocsCurrentV0(packet) {
		t.Fatalf("doc outside current catalog accepted")
	}
}
