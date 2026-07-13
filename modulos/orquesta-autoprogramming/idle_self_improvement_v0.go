package orquestaautoprogramming

import (
	"strconv"
	"strings"
)

const (
	OrquestaServerIdleSelfImprovementAfterSecondsEnvV0 = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS"
	OrquestaServerIdleSelfImprovementTargetQueueEnvV0  = "ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_TARGET_QUEUE"

	AutoprogrammingIdleSelfImprovementDefaultAfterSecondsV0 = 60
	AutoprogrammingIdleSelfImprovementDefaultTargetQueueV0  = 1

	AutoprogrammingExternalProjectionOutboxPendingV0           = "outbox_pending"
	AutoprogrammingExternalProjectionWaitExternalV0            = "wait_external"
	AutoprogrammingExternalProjectionExternalProcessVerifiedV0 = "external_process_verified"
	AutoprogrammingExternalProjectionNoExternalWorkV0          = "no_external_work"
	AutoprogrammingBacklogPlannerSkipVisibleInQueueV0          = "visible_in_queue"
	AutoprogrammingBacklogPlannerSkipNarrativeSectionV0        = "narrative_section"
	AutoprogrammingBacklogPlannerScannerAreaV0                 = "backlog_scan"
	AutoprogrammingBacklogPlannerScannerTaskRefV0              = "task-ref-backlog-scanner"
)

const (
	AutoprogrammingIdleBudgetUnconfiguredV0 = "budget_unconfigured"
	AutoprogrammingIdleBudgetWithinV0       = "within_budget"
	AutoprogrammingIdleBudgetDeferredV0     = "budget_deferred"
	AutoprogrammingIdleBudgetDegradedV0     = "budget_degraded"
)

type AutoprogrammingIdleSelfImprovementConfigV0 struct {
	AfterSeconds int `json:"after_seconds"`
	TargetQueue  int `json:"target_queue"`
}

type AutoprogrammingIdleSelfImprovementConfigResultV0 struct {
	Config AutoprogrammingIdleSelfImprovementConfigV0 `json:"config"`
	Issues []AutoprogrammingRequestIssueV0            `json:"issues,omitempty"`
}

type AutoprogrammingIdleSelfImprovementDecisionInputV0 struct {
	Config         AutoprogrammingIdleSelfImprovementConfigV0 `json:"config"`
	IdleForSeconds int                                        `json:"idle_for_seconds"`
	QueueSize      int                                        `json:"queue_size"`
	FreeCapacity   int                                        `json:"free_capacity"`
}

type AutoprogrammingIdleSelfImprovementDecisionV0 struct {
	Prepare        bool   `json:"prepare"`
	Reason         string `json:"reason"`
	IdleTriggered  bool   `json:"idle_triggered"`
	QueueTriggered bool   `json:"queue_triggered"`
	Disabled       bool   `json:"disabled"`
}

type AutoprogrammingIdleSelfImprovementBudgetConfigV0 struct {
	MaxGoalsPerDay              int   `json:"max_goals_per_day,omitempty"`
	MaxContextBudgetBytesPerDay int64 `json:"max_context_budget_bytes_per_day,omitempty"`
}

type AutoprogrammingIdleSelfImprovementBudgetUsageV0 struct {
	GoalsUsedToday                    int   `json:"goals_used_today,omitempty"`
	ContextBudgetBytesUsedToday       int64 `json:"context_budget_bytes_used_today,omitempty"`
	EstimatedNextContextBudgetBytes   int64 `json:"estimated_next_context_budget_bytes,omitempty"`
	PromptCacheCachedInputTokensToday int64 `json:"prompt_cache_cached_input_tokens_today,omitempty"`
}

type AutoprogrammingIdleSelfImprovementBudgetDecisionInputV0 struct {
	Config         AutoprogrammingIdleSelfImprovementBudgetConfigV0 `json:"config,omitempty"`
	Usage          AutoprogrammingIdleSelfImprovementBudgetUsageV0  `json:"usage,omitempty"`
	RequestedGoals int                                              `json:"requested_goals,omitempty"`
}

type AutoprogrammingIdleSelfImprovementBudgetDecisionV0 struct {
	Prepare                              bool   `json:"prepare"`
	Degraded                             bool   `json:"degraded,omitempty"`
	Reason                               string `json:"reason"`
	RequestedGoals                       int    `json:"requested_goals,omitempty"`
	AllowedGoals                         int    `json:"allowed_goals,omitempty"`
	MaxGoalsPerDay                       int    `json:"max_goals_per_day,omitempty"`
	GoalsUsedToday                       int    `json:"goals_used_today,omitempty"`
	GoalsRemainingToday                  int    `json:"goals_remaining_today,omitempty"`
	MaxContextBudgetBytesPerDay          int64  `json:"max_context_budget_bytes_per_day,omitempty"`
	ContextBudgetBytesUsedToday          int64  `json:"context_budget_bytes_used_today,omitempty"`
	ContextBudgetBytesRemainingToday     int64  `json:"context_budget_bytes_remaining_today,omitempty"`
	EstimatedNextContextBudgetBytes      int64  `json:"estimated_next_context_budget_bytes,omitempty"`
	PromptCacheCachedInputTokensToday    int64  `json:"prompt_cache_cached_input_tokens_today,omitempty"`
	ContextBudgetEstimateEvidencePresent bool   `json:"context_budget_estimate_evidence_present,omitempty"`
}

type AutoprogrammingBacklogPlannerEntryV0 struct {
	TaskRef            string   `json:"task_ref,omitempty"`
	SectionRef         string   `json:"section_ref,omitempty"`
	Area               string   `json:"area,omitempty"`
	Title              string   `json:"title,omitempty"`
	Narrative          bool     `json:"narrative,omitempty"`
	WriteSet           []string `json:"write_set,omitempty"`
	RequiredTests      []string `json:"required_tests,omitempty"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	ContextRefs        []string `json:"context_refs,omitempty"`
}

type AutoprogrammingVisibleQueueItemV0 struct {
	TaskRef    string `json:"task_ref,omitempty"`
	SectionRef string `json:"section_ref,omitempty"`
}

type AutoprogrammingBacklogPlannerInputV0 struct {
	Entries        []AutoprogrammingBacklogPlannerEntryV0 `json:"entries,omitempty"`
	VisibleQueue   []AutoprogrammingVisibleQueueItemV0    `json:"visible_queue,omitempty"`
	CreateScanner  bool                                   `json:"create_scanner,omitempty"`
	ScannerTaskRef string                                 `json:"scanner_task_ref,omitempty"`
	BacklogScanRef string                                 `json:"backlog_scan_ref,omitempty"`
	BacklogScan    AutoprogrammingBacklogScanV0           `json:"backlog_scan,omitempty"`
}

type AutoprogrammingBacklogPlannerSkippedV0 struct {
	TaskRef    string `json:"task_ref,omitempty"`
	SectionRef string `json:"section_ref,omitempty"`
	Reason     string `json:"reason"`
}

type AutoprogrammingBacklogPlannerResultV0 struct {
	Tasks   []AutoprogrammingBacklogPlannerEntryV0   `json:"tasks,omitempty"`
	Scanner *AutoprogrammingBacklogPlannerEntryV0    `json:"scanner,omitempty"`
	Skipped []AutoprogrammingBacklogPlannerSkippedV0 `json:"skipped,omitempty"`
}

type AutoprogrammingExternalWorkProjectionInputV0 struct {
	OutboxPendingRefs       []string `json:"outbox_pending_refs,omitempty"`
	WaitExternalRefs        []string `json:"wait_external_refs,omitempty"`
	ExternalProcessRefs     []string `json:"external_process_refs,omitempty"`
	ExternalProcessVerified bool     `json:"external_process_verified,omitempty"`
}

type AutoprogrammingExternalWorkProjectionV0 struct {
	State string   `json:"state"`
	Refs  []string `json:"refs,omitempty"`
}

func ResolveAutoprogrammingIdleSelfImprovementConfigV0(
	values map[string]string,
) AutoprogrammingIdleSelfImprovementConfigResultV0 {
	result := AutoprogrammingIdleSelfImprovementConfigResultV0{
		Config: AutoprogrammingIdleSelfImprovementConfigV0{
			AfterSeconds: AutoprogrammingIdleSelfImprovementDefaultAfterSecondsV0,
			TargetQueue:  AutoprogrammingIdleSelfImprovementDefaultTargetQueueV0,
		},
	}
	if values == nil {
		return result
	}
	result.Config.AfterSeconds, result.Issues = autoprogrammingIdleConfigIntV0(
		values,
		OrquestaServerIdleSelfImprovementAfterSecondsEnvV0,
		result.Config.AfterSeconds,
		result.Issues,
	)
	result.Config.TargetQueue, result.Issues = autoprogrammingIdleConfigIntV0(
		values,
		OrquestaServerIdleSelfImprovementTargetQueueEnvV0,
		result.Config.TargetQueue,
		result.Issues,
	)
	return result
}

func DecideAutoprogrammingIdleSelfImprovementV0(
	input AutoprogrammingIdleSelfImprovementDecisionInputV0,
) AutoprogrammingIdleSelfImprovementDecisionV0 {
	config := input.Config
	idleDisabled := config.AfterSeconds == 0
	if config.AfterSeconds < 0 {
		config.AfterSeconds = AutoprogrammingIdleSelfImprovementDefaultAfterSecondsV0
	}
	if config.TargetQueue < 0 {
		config.TargetQueue = AutoprogrammingIdleSelfImprovementDefaultTargetQueueV0
	}
	idleTriggered := !idleDisabled && input.IdleForSeconds >= config.AfterSeconds
	queueTriggered := input.FreeCapacity > 0 &&
		config.TargetQueue > 0 &&
		input.QueueSize < config.TargetQueue
	switch {
	case idleTriggered:
		return AutoprogrammingIdleSelfImprovementDecisionV0{
			Prepare:        true,
			Reason:         "idle_after_seconds_reached",
			IdleTriggered:  true,
			QueueTriggered: queueTriggered,
		}
	case queueTriggered:
		return AutoprogrammingIdleSelfImprovementDecisionV0{
			Prepare:        true,
			Reason:         "free_capacity_below_target_queue",
			QueueTriggered: true,
		}
	default:
		if idleDisabled {
			return AutoprogrammingIdleSelfImprovementDecisionV0{
				Reason:   "idle_self_improvement_disabled",
				Disabled: true,
			}
		}
		return AutoprogrammingIdleSelfImprovementDecisionV0{Reason: "primary_work_not_idle"}
	}
}

func DecideAutoprogrammingIdleSelfImprovementBudgetV0(
	input AutoprogrammingIdleSelfImprovementBudgetDecisionInputV0,
) AutoprogrammingIdleSelfImprovementBudgetDecisionV0 {
	config := normalizeAutoprogrammingIdleBudgetConfigV0(input.Config)
	usage := normalizeAutoprogrammingIdleBudgetUsageV0(input.Usage)
	requested := input.RequestedGoals
	if requested <= 0 {
		requested = 1
	}
	decision := AutoprogrammingIdleSelfImprovementBudgetDecisionV0{
		Prepare:                              true,
		Reason:                               AutoprogrammingIdleBudgetUnconfiguredV0,
		RequestedGoals:                       requested,
		AllowedGoals:                         requested,
		MaxGoalsPerDay:                       config.MaxGoalsPerDay,
		GoalsUsedToday:                       usage.GoalsUsedToday,
		MaxContextBudgetBytesPerDay:          config.MaxContextBudgetBytesPerDay,
		ContextBudgetBytesUsedToday:          usage.ContextBudgetBytesUsedToday,
		EstimatedNextContextBudgetBytes:      usage.EstimatedNextContextBudgetBytes,
		PromptCacheCachedInputTokensToday:    usage.PromptCacheCachedInputTokensToday,
		ContextBudgetEstimateEvidencePresent: usage.EstimatedNextContextBudgetBytes > 0,
	}
	if config.MaxGoalsPerDay <= 0 && config.MaxContextBudgetBytesPerDay <= 0 {
		return decision
	}
	decision.Reason = AutoprogrammingIdleBudgetWithinV0
	allowed := requested
	if config.MaxGoalsPerDay > 0 {
		remaining := config.MaxGoalsPerDay - usage.GoalsUsedToday
		if remaining < 0 {
			remaining = 0
		}
		decision.GoalsRemainingToday = remaining
		if remaining <= 0 {
			decision.Prepare = false
			decision.AllowedGoals = 0
			decision.Reason = AutoprogrammingIdleBudgetDeferredV0
			return decision
		}
		if remaining < allowed {
			allowed = remaining
		}
	}
	if config.MaxContextBudgetBytesPerDay > 0 {
		remaining := config.MaxContextBudgetBytesPerDay - usage.ContextBudgetBytesUsedToday
		if remaining < 0 {
			remaining = 0
		}
		decision.ContextBudgetBytesRemainingToday = remaining
		if remaining <= 0 {
			decision.Prepare = false
			decision.AllowedGoals = 0
			decision.Reason = AutoprogrammingIdleBudgetDeferredV0
			return decision
		}
		if usage.EstimatedNextContextBudgetBytes > 0 {
			allowedByContext := int(remaining / usage.EstimatedNextContextBudgetBytes)
			if allowedByContext <= 0 {
				decision.Prepare = false
				decision.AllowedGoals = 0
				decision.Reason = AutoprogrammingIdleBudgetDeferredV0
				return decision
			}
			if allowedByContext < allowed {
				allowed = allowedByContext
			}
		}
	}
	if allowed <= 0 {
		decision.Prepare = false
		decision.AllowedGoals = 0
		decision.Reason = AutoprogrammingIdleBudgetDeferredV0
		return decision
	}
	decision.AllowedGoals = allowed
	if allowed < requested {
		decision.Degraded = true
		decision.Reason = AutoprogrammingIdleBudgetDegradedV0
	}
	return decision
}

func PlanAutoprogrammingBacklogSelfImprovementV0(
	input AutoprogrammingBacklogPlannerInputV0,
) AutoprogrammingBacklogPlannerResultV0 {
	visible := autoprogrammingVisibleBacklogQueueSetV0(input.VisibleQueue)
	result := AutoprogrammingBacklogPlannerResultV0{}
	for _, entry := range input.Entries {
		entry = normalizeAutoprogrammingBacklogPlannerEntryV0(entry)
		if entry.TaskRef == "" && entry.SectionRef == "" {
			continue
		}
		if autoprogrammingBacklogPlannerEntryNarrativeV0(entry) {
			result.Skipped = append(result.Skipped, AutoprogrammingBacklogPlannerSkippedV0{
				TaskRef:    entry.TaskRef,
				SectionRef: entry.SectionRef,
				Reason:     AutoprogrammingBacklogPlannerSkipNarrativeSectionV0,
			})
			continue
		}
		if autoprogrammingBacklogEntryVisibleInQueueV0(entry, visible) {
			result.Skipped = append(result.Skipped, AutoprogrammingBacklogPlannerSkippedV0{
				TaskRef:    entry.TaskRef,
				SectionRef: entry.SectionRef,
				Reason:     AutoprogrammingBacklogPlannerSkipVisibleInQueueV0,
			})
			continue
		}
		result.Tasks = append(result.Tasks, entry)
	}
	if input.CreateScanner && !autoprogrammingScannerVisibleInQueueV0(input.ScannerTaskRef, input.BacklogScanRef, visible) {
		scanner := autoprogrammingBacklogScannerEntryV0(input)
		result.Scanner = &scanner
	}
	return result
}

func normalizeAutoprogrammingIdleBudgetConfigV0(
	config AutoprogrammingIdleSelfImprovementBudgetConfigV0,
) AutoprogrammingIdleSelfImprovementBudgetConfigV0 {
	if config.MaxGoalsPerDay < 0 {
		config.MaxGoalsPerDay = 0
	}
	if config.MaxContextBudgetBytesPerDay < 0 {
		config.MaxContextBudgetBytesPerDay = 0
	}
	return config
}

func normalizeAutoprogrammingIdleBudgetUsageV0(
	usage AutoprogrammingIdleSelfImprovementBudgetUsageV0,
) AutoprogrammingIdleSelfImprovementBudgetUsageV0 {
	if usage.GoalsUsedToday < 0 {
		usage.GoalsUsedToday = 0
	}
	if usage.ContextBudgetBytesUsedToday < 0 {
		usage.ContextBudgetBytesUsedToday = 0
	}
	if usage.EstimatedNextContextBudgetBytes < 0 {
		usage.EstimatedNextContextBudgetBytes = 0
	}
	if usage.PromptCacheCachedInputTokensToday < 0 {
		usage.PromptCacheCachedInputTokensToday = 0
	}
	return usage
}

func ProjectAutoprogrammingExternalWorkV0(
	input AutoprogrammingExternalWorkProjectionInputV0,
) AutoprogrammingExternalWorkProjectionV0 {
	externalRefs := compactStringsV0(input.ExternalProcessRefs)
	if input.ExternalProcessVerified && len(externalRefs) > 0 {
		return AutoprogrammingExternalWorkProjectionV0{
			State: AutoprogrammingExternalProjectionExternalProcessVerifiedV0,
			Refs:  externalRefs,
		}
	}
	waitRefs := compactStringsV0(input.WaitExternalRefs)
	if len(waitRefs) > 0 {
		return AutoprogrammingExternalWorkProjectionV0{
			State: AutoprogrammingExternalProjectionWaitExternalV0,
			Refs:  waitRefs,
		}
	}
	outboxRefs := compactStringsV0(input.OutboxPendingRefs)
	if len(outboxRefs) > 0 {
		return AutoprogrammingExternalWorkProjectionV0{
			State: AutoprogrammingExternalProjectionOutboxPendingV0,
			Refs:  outboxRefs,
		}
	}
	return AutoprogrammingExternalWorkProjectionV0{State: AutoprogrammingExternalProjectionNoExternalWorkV0}
}

func autoprogrammingIdleConfigIntV0(
	values map[string]string,
	name string,
	fallback int,
	issues []AutoprogrammingRequestIssueV0,
) (int, []AutoprogrammingRequestIssueV0) {
	raw, ok := values[name]
	if !ok {
		return fallback, issues
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback, issues
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return fallback, append(issues, autoprogrammingRequestIssueV0(
			"idle_self_improvement_config_invalid",
			name,
			"valor entero no negativo requerido",
		))
	}
	return value, issues
}

func normalizeAutoprogrammingBacklogPlannerEntryV0(
	entry AutoprogrammingBacklogPlannerEntryV0,
) AutoprogrammingBacklogPlannerEntryV0 {
	entry.TaskRef = strings.TrimSpace(entry.TaskRef)
	entry.SectionRef = strings.TrimSpace(entry.SectionRef)
	entry.Area = normalizeAutoprogrammingTaskAreaV0(entry.Area)
	entry.Title = strings.TrimSpace(entry.Title)
	entry.WriteSet = compactStringsV0(entry.WriteSet)
	entry.RequiredTests = compactStringsV0(entry.RequiredTests)
	entry.AcceptanceCriteria = compactStringsV0(entry.AcceptanceCriteria)
	entry.ContextRefs = compactStringsV0(entry.ContextRefs)
	return entry
}

func autoprogrammingVisibleBacklogQueueSetV0(
	items []AutoprogrammingVisibleQueueItemV0,
) map[string]bool {
	out := make(map[string]bool, len(items)*2)
	for _, item := range items {
		if taskRef := strings.TrimSpace(item.TaskRef); taskRef != "" {
			out["task:"+taskRef] = true
		}
		if sectionRef := strings.TrimSpace(item.SectionRef); sectionRef != "" {
			out["section:"+sectionRef] = true
		}
	}
	return out
}

func autoprogrammingBacklogEntryVisibleInQueueV0(
	entry AutoprogrammingBacklogPlannerEntryV0,
	visible map[string]bool,
) bool {
	return (entry.TaskRef != "" && visible["task:"+entry.TaskRef]) ||
		(entry.SectionRef != "" && visible["section:"+entry.SectionRef])
}

func autoprogrammingBacklogPlannerEntryNarrativeV0(
	entry AutoprogrammingBacklogPlannerEntryV0,
) bool {
	if entry.Narrative {
		return true
	}
	// El scanner puede adjuntar refs de procedencia a una seccion documental.
	// Esas refs no convierten la seccion en una tarea ejecutable: el contrato
	// posterior exige un task_ref para conservar causalidad e idempotencia.
	return entry.TaskRef == ""
}

func autoprogrammingScannerVisibleInQueueV0(
	scannerTaskRef string,
	backlogScanRef string,
	visible map[string]bool,
) bool {
	scannerTaskRef = strings.TrimSpace(scannerTaskRef)
	if scannerTaskRef == "" {
		scannerTaskRef = AutoprogrammingBacklogPlannerScannerTaskRefV0
	}
	if visible["task:"+scannerTaskRef] || visible["section:backlog_scanner"] {
		return true
	}
	backlogScanRef = strings.TrimSpace(backlogScanRef)
	return backlogScanRef != "" && visible["section:"+backlogScanRef]
}

func autoprogrammingBacklogScannerEntryV0(
	input AutoprogrammingBacklogPlannerInputV0,
) AutoprogrammingBacklogPlannerEntryV0 {
	taskRef := strings.TrimSpace(input.ScannerTaskRef)
	if taskRef == "" {
		taskRef = AutoprogrammingBacklogPlannerScannerTaskRefV0
	}
	entry := AutoprogrammingBacklogPlannerEntryV0{
		TaskRef:    taskRef,
		SectionRef: "backlog_scanner",
		Area:       AutoprogrammingBacklogPlannerScannerAreaV0,
		Title:      "Escaneo backlog nuevos",
		ContextRefs: []string{
			"backlog_scanner:create_new_gap_scan",
		},
	}
	if scanRef := strings.TrimSpace(input.BacklogScanRef); scanRef != "" {
		entry.ContextRefs = append(entry.ContextRefs, "backlog_scan_ref:"+scanRef)
	}
	if input.BacklogScan.Epoch != "" {
		entry.ContextRefs = append(entry.ContextRefs, "backlog_scan_epoch:"+input.BacklogScan.Epoch)
	}
	for _, ref := range input.BacklogScan.ReservationRefs {
		entry.ContextRefs = append(entry.ContextRefs, "backlog_scan_reservation_ref:"+ref)
	}
	for _, doc := range input.BacklogScan.Documents {
		entry.ContextRefs = append(entry.ContextRefs, "backlog_scan_doc:"+doc.Path+
			":line:"+strconv.Itoa(doc.StartLine)+":sha256:"+doc.SHA256)
	}
	entry.ContextRefs = compactStringsV0(entry.ContextRefs)
	return entry
}
