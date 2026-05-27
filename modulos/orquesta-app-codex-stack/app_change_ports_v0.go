package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

const (
	appChangeDirectorMaxSummaryLenV0     = 700
	appChangeDirectorMaxOptionsV0        = 5
	appChangeDirectorMaxOptionLenV0      = 180
	appChangeDirectorMaxEvidenceRefsV0   = 10
	appChangeDirectorMaxEvidenceRefLenV0 = 180
)

func appChangePortsV0(config ConfigV0) orquestaappchange.AppChangePortsV0 {
	ports := config.AppChange
	if ports.Store == nil {
		ports.Store = config.Stores.AppChangeStore
	}
	if ports.DirectorNotifier == nil {
		ports.DirectorNotifier = workflowAppChangeNotifierV0{config: config}
	}
	return ports
}

type workflowAppChangeNotifierV0 struct {
	config ConfigV0
}

func (notifier workflowAppChangeNotifierV0) NotifyAppChangeRequestedV0(
	ctx context.Context,
	record orquestaappchange.AppChangeRecordV0,
) (orquestaappchange.AppChangeDirectorNotificationV0, error) {
	command, err := appChangeAskDirectorCommandV0(notifier.config, record)
	if err != nil {
		return orquestaappchange.AppChangeDirectorNotificationV0{}, err
	}
	result, err := orquestacionnucleoapp.HandleStoredWorkflowCommandV0(
		ctx,
		notifier.config.Stores.RunStore,
		notifier.config.Stores.EventSink,
		command,
	)
	if err != nil {
		return orquestaappchange.AppChangeDirectorNotificationV0{}, err
	}
	if len(result.Outbox) > 0 {
		_, err = orquestadirectorcycleoutbox.RecordDirectorCycleOutboxV0(
			ctx,
			orquestadirectorcycleoutbox.DirectorCycleOutboxRecordInputV0{
				Ledger:     notifier.config.Stores.OutboxLedger,
				RunRef:     record.Request.RunRef,
				TargetPort: orquestacoreworkflow.OutboxTargetDirectorV0,
				Messages:   result.Outbox,
			},
		)
		if err != nil {
			return orquestaappchange.AppChangeDirectorNotificationV0{}, err
		}
	}
	return orquestaappchange.AppChangeDirectorNotificationV0{
		DirectorQuestionRef: appChangeQuestionRefV0(record.Request.ChangeRef),
		EvidenceRefs:        appChangeEvidenceRefsV0(record.Request),
	}, nil
}

func appChangeAskDirectorCommandV0(
	config ConfigV0,
	record orquestaappchange.AppChangeRecordV0,
) (orquestacoreworkflow.OrchestrationCommandV0, error) {
	request := record.Request
	return orquestacoreworkflow.NewAskDirectorCommandV0(
		orquestacoreworkflow.OrchestrationCommandMetaV0{
			CommandID:      "cmd-app-change-" + request.ChangeRef,
			RunID:          request.RunRef,
			IdempotencyKey: "idem-app-change-" + request.ChangeRef,
			CorrelationID:  firstAppChangeNonEmptyV0(request.CorrelationID, request.RequestID, request.ChangeRef),
			RequestedBy:    firstAppChangeNonEmptyV0(record.RequestedBy, config.Capacity.RequestedBy),
			OccurredAt:     config.Capacity.OccurredAt,
		},
		orquestacoreworkflow.AskDirectorCommandPayloadV0{
			QuestionID:   appChangeQuestionRefV0(request.ChangeRef),
			SourceGroup:  "user",
			TargetGroup:  "director",
			Summary:      appChangeDirectorSummaryV0(request),
			Options:      appChangeDirectorOptionsV0(request),
			EvidenceRefs: appChangeDirectorEvidenceRefsV0(request),
			Blocking:     false,
		},
	)
}

func appChangeQuestionRefV0(changeRef string) string {
	return "question-ref-app-change-" + strings.TrimSpace(changeRef)
}

func appChangeDirectorSummaryV0(request orquestaappchange.AppChangeRequestV0) string {
	parts := []string{"Cambio solicitado", request.UserIntent}
	if request.TargetArea != "" {
		parts = append(parts, "area "+request.TargetArea)
	}
	if request.AppRef != "" {
		parts = append(parts, "app "+request.AppRef)
	}
	return appChangeCompactTextV0(
		appChangeDirectorSafeTextV0(strings.Join(parts, ": ")),
		appChangeDirectorMaxSummaryLenV0,
	)
}

func appChangeEvidenceRefsV0(request orquestaappchange.AppChangeRequestV0) []string {
	refs := []string{request.ChangeRef}
	refs = append(refs, request.CurrentStateRefs...)
	refs = append(refs, request.MetadataRefs...)
	refs = append(refs, appChangeExternalWorkRefsV0(request.ExternalWork)...)
	return compactCodexStackStringsV0(refs)
}

func appChangeDirectorOptionsV0(request orquestaappchange.AppChangeRequestV0) []string {
	criteria := appChangeDirectorSafeTextsV0(request.AcceptanceCriteria)
	if len(criteria) == 0 {
		criteria = appChangeDirectorSafeTextsV0(request.Constraints)
	}
	if len(criteria) == 0 {
		return nil
	}
	if len(criteria) <= appChangeDirectorMaxOptionsV0 {
		return appChangeCompactListV0(criteria, appChangeDirectorMaxOptionsV0, appChangeDirectorMaxOptionLenV0)
	}
	kept := appChangeDirectorMaxOptionsV0 - 1
	options := appChangeCompactListV0(criteria[:kept], kept, appChangeDirectorMaxOptionLenV0)
	options = append(options, fmt.Sprintf("contrato completo persistido en app_change: %d criterios", len(criteria)))
	return options
}

func appChangeDirectorEvidenceRefsV0(request orquestaappchange.AppChangeRequestV0) []string {
	return appChangeCompactListV0(
		appChangeDirectorSafeRefsV0(appChangeEvidenceRefsV0(request)),
		appChangeDirectorMaxEvidenceRefsV0,
		appChangeDirectorMaxEvidenceRefLenV0,
	)
}

func appChangeExternalWorkRefsV0(
	work *orquestaappchange.AppChangeExternalWorkV0,
) []string {
	if work == nil {
		return nil
	}
	refs := []string{work.ProjectRef, work.WorkKind}
	refs = append(refs, work.InterfaceRefs...)
	refs = append(refs, work.WorkRefs...)
	return refs
}

func firstAppChangeNonEmptyV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func appChangeCompactListV0(values []string, maxItems int, maxLen int) []string {
	if maxItems <= 0 || maxLen <= 0 {
		return nil
	}
	compacted := compactCodexStackStringsV0(values)
	if len(compacted) > maxItems {
		compacted = compacted[:maxItems]
	}
	out := make([]string, 0, len(compacted))
	for _, value := range compacted {
		out = append(out, appChangeCompactTextV0(value, maxLen))
	}
	return out
}

func appChangeCompactTextV0(value string, maxLen int) string {
	value = strings.TrimSpace(value)
	if maxLen <= 0 || len(value) <= maxLen {
		return value
	}
	const suffix = "..."
	if maxLen <= len(suffix) {
		return value[:maxLen]
	}
	return strings.TrimSpace(value[:maxLen-len(suffix)]) + suffix
}

func appChangeDirectorSafeTextsV0(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		safe := appChangeDirectorSafeTextV0(value)
		if safe != "" {
			out = append(out, safe)
		}
	}
	return compactCodexStackStringsV0(out)
}

func appChangeDirectorSafeRefsV0(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		safe := appChangeDirectorSafeRefV0(value)
		if safe != "" {
			out = append(out, safe)
		}
	}
	return compactCodexStackStringsV0(out)
}
