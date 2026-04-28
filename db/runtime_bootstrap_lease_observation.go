package db

import (
	"strings"
	"time"

	"orquesta/runtimeagente"
	"orquesta/runtimepolicy"
)

func bootstrapRuntimeLeaseEvidence(handle *RuntimeHandle, runtime *RuntimeInstance, order, startOrder *RuntimeOrder) bootstrapLeaseEvidence {
	snap := runtimeWorkerSnapshot(handle, runtime)
	if snap == nil || !snap.Alive() || snap.IsHeartbeatStale(time.Now().UTC(), time.Minute) {
		return bootstrapLeaseEvidence{}
	}
	baseline := runtimeBootstrapLeaseBaseline(order, startOrder)
	state := strings.ToLower(strings.TrimSpace(snap.EffectiveState()))
	deliveredAt := time.Time{}
	if readyAt := snap.ReadyTime(); readyAt != nil {
		deliveredAt = readyAt.UTC()
	} else if hb := snap.HeartbeatTime(); hb != nil {
		deliveredAt = hb.UTC()
	} else if updated := snap.UpdatedTime(); updated != nil {
		deliveredAt = updated.UTC()
	} else if state == "ready" || state == "running" || state == "starting" {
		if hb := snap.HeartbeatTime(); hb != nil {
			deliveredAt = hb.UTC()
		} else if updated := snap.UpdatedTime(); updated != nil {
			deliveredAt = updated.UTC()
		}
	}
	if deliveredAt.IsZero() {
		return bootstrapLeaseEvidence{}
	}
	if !baseline.IsZero() && deliveredAt.Before(baseline) {
		return bootstrapLeaseEvidence{}
	}
	evidence := bootstrapLeaseEvidence{
		delivered:   true,
		deliveredAt: deliveredAt,
	}
	if progressAt := snap.LastProgressTime(); progressAt != nil {
		progress := progressAt.UTC()
		if baseline.IsZero() || progress.After(baseline) {
			evidence.consumed = true
			return evidence
		}
	}
	if workerSnapshotAllowsBootstrapTMUXConsumeFromOutput(handle, snap) {
		if outputAt := snap.LastOutputTime(); outputAt != nil {
			output := outputAt.UTC()
			if baseline.IsZero() || output.After(baseline) {
				evidence.consumed = true
				return evidence
			}
		}
	}
	return evidence
}

func workerSnapshotAllowsBootstrapTMUXConsumeFromOutput(handle *RuntimeHandle, snap *runtimeagente.WorkerSnapshot) bool {
	if snap == nil {
		return false
	}
	deliveryMode := runtimeagente.NormalizeMailboxDeliveryMode(snap.MailboxDeliveryMode())
	if deliveryMode == "" {
		deliveryMode = runtimeagente.NormalizeMailboxDeliveryMode(RuntimeHandleMailboxDeliveryMode(handle))
	}
	if deliveryMode != runtimeagente.MailboxDeliveryBootstrapOnly {
		return false
	}
	driver := strings.TrimSpace(snap.Driver())
	if driver == "" && handle != nil {
		driver = strings.TrimSpace(stringFromMap(mapFromJSON(handle.MetadataJSON), "driver", ""))
	}
	if !strings.EqualFold(driver, "tmux_cli_session") {
		return false
	}
	transport := strings.TrimSpace(snap.Transport())
	if transport == "" && handle != nil {
		transport = strings.TrimSpace(handle.Transporte)
	}
	if !strings.EqualFold(transport, "tmux") {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(snap.EffectiveState())) {
	case "ready", "running":
		return true
	default:
		return false
	}
}

func runtimeWorkerSnapshot(handle *RuntimeHandle, runtime *RuntimeInstance) *runtimeagente.WorkerSnapshot {
	if handle == nil {
		return nil
	}
	metaJSON := strings.TrimSpace(handle.MetadataJSON)
	if metaJSON == "" {
		return nil
	}
	if snap, err := runtimeagente.LoadWorkerSnapshotFromMetadataJSON(metaJSON); err == nil && snap != nil {
		return snap
	}
	return nil
}

func runtimeBootstrapLeaseBaseline(order, startOrder *RuntimeOrder) time.Time {
	if runtimeBootstrapLeaseLinkedToStart(order, startOrder) {
		return runtimeBootstrapLeaseBaselineStartOrder(startOrder)
	}
	latest := time.Time{}
	for _, candidate := range []time.Time{
		func() time.Time {
			if startOrder != nil && startOrder.StartedAt != nil {
				return startOrder.StartedAt.UTC()
			}
			return time.Time{}
		}(),
		func() time.Time {
			if order != nil && order.StartedAt != nil {
				return order.StartedAt.UTC()
			}
			return time.Time{}
		}(),
		func() time.Time {
			if startOrder != nil {
				return startOrder.CreatedAt.UTC()
			}
			return time.Time{}
		}(),
		func() time.Time {
			if order != nil {
				return order.CreatedAt.UTC()
			}
			return time.Time{}
		}(),
	} {
		if candidate.IsZero() {
			continue
		}
		if latest.IsZero() || candidate.After(latest) {
			latest = candidate
		}
	}
	return latest
}

func runtimeBootstrapLeaseBaselineStartOrder(startOrder *RuntimeOrder) time.Time {
	if startOrder == nil {
		return time.Time{}
	}
	if startOrder.StartedAt != nil && !startOrder.StartedAt.IsZero() {
		return startOrder.StartedAt.UTC()
	}
	if !startOrder.CreatedAt.IsZero() {
		return startOrder.CreatedAt.UTC()
	}
	return time.Time{}
}

func AckBootstrapRuntimeLeaseByEvidence(handle *RuntimeHandle, runtime *RuntimeInstance, ackSource string) error {
	order, startOrderID, mailboxIDs, sesionID, err := resolverBootstrapRuntimeLeasePendiente(handle, runtime)
	if err != nil || order == nil {
		return err
	}
	var startOrder *RuntimeOrder
	if startOrderID > 0 {
		startOrder, err = GetRuntimeOrder(startOrderID)
		if err != nil {
			return err
		}
	}
	evidence := bootstrapRuntimeLeaseEvidence(handle, runtime, order, startOrder)
	if !evidence.delivered {
		return nil
	}
	if err := runtimePromoverEstadoObservadoDesdeHandle(handle, runtime); err != nil {
		return err
	}
	if evidence.consumed {
		if err := marcarBootstrapRuntimeLeaseEntregado(startOrderID, order.ID, mailboxIDs, sesionID, ackSource, evidence.deliveredAt); err != nil {
			return err
		}
		return AckBootstrapRuntimeLease(startOrderID, order.ID, mailboxIDs, sesionID, ackSource)
	}
	if err := marcarBootstrapRuntimeLeaseObservada(startOrderID, order.ID, mailboxIDs, sesionID, ackSource, evidence.deliveredAt); err != nil {
		return err
	}
	return nil
}

func RuntimeMailboxCubiertoPorBootstrapPendiente(mailboxID int64, handle *RuntimeHandle, runtime *RuntimeInstance) (bool, int64, int64, error) {
	if mailboxID <= 0 {
		return false, 0, 0, nil
	}
	order, startOrderID, mailboxIDs, _, err := resolverBootstrapRuntimeLeasePendienteParaMailbox(mailboxID, handle, runtime)
	if err != nil || order == nil {
		return false, 0, 0, err
	}
	var inherited map[string]any
	if startOrderID > 0 {
		startOrder, err := GetRuntimeOrder(startOrderID)
		if err != nil {
			return false, 0, 0, err
		}
		if startOrder != nil {
			inherited = mapFromJSON(startOrder.ResultadoJSON)
		}
	}
	if !runtimeBootstrapLeaseStateBlocksMailbox(mapFromJSON(order.ResultadoJSON), inherited) {
		return false, 0, 0, nil
	}
	for _, id := range mailboxIDs {
		if id == mailboxID {
			return true, order.ID, startOrderID, nil
		}
	}
	return false, 0, 0, nil
}

func RuntimeMailboxEntregadoPorBootstrapObservado(mailboxID int64, handle *RuntimeHandle, runtime *RuntimeInstance) (bool, int64, int64, error) {
	if mailboxID <= 0 {
		return false, 0, 0, nil
	}
	order, startOrderID, mailboxIDs, _, err := resolverBootstrapRuntimeLeaseObservadaParaMailbox(mailboxID, handle, runtime)
	if err != nil || order == nil {
		return false, 0, 0, err
	}
	for _, id := range mailboxIDs {
		if id == mailboxID {
			return true, order.ID, startOrderID, nil
		}
	}
	return false, 0, 0, nil
}

func runtimeBootstrapLeaseTieneReceiptUtil(result map[string]any) bool {
	return runtimepolicy.RuntimeBootstrapLeaseHasUsefulReceipt(result)
}

func runtimeBootstrapLeaseTieneCoberturaDeclarada(result map[string]any) bool {
	return runtimepolicy.RuntimeBootstrapLeaseHasDeclaredCoverage(result)
}

func runtimeBootstrapLeaseStateBlocksMailbox(result map[string]any, inherited map[string]any) bool {
	return runtimepolicy.RuntimeBootstrapLeaseStateBlocksMailbox(result, inherited)
}

func runtimeOrderMantieneBootstrapLeasePendiente(result map[string]any) bool {
	return runtimepolicy.RuntimeOrderKeepsBootstrapLeasePending(result)
}

func resolverBootstrapRuntimeLeaseObservada(handle *RuntimeHandle, runtime *RuntimeInstance) (*RuntimeOrder, int64, []int64, int64, error) {
	return resolverBootstrapRuntimeLeaseConFiltro(0, handle, runtime, true)
}

func resolverBootstrapRuntimeLeaseObservadaParaMailbox(mailboxID int64, handle *RuntimeHandle, runtime *RuntimeInstance) (*RuntimeOrder, int64, []int64, int64, error) {
	return resolverBootstrapRuntimeLeaseConFiltro(mailboxID, handle, runtime, true)
}

func resolverBootstrapRuntimeLeasePendiente(handle *RuntimeHandle, runtime *RuntimeInstance) (*RuntimeOrder, int64, []int64, int64, error) {
	return resolverBootstrapRuntimeLeaseConFiltro(0, handle, runtime, false)
}

func resolverBootstrapRuntimeLeasePendienteParaMailbox(mailboxID int64, handle *RuntimeHandle, runtime *RuntimeInstance) (*RuntimeOrder, int64, []int64, int64, error) {
	return resolverBootstrapRuntimeLeaseConFiltro(mailboxID, handle, runtime, false)
}
