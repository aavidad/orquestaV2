package orquestacionnucleoapp

import "testing"

func TestEvaluateAutoprogrammingReviewGateV0(t *testing.T) {
	requiredTest := "go test ./orquestacionnucleoapp -run TestEvaluateAutoprogrammingReviewGateV0 -count=1"

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
				{Path: "orquestacionnucleoapp/autoprogramming_review_gate_v0.go", LineCount: 300},
			},
			WriteSet: []string{"orquestacionnucleoapp/autoprogramming_review_gate_v0.go"},
		})

		if !result.Accepted {
			t.Fatalf("accepted=false issues=%+v", result.Issues)
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
		assertAutoprogrammingReviewGateIssueV0(t, result, "ack_missing")
	})

	t.Run("rechaza tests fallidos", func(t *testing.T) {
		result := EvaluateAutoprogrammingReviewGateV0(validAutoprogrammingReviewGateInputV0(requiredTest, func(input *AutoprogrammingReviewGateInputV0) {
			input.Tests[0].Passed = false
		}))

		if result.Accepted {
			t.Fatalf("accepted=true")
		}
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
		assertAutoprogrammingReviewGateIssueV0(t, result, "required_test_missing")
	})

	t.Run("rechaza ficheros sobre maximo de lineas", func(t *testing.T) {
		result := EvaluateAutoprogrammingReviewGateV0(validAutoprogrammingReviewGateInputV0(requiredTest, func(input *AutoprogrammingReviewGateInputV0) {
			input.Files[0].LineCount = 301
		}))

		if result.Accepted {
			t.Fatalf("accepted=true")
		}
		assertAutoprogrammingReviewGateIssueV0(t, result, "file_too_large")
	})

	t.Run("rechaza ficheros fuera del write-set", func(t *testing.T) {
		result := EvaluateAutoprogrammingReviewGateV0(validAutoprogrammingReviewGateInputV0(requiredTest, func(input *AutoprogrammingReviewGateInputV0) {
			input.Files[0].Path = "orquestacionnucleoapp/fuera.go"
		}))

		if result.Accepted {
			t.Fatalf("accepted=true")
		}
		assertAutoprogrammingReviewGateIssueV0(t, result, "file_outside_write_set")
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
			{Path: "orquestacionnucleoapp/autoprogramming_review_gate_v0.go", LineCount: 300},
		},
		WriteSet: []string{"orquestacionnucleoapp/autoprogramming_review_gate_v0.go"},
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
