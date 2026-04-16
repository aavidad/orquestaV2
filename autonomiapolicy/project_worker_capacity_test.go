package autonomiapolicy

import "testing"

func TestProjectAdmitsWorkerAutonomy(t *testing.T) {
	t.Run("project_disabled_allows", func(t *testing.T) {
		if !ProjectAdmitsWorkerAutonomy(false, 1, 100, false, false) {
			t.Fatalf("disabled project should allow by default")
		}
	})

	t.Run("project_full_blocks_without_self_assignment", func(t *testing.T) {
		if ProjectAdmitsWorkerAutonomy(true, 1, 1, false, false) {
			t.Fatalf("full project should block another worker")
		}
	})

	t.Run("self_active_assignment_is_not_double_counted", func(t *testing.T) {
		if !ProjectAdmitsWorkerAutonomy(true, 1, 1, true, false) {
			t.Fatalf("self active assignment should not double count against the quota")
		}
	})

	t.Run("project_full_with_other_workers_still_blocks_even_if_self_active", func(t *testing.T) {
		if ProjectAdmitsWorkerAutonomy(true, 1, 2, true, false) {
			t.Fatalf("project full with other workers should still block")
		}
	})

	t.Run("reserved_self_assignment_keeps_current_count", func(t *testing.T) {
		if ProjectAdmitsWorkerAutonomy(true, 1, 1, true, true) {
			t.Fatalf("reserved self assignment should not bypass the quota")
		}
	})
}
