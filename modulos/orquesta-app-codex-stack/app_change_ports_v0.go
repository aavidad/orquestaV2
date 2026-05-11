package orquestaappcodexstack

import (
	"context"
	"strings"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadirectorcycleoutbox "orquesta/modulos/orquesta-director-cycle-outbox"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
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
			Summary:      appChangeSummaryV0(request),
			Options:      append([]string(nil), request.AcceptanceCriteria...),
			EvidenceRefs: appChangeEvidenceRefsV0(request),
			Blocking:     false,
		},
	)
}

func appChangeQuestionRefV0(changeRef string) string {
	return "question-ref-app-change-" + strings.TrimSpace(changeRef)
}

func appChangeSummaryV0(request orquestaappchange.AppChangeRequestV0) string {
	parts := []string{"Cambio solicitado", request.UserIntent}
	if request.TargetArea != "" {
		parts = append(parts, "area "+request.TargetArea)
	}
	if request.AppRef != "" {
		parts = append(parts, "app "+request.AppRef)
	}
	return strings.Join(parts, ": ")
}

func appChangeEvidenceRefsV0(request orquestaappchange.AppChangeRequestV0) []string {
	refs := []string{request.ChangeRef}
	refs = append(refs, request.CurrentStateRefs...)
	refs = append(refs, request.MetadataRefs...)
	return compactCodexStackStringsV0(refs)
}

func firstAppChangeNonEmptyV0(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func compactCodexStackStringsV0(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	return out
}
