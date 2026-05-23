package orquestaappdirectorintake

import "strings"

func BuildHumanDirectorReviewablePlanV0(
	request HumanDirectorWorkIntakeRequestV0,
) HumanDirectorReviewablePlanResultV0 {
	request = normalizeHumanDirectorWorkIntakeRequestV0(request)
	issues := validateHumanDirectorWorkIntakeRequestV0(request)
	plan := humanDirectorReviewablePlanSkeletonV0(request)
	if len(issues) > 0 {
		plan.Status = HumanDirectorPlanStatusInvalidV0
		return HumanDirectorReviewablePlanResultV0{Accepted: false, Plan: plan, Issues: issues}
	}

	writeSet, writeIssues := normalizeHumanDirectorWriteSetHintsV0(request.Hints.WriteSet)
	overlapped, clean := splitHumanDirectorWriteSetOverlapsV0(writeSet)
	for _, issue := range writeIssues {
		plan.Steps = append(plan.Steps, humanDirectorPlanStepV0(
			request,
			HumanDirectorPlanActionRequestReviewV0,
			"Revisar pista de alcance insegura",
			"El director debe revisar una pista de write-set antes de preparar trabajo de codigo.",
			nil,
			nil,
			issue.Detail,
		))
	}

	if request.Request.Objective == "" {
		plan.Steps = append(plan.Steps, humanDirectorPlanStepV0(
			request,
			HumanDirectorPlanActionRequestReviewV0,
			"Aclarar objetivo humano",
			"La peticion necesita un objetivo antes de preparar trabajo de codigo.",
			nil,
			nil,
			"objetivo ausente",
		))
	}

	if len(overlapped) > 0 {
		study := humanDirectorPlanStepV0(
			request,
			HumanDirectorPlanActionStudyBeforeV0,
			"Estudiar solapes de alcance",
			"Dividir o secuenciar areas solapadas antes de preparar trabajo de codigo.",
			writeSet,
			nil,
			"write-set con rutas solapadas",
		)
		plan.Steps = append(plan.Steps, study)
		if len(clean) > 0 && request.Request.Objective != "" {
			plan.Steps = append(plan.Steps, humanDirectorPlanStepV0(
				request,
				HumanDirectorPlanActionExecuteNowV0,
				"Preparar trabajo no solapado",
				request.Request.Objective,
				clean,
				nil,
				"alcance no solapado",
			))
		}
		plan.Steps = append(plan.Steps, humanDirectorPlanStepV0(
			request,
			HumanDirectorPlanActionPostponeOverlapV0,
			"Posponer alcance solapado",
			"Crear fase posterior o replan tras estudiar dependencias.",
			overlapped,
			[]string{study.StepRef},
			firstDirectorIntakeValueV0(request.Hints.DeferredReason, "solape detectado"),
		))
		return humanDirectorReviewablePlanResultV0(plan)
	}

	if needsHumanDirectorStudyBeforeV0(request, writeSet) {
		plan.Steps = append(plan.Steps, humanDirectorPlanStepV0(
			request,
			HumanDirectorPlanActionStudyBeforeV0,
			"Estudiar peticion amplia",
			"El director debe dividir, normalizar alcance y decidir fases antes de preparar codigo.",
			writeSet,
			nil,
			"alcance amplio o incompleto",
		))
	} else if request.Request.Objective != "" && len(writeSet) > 0 {
		plan.Steps = append(plan.Steps, humanDirectorPlanStepV0(
			request,
			HumanDirectorPlanActionExecuteNowV0,
			"Preparar trabajo acotado",
			request.Request.Objective,
			writeSet,
			nil,
			"alcance suficiente",
		))
	}

	for _, reason := range request.Hints.ReviewNeeded {
		plan.Steps = append(plan.Steps, humanDirectorPlanStepV0(
			request,
			HumanDirectorPlanActionRequestReviewV0,
			"Solicitar revision humana",
			"Confirmar frontera antes de preparar trabajo de codigo.",
			nil,
			nil,
			reason,
		))
	}
	return humanDirectorReviewablePlanResultV0(plan)
}

func normalizeHumanDirectorWorkIntakeRequestV0(
	request HumanDirectorWorkIntakeRequestV0,
) HumanDirectorWorkIntakeRequestV0 {
	request.SchemaVersion = strings.TrimSpace(request.SchemaVersion)
	request.RequestRef = strings.TrimSpace(request.RequestRef)
	request.ProjectRef = strings.TrimSpace(request.ProjectRef)
	request.WorktreeRef = strings.TrimSpace(request.WorktreeRef)
	request.BranchRef = strings.TrimSpace(request.BranchRef)
	request.OccurredAt = strings.TrimSpace(request.OccurredAt)
	request.CorrelationID = strings.TrimSpace(request.CorrelationID)
	request.RequestedBy = strings.TrimSpace(request.RequestedBy)
	request.Request.Title = compactDirectorTaskSummaryTextV0(request.Request.Title)
	request.Request.Objective = compactDirectorTaskSummaryTextV0(request.Request.Objective)
	request.Request.Context = compactDirectorIntakeStringsV0(request.Request.Context)
	request.Request.AcceptanceCriteria = compactDirectorIntakeStringsV0(request.Request.AcceptanceCriteria)
	request.Request.RequiredTests = compactDirectorIntakeStringsV0(request.Request.RequiredTests)
	request.Request.CompactRules = compactDirectorIntakeStringsV0(request.Request.CompactRules)
	request.Rules = compactDirectorIntakeStringsV0(request.Rules)
	request.ContextRefs = compactDirectorIntakeStringsV0(request.ContextRefs)
	request.Hints.Areas = normalizeHumanDirectorAreasV0(request.Hints.Areas)
	request.Hints.OpaqueRefs = compactDirectorIntakeStringsV0(request.Hints.OpaqueRefs)
	request.Hints.CanRepairSafely = compactDirectorIntakeStringsV0(request.Hints.CanRepairSafely)
	request.Hints.ReviewNeeded = compactDirectorIntakeStringsV0(request.Hints.ReviewNeeded)
	request.Hints.DeferredReason = compactDirectorTaskSummaryTextV0(request.Hints.DeferredReason)
	if request.RequestedBy == "" {
		request.RequestedBy = "orquesta-app-director-intake-human-work"
	}
	return request
}

func validateHumanDirectorWorkIntakeRequestV0(
	request HumanDirectorWorkIntakeRequestV0,
) []HumanDirectorPlanIssueV0 {
	var issues []HumanDirectorPlanIssueV0
	for _, item := range []struct {
		field string
		value string
	}{
		{"schema_version", request.SchemaVersion},
		{"request_ref", request.RequestRef},
		{"project_ref", request.ProjectRef},
		{"worktree_ref", request.WorktreeRef},
		{"branch_ref", request.BranchRef},
	} {
		if item.value == "" {
			issues = append(issues, humanDirectorPlanIssueV0("required", item.field, "campo obligatorio"))
		}
	}
	if request.SchemaVersion != "" && request.SchemaVersion != HumanDirectorWorkIntakeSchemaVersionV0 {
		issues = append(issues, humanDirectorPlanIssueV0("schema_version_invalid", "schema_version", request.SchemaVersion))
	}
	return issues
}

func humanDirectorReviewablePlanSkeletonV0(
	request HumanDirectorWorkIntakeRequestV0,
) HumanDirectorReviewablePlanV0 {
	refs := []string{
		"request_ref:" + request.RequestRef,
		"project_ref:" + request.ProjectRef,
		"worktree_ref:" + request.WorktreeRef,
		"branch_ref:" + request.BranchRef,
	}
	refs = append(refs, request.ContextRefs...)
	refs = append(refs, request.Hints.OpaqueRefs...)
	return HumanDirectorReviewablePlanV0{
		SchemaVersion:  HumanDirectorReviewablePlanSchemaVersionV0,
		RequestRef:     request.RequestRef,
		ProjectRef:     request.ProjectRef,
		WorktreeRef:    request.WorktreeRef,
		BranchRef:      request.BranchRef,
		Status:         HumanDirectorPlanStatusReadyForReviewV0,
		ReviewRequired: true,
		Goal:           request.Request.Objective,
		Rules:          compactDirectorIntakeStringsV0(append(request.Rules, request.Request.CompactRules...)),
		Limits:         request.Limits,
		ContextRefs:    compactDirectorIntakeStringsV0(refs),
		EvidenceRefs: []string{
			"evidence-ref-human-director-work-intake-v0",
			"evidence-ref-" + safeDirectorIntakeRefPartV0(request.RequestRef),
		},
	}
}

func humanDirectorPlanStepV0(
	request HumanDirectorWorkIntakeRequestV0,
	action string,
	title string,
	objective string,
	writeSet []string,
	dependsOn []string,
	reason string,
) HumanDirectorPlanStepV0 {
	stepRef := "step-" + safeDirectorIntakeRefPartV0(request.RequestRef+"-"+action+"-"+title)
	area := ""
	if len(request.Hints.Areas) == 1 {
		area = request.Hints.Areas[0]
	}
	return HumanDirectorPlanStepV0{
		StepRef:            stepRef,
		Action:             action,
		Title:              title,
		Objective:          compactDirectorTaskSummaryTextV0(objective),
		Area:               area,
		WriteSet:           append([]string(nil), writeSet...),
		DependsOnStepRefs:  append([]string(nil), dependsOn...),
		Reason:             compactDirectorTaskSummaryTextV0(reason),
		AcceptanceCriteria: append([]string(nil), request.Request.AcceptanceCriteria...),
		RequiredTests:      append([]string(nil), request.Request.RequiredTests...),
		ContextRefs:        append([]string(nil), request.ContextRefs...),
		SafeRepairAllowed:  len(request.Hints.CanRepairSafely) > 0,
	}
}

func humanDirectorReviewablePlanResultV0(
	plan HumanDirectorReviewablePlanV0,
) HumanDirectorReviewablePlanResultV0 {
	if len(plan.Steps) == 0 {
		plan.Steps = append(plan.Steps, HumanDirectorPlanStepV0{
			StepRef: "step-" + safeDirectorIntakeRefPartV0(plan.RequestRef+"-review"),
			Action:  HumanDirectorPlanActionRequestReviewV0,
			Title:   "Solicitar revision humana",
			Reason:  "sin pasos derivados",
		})
	}
	return HumanDirectorReviewablePlanResultV0{Accepted: true, Plan: plan}
}

func humanDirectorPlanIssueV0(code string, field string, detail string) HumanDirectorPlanIssueV0 {
	return HumanDirectorPlanIssueV0{Code: code, Field: field, Detail: detail}
}
