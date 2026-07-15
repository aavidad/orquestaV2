package application

import "testing"

func TestRecoveryRefsRejectConcreteOrNonCanonicalNames(t *testing.T) {
	validBackup := "backup:sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if ref, err := NewBackupRef(validBackup); err != nil || ref.String() != validBackup {
		t.Fatalf("valid backup ref = %q err=%v", ref.String(), err)
	}
	for _, invalid := range []string{"", "backup:sha256:ABC", "file:/tmp/state.sqlite", "backup:sha256:" + validBackup} {
		if _, err := NewBackupRef(invalid); err == nil {
			t.Errorf("invalid backup ref accepted: %q", invalid)
		}
	}
	if ref, err := NewRecoveryTargetRef("recovery-target:offline-1"); err != nil || ref.String() == "" {
		t.Fatalf("valid target ref = %q err=%v", ref.String(), err)
	}
	for _, invalid := range []string{"", "../escape", "recovery-target:../escape", "recovery-target:/tmp/state"} {
		if _, err := NewRecoveryTargetRef(invalid); err == nil {
			t.Errorf("invalid target ref accepted: %q", invalid)
		}
	}
}
