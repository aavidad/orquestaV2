package orquestaappcodexstack

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

const defaultCodexStackUsageLogMaxBytesV0 int64 = 64 * 1024

type codexStackUsageReportStatusV0 string

const (
	codexStackUsageReportOKV0          codexStackUsageReportStatusV0 = "ok"
	codexStackUsageReportMissingV0     codexStackUsageReportStatusV0 = "missing"
	codexStackUsageReportEmptyV0       codexStackUsageReportStatusV0 = "empty"
	codexStackUsageReportUnreadableV0  codexStackUsageReportStatusV0 = "unreadable"
	codexStackUsageReportNotRedactedV0 codexStackUsageReportStatusV0 = "not_redacted"
)

type CodexStackRuntimeUsageMetricsSourceV0 struct {
	Store    CodexReceiptStorePortV0
	MaxBytes int64
}

var _ CodexStackAgentUsageMetricsProviderPortV0 = CodexStackRuntimeUsageMetricsSourceV0{}

func (source CodexStackRuntimeUsageMetricsSourceV0) BuildCodexStackAgentUsageMetricsV0(
	ctx context.Context,
	request CodexStackAgentUsageMetricsRequestV0,
) ([]CodexStackAgentUsageMetricV0, error) {
	if source.Store == nil {
		return nil, nil
	}
	descriptors, err := source.Store.ListCodexReceiptDescriptorsV0(ctx, orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{
		RunID:         strings.TrimSpace(request.RunID),
		StartedAgents: compactStringsV0(request.AgentRefs),
		CorrelationID: strings.TrimSpace(request.CorrelationID),
		EvidenceRefs:  compactStringsV0(request.EvidenceRefs),
	})
	if err != nil {
		return nil, errors.New("codex_usage_accounting_source_unavailable")
	}
	out := make([]CodexStackAgentUsageMetricV0, 0, len(descriptors))
	for _, descriptor := range descriptors {
		metric, ok := source.metricFromDescriptorV0(descriptor)
		if ok {
			out = append(out, metric)
		}
	}
	return out, nil
}

func (source CodexStackRuntimeUsageMetricsSourceV0) metricFromDescriptorV0(
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) (CodexStackAgentUsageMetricV0, bool) {
	runtimeDir := filepath.Dir(strings.TrimSpace(descriptor.AckPath))
	if runtimeDir == "." || runtimeDir == "" {
		return CodexStackAgentUsageMetricV0{}, false
	}
	report, status := source.readRedactedUsageReportV0(filepath.Join(runtimeDir, orquestaruntimecodex.CodexUsageAccountingFileNameV0))
	if status != codexStackUsageReportOKV0 {
		return source.unavailableMetricFromDescriptorV0(descriptor, status), true
	}
	snapshot := orquestaruntimecodex.BuildCodexUsageAccountingSnapshotV0([]string{report})
	if !snapshot.Observed {
		return source.unavailableMetricFromDescriptorV0(descriptor, codexStackUsageReportEmptyV0), true
	}
	return CodexStackAgentUsageMetricV0{
		AgentRequestID:   strings.TrimSpace(descriptor.AgentRef),
		QuotaStatus:      snapshot.QuotaStatus,
		QuotaRemaining:   snapshot.QuotaRemaining,
		QuotaLimit:       snapshot.QuotaLimit,
		PromptTokens:     snapshot.PromptTokens,
		CompletionTokens: snapshot.CompletionTokens,
		TotalTokens:      snapshot.TotalTokens,
		EvidenceRefs: compactStringsV0(append(
			[]string{codexStackUsageEvidenceRefV0(descriptor.DescriptorRef)},
			codexStackUsageQuotaReasonEvidenceRefsV0(snapshot.QuotaReason)...,
		)),
	}, true
}

func (source CodexStackRuntimeUsageMetricsSourceV0) unavailableMetricFromDescriptorV0(
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
	status codexStackUsageReportStatusV0,
) CodexStackAgentUsageMetricV0 {
	return CodexStackAgentUsageMetricV0{
		AgentRequestID: strings.TrimSpace(descriptor.AgentRef),
		QuotaStatus:    orquestaruntimecodex.CodexUsageQuotaUnknownV0,
		EvidenceRefs: compactStringsV0([]string{
			codexStackUsageEvidenceRefV0(descriptor.DescriptorRef),
			"quota_observed_unavailable",
			codexStackUsageReportStatusEvidenceRefV0(status),
		}),
	}
}

func (source CodexStackRuntimeUsageMetricsSourceV0) readRedactedUsageReportV0(
	path string,
) (string, codexStackUsageReportStatusV0) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", codexStackUsageReportMissingV0
		}
		return "", codexStackUsageReportUnreadableV0
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.Size() <= 0 {
		if err != nil {
			return "", codexStackUsageReportUnreadableV0
		}
		return "", codexStackUsageReportEmptyV0
	}
	maxBytes := source.MaxBytes
	if maxBytes <= 0 {
		maxBytes = defaultCodexStackUsageLogMaxBytesV0
	}
	size := info.Size()
	offset := int64(0)
	if size > maxBytes {
		offset = size - maxBytes
		size = maxBytes
	}
	buf := make([]byte, int(size))
	if _, err := file.ReadAt(buf, offset); err != nil {
		return "", codexStackUsageReportUnreadableV0
	}
	report := string(buf)
	if !orquestaruntimecodex.CodexUsageAccountingReportRedactedV0(report) {
		return "", codexStackUsageReportNotRedactedV0
	}
	return report, codexStackUsageReportOKV0
}

func codexStackUsageEvidenceRefV0(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "usage-accounting-ref-codex-runtime"
	}
	return "usage-accounting-ref-" + b.String()
}

func codexStackUsageQuotaReasonEvidenceRefsV0(reason string) []string {
	switch strings.TrimSpace(reason) {
	case orquestaruntimecodex.CodexUsageQuotaReasonNotReportedV0:
		return []string{
			"quota_observed_unavailable",
			"usage-accounting-quota-unknown-no-real-quota",
		}
	case orquestaruntimecodex.CodexUsageQuotaReasonUnknownReportedV0:
		return []string{
			"quota_observed_unavailable",
			"usage-accounting-quota-reported-unknown",
		}
	case orquestaruntimecodex.CodexUsageQuotaReasonObservedUnavailableV0:
		return []string{"quota_observed_unavailable"}
	default:
		return nil
	}
}

func codexStackUsageReportStatusEvidenceRefV0(status codexStackUsageReportStatusV0) string {
	switch status {
	case codexStackUsageReportMissingV0:
		return "usage-accounting-report-missing"
	case codexStackUsageReportEmptyV0:
		return "usage-accounting-report-empty"
	case codexStackUsageReportUnreadableV0:
		return "usage-accounting-report-unreadable"
	case codexStackUsageReportNotRedactedV0:
		return "usage-accounting-report-rejected-not-redacted"
	default:
		return ""
	}
}
