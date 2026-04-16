package autonomiapolicy

import "testing"

func TestSupervisorRoleScore(t *testing.T) {
	if got := SupervisorRoleScore("supervisor"); got != 0 {
		t.Fatalf("unexpected score for supervisor: %d", got)
	}
	if got := SupervisorRoleScore("Programador"); got != 2 {
		t.Fatalf("unexpected score for programador: %d", got)
	}
	if got := SupervisorRoleScore("desconocido"); got != 4 {
		t.Fatalf("unexpected default score: %d", got)
	}
}
