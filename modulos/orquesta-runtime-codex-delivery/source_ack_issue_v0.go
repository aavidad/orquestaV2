package orquestaruntimecodexdelivery

import (
	"strings"

	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
)

func codexReceiptIssueMeansAckWithoutDeliveryV0(
	issue orquestaruntime.ExternalAgentConnectorErrorV0,
) bool {
	return string(issue.Code) == string(orquestaruntimecodex.CodexConnectorAckInvalidV0) &&
		strings.TrimSpace(issue.Field) == "status" &&
		stringInCodexDeliverySetV0(issue.Evidence, "status_not_completed")
}

func codexReceiptIssueMeansAckNotReadyV0(
	issue orquestaruntime.ExternalAgentConnectorErrorV0,
) bool {
	return string(issue.Code) == string(orquestaruntimecodex.CodexConnectorAckInvalidV0) &&
		strings.TrimSpace(issue.Field) == orquestaruntimecodex.CodexAgentAckFileNameV0 &&
		issue.Retryable &&
		stringInCodexDeliverySetV0(issue.Evidence, "ack_not_ready")
}
