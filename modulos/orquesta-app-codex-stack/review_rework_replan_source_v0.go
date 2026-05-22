package orquestaappcodexstack

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	orquestacorereplanner "orquesta/modulos/orquesta-core-replanner"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

type ReviewReworkReplanSourceV0 struct {
	Store    orquestaruntimecodexdelivery.CodexReceiptDescriptorStorePortV0
	Capacity CapacityConfigV0
}

var _ orquestacionnucleoapp.ReviewReworkReplanPlanProviderPortV0 = ReviewReworkReplanSourceV0{}

type reviewReworkProjectionV0 struct {
	ReworkRequestRef string
	ReviewResultRef  string
	ReviewRequestID  string
	DeliveryRef      string
}

type reviewResultProjectionV0 struct {
	ReviewResultRef string
	Status          orquestacoreworkflow.ReviewResultStatusV0
	ReviewRequestID string
	DeliveryRef     string
}

func (source ReviewReworkReplanSourceV0) BuildReviewReworkReplanPlansV0(
	ctx context.Context,
	request orquestacionnucleoapp.ReviewReworkReplanPlanRequestV0,
) ([]orquestacionnucleoapp.ReviewReworkReplanPlanV0, error) {
	if source.Store == nil {
		return nil, fmt.Errorf("review_rework_replan_source: receipt_store requerido")
	}
	descriptors, err := source.Store.ListCodexReceiptDescriptorsV0(
		ctx,
		orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{RunID: request.Run.RunID},
	)
	if err != nil {
		return nil, err
	}
	plans := make([]orquestacionnucleoapp.ReviewReworkReplanPlanV0, 0, len(request.Run.ReworkRequests))
	for _, raw := range request.Run.ReworkRequests {
		rework, ok := parseReviewReworkProjectionV0(raw)
		if !ok || reworkRetryAgentAlreadyRequestedV0(request.Run, rework.ReworkRequestRef) {
			continue
		}
		result, ok := reviewResultForReworkV0(request.Run, rework)
		if !ok || !reviewResultNeedsReworkV0(result.Status) {
			continue
		}
		plans = append(plans, source.planForReworkV0(request, rework, result, descriptors))
	}
	return plans, nil
}

func (source ReviewReworkReplanSourceV0) planForReworkV0(
	request orquestacionnucleoapp.ReviewReworkReplanPlanRequestV0,
	rework reviewReworkProjectionV0,
	result reviewResultProjectionV0,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) orquestacionnucleoapp.ReviewReworkReplanPlanV0 {
	taskRef, fallback := taskRefForReworkDeliveryV0(request.Run, rework.DeliveryRef, descriptors)
	suffix := reviewReworkReplanSafeRefV0(rework.ReworkRequestRef)
	missingTargets := reviewReworkMissingWriteSetTargetsV0(rework.DeliveryRef, descriptors)
	evidence := reviewReworkReplanEvidenceRefsV0(request, rework, result, fallback, missingTargets)
	summary := reviewReworkReplanSummaryV0(missingTargets)
	return orquestacionnucleoapp.ReviewReworkReplanPlanV0{
		CandidateRef:               "review-rework-replan-candidate-ref-" + suffix,
		ReplanRef:                  "replan-ref-" + suffix,
		SignalRef:                  "review-rework-signal-ref-" + suffix,
		ReworkRequestRef:           rework.ReworkRequestRef,
		TaskRef:                    taskRef,
		ReasonRef:                  "reason-ref-" + reviewReworkReplanSafeRefV0(result.ReviewResultRef),
		RequestedAction:            orquestacorereplanner.ReplanActionRetryTaskV0,
		CapacityRequestRef:         "capacity-ref-" + suffix,
		AgentRequestID:             "agent-ref-" + suffix,
		AgentRole:                  "implementacion",
		MinimumRecommendedCapacity: reviewReworkReplanCapacityV0(source.Capacity),
		Summary:                    summary,
		EvidenceRefs:               evidence,
		ReviewResult: orquestacoreworkflow.ReviewResultV0{
			ReviewResultRef: result.ReviewResultRef,
			ReviewRequestID: result.ReviewRequestID,
			DeliveryRef:     result.DeliveryRef,
			Status:          result.Status,
			Summary:         summary,
			EvidenceRefs:    evidence,
			QualityGateRef:  "quality-gate-ref-" + reviewReworkReplanSafeRefV0(result.DeliveryRef),
		},
	}
}

func taskRefForReworkDeliveryV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	deliveryRef string,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) (string, bool) {
	deliveryRef = strings.TrimSpace(deliveryRef)
	for _, descriptor := range descriptors {
		if strings.TrimSpace(descriptor.Spec.AgentPacket.DeliveryRefs.AckRef) != deliveryRef {
			continue
		}
		taskRef := strings.TrimSpace(descriptor.Spec.AgentPacket.Task.TaskRef)
		if taskRef != "" {
			return taskRef, false
		}
	}
	return firstReviewReworkTaskRefV0(run), true
}

func firstReviewReworkTaskRefV0(run orquestacoreworkflow.OrchestrationRunV0) string {
	tasks := compactStringsV0(run.Tasks)
	if len(tasks) == 0 {
		return ""
	}
	return tasks[0]
}

func reviewResultForReworkV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	rework reviewReworkProjectionV0,
) (reviewResultProjectionV0, bool) {
	for _, raw := range run.ReviewResults {
		result, ok := parseReviewResultProjectionV0(raw)
		if !ok || result.ReviewResultRef != rework.ReviewResultRef {
			continue
		}
		if result.ReviewRequestID == rework.ReviewRequestID &&
			result.DeliveryRef == rework.DeliveryRef {
			return result, true
		}
	}
	return reviewResultProjectionV0{}, false
}

func parseReviewReworkProjectionV0(value string) (reviewReworkProjectionV0, bool) {
	reworkRef, tail, ok := strings.Cut(strings.TrimSpace(value), "#review_result:")
	if !ok {
		return reviewReworkProjectionV0{}, false
	}
	resultRef, tail, ok := strings.Cut(tail, "#review_request:")
	if !ok {
		return reviewReworkProjectionV0{}, false
	}
	reviewRef, deliveryRef, ok := strings.Cut(tail, "#delivery:")
	if !ok {
		return reviewReworkProjectionV0{}, false
	}
	projection := reviewReworkProjectionV0{
		ReworkRequestRef: strings.TrimSpace(reworkRef),
		ReviewResultRef:  strings.TrimSpace(resultRef),
		ReviewRequestID:  strings.TrimSpace(reviewRef),
		DeliveryRef:      strings.TrimSpace(deliveryRef),
	}
	return projection, projection.ReworkRequestRef != "" && projection.ReviewResultRef != "" &&
		projection.ReviewRequestID != "" && projection.DeliveryRef != ""
}

func parseReviewResultProjectionV0(value string) (reviewResultProjectionV0, bool) {
	resultRef, tail, ok := strings.Cut(strings.TrimSpace(value), "#review_result:")
	if !ok {
		return reviewResultProjectionV0{}, false
	}
	status, tail, ok := strings.Cut(tail, "#review_request:")
	if !ok {
		return reviewResultProjectionV0{}, false
	}
	reviewRef, deliveryRef, ok := strings.Cut(tail, "#delivery:")
	if !ok {
		return reviewResultProjectionV0{}, false
	}
	projection := reviewResultProjectionV0{
		ReviewResultRef: strings.TrimSpace(resultRef),
		Status:          orquestacoreworkflow.ReviewResultStatusV0(strings.TrimSpace(status)),
		ReviewRequestID: strings.TrimSpace(reviewRef),
		DeliveryRef:     strings.TrimSpace(deliveryRef),
	}
	return projection, projection.ReviewResultRef != "" &&
		projection.ReviewRequestID != "" && projection.DeliveryRef != ""
}

func reworkRetryAgentAlreadyRequestedV0(
	run orquestacoreworkflow.OrchestrationRunV0,
	reworkRef string,
) bool {
	agentRef := "agent-ref-" + reviewReworkReplanSafeRefV0(reworkRef)
	return reviewReworkReplanStringInSetV0(run.Agents, agentRef) ||
		reviewReworkReplanStringInSetV0(run.StartedAgents, agentRef)
}

func reviewReworkReplanStringInSetV0(values []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, value := range values {
		if strings.TrimSpace(value) == want {
			return true
		}
	}
	return false
}

func reviewResultNeedsReworkV0(status orquestacoreworkflow.ReviewResultStatusV0) bool {
	return status == orquestacoreworkflow.ReviewResultStatusChangesRequestedV0 ||
		status == orquestacoreworkflow.ReviewResultStatusRejectedV0
}

func reviewReworkReplanCapacityV0(
	config CapacityConfigV0,
) orquestacoreworkflow.OrchestrationCapacityRecommendationV0 {
	if strings.TrimSpace(string(config.Tier)) != "" {
		return config.Tier
	}
	return orquestacoreworkflow.OrchestrationCapacityHighV0
}

func reviewReworkReplanEvidenceRefsV0(
	request orquestacionnucleoapp.ReviewReworkReplanPlanRequestV0,
	rework reviewReworkProjectionV0,
	result reviewResultProjectionV0,
	fallback bool,
	missingTargets []string,
) []string {
	refs := []string{
		"evidence-ref-review-rework-replan",
		rework.ReworkRequestRef,
		result.ReviewResultRef,
		rework.DeliveryRef,
	}
	if fallback {
		refs = append(refs, "evidence-ref-review-rework-task-fallback")
	}
	for _, target := range reviewReworkTargetEvidenceRefsV0(missingTargets) {
		refs = append(refs, target)
	}
	return compactStringsV0(append(refs, reviewReworkReplanNeutralEvidenceRefsV0(request.EvidenceRefs)...))
}

func reviewReworkReplanSummaryV0(missingTargets []string) string {
	missingTargets = compactStringsV0(missingTargets)
	if len(missingTargets) == 0 {
		return "Repetir tarea tras revision no aceptada."
	}
	return "Corregir entrega tras revision; conservar lo valido y completar faltantes: " +
		strings.Join(reviewReworkSummaryTargetsV0(missingTargets), ", ") + "."
}

func reviewReworkSummaryTargetsV0(targets []string) []string {
	targets = compactStringsV0(targets)
	if len(targets) <= 6 {
		return targets
	}
	return append(append([]string(nil), targets[:6]...), "mas")
}

func reviewReworkTargetEvidenceRefsV0(targets []string) []string {
	targets = compactStringsV0(targets)
	refs := make([]string, 0, len(targets))
	for index, target := range targets {
		if index >= 8 {
			break
		}
		refs = append(refs, "review-rework-missing-"+reviewReworkReplanSafeOpaqueTargetV0(target))
	}
	return refs
}

func reviewReworkMissingWriteSetTargetsV0(
	deliveryRef string,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) []string {
	deliveryRef = strings.TrimSpace(deliveryRef)
	missing := make([]string, 0)
	for _, descriptor := range descriptors {
		if strings.TrimSpace(descriptor.Spec.AgentPacket.DeliveryRefs.AckRef) != deliveryRef {
			continue
		}
		projectDir := strings.TrimSpace(descriptor.ProjectWorkDir)
		if projectDir == "" {
			continue
		}
		for _, target := range compactStringsV0(descriptor.Spec.AgentPacket.Task.WriteSet) {
			if reviewReworkProjectTargetExistsV0(projectDir, target) {
				continue
			}
			missing = append(missing, target)
		}
	}
	return compactStringsV0(missing)
}

func reviewReworkProjectTargetExistsV0(projectDir string, rawTarget string) bool {
	target, ok := reviewReworkRelTargetV0(rawTarget)
	if !ok {
		return false
	}
	if reviewReworkTargetHasGlobV0(target) {
		return reviewReworkProjectGlobHasFileV0(projectDir, target)
	}
	info, err := os.Stat(filepath.Join(projectDir, filepath.FromSlash(target)))
	if err != nil {
		return false
	}
	if !info.IsDir() {
		return info.Size() > 0
	}
	return reviewReworkDirHasFileV0(filepath.Join(projectDir, filepath.FromSlash(target)))
}

func reviewReworkProjectGlobHasFileV0(projectDir string, pattern string) bool {
	re, err := reviewReworkGlobRegexpV0(pattern)
	if err != nil {
		return false
	}
	found := false
	_ = filepath.WalkDir(projectDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry == nil || found {
			return nil
		}
		if entry.IsDir() {
			if reviewReworkSkipProjectDirV0(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil || info.Size() == 0 {
			return nil
		}
		rel, err := filepath.Rel(projectDir, path)
		if err == nil && re.MatchString(filepath.ToSlash(rel)) {
			found = true
		}
		return nil
	})
	return found
}

func reviewReworkDirHasFileV0(dir string) bool {
	found := false
	_ = filepath.WalkDir(dir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry == nil || found {
			return nil
		}
		if entry.IsDir() {
			if path != dir && reviewReworkSkipProjectDirV0(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err == nil && info.Size() > 0 {
			found = true
		}
		return nil
	})
	return found
}

func reviewReworkRelTargetV0(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" ||
		strings.Contains(value, "://") ||
		strings.HasPrefix(value, "~") ||
		strings.Contains(value, "$HOME") ||
		strings.ContainsAny(value, "\x00\r\n") ||
		filepath.IsAbs(value) {
		return "", false
	}
	cleaned := filepath.ToSlash(filepath.Clean(value))
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", false
	}
	return cleaned, true
}

func reviewReworkTargetHasGlobV0(value string) bool {
	return strings.ContainsAny(value, "*?[")
}

func reviewReworkGlobRegexpV0(pattern string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		if strings.HasPrefix(pattern[i:], "**") {
			b.WriteString(".*")
			i++
			continue
		}
		switch pattern[i] {
		case '*':
			b.WriteString("[^/]*")
		case '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(string(pattern[i])))
		}
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}

func reviewReworkSkipProjectDirV0(name string) bool {
	switch name {
	case ".git", ".orquesta-runtime", ".orquesta-codex-runtime":
		return true
	default:
		return false
	}
}

func reviewReworkReplanSafeOpaqueTargetV0(value string) string {
	value = strings.TrimSpace(filepath.ToSlash(value))
	replacer := strings.NewReplacer(
		"\\", "-",
		"/", "-",
		" ", "-",
		"*", "star",
		"?", "q",
		"[", "-",
		"]", "-",
	)
	value = strings.Trim(replacer.Replace(value), "-")
	if value == "" {
		return "target"
	}
	if len(value) > 120 {
		return value[:120]
	}
	return value
}

func reviewReworkReplanSafeRefV0(value string) string {
	value = strings.TrimSpace(value)
	replacer := strings.NewReplacer("\\", "-", "/", "-", " ", "-", "#", "-", ":", "-")
	value = strings.Trim(replacer.Replace(value), "-")
	if value == "" {
		return "sin-ref"
	}
	return value
}

func reviewReworkReplanNeutralEvidenceRefsV0(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range compactStringsV0(values) {
		if reviewReworkReplanEvidenceRefIsNeutralV0(value) {
			out = append(out, value)
		}
	}
	return out
}

func reviewReworkReplanEvidenceRefIsNeutralV0(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	if lower == "" {
		return false
	}
	for _, token := range reviewReworkReplanForbiddenEvidenceTokensV0() {
		if reviewReworkReplanHasEvidenceTokenV0(lower, token) {
			return false
		}
	}
	return true
}

func reviewReworkReplanForbiddenEvidenceTokensV0() []string {
	return []string{
		"db", "database", "sql", "dsn",
		"runtime", "provider", "proveedor", "model", "modelo",
		"home", "oauth", "codex", "claude", "ollama", "vllm",
		"adapter", "adaptador", "filesystem", "git", "docker", "tmux",
		"secret", "secreto", "token", "password", "credential", "credencial", "api_key",
	}
}

func reviewReworkReplanHasEvidenceTokenV0(value string, token string) bool {
	token = strings.ToLower(strings.TrimSpace(token))
	if token == "" {
		return false
	}
	start := 0
	for {
		index := strings.Index(value[start:], token)
		if index < 0 {
			return false
		}
		absolute := start + index
		if reviewReworkReplanTokenBoundaryV0(value, absolute, absolute+len(token)) {
			return true
		}
		start = absolute + len(token)
	}
}

func reviewReworkReplanTokenBoundaryV0(value string, start int, end int) bool {
	return (start == 0 || !reviewReworkReplanTokenCharV0(value[start-1])) &&
		(end >= len(value) || !reviewReworkReplanTokenCharV0(value[end]))
}

func reviewReworkReplanTokenCharV0(value byte) bool {
	return (value >= 'a' && value <= 'z') ||
		(value >= '0' && value <= '9') ||
		value == '_'
}
