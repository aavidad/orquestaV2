package orquestaappcodexstack

import (
	"context"
	"fmt"
	"strings"
	"time"

	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimecodex "orquesta/modulos/orquesta-runtime-codex"
	orquestaruntimecodexdelivery "orquesta/modulos/orquesta-runtime-codex-delivery"
)

type AckAwareWaiterV0 struct {
	Store    orquestaruntimecodexdelivery.CodexReceiptDescriptorStorePortV0
	Interval time.Duration
}

func ackWaiterV0(config ConfigV0) AckAwareWaiterV0 {
	return AckAwareWaiterV0{
		Store:    config.Stores.ReceiptStore,
		Interval: config.Codex.WaitInterval,
	}
}

func (waiter AckAwareWaiterV0) WaitExternalProgressV0(
	ctx context.Context,
	request orquestacionnucleoapp.ExternalProgressWaitRequestV0,
) (orquestacionnucleoapp.ExternalProgressWaitResultV0, error) {
	ready, err := waiter.allStartedAgentsHaveAckV0(ctx, request)
	if err != nil {
		return orquestacionnucleoapp.ExternalProgressWaitResultV0{}, err
	}
	if !ready {
		if err := waiter.waitOneTickV0(ctx); err != nil {
			return orquestacionnucleoapp.ExternalProgressWaitResultV0{}, err
		}
	}
	return orquestacionnucleoapp.ExternalProgressWaitResultV0{
		Continue:     true,
		EvidenceRefs: []string{"evidence-ref-app-stack-wait"},
	}, nil
}

func (waiter AckAwareWaiterV0) allStartedAgentsHaveAckV0(
	ctx context.Context,
	request orquestacionnucleoapp.ExternalProgressWaitRequestV0,
) (bool, error) {
	started := compactStringsV0(request.LastResult.Run.StartedAgents)
	if len(started) == 0 {
		return false, nil
	}
	descriptors, err := waiter.Store.ListCodexReceiptDescriptorsV0(
		ctx,
		orquestaruntimecodexdelivery.CodexReceiptDescriptorRequestV0{
			RunID:         request.RunRef,
			StartedAgents: started,
		},
	)
	if err != nil {
		return false, err
	}
	return descriptorsHaveReadyAckV0(started, descriptors)
}

func descriptorsHaveReadyAckV0(
	started []string,
	descriptors []orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) (bool, error) {
	if len(descriptors) < len(started) {
		return false, nil
	}
	ready := map[string]bool{}
	for _, descriptor := range descriptors {
		ok, err := descriptorAckReadyV0(descriptor)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
		ready[strings.TrimSpace(descriptor.AgentRef)] = true
	}
	for _, agentRef := range started {
		if !ready[agentRef] {
			return false, nil
		}
	}
	return true, nil
}

func descriptorAckReadyV0(
	descriptor orquestaruntimecodexdelivery.CodexReceiptDescriptorV0,
) (bool, error) {
	_, issues := orquestaruntimecodex.ReadCodexDeliveryObservationFileV0(
		descriptor.AckPath,
		descriptor.Spec,
	)
	if len(issues) == 0 {
		return true, nil
	}
	if ackIssueRetryableV0(issues[0]) {
		return false, nil
	}
	return false, fmt.Errorf("ack invalido: %s:%s", issues[0].Code, issues[0].Field)
}

func ackIssueRetryableV0(issue orquestaruntime.ExternalAgentConnectorErrorV0) bool {
	return issue.Code == orquestaruntime.ExternalAgentConnectorErrorCodeV0(orquestaruntimecodex.CodexConnectorAckInvalidV0) &&
		strings.TrimSpace(issue.Field) == orquestaruntimecodex.CodexAgentAckFileNameV0 &&
		issue.Retryable
}

func (waiter AckAwareWaiterV0) waitOneTickV0(ctx context.Context) error {
	timer := time.NewTimer(waiter.Interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
