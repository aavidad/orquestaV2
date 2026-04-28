package db

import (
	"encoding/json"
	"time"
)

func int64FromAny(raw any) int64 {
	switch v := raw.(type) {
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	case json.Number:
		n, _ := v.Int64()
		return n
	default:
		return 0
	}
}

func int64SliceFromAny(raw any) []int64 {
	values, ok := raw.([]any)
	if !ok || len(values) == 0 {
		return nil
	}
	out := make([]int64, 0, len(values))
	for _, item := range values {
		if id := int64FromAny(item); id > 0 {
			out = append(out, id)
		}
	}
	return out
}

func runtimeBootstrapLeaseFromOrder(order *RuntimeOrder) (startOrderID int64, mailboxIDs []int64, sesionID int64) {
	if order == nil {
		return 0, nil, 0
	}
	result := mapFromJSON(order.ResultadoJSON)
	startOrderID = int64FromAny(result["start_order_id"])
	mailboxIDs = int64SliceFromAny(result["mailbox_ids"])
	sesionID = int64FromAny(result["sesion_id"])
	return startOrderID, mailboxIDs, sesionID
}

func runtimeBootstrapLeaseMailboxIDs(order, startOrder *RuntimeOrder) []int64 {
	_, mailboxIDs, _ := runtimeBootstrapLeaseFromOrder(order)
	if len(mailboxIDs) > 0 {
		return mailboxIDs
	}
	if startOrder == nil {
		return nil
	}
	_, mailboxIDs, _ = runtimeBootstrapLeaseFromOrder(startOrder)
	return mailboxIDs
}

func runtimeBootstrapLeaseMailboxIDsConFallbackFuente(order, startOrder *RuntimeOrder) [][]int64 {
	candidates := make([][]int64, 0, 2)
	mailboxIDs := runtimeBootstrapLeaseMailboxIDs(order, startOrder)
	if len(mailboxIDs) > 0 {
		candidates = append(candidates, mailboxIDs)
	}
	if startOrder == nil || !runtimeBootstrapLeaseLinkedToStart(order, startOrder) {
		return candidates
	}
	sourceMailboxIDs := runtimeBootstrapLeaseMailboxIDs(startOrder, nil)
	if len(sourceMailboxIDs) == 0 {
		return candidates
	}
	same := len(mailboxIDs) == len(sourceMailboxIDs)
	if same {
		for i := range mailboxIDs {
			if mailboxIDs[i] != sourceMailboxIDs[i] {
				same = false
				break
			}
		}
	}
	if !same {
		candidates = append(candidates, sourceMailboxIDs)
	}
	return candidates
}

type bootstrapLeaseEvidence struct {
	delivered   bool
	deliveredAt time.Time
	consumed    bool
}

func runtimeOrderTiposBasicos() []string {
	return []string{"sync_status", "checkpoint", "nudge", "discordia"}
}

func runtimeOrderTiposDespachables() []string {
	return []string{
		"sync_status",
		"checkpoint",
		"nudge",
		"discordia",
		"start",
		"pause",
		"resume",
		"stop",
		"restart",
	}
}

func runtimeOrderTiposDiferibles() []string {
	return []string{"send_instruction"}
}
