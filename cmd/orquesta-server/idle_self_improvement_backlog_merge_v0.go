package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
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
	return planner.withBacklogMergeLeaseV0(request, "", 1, planner.backlogScannerDocumentRefsV0())
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
	return request
}

func (planner idleSelfImprovementBacklogPlannerV0) withBacklogMergeLeaseV0(
	request orquestaserver.IdleSelfImprovementRequestV0,
	sectionRef string,
	line int,
	docs []string,
) orquestaserver.IdleSelfImprovementRequestV0 {
	request.BacklogScanDocs = planner.backlogScanDocumentsV0(sectionRef, line, docs)
	request.BacklogScanEpoch = backlogScanEpochV0(request.BacklogScanDocs)
	request.ReservationRefs = backlogScanReservationRefsV0(request.BacklogScanEpoch, request.WriteSet)
	request.ContextRefs = compactServerStackStringsV0(append(
		request.ContextRefs,
		backlogScanContextRefsV0(request)...,
	))
	request.AcceptanceCriteria = compactServerStackStringsV0(append(
		request.AcceptanceCriteria,
		backlogScanAcceptanceCriteriaV0(request)...,
	))
	request.EvidenceRefs = compactServerStackStringsV0(append(
		request.EvidenceRefs,
		"evidence-ref-autoprogramming-backlog-merge-lease",
	))
	return request
}

func (planner idleSelfImprovementBacklogPlannerV0) backlogScanDocumentsV0(
	sectionRef string,
	line int,
	docs []string,
) []orquestaserver.BacklogScanDocumentV0 {
	out := make([]orquestaserver.BacklogScanDocumentV0, 0, len(docs))
	for _, rel := range compactServerStackStringsV0(docs) {
		hash, missing := planner.backlogScanDocumentHashV0(rel)
		out = append(out, orquestaserver.BacklogScanDocumentV0{
			Path:       rel,
			StartLine:  line,
			SHA256:     hash,
			Missing:    missing,
			SectionRef: sectionRef,
		})
	}
	return out
}

func (planner idleSelfImprovementBacklogPlannerV0) backlogScanDocumentHashV0(rel string) (string, bool) {
	body, err := os.ReadFile(filepath.Join(strings.TrimSpace(planner.ProjectWorkDir), rel))
	if err != nil {
		return backlogScanHashV0("missing:" + rel), true
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

func backlogScanContextRefsV0(request orquestaserver.IdleSelfImprovementRequestV0) []string {
	refs := []string{"backlog_scan_epoch:" + request.BacklogScanEpoch}
	for _, ref := range request.ReservationRefs {
		refs = append(refs, "backlog_scan_reservation_ref:"+ref)
	}
	for _, doc := range request.BacklogScanDocs {
		refs = append(refs, backlogScanDocTokenV0(doc))
	}
	return refs
}

func backlogScanAcceptanceCriteriaV0(request orquestaserver.IdleSelfImprovementRequestV0) []string {
	out := []string{
		"backlog_scan_epoch:" + request.BacklogScanEpoch,
		"si los documentos del scanner cambiaron desde esta foto, ACK failed con CONSULTA AL DIRECTOR: rebase/merge pendiente",
	}
	for _, ref := range request.ReservationRefs {
		out = append(out, "backlog_scan_reservation_ref:"+ref)
	}
	for _, doc := range request.BacklogScanDocs {
		out = append(out, backlogScanDocTokenV0(doc))
	}
	return out
}

func backlogScanDocTokenV0(doc orquestaserver.BacklogScanDocumentV0) string {
	return "backlog_scan_doc:" + doc.Path +
		":line:" + strconv.Itoa(doc.StartLine) +
		":sha256:" + doc.SHA256
}

func backlogScanHashV0(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])
}

func (planner idleSelfImprovementBacklogPlannerV0) packetBacklogDocsCurrentV0(
	packet orquestaruntime.AgentStartPacketV0,
) bool {
	for _, expected := range backlogScanDocsFromPacketV0(packet) {
		current, _ := planner.backlogScanDocumentHashV0(expected.Path)
		if current != expected.SHA256 {
			return false
		}
	}
	return true
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
