package runtimepolicy

import "testing"

func TestRuntimeBootstrapLeaseHasUsefulReceipt(t *testing.T) {
	cases := []struct {
		name   string
		result map[string]any
		want   bool
	}{
		{"vacio", map[string]any{}, false},
		{"nil", nil, false},
		{"receipt_source presente", map[string]any{"receipt_source": "git_worktree"}, true},
		{"receipt_source vacio", map[string]any{"receipt_source": ""}, false},
		{"delivery_receipt_at presente", map[string]any{"delivery_receipt_at": "2026-04-14T10:00:00Z"}, true},
		{"delivery_receipt_at vacio", map[string]any{"delivery_receipt_at": "  "}, false},
		{"delivery_state delivered", map[string]any{"delivery_state": "delivered"}, true},
		{"delivery_state DELIVERED mayusculas", map[string]any{"delivery_state": "DELIVERED"}, true},
		{"delivery_state waiting_for_evidence", map[string]any{"delivery_state": "waiting_for_evidence"}, false},
	}
	for _, tc := range cases {
		if got := RuntimeBootstrapLeaseHasUsefulReceipt(tc.result); got != tc.want {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}

func TestRuntimeBootstrapLeaseHasDeclaredCoverage(t *testing.T) {
	cases := []struct {
		name   string
		result map[string]any
		want   bool
	}{
		{"vacio", map[string]any{}, false},
		{"nil", nil, false},
		{"mailbox_ids presente", map[string]any{"mailbox_ids": []any{int64(1)}}, true},
		{"mailbox_ids slice vacio", map[string]any{"mailbox_ids": []any{}}, false},
		{"start_order_id presente", map[string]any{"start_order_id": int64(42)}, true},
		{"start_order_id cero", map[string]any{"start_order_id": int64(0)}, false},
	}
	for _, tc := range cases {
		if got := RuntimeBootstrapLeaseHasDeclaredCoverage(tc.result); got != tc.want {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}

func TestRuntimeBootstrapLeaseStateBlocksMailbox(t *testing.T) {
	wfe := map[string]any{"lease_state": "waiting_for_evidence"}
	wfeWithReceipt := map[string]any{"lease_state": "waiting_for_evidence", "receipt_source": "git_worktree"}
	acked := map[string]any{"lease_state": "acked"}
	inheritedReceipt := map[string]any{"receipt_source": "git_worktree"}

	cases := []struct {
		name      string
		result    map[string]any
		inherited map[string]any
		want      bool
	}{
		{"wfe sin receipt bloquea", wfe, nil, true},
		{"wfe con receipt propio no bloquea", wfeWithReceipt, nil, false},
		{"wfe con receipt heredado no bloquea", wfe, inheritedReceipt, false},
		{"acked no bloquea", acked, nil, false},
		{"vacio no bloquea", map[string]any{}, nil, false},
		{"nil no bloquea", nil, nil, false},
	}
	for _, tc := range cases {
		if got := RuntimeBootstrapLeaseStateBlocksMailbox(tc.result, tc.inherited); got != tc.want {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}

func TestRuntimeOrderKeepsBootstrapLeasePending(t *testing.T) {
	cases := []struct {
		name   string
		result map[string]any
		want   bool
	}{
		{"wfe con mailbox_ids", map[string]any{"lease_state": "waiting_for_evidence", "mailbox_ids": []any{int64(1)}}, true},
		{"wfe con start_order_id", map[string]any{"lease_state": "waiting_for_evidence", "start_order_id": int64(7)}, true},
		{"delivered con cobertura", map[string]any{"lease_state": "delivered", "mailbox_ids": []any{int64(2)}}, true},
		{"acked con cobertura", map[string]any{"lease_state": "acked", "mailbox_ids": []any{int64(1)}}, false},
		{"wfe sin cobertura", map[string]any{"lease_state": "waiting_for_evidence"}, false},
		{"vacio", map[string]any{}, false},
		{"nil", nil, false},
	}
	for _, tc := range cases {
		if got := RuntimeOrderKeepsBootstrapLeasePending(tc.result); got != tc.want {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}
