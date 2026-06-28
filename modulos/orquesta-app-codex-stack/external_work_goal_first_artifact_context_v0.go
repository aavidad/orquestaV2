package orquestaappcodexstack

import (
	"context"
	"sort"
	"strconv"
	"strings"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestaexternalworkrun "orquesta/modulos/orquesta-external-work-run"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	externalWorkGoalFirstAcceptedArtifactContextEvidenceV0 = "evidence-ref-external-work-goal-first-accepted-artifact-context"
	externalWorkGoalFirstAcceptedArtifactContextMaxV0      = 24
	externalWorkGoalFirstAcceptedArtifactSummaryMaxV0      = 240
	externalWorkGoalFirstAcceptedArtifactPurposeMaxV0      = 900
)

func (executor CodexStackExternalWorkGoalFirstExecutorV0) externalWorkGoalFirstEnrichSpecV0(
	ctx context.Context,
	request orquestaexternalworkrun.StartExternalWorkRunRequestV0,
	spec orquestagoal.GoalWorkSpecV0,
) orquestagoal.GoalWorkSpecV0 {
	if !externalWorkGoalFirstNeedsAcceptedArtifactContextV0(spec) ||
		executor.DomainSubmissionLedger == nil {
		return spec
	}
	records, err := executor.DomainSubmissionLedger.ListDomainWorkArtifactSubmissionsV0(
		ctx,
		DomainWorkArtifactSubmissionRecordFilterV0{
			Status: DomainWorkArtifactSubmissionStatusAcceptedV0,
		},
	)
	if err != nil {
		return spec
	}
	records = externalWorkGoalFirstAcceptedArtifactContextRecordsV0(request, spec, records)
	if len(records) == 0 {
		return spec
	}
	spec.ContextRefs = append(spec.ContextRefs, externalWorkGoalFirstAcceptedArtifactContextRefsV0(request, spec, records)...)
	spec.AcceptanceCriteria = append(
		spec.AcceptanceCriteria,
		"Usar accepted_domain_artifact_manifest y accepted_domain_artifact como inventario publico de artefactos ya aceptados por Orquesta para el cierre; leer solo rutas relativas del workspace, no internals de OPES.",
	)
	spec.EvidenceRefs = compactStringsV0(append(
		spec.EvidenceRefs,
		externalWorkGoalFirstAcceptedArtifactContextEvidenceV0,
	))
	return orquestagoal.NormalizeGoalWorkSpecV0(spec)
}

func externalWorkGoalFirstNeedsAcceptedArtifactContextV0(spec orquestagoal.GoalWorkSpecV0) bool {
	workKind := strings.ToLower(strings.TrimSpace(spec.WorkKind))
	switch workKind {
	case "finalize_temario_package", "close_temario_package", "finalize_syllabus_package":
		return true
	}
	for _, contract := range spec.ArtifactContracts {
		artifactType := strings.ToLower(strings.TrimSpace(contract.ArtifactType))
		switch artifactType {
		case "final_domain_package", "completed_syllabus_package":
			return true
		}
	}
	return false
}

func externalWorkGoalFirstAcceptedArtifactContextRecordsV0(
	request orquestaexternalworkrun.StartExternalWorkRunRequestV0,
	spec orquestagoal.GoalWorkSpecV0,
	records []DomainWorkArtifactSubmissionRecordV0,
) []DomainWorkArtifactSubmissionRecordV0 {
	domainRef := strings.TrimSpace(spec.DomainRef)
	correlationID := strings.TrimSpace(request.CorrelationID)
	currentJobRef := ""
	if request.AppChangeRequest.ExternalWork != nil {
		currentJobRef = strings.TrimSpace(request.AppChangeRequest.ExternalWork.JobRef)
	}
	out := make([]DomainWorkArtifactSubmissionRecordV0, 0, len(records))
	for _, record := range records {
		record = normalizeDomainWorkArtifactSubmissionRecordV0(record)
		if record.Status != DomainWorkArtifactSubmissionStatusAcceptedV0 ||
			record.DomainRef != domainRef ||
			record.CorrelationID != correlationID ||
			record.JobRef == "" ||
			record.JobRef == currentJobRef ||
			record.ArtifactType == "" {
			continue
		}
		out = append(out, record)
	}
	sort.SliceStable(out, func(i, j int) bool {
		left := strings.Join([]string{out[i].RecordedAt, out[i].ArtifactType, out[i].JobRef, out[i].ReceiptRef}, "\x00")
		right := strings.Join([]string{out[j].RecordedAt, out[j].ArtifactType, out[j].JobRef, out[j].ReceiptRef}, "\x00")
		return left < right
	})
	if len(out) > externalWorkGoalFirstAcceptedArtifactContextMaxV0 {
		out = out[len(out)-externalWorkGoalFirstAcceptedArtifactContextMaxV0:]
	}
	return out
}

func externalWorkGoalFirstAcceptedArtifactContextRefsV0(
	request orquestaexternalworkrun.StartExternalWorkRunRequestV0,
	spec orquestagoal.GoalWorkSpecV0,
	records []DomainWorkArtifactSubmissionRecordV0,
) []orquestagoal.GoalContextRefV0 {
	refs := make([]orquestagoal.GoalContextRefV0, 0, len(records)+1)
	refs = append(refs, orquestagoal.GoalContextRefV0{
		Kind: "accepted_domain_artifact_manifest",
		Ref:  "accepted-domain-artifacts-" + safeDomainWorkEvidenceRefV0(request.CorrelationID),
		Purpose: strings.TrimSpace(
			"Inventario de artefactos aceptados previos para " + strings.TrimSpace(spec.WorkKind) +
				"; count=" + shortDomainWorkAcceptedArtifactCountV0(len(records)) +
				"; correlation_id=" + strings.TrimSpace(request.CorrelationID) +
				"; usar las refs accepted_domain_artifact como read-set relativo del workspace.",
		),
	})
	for _, record := range records {
		refs = append(refs, orquestagoal.GoalContextRefV0{
			Kind:    "accepted_domain_artifact",
			Ref:     externalWorkGoalFirstAcceptedArtifactContextRefV0(record),
			Purpose: externalWorkGoalFirstAcceptedArtifactPurposeV0(spec, record),
		})
	}
	return refs
}

func externalWorkGoalFirstAcceptedArtifactContextRefV0(
	record DomainWorkArtifactSubmissionRecordV0,
) string {
	return "accepted-domain-artifact-" +
		safeDomainWorkEvidenceRefV0(record.ArtifactType) +
		"-" +
		safeDomainWorkEvidenceRefV0(record.JobRef)
}

func externalWorkGoalFirstAcceptedArtifactPurposeV0(
	spec orquestagoal.GoalWorkSpecV0,
	record DomainWorkArtifactSubmissionRecordV0,
) string {
	parts := []string{
		"artifact_type=" + strings.TrimSpace(record.ArtifactType),
		"job_ref=" + strings.TrimSpace(record.JobRef),
	}
	if receiptRef := strings.TrimSpace(record.ReceiptRef); receiptRef != "" {
		parts = append(parts, "receipt_ref="+receiptRef)
	}
	if artifactRef := strings.TrimSpace(record.ArtifactRef); artifactRef != "" {
		parts = append(parts, "artifact_ref="+artifactRef)
	}
	if workKind := externalWorkGoalFirstAcceptedArtifactWorkKindV0(record); workKind != "" {
		parts = append(parts,
			"artifact_dir="+externalWorkGoalFirstAcceptedArtifactDirV0(spec, record, workKind),
			"path_candidates="+externalWorkGoalFirstAcceptedArtifactPathCandidatesV0(spec, record, workKind),
		)
	}
	if fieldNames := externalWorkGoalFirstAcceptedArtifactPayloadFieldNamesV0(record.PayloadFields); fieldNames != "" {
		parts = append(parts, "payload_fields="+fieldNames)
	}
	if summary := trimExternalWorkGoalFirstContextStringV0(
		record.Summary,
		externalWorkGoalFirstAcceptedArtifactSummaryMaxV0,
	); summary != "" {
		parts = append(parts, "summary="+summary)
	}
	return trimExternalWorkGoalFirstContextStringV0(
		strings.Join(compactStringsV0(parts), "; "),
		externalWorkGoalFirstAcceptedArtifactPurposeMaxV0,
	)
}

func externalWorkGoalFirstAcceptedArtifactWorkKindV0(
	record DomainWorkArtifactSubmissionRecordV0,
) string {
	for _, name := range []string{"source_work_kind", "work_kind", "job_type"} {
		if value := externalWorkGoalFirstDomainFieldStringV0(record.PayloadFields, name); value != "" {
			return value
		}
	}
	return ""
}

func externalWorkGoalFirstAcceptedArtifactDirV0(
	spec orquestagoal.GoalWorkSpecV0,
	record DomainWorkArtifactSubmissionRecordV0,
	workKind string,
) string {
	return "external/" +
		safeDomainWorkEvidenceRefV0(firstExternalWorkGoalFirstValueV0(spec.DomainRef, spec.ProjectRef, "domain")) +
		"/" +
		safeDomainWorkEvidenceRefV0(workKind) +
		"/" +
		safeDomainWorkEvidenceRefV0(record.JobRef)
}

func externalWorkGoalFirstAcceptedArtifactPathCandidatesV0(
	spec orquestagoal.GoalWorkSpecV0,
	record DomainWorkArtifactSubmissionRecordV0,
	workKind string,
) string {
	dir := externalWorkGoalFirstAcceptedArtifactDirV0(spec, record, workKind)
	artifactType := safeDomainWorkEvidenceRefV0(record.ArtifactType)
	return strings.Join([]string{
		dir + "/" + artifactType + ".json",
		dir + "/" + artifactType + ".md",
		dir + "/artifact.json",
		dir + "/artifact.md",
	}, "|")
}

func externalWorkGoalFirstAcceptedArtifactPayloadFieldNamesV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
) string {
	names := make([]string, 0, len(fields))
	for _, field := range fields {
		name := strings.TrimSpace(field.Name)
		if name == "" {
			continue
		}
		names = append(names, name)
		if len(names) >= 16 {
			break
		}
	}
	return strings.Join(compactStringsV0(names), ",")
}

func externalWorkGoalFirstDomainFieldStringV0(
	fields []orquestadomainwork.DomainWorkFieldV0,
	name string,
) string {
	name = strings.TrimSpace(name)
	for _, field := range fields {
		if strings.TrimSpace(field.Name) != name {
			continue
		}
		if value := strings.TrimSpace(field.Value); value != "" {
			return value
		}
		for _, value := range field.Values {
			if value = strings.TrimSpace(value); value != "" {
				return value
			}
		}
		if value := strings.TrimSpace(string(field.ValueJSON)); value != "" {
			return value
		}
	}
	return ""
}

func shortDomainWorkAcceptedArtifactCountV0(count int) string {
	return strconv.Itoa(count)
}

func trimExternalWorkGoalFirstContextStringV0(value string, maxBytes int) string {
	value = strings.TrimSpace(value)
	if maxBytes <= 0 || len([]byte(value)) <= maxBytes {
		return value
	}
	runes := []rune(value)
	for len(runes) > 0 && len([]byte(string(runes)+"...")) > maxBytes {
		runes = runes[:len(runes)-1]
	}
	return strings.TrimSpace(string(runes)) + "..."
}
