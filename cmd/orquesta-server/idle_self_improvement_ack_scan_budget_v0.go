package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	orquestaserver "orquesta/modulos/orquesta-server"
)

type idleSelfImprovementACKRuntimeScanV0 struct {
	runtimeDir       string
	deadline         time.Time
	exhausted        bool
	ambiguousRunRefs []string
}

type idleSelfImprovementACKCandidateV0 struct {
	ackPath  string
	runRef   string
	agentRef string
}

func (scan *idleSelfImprovementACKRuntimeScanV0) runtimeACKCandidatesV0() []idleSelfImprovementACKCandidateV0 {
	entries, exhausted, err := idleSelfImprovementReadDirBudgetedV0(
		scan.runtimeDir,
		idleSelfImprovementAckScanMaxRuntimeDirsV0,
	)
	if err != nil {
		return nil
	}
	scan.exhausted = scan.exhausted || exhausted
	var candidates []idleSelfImprovementACKCandidateV0
	for _, entry := range entries {
		if scan.deadlineExceededV0() {
			break
		}
		if !entry.IsDir() || !idleSelfImprovementIsBacklogRunDirV0(entry.Name()) {
			continue
		}
		candidates = append(candidates, scan.runACKCandidatesV0(entry.Name())...)
	}
	return candidates
}

func (scan *idleSelfImprovementACKRuntimeScanV0) runACKCandidatesV0(runRef string) []idleSelfImprovementACKCandidateV0 {
	runPath := filepath.Join(scan.runtimeDir, runRef)
	entries, exhausted, err := idleSelfImprovementReadDirBudgetedV0(
		runPath,
		idleSelfImprovementAckScanMaxRunEntriesV0,
	)
	if err != nil {
		scan.ambiguousRunRefs = append(scan.ambiguousRunRefs, runRef)
		return nil
	}
	scan.exhausted = scan.exhausted || exhausted
	var candidates []idleSelfImprovementACKCandidateV0
	agentDirs := 0
	for _, entry := range entries {
		if scan.deadlineExceededV0() {
			break
		}
		if !entry.IsDir() || !idleSelfImprovementIsAgentControlDirV0(entry.Name()) {
			continue
		}
		agentDirs++
		if agentDirs > idleSelfImprovementAckScanMaxAgentDirsPerRunV0 {
			scan.exhausted = true
			break
		}
		ackPath := filepath.Join(runPath, entry.Name(), "agent_ack.json")
		if _, err := os.Stat(ackPath); err != nil {
			if !os.IsNotExist(err) {
				scan.ambiguousRunRefs = append(scan.ambiguousRunRefs, runRef)
			}
			continue
		}
		candidates = append(candidates, idleSelfImprovementACKCandidateV0{
			ackPath:  ackPath,
			runRef:   runRef,
			agentRef: entry.Name(),
		})
	}
	return candidates
}

func idleSelfImprovementReadDirBudgetedV0(path string, maxEntries int) ([]os.DirEntry, bool, error) {
	if maxEntries <= 0 {
		return nil, true, nil
	}
	dir, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	defer dir.Close()
	entries, err := dir.ReadDir(maxEntries + 1)
	if err != nil {
		return nil, false, err
	}
	if len(entries) > maxEntries {
		return entries[:maxEntries], true, nil
	}
	return entries, false, nil
}

func (scan *idleSelfImprovementACKRuntimeScanV0) deadlineExceededV0() bool {
	if scan.deadline.IsZero() || time.Now().Before(scan.deadline) {
		return false
	}
	scan.exhausted = true
	return true
}

func idleSelfImprovementIsBacklogRunDirV0(name string) bool {
	return strings.HasPrefix(strings.TrimSpace(name), "request-ref-autoprogramming-backlog-")
}

func idleSelfImprovementIsAgentControlDirV0(name string) bool {
	name = strings.TrimSpace(name)
	return strings.HasPrefix(name, "agent-") || strings.HasPrefix(name, "agent-ref-")
}

func idleSelfImprovementBacklogACKDocStaleCollisionV0(runRef string) orquestaserver.BacklogScanCollisionV0 {
	return orquestaserver.BacklogScanCollisionV0{
		Code:       "backlog_docs_changed_after_plan",
		RequestRef: idleSelfImprovementNormalizeQueuedRequestRefV0(runRef),
		Message:    "ACK completado sobre foto documental obsoleta; requiere rebase/merge",
		EvidenceRefs: []string{
			"evidence-ref-autoprogramming-backlog-doc-merge-pending",
		},
	}
}

func idleSelfImprovementBacklogACKScanBudgetCollisionV0() orquestaserver.BacklogScanCollisionV0 {
	return orquestaserver.BacklogScanCollisionV0{
		Code:    "backlog_ack_scan_budget_exhausted",
		Message: "scan acotado de ACKs de automejora agoto presupuesto",
		EvidenceRefs: []string{
			idleSelfImprovementAckScanBudgetEvidenceRefV0,
		},
	}
}

func idleSelfImprovementBacklogACKAmbiguousCollisionV0(runRef string) orquestaserver.BacklogScanCollisionV0 {
	return orquestaserver.BacklogScanCollisionV0{
		Code:       "backlog_ack_scan_ambiguous",
		RequestRef: idleSelfImprovementNormalizeQueuedRequestRefV0(runRef),
		Message:    "ACK de automejora no puede usarse para deduplicar backlog",
		EvidenceRefs: []string{
			idleSelfImprovementAckAmbiguousEvidenceRefV0,
		},
	}
}

func (planner idleSelfImprovementBacklogPlannerV0) backlogACKScanRecoveryRequestV0(
	base orquestaserver.IdleSelfImprovementRequestV0,
	request orquestaserver.IdleSelfImprovementPlanRequestV0,
	collisions []orquestaserver.BacklogScanCollisionV0,
) (orquestaserver.IdleSelfImprovementPlanResultV0, bool) {
	code := idleSelfImprovementACKScanBlockingCodeV0(collisions)
	if code == "" {
		return orquestaserver.IdleSelfImprovementPlanResultV0{}, false
	}
	scanner := idleSelfImprovementBacklogScannerRequestV0(base, request)
	scanner = planner.withBacklogScannerMergeLeaseV0(scanner)
	return orquestaserver.IdleSelfImprovementPlanResultV0{
		Requests:     []orquestaserver.IdleSelfImprovementRequestV0{scanner},
		EvidenceRefs: idleSelfImprovementACKScanRecoveryEvidenceRefsV0(collisions),
		Collisions:   collisions,
		Message:      code,
	}, true
}

func idleSelfImprovementACKScanRecoveryEvidenceRefsV0(
	collisions []orquestaserver.BacklogScanCollisionV0,
) []string {
	refs := []string{
		"evidence-ref-autoprogramming-backlog-doc",
		"evidence-ref-autoprogramming-backlog-scanner",
	}
	for _, collision := range collisions {
		refs = append(refs, collision.EvidenceRefs...)
	}
	return compactServerStackStringsV0(refs)
}

func idleSelfImprovementACKScanBlockingCodeV0(
	collisions []orquestaserver.BacklogScanCollisionV0,
) string {
	_ = collisions
	return ""
}
