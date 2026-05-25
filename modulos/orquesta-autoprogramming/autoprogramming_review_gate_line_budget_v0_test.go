package orquestaautoprogramming

import "testing"

func TestAutoprogrammingReviewGateStrictGoLineBudgetV0(t *testing.T) {
	requiredTest := "go test ./modulos/orquesta-autoprogramming -run TestStrictGoLineBudget -count=1"

	t.Run("bloquea fichero nuevo sobre limite sin fuente real", func(t *testing.T) {
		result := EvaluateAutoprogrammingReviewGateV0(validAutoprogrammingReviewGateInputV0(requiredTest, func(input *AutoprogrammingReviewGateInputV0) {
			input.StrictGoLineBudget = true
			input.Files[0].LineCount = 301
		}))

		assertAutoprogrammingReviewGateBlockingV0(t, result)
		assertAutoprogrammingReviewGateIssueV0(t, result, AutoprogrammingReviewGateStrictLineBudgetIssueV0)
	})

	t.Run("bloquea crecimiento sobre baseline historico", func(t *testing.T) {
		result := EvaluateAutoprogrammingReviewGateV0(validAutoprogrammingReviewGateInputV0(requiredTest, func(input *AutoprogrammingReviewGateInputV0) {
			input.StrictGoLineBudget = true
			input.Files[0].LineCount = 321
			input.Files[0].BaselineLineCount = 320
			input.Files[0].LineCountSource = "snapshot"
		}))

		assertAutoprogrammingReviewGateBlockingV0(t, result)
		assertAutoprogrammingReviewGateIssueV0(t, result, AutoprogrammingReviewGateStrictLineBudgetIssueV0)
	})

	t.Run("acepta deuda historica que no crece", func(t *testing.T) {
		result := EvaluateAutoprogrammingReviewGateV0(validAutoprogrammingReviewGateInputV0(requiredTest, func(input *AutoprogrammingReviewGateInputV0) {
			input.StrictGoLineBudget = true
			input.Files[0].LineCount = 320
			input.Files[0].BaselineLineCount = 320
			input.Files[0].LineCountSource = "worktree"
		}))

		if !result.Accepted || result.RequiresFollowup {
			t.Fatalf("result=%+v", result)
		}
	})

	t.Run("followup de particion aceptado desbloquea cierre", func(t *testing.T) {
		result := EvaluateAutoprogrammingReviewGateV0(validAutoprogrammingReviewGateInputV0(requiredTest, func(input *AutoprogrammingReviewGateInputV0) {
			input.StrictGoLineBudget = true
			input.Files[0].LineCount = 321
			input.Files[0].BaselineLineCount = 320
			input.Files[0].LineCountSource = "snapshot"
			input.AcceptedPartitionFollowups = []string{input.Files[0].Path}
		}))

		if !result.Accepted || result.RequiresFollowup {
			t.Fatalf("result=%+v", result)
		}
	})

	t.Run("fuente real sin modo estricto conserva advisory legacy", func(t *testing.T) {
		result := EvaluateAutoprogrammingReviewGateV0(validAutoprogrammingReviewGateInputV0(requiredTest, func(input *AutoprogrammingReviewGateInputV0) {
			input.Files[0].LineCount = 321
			input.Files[0].BaselineLineCount = 320
			input.Files[0].LineCountSource = "project_file"
		}))

		assertAutoprogrammingReviewGateFollowupV0(t, result)
		assertAutoprogrammingReviewGateIssueV0(t, result, "file_too_large")
	})
}
