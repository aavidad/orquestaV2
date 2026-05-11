package orquestaruntimecodexdelivery

import (
	"testing"
	"time"
)

func TestCodexBudgetActivityV0ActivoPasadoDeTiempoNoEsNoActivity(t *testing.T) {
	now := codexBudgetActivityNowForTestV0()
	got := ClassifyCodexBudgetActivityV0(CodexBudgetActivityInputV0{
		StartedAt:       now.Add(-40 * time.Minute),
		LastActivity:    now.Add(-1 * time.Minute),
		MaxExpected:     30 * time.Minute,
		NoActivityLimit: 5 * time.Minute,
		ObservedAt:      now,
	})

	if got.Status != CodexBudgetActivityOverBudgetButActiveV0 ||
		got.Reason != CodexBudgetActivityReasonOverBudgetButActiveV0 {
		t.Fatalf("classification=%+v", got)
	}
}

func TestCodexBudgetActivityV0SinActividadPasadoPresupuestoMarcaNoActivity(t *testing.T) {
	now := codexBudgetActivityNowForTestV0()
	got := ClassifyCodexBudgetActivityV0(CodexBudgetActivityInputV0{
		StartedAt:       now.Add(-40 * time.Minute),
		MaxExpected:     30 * time.Minute,
		NoActivityLimit: 5 * time.Minute,
		ObservedAt:      now,
	})

	if got.Status != CodexBudgetActivityOverBudgetNoActivityV0 ||
		got.Reason != CodexBudgetActivityReasonOverBudgetNoActivityV0 {
		t.Fatalf("classification=%+v", got)
	}
}

func TestCodexBudgetActivityV0DentroPresupuestoWorkingStalledPorActividad(t *testing.T) {
	now := codexBudgetActivityNowForTestV0()
	tests := []struct {
		name    string
		input   CodexBudgetActivityInputV0
		want    CodexBudgetActivityStatusV0
		wantWhy CodexBudgetActivityReasonV0
	}{
		{
			name: "working",
			input: CodexBudgetActivityInputV0{
				StartedAt:       now.Add(-10 * time.Minute),
				LastActivity:    now.Add(-1 * time.Minute),
				MaxExpected:     30 * time.Minute,
				NoActivityLimit: 5 * time.Minute,
				ObservedAt:      now,
			},
			want:    CodexBudgetActivityWorkingV0,
			wantWhy: CodexBudgetActivityReasonWorkingV0,
		},
		{
			name: "stalled",
			input: CodexBudgetActivityInputV0{
				StartedAt:       now.Add(-10 * time.Minute),
				LastActivity:    now.Add(-7 * time.Minute),
				MaxExpected:     30 * time.Minute,
				NoActivityLimit: 5 * time.Minute,
				ObservedAt:      now,
			},
			want:    CodexBudgetActivityStalledV0,
			wantWhy: CodexBudgetActivityReasonStalledV0,
		},
		{
			name: "last_ack_counts_as_activity",
			input: CodexBudgetActivityInputV0{
				StartedAt:       now.Add(-10 * time.Minute),
				LastAck:         now.Add(-1 * time.Minute),
				MaxExpected:     30 * time.Minute,
				NoActivityLimit: 5 * time.Minute,
				ObservedAt:      now,
			},
			want:    CodexBudgetActivityWorkingV0,
			wantWhy: CodexBudgetActivityReasonWorkingV0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyCodexBudgetActivityV0(tt.input)
			if got.Status != tt.want || got.Reason != tt.wantWhy {
				t.Fatalf("classification=%+v want=%s/%s", got, tt.want, tt.wantWhy)
			}
		})
	}
}

func codexBudgetActivityNowForTestV0() time.Time {
	return time.Date(2026, 5, 11, 10, 0, 0, 0, time.UTC)
}
