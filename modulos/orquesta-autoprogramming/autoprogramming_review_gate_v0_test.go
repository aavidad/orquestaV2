package orquestaautoprogramming

import "testing"

func TestEvaluateAutoprogrammingReviewGateV0(t *testing.T) {
	requiredTest := "go test ./modulos/orquesta-autoprogramming -run TestEvaluateAutoprogrammingReviewGateV0 -count=1"

	t.Run("acepta ack completed tests verdes y ficheros pequenos", func(t *testing.T) {
		result := EvaluateAutoprogrammingReviewGateV0(AutoprogrammingReviewGateInputV0{
			ACK: AutoprogrammingReviewGateACKV0{
				Present: true,
				Status:  AutoprogrammingReviewGateACKCompletedV0,
			},
			RequiredTests: []string{requiredTest},
			Tests: []AutoprogrammingReviewGateTestV0{
				{Command: requiredTest, Passed: true},
			},
			Files: []AutoprogrammingReviewGateFileV0{
				{Path: "modulos/orquesta-autoprogramming/autoprogramming_review_gate_v0.go", LineCount: 300},
			},
			WriteSet: []string{"modulos/orquesta-autoprogramming/autoprogramming_review_gate_v0.go"},
		})

		if !result.Accepted {
			t.Fatalf("accepted=false issues=%+v", result.Issues)
		}
		if !result.PreserveOutput || result.RequiresFollowup {
			t.Fatalf("preserve_output=%v requires_followup=%v", result.PreserveOutput, result.RequiresFollowup)
		}
		if result.RecommendedAction != AutoprogrammingReviewGateActionAcceptV0 {
			t.Fatalf("recommended_action=%q", result.RecommendedAction)
		}
		if len(result.Issues) != 0 {
			t.Fatalf("issues=%+v", result.Issues)
		}
	})

	t.Run("rechaza entrega sin ack", func(t *testing.T) {
		result := EvaluateAutoprogrammingReviewGateV0(validAutoprogrammingReviewGateInputV0(requiredTest, func(input *AutoprogrammingReviewGateInputV0) {
			input.ACK = AutoprogrammingReviewGateACKV0{}
		}))

		if result.Accepted {
			t.Fatalf("accepted=true")
		}
		assertAutoprogrammingReviewGateBlockingV0(t, result)
		assertAutoprogrammingReviewGateIssueV0(t, result, "ack_missing")
	})

	t.Run("rechaza tests fallidos", func(t *testing.T) {
		result := EvaluateAutoprogrammingReviewGateV0(validAutoprogrammingReviewGateInputV0(requiredTest, func(input *AutoprogrammingReviewGateInputV0) {
			input.Tests[0].Passed = false
		}))

		if result.Accepted {
			t.Fatalf("accepted=true")
		}
		assertAutoprogrammingReviewGateBlockingV0(t, result)
		assertAutoprogrammingReviewGateIssueV0(t, result, "required_test_failed")
		assertAutoprogrammingReviewGateIssueV0(t, result, "test_failed")
	})

	t.Run("rechaza test obligatorio ausente", func(t *testing.T) {
		result := EvaluateAutoprogrammingReviewGateV0(validAutoprogrammingReviewGateInputV0(requiredTest, func(input *AutoprogrammingReviewGateInputV0) {
			input.Tests = nil
		}))

		if result.Accepted {
			t.Fatalf("accepted=true")
		}
		assertAutoprogrammingReviewGateBlockingV0(t, result)
		assertAutoprogrammingReviewGateIssueV0(t, result, "required_test_missing")
	})

	t.Run("acepta ficheros sobre maximo como aviso no bloqueante", func(t *testing.T) {
		result := EvaluateAutoprogrammingReviewGateV0(validAutoprogrammingReviewGateInputV0(requiredTest, func(input *AutoprogrammingReviewGateInputV0) {
			input.Files[0].LineCount = 301
		}))

		if !result.Accepted {
			t.Fatalf("accepted=false issues=%+v", result.Issues)
		}
		if !result.PreserveOutput || !result.RequiresFollowup {
			t.Fatalf("preserve_output=%v requires_followup=%v", result.PreserveOutput, result.RequiresFollowup)
		}
		assertAutoprogrammingReviewGateFollowupV0(t, result)
		assertAutoprogrammingReviewGateIssueV0(t, result, "file_too_large")
	})

	t.Run("acepta ficheros fuera del write-set como aviso no bloqueante", func(t *testing.T) {
		result := EvaluateAutoprogrammingReviewGateV0(validAutoprogrammingReviewGateInputV0(requiredTest, func(input *AutoprogrammingReviewGateInputV0) {
			input.Files[0].Path = "modulos/orquesta-autoprogramming/fuera.go"
		}))

		if !result.Accepted {
			t.Fatalf("accepted=false issues=%+v", result.Issues)
		}
		if !result.PreserveOutput || !result.RequiresFollowup {
			t.Fatalf("preserve_output=%v requires_followup=%v", result.PreserveOutput, result.RequiresFollowup)
		}
		assertAutoprogrammingReviewGateFollowupV0(t, result)
		assertAutoprogrammingReviewGateIssueV0(t, result, "file_outside_write_set")
	})

	t.Run("no marca ruta hija del write-set como fuera de alcance", func(t *testing.T) {
		result := EvaluateAutoprogrammingReviewGateV0(validAutoprogrammingReviewGateInputV0(requiredTest, func(input *AutoprogrammingReviewGateInputV0) {
			input.WriteSet = []string{"modulos/orquesta-autoprogramming"}
			input.Files[0].Path = "modulos/orquesta-autoprogramming/review_gate.go"
		}))

		if !result.Accepted || result.RequiresFollowup {
			t.Fatalf("result=%+v", result)
		}
		assertAutoprogrammingReviewGateNoIssueV0(t, result, "file_outside_write_set")
	})

	t.Run("no marca glob compatible del write-set como fuera de alcance", func(t *testing.T) {
		result := EvaluateAutoprogrammingReviewGateV0(validAutoprogrammingReviewGateInputV0(requiredTest, func(input *AutoprogrammingReviewGateInputV0) {
			input.WriteSet = []string{"modulos/orquesta-autoprogramming/*.go"}
			input.Files[0].Path = "modulos/orquesta-autoprogramming/review_gate.go"
		}))

		if !result.Accepted || result.RequiresFollowup {
			t.Fatalf("result=%+v", result)
		}
		assertAutoprogrammingReviewGateNoIssueV0(t, result, "file_outside_write_set")
	})
}

func validAutoprogrammingReviewGateInputV0(
	requiredTest string,
	mutate func(*AutoprogrammingReviewGateInputV0),
) AutoprogrammingReviewGateInputV0 {
	input := AutoprogrammingReviewGateInputV0{
		ACK: AutoprogrammingReviewGateACKV0{
			Present: true,
			Status:  AutoprogrammingReviewGateACKCompletedV0,
		},
		RequiredTests: []string{requiredTest},
		Tests: []AutoprogrammingReviewGateTestV0{
			{Command: requiredTest, Passed: true},
		},
		Files: []AutoprogrammingReviewGateFileV0{
			{Path: "modulos/orquesta-autoprogramming/autoprogramming_review_gate_v0.go", LineCount: 300},
		},
		WriteSet: []string{"modulos/orquesta-autoprogramming/autoprogramming_review_gate_v0.go"},
	}
	if mutate != nil {
		mutate(&input)
	}
	return input
}

func assertAutoprogrammingReviewGateIssueV0(
	t *testing.T,
	result AutoprogrammingReviewGateResultV0,
	code string,
) {
	t.Helper()
	for _, issue := range result.Issues {
		if issue.Code == code {
			return
		}
	}
	t.Fatalf("issue %q no encontrado: %+v", code, result.Issues)
}

func assertAutoprogrammingReviewGateNoIssueV0(
	t *testing.T,
	result AutoprogrammingReviewGateResultV0,
	code string,
) {
	t.Helper()
	for _, issue := range result.Issues {
		if issue.Code == code {
			t.Fatalf("issue %q inesperado: %+v", code, result.Issues)
		}
	}
}

func assertAutoprogrammingReviewGateBlockingV0(
	t *testing.T,
	result AutoprogrammingReviewGateResultV0,
) {
	t.Helper()
	if result.PreserveOutput || result.RequiresFollowup {
		t.Fatalf("preserve_output=%v requires_followup=%v", result.PreserveOutput, result.RequiresFollowup)
	}
	if result.RecommendedAction != AutoprogrammingReviewGateActionBlockClosureV0 {
		t.Fatalf("recommended_action=%q", result.RecommendedAction)
	}
}

func assertAutoprogrammingReviewGateFollowupV0(
	t *testing.T,
	result AutoprogrammingReviewGateResultV0,
) {
	t.Helper()
	if !result.Accepted || !result.PreserveOutput || !result.RequiresFollowup {
		t.Fatalf("soft rail result=%+v", result)
	}
	if result.RecommendedAction != AutoprogrammingReviewGateActionRequestFollowupReviewV0 {
		t.Fatalf("recommended_action=%q", result.RecommendedAction)
	}
}
