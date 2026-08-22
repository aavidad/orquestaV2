package agentmicrovm

import (
	"context"
	"testing"

	"orquesta/internal/ports"
)

func TestAgentMicroVMLifecycleContractsRemainCompleteAndFailClosed(t *testing.T) {
	var lifecycle ports.AgentEnvironmentLifecycle = (*Adapter)(nil)
	var reconciler ports.AgentEnvironmentLifecycleReconciler = (*Adapter)(nil)
	adapter := &Adapter{}

	if _, err := lifecycle.(*Adapter).Quiesce(context.Background(), ports.AgentQuiesceRequest{}); ErrorCode(err) != CodeLifecycleReceiptUnavailable {
		t.Fatalf("Quiesce code=%q err=%v", ErrorCode(err), err)
	}
	if _, err := adapter.Close(context.Background(), ports.AgentCloseRequest{}); ErrorCode(err) != CodeLifecycleReceiptUnavailable {
		t.Fatalf("Close code=%q err=%v", ErrorCode(err), err)
	}
	if _, err := reconciler.(*Adapter).ReconcileQuiesce(context.Background(), ports.AgentQuiesceRequest{}); ErrorCode(err) != CodeLifecycleReceiptUnavailable {
		t.Fatalf("ReconcileQuiesce code=%q err=%v", ErrorCode(err), err)
	}
	if _, err := adapter.ReconcilePreserve(context.Background(), ports.AgentPreserveRequest{}); ErrorCode(err) != CodePreserveRecoveryUnsupported {
		t.Fatalf("ReconcilePreserve code=%q err=%v", ErrorCode(err), err)
	}
	if _, err := adapter.ReconcileClose(context.Background(), ports.AgentCloseRequest{}); ErrorCode(err) != CodeLifecycleReceiptUnavailable {
		t.Fatalf("ReconcileClose code=%q err=%v", ErrorCode(err), err)
	}
}
