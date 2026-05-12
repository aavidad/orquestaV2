package orquestadirectortickinput

import (
	"reflect"
	"strings"
	"testing"
)

func TestProjectionRefsV0ExtraeRefsCanonicas(t *testing.T) {
	cases := map[string]struct {
		got  []string
		want []string
	}{
		"capacity": {
			got:  schedulerCapacityDecisionRefsV0([]string{" capacity-ref-001#capacity_decision:decision-ref-001 "}),
			want: []string{"capacity-ref-001"},
		},
		"gate": {
			got:  schedulerConcurrencyGateRefsV0([]string{"gate-ref-001#decision:allow_request_agent#plan:plan-ref-001"}),
			want: []string{"gate-ref-001"},
		},
		"lease": {
			got:  schedulerExpiredLeaseRefsV0([]string{"lease-ref-001#agent:agent-ref-001#action:stop_agent"}),
			want: []string{"lease-ref-001"},
		},
		"replan": {
			got:  schedulerReplanRefsV0([]string{"replan-ref-001#source:agent-ref-001#task:task-ref-001"}),
			want: []string{"replan-ref-001"},
		},
	}
	for name, tc := range cases {
		if !reflect.DeepEqual(tc.got, tc.want) {
			t.Fatalf("%s got=%v want=%v", name, tc.got, tc.want)
		}
	}
}

func TestProjectionRefsV0CompactaDuplicados(t *testing.T) {
	got := schedulerCapacityDecisionRefsV0([]string{
		"capacity-ref-001#capacity_decision:decision-ref-001",
		"capacity-ref-001",
		" ",
	})
	if !reflect.DeepEqual(got, []string{"capacity-ref-001"}) {
		t.Fatalf("got=%v", got)
	}
}

func TestProjectionRefsV0AcotaRefsLargasPersistidas(t *testing.T) {
	got := schedulerConcurrencyGateRefsV0([]string{
		"concurrency_gate:" + strings.Repeat("claim-ref-largo-", 80) + "#decision:allow_request_agent",
	})
	if len(got) != 1 || len(got[0]) > maxTickInputStringV0 {
		t.Fatalf("projection larga no acotada: %v", got)
	}
}
