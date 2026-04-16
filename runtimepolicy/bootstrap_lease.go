package runtimepolicy

import "strings"

func RuntimeBootstrapLeaseHasUsefulReceipt(result map[string]any) bool {
	if len(result) == 0 {
		return false
	}
	if strings.TrimSpace(mapValueString(result, "receipt_source", "")) != "" {
		return true
	}
	if strings.TrimSpace(mapValueString(result, "delivery_receipt_at", "")) != "" {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(mapValueString(result, "delivery_state", "")), "delivered")
}

func RuntimeBootstrapLeaseHasDeclaredCoverage(result map[string]any) bool {
	if len(result) == 0 {
		return false
	}
	if len(int64SliceFromAny(result["mailbox_ids"])) > 0 {
		return true
	}
	return int64FromAny(result["start_order_id"]) > 0
}

func RuntimeBootstrapLeaseStateBlocksMailbox(result map[string]any, inherited map[string]any) bool {
	if len(result) == 0 {
		return false
	}
	if RuntimeBootstrapLeaseHasUsefulReceipt(result) {
		return false
	}
	if RuntimeBootstrapLeaseHasUsefulReceipt(inherited) {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(mapValueString(result, "lease_state", "")), "waiting_for_evidence")
}

func RuntimeOrderKeepsBootstrapLeasePending(result map[string]any) bool {
	if len(result) == 0 {
		return false
	}
	if !RuntimeBootstrapLeaseHasDeclaredCoverage(result) {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(mapValueString(result, "lease_state", ""))) {
	case "waiting_for_evidence", "delivered":
		return true
	default:
		return false
	}
}

func int64FromAny(raw any) int64 {
	switch v := raw.(type) {
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case float32:
		return int64(v)
	case float64:
		return int64(v)
	}
	return 0
}

func int64SliceFromAny(raw any) []int64 {
	switch v := raw.(type) {
	case []int64:
		return append([]int64(nil), v...)
	case []any:
		out := make([]int64, 0, len(v))
		for _, item := range v {
			if id := int64FromAny(item); id > 0 {
				out = append(out, id)
			}
		}
		return out
	}
	return nil
}
