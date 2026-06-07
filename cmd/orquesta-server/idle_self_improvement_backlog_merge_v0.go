package main

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaserver "orquesta/modulos/orquesta-server"
)

var idleSelfImprovementBacklogScannerDocsV0 = []string{
	idleSelfImprovementBacklogDocRelV0,
	"docs/rail_errors_observados_2026-05-23.md",
	"docs/duplicaciones_railes_pendientes_2026-05-24.md",
}

func (planner idleSelfImprovementBacklogPlannerV0) withBacklogScannerMergeLeaseV0(
	request orquestaserver.IdleSelfImprovementRequestV0,
) orquestaserver.IdleSelfImprovementRequestV0 {
	request = planner.withBacklogMergeLeaseV0(request, "", 1, planner.backlogScannerDocumentRefsV0())
	request = planner.withBacklogScannerCanonicalPreflightV0(request)
	request.ContextRefs = compactServerStackStringsV0(append(
		request.ContextRefs,
		planner.federatedBacklogNonExecutableContextRefsV0()...,
	))
	return request
}

func (planner idleSelfImprovementBacklogPlannerV0) withBacklogSectionMergeLeaseV0(
	request orquestaserver.IdleSelfImprovementRequestV0,
	section idleSelfImprovementBacklogSectionV0,
) orquestaserver.IdleSelfImprovementRequestV0 {
	line := section.SourceLine
	if line <= 0 {
		line = 1
	}
	sourcePath := firstNonEmptyServerStackV0(section.SourcePath, idleSelfImprovementBacklogDocRelV0)
	request = planner.withBacklogMergeLeaseV0(request, section.Ref, line, []string{sourcePath})
	request.ContextRefs = compactServerStackStringsV0(append(request.ContextRefs,
		"backlog_section_fingerprint:"+idleSelfImprovementBacklogHashV0(
			idleSelfImprovementBacklogSectionFingerprintV0(section),
		),
	))
	request.ContextRefs = compactServerStackStringsV0(append(
		request.ContextRefs,
		idleSelfImprovementBacklogProposalContextRefsV0(section)...,
	))
	return idleSelfImprovementWithBacklogTaskIDAliasV0(request, section)
}

func (planner idleSelfImprovementBacklogPlannerV0) withBacklogMergeLeaseV0(
	request orquestaserver.IdleSelfImprovementRequestV0,
	sectionRef string,
	line int,
	docs []string,
) orquestaserver.IdleSelfImprovementRequestV0 {
	request.BacklogScanDocs = planner.backlogScanDocumentsV0(sectionRef, line, docs)
	request.BacklogScanEpoch = backlogScanEpochV0(request.BacklogScanDocs)
	request.BacklogScanRef = backlogScanRefV0(request, sectionRef)
	allocation := planner.backlogTaskIDAllocationV0(request, sectionRef)
	request.ReservationRefs = backlogScanReservationRefsV0(request.BacklogScanEpoch, request.WriteSet)
	request.ReservationRefs = compactServerStackStringsV0(append(request.ReservationRefs, allocation.ReservationRef))
	request.ContextRefs = compactServerStackStringsV0(append(
		request.ContextRefs,
		backlogScanContextRefsV0(request)...,
	))
	request.ContextRefs = compactServerStackStringsV0(append(request.ContextRefs, allocation.ContextRefs()...))
	request.AcceptanceCriteria = compactServerStackStringsV0(append(
		request.AcceptanceCriteria,
		backlogScanAcceptanceCriteriaV0(request)...,
	))
	request.AcceptanceCriteria = compactServerStackStringsV0(append(
		request.AcceptanceCriteria,
		allocation.AcceptanceCriteria()...,
	))
	request.EvidenceRefs = compactServerStackStringsV0(append(
		request.EvidenceRefs,
		"evidence-ref-autoprogramming-backlog-merge-lease",
		"evidence-ref-autoprogramming-backlog-task-id-allocation-lease",
	))
	return request
}

func (planner idleSelfImprovementBacklogPlannerV0) backlogScanDocumentsV0(
	sectionRef string,
	line int,
	docs []string,
) []orquestaserver.BacklogScanDocumentV0 {
	docs = compactServerStackStringsV0(docs)
	if len(docs) > idleSelfImprovementBacklogScanMaxDocsV0 {
		docs = docs[:idleSelfImprovementBacklogScanMaxDocsV0]
	}
	out := make([]orquestaserver.BacklogScanDocumentV0, 0, len(docs))
	for _, rel := range docs {
		clean, ok := backlogScanCleanDocumentRelV0(rel)
		if !ok {
			continue
		}
		hash, missing := planner.backlogScanDocumentHashV0(rel)
		scanEntries, entryDigest, entryIssues := planner.backlogScanEntrySummaryV0(clean, missing)
		out = append(out, orquestaserver.BacklogScanDocumentV0{
			Path:            clean,
			StartLine:       line,
			SHA256:          hash,
			Missing:         missing,
			SectionRef:      sectionRef,
			ScanEntries:     scanEntries,
			ScanEntryDigest: entryDigest,
			ScanEntryCount:  len(scanEntries),
			ScanEntryIssues: entryIssues,
		})
	}
	return out
}

func (planner idleSelfImprovementBacklogPlannerV0) backlogScanEntrySummaryV0(
	rel string,
	missing bool,
) ([]orquestaserver.BacklogScanEntryV0, string, []orquestaserver.BacklogScanEntryIssueV0) {
	if missing {
		return nil, "", nil
	}
	body, err := planner.readBacklogScanDocumentTextV0(rel)
	if err != nil {
		return nil, "", nil
	}
	entries, issues := backlogScanEntryRefsForDocumentV0(rel, body)
	return backlogScanPublicEntriesV0(entries), backlogScanEntryDigestV0(entries), issues
}

func (planner idleSelfImprovementBacklogPlannerV0) backlogScanDocumentHashV0(rel string) (string, bool) {
	clean, ok := backlogScanCleanDocumentRelV0(rel)
	if !ok {
		return backlogScanHashV0("blocked:" + strings.TrimSpace(rel)), true
	}
	body, err := planner.readBacklogScanDocumentBytesV0(clean)
	if err != nil {
		return backlogScanHashV0("missing:" + clean), true
	}
	return backlogScanHashV0(string(body)), false
}

func backlogScanEpochV0(docs []orquestaserver.BacklogScanDocumentV0) string {
	var parts []string
	for _, doc := range docs {
		parts = append(parts, doc.Path+"|"+strconv.Itoa(doc.StartLine)+"|"+doc.SHA256)
	}
	return "backlog-scan-epoch-" + backlogScanHashV0(strings.Join(parts, "\n"))[:12]
}

func backlogScanReservationRefsV0(epoch string, writeSet []string) []string {
	seed := strings.TrimSpace(epoch) + "|" + strings.Join(compactServerStackStringsV0(writeSet), "|")
	return []string{"reservation-ref-backlog-scan-doc-merge-" + backlogScanHashV0(seed)[:12]}
}

func backlogScanRefV0(request orquestaserver.IdleSelfImprovementRequestV0, sectionRef string) string {
	seed := strings.Join([]string{
		strings.TrimSpace(request.RequestRef),
		strings.TrimSpace(request.CorrelationID),
		strings.TrimSpace(request.BacklogScanEpoch),
		strings.TrimSpace(sectionRef),
	}, "|")
	return "scan-ref-backlog-" + backlogScanHashV0(seed)[:12]
}

func backlogScanContextRefsV0(request orquestaserver.IdleSelfImprovementRequestV0) []string {
	refs := []string{
		"backlog_scan_epoch:" + request.BacklogScanEpoch,
		"backlog_scan_ref:" + request.BacklogScanRef,
	}
	for _, ref := range request.ReservationRefs {
		refs = append(refs, "backlog_scan_reservation_ref:"+ref)
	}
	for _, doc := range request.BacklogScanDocs {
		refs = append(refs, backlogScanDocTokenV0(doc))
		refs = append(refs, backlogScanEntryTokensV0(doc)...)
	}
	return refs
}

func backlogScanAcceptanceCriteriaV0(request orquestaserver.IdleSelfImprovementRequestV0) []string {
	out := []string{
		"backlog_scan_epoch:" + request.BacklogScanEpoch,
		"backlog_scan_ref:" + request.BacklogScanRef,
		"si los documentos del scanner cambiaron desde esta foto, conservar propuesta como borrador y anotar rebase/merge pendiente",
		"el ACK debe citar backlog_scan_ref, lineas y hashes de docs para bloques Escaneo backlog nuevos",
	}
	for _, ref := range request.ReservationRefs {
		out = append(out, "backlog_scan_reservation_ref:"+ref)
	}
	for _, doc := range request.BacklogScanDocs {
		out = append(out, backlogScanDocTokenV0(doc))
		out = append(out, backlogScanEntryTokensV0(doc)...)
		if len(doc.ScanEntryIssues) > 0 {
			out = append(out, "si hay backlog_scan_entry_issue, conservar propuesta como borrador y anotar rebase/merge pendiente")
		}
	}
	return out
}

func backlogScanDocTokenV0(doc orquestaserver.BacklogScanDocumentV0) string {
	return "backlog_scan_doc:" + doc.Path +
		":line:" + strconv.Itoa(doc.StartLine) +
		":sha256:" + doc.SHA256
}

func backlogScanEntryTokensV0(doc orquestaserver.BacklogScanDocumentV0) []string {
	out := []string{}
	for _, entry := range doc.ScanEntries {
		out = append(out,
			"backlog_scan_entry:"+entry.Ref+
				":line:"+strconv.Itoa(entry.Line)+
				":hash:"+entry.BlockHash,
		)
	}
	if doc.ScanEntryDigest != "" {
		out = append(out,
			"backlog_scan_entries:"+doc.Path+
				":count:"+strconv.Itoa(doc.ScanEntryCount)+
				":sha256:"+doc.ScanEntryDigest,
		)
	}
	for _, issue := range doc.ScanEntryIssues {
		out = append(out, "backlog_scan_entry_issue:"+
			issue.Code+
			":line:"+strconv.Itoa(issue.Line)+
			":ref:"+issue.ScanEntryRef)
	}
	return out
}

func backlogScanHashV0(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])
}

func (planner idleSelfImprovementBacklogPlannerV0) packetBacklogDocsCurrentV0(
	packet orquestaruntime.AgentStartPacketV0,
) bool {
	expectedDocs := backlogScanDocsFromPacketV0(packet)
	if len(expectedDocs) > idleSelfImprovementBacklogScanMaxDocsV0 {
		return false
	}
	canonicalDocs := planner.backlogScanCanonicalDocumentSetV0()
	for _, expected := range expectedDocs {
		clean, ok := backlogScanCleanDocumentRelV0(expected.Path)
		if !ok || !canonicalDocs[clean] || strings.TrimSpace(expected.SHA256) == "" {
			return false
		}
		current, missing := planner.backlogScanDocumentHashV0(expected.Path)
		if missing || current != expected.SHA256 {
			return false
		}
	}
	return true
}

func (planner idleSelfImprovementBacklogPlannerV0) backlogScanCanonicalDocumentSetV0() map[string]bool {
	return backlogScanCanonicalDocSetV0(append(
		append([]string(nil), idleSelfImprovementBacklogScannerDocsV0...),
		planner.backlogScannerDocumentRefsV0()...,
	))
}

func backlogScanDocsFromPacketV0(packet orquestaruntime.AgentStartPacketV0) []orquestaserver.BacklogScanDocumentV0 {
	var docs []orquestaserver.BacklogScanDocumentV0
	for _, value := range packet.Task.DoneCriteria {
		if doc, ok := backlogScanDocFromTextV0(value); ok {
			docs = append(docs, doc)
		}
	}
	return docs
}

func backlogScanDocFromTextV0(value string) (orquestaserver.BacklogScanDocumentV0, bool) {
	_, rest, ok := strings.Cut(strings.TrimSpace(value), "backlog_scan_doc:")
	if !ok {
		return orquestaserver.BacklogScanDocumentV0{}, false
	}
	path, rest, ok := strings.Cut(rest, ":line:")
	if !ok {
		return orquestaserver.BacklogScanDocumentV0{}, false
	}
	lineText, hash, ok := strings.Cut(rest, ":sha256:")
	if !ok {
		return orquestaserver.BacklogScanDocumentV0{}, false
	}
	line, _ := strconv.Atoi(strings.TrimSpace(lineText))
	return orquestaserver.BacklogScanDocumentV0{
		Path:      strings.TrimSpace(path),
		StartLine: line,
		SHA256:    strings.TrimSpace(hash),
	}, true
}
