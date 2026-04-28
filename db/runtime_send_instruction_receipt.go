package db

import (
	"strings"
	"time"

	"orquesta/runtimeagente"
)

func runtimeOrderSendInstructionReceiptEvidence(order *RuntimeOrder, payload map[string]any, msg *RuntimeMailboxMessage) (bool, string, time.Time, error) {
	if order == nil || !runtimeOrderSendInstructionProvieneMailbox(payload) {
		return false, "", time.Time{}, nil
	}
	baseline := runtimeOrderSendInstructionReceiptBaseline(order, msg)
	if msg != nil && msg.ConsumedAt != nil && !msg.ConsumedAt.IsZero() {
		consumedAt := msg.ConsumedAt.UTC()
		if consumedAt.After(baseline) || consumedAt.Equal(baseline) {
			return true, "runtime_mailbox_consumed", consumedAt, nil
		}
	}
	handle, err := resolverHandleParaOrden(order)
	if err != nil {
		return false, "", time.Time{}, err
	}
	runtime, err := resolverRuntimeParaOrden(order)
	if err != nil {
		return false, "", time.Time{}, err
	}
	runtime, handle, err = resolverDestinoRuntimeOrderSendInstruction(order, runtime, handle)
	if err != nil || handle == nil {
		return false, "", time.Time{}, err
	}
	if confirmed, source, at, err := runtimeOrderSendInstructionDispatchLedgerEvidence(handle, order, payload); err != nil {
		return false, "", time.Time{}, err
	} else if confirmed {
		return true, source, at, nil
	}
	if runtimeOrderSendInstructionEsMicroprogramacion(payload) {
		if runtimeOrderSendInstructionPermiteReceiptTranscriptPatch(payload) {
			confirmed, receiptSource, receiptAt, err := runtimeOrderSendInstructionTranscriptPatchEvidence(runtime, handle, baseline)
			if err != nil {
				return false, "", time.Time{}, err
			}
			if confirmed {
				return true, receiptSource, receiptAt, nil
			}
		}
	}
	snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(handle.MetadataJSON)
	if err != nil || snap == nil {
		return false, "", time.Time{}, err
	}
	if runtimeOrderSendInstructionUsaTMUXPaneActivityEvidence(handle, payload) {
		if fresh, freshErr := runtimeOrderSendInstructionLoadWorkerSnapshotFresh(handle.MetadataJSON); freshErr != nil {
			return false, "", time.Time{}, freshErr
		} else if fresh != nil {
			snap = fresh
		}
	}
	if runtimeOrderSendInstructionEsMicroprogramacion(payload) {
		confirmed, receiptSource, receiptAt, err := runtimeOrderSendInstructionTMUXPanePatchEvidence(handle, snap, baseline)
		if err != nil {
			return false, "", time.Time{}, err
		}
		if confirmed {
			return true, receiptSource, receiptAt, nil
		}
	}
	if runtimeOrderSendInstructionPermiteReceiptLastProgress(payload) {
		if progress := snap.LastProgressTime(); progress != nil {
			progressAt := progress.UTC()
			if progressAt.After(baseline) {
				return true, "last_progress", progressAt, nil
			}
		}
	}
	if runtimeOrderSendInstructionEsMicroprogramacion(payload) {
		return false, "", time.Time{}, nil
	}
	if confirmed, receiptSource, receiptAt, err := runtimeOrderSendInstructionPremiumActivityEvidence(runtime, handle, snap, payload, baseline); err != nil {
		return false, "", time.Time{}, err
	} else if confirmed {
		return true, receiptSource, receiptAt, nil
	}
	if output := snap.LastOutputTime(); output != nil && runtimeOrderSendInstructionPermiteReceiptLastOutput(handle, snap, payload) {
		outputAt := output.UTC()
		if outputAt.After(baseline) {
			return true, "last_output", outputAt, nil
		}
	}
	return false, "", time.Time{}, nil
}

func runtimeOrderSendInstructionReceiptBlocked(order *RuntimeOrder, payload map[string]any, msg *RuntimeMailboxMessage, now time.Time) (bool, string, error) {
	if order == nil || msg == nil || !runtimeOrderSendInstructionProvieneMailbox(payload) {
		return false, "", nil
	}
	handle, err := resolverHandleParaOrden(order)
	if err != nil {
		return false, "", err
	}
	runtime, err := resolverRuntimeParaOrden(order)
	if err != nil {
		return false, "", err
	}
	runtime, handle, err = resolverDestinoRuntimeOrderSendInstruction(order, runtime, handle)
	if err != nil {
		return false, "", err
	}
	if handle == nil {
		return true, "worker_missing_after_notified_dispatch", nil
	}
	if runtimeOrderSendInstructionReceiptBloqueoIgnoraSnapshot(handle) {
		return false, "", nil
	}
	snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(handle.MetadataJSON)
	if err != nil {
		return false, "", err
	}
	if snap == nil {
		return true, "worker_snapshot_missing_after_notified_dispatch", nil
	}
	view := snap.View(now.UTC(), time.Minute)
	if view == nil {
		return true, "worker_view_missing_after_notified_dispatch", nil
	}
	if view.HeartbeatStale {
		return true, "worker_heartbeat_stale_after_notified_dispatch", nil
	}
	if !view.Alive {
		return true, "worker_not_alive_after_notified_dispatch", nil
	}
	switch strings.ToLower(strings.TrimSpace(view.State)) {
	case "failed", "stopped", "exited", "closed", "stale":
		return true, "worker_" + strings.ToLower(strings.TrimSpace(view.State)) + "_after_notified_dispatch", nil
	default:
		return false, "", nil
	}
}

func reconciliarRuntimeOrderSendInstructionMailboxEntregado(order *RuntimeOrder, payload map[string]any, msg *RuntimeMailboxMessage, now time.Time) (bool, error) {
	if order == nil || msg == nil || !strings.EqualFold(strings.TrimSpace(msg.Estado), "entregado") {
		return false, nil
	}
	if runtimeOrderSendInstructionEsMicroprogramacion(payload) {
		baseline := runtimeOrderSendInstructionNotifiedAt(order, msg)
		if baseline.IsZero() {
			baseline = msg.CreatedAt.UTC()
		}
		handle, err := resolverHandleParaOrden(order)
		if err != nil {
			return true, err
		}
		runtime, err := resolverRuntimeParaOrden(order)
		if err != nil {
			return true, err
		}
		runtime, handle, err = resolverDestinoRuntimeOrderSendInstruction(order, runtime, handle)
		if err != nil {
			return true, err
		}
		blockedByAgent, blockedDetail, blockedAt, err := runtimeOrderSendInstructionTranscriptBlockedByAgent(runtime, handle, baseline)
		if err != nil {
			return true, err
		}
		if blockedByAgent {
			if err := MarcarRuntimeMailboxConsumido(msg.ID); err != nil {
				return true, err
			}
			if err := fallarRuntimeOrderSendInstructionPorBloqueoAgente(order, payload, blockedDetail, blockedAt); err != nil {
				return true, err
			}
			return true, nil
		}
	}
	confirmed, receiptSource, receiptAt, err := runtimeOrderSendInstructionReceiptEvidence(order, payload, msg)
	if err != nil {
		return true, err
	}
	if confirmed {
		if err := MarcarRuntimeMailboxConsumido(msg.ID); err != nil {
			return true, err
		}
		if err := completarRuntimeOrderSendInstructionEntregadaPorReceipt(order, payload, receiptSource, receiptAt); err != nil {
			return true, err
		}
		return true, nil
	}
	if absorbed, reason, err := RuntimeOrderSendInstructionAbsorbidaPorTrabajoVivo(order, now); err != nil {
		return true, err
	} else if absorbed {
		if err := MarcarRuntimeMailboxConsumido(msg.ID); err != nil {
			return true, err
		}
		if err := completarRuntimeOrderSendInstructionSupersedida(order, payload, reason); err != nil {
			return true, err
		}
		return true, nil
	}
	blocked, blockedReason, err := runtimeOrderSendInstructionReceiptBlocked(order, payload, msg, now)
	if err != nil {
		return true, err
	}
	if blocked {
		runtimeOrderSendInstructionAutoStopLocalOllama(order, payload)
		if err := RearmarRuntimeMailboxPendiente(msg.ID); err != nil {
			return true, err
		}
		if err := reencolarRuntimeOrderSendInstruction(order, payload, blockedReason); err != nil {
			return true, err
		}
		return true, nil
	}
	if runtimeOrderSendInstructionReceiptTimedOut(order, msg, now) {
		runtimeOrderSendInstructionAutoStopLocalOllama(order, payload)
		if err := RearmarRuntimeMailboxPendiente(msg.ID); err != nil {
			return true, err
		}
		if err := reencolarRuntimeOrderSendInstruction(order, payload, "delivery receipt timeout"); err != nil {
			return true, err
		}
		return true, nil
	}
	if err := retenerRuntimeOrderSendInstructionNotificada(order, payload, "mailbox notificado pendiente de recibo", runtimeOrderSendInstructionNotifiedAt(order, msg)); err != nil {
		return true, err
	}
	return true, nil
}
