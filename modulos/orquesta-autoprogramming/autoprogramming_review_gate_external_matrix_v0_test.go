package orquestaautoprogramming

import (
	"fmt"
	"testing"
)

func TestAutoprogrammingReviewGateExternalMatrixV0(t *testing.T) {
	requiredTest := "go test ./modulos/orquesta-autoprogramming -run TestAutoprogrammingReviewGateExternalMatrixV0 -count=1"

	for lines := 1; lines <= 520; lines++ {
		t.Run(fmt.Sprintf("file-lines-%03d", lines), func(t *testing.T) {
			result := EvaluateAutoprogrammingReviewGateV0(validAutoprogrammingReviewGateInputV0(requiredTest, func(input *AutoprogrammingReviewGateInputV0) {
				input.Files[0].LineCount = lines
			}))
			if !result.Accepted {
				t.Fatalf("lineas=%d accepted=false issues=%+v", lines, result.Issues)
			}
			if lines > AutoprogrammingReviewGateDefaultMaxLinesV0 {
				assertAutoprogrammingReviewGateIssueV0(t, result, "file_too_large")
				assertAutoprogrammingReviewGateFollowupV0(t, result)
			}
		})
	}

	for index := 0; index < 260; index++ {
		path := fmt.Sprintf("modulos/orquesta-autoprogramming/extra-%03d.go", index)
		t.Run(fmt.Sprintf("outside-write-set-%03d", index), func(t *testing.T) {
			result := EvaluateAutoprogrammingReviewGateV0(validAutoprogrammingReviewGateInputV0(requiredTest, func(input *AutoprogrammingReviewGateInputV0) {
				input.Files[0].Path = path
			}))
			if !result.Accepted {
				t.Fatalf("path=%q accepted=false issues=%+v", path, result.Issues)
			}
			assertAutoprogrammingReviewGateIssueV0(t, result, "file_outside_write_set")
			assertAutoprogrammingReviewGateFollowupV0(t, result)
		})
	}

	for index := 0; index < 260; index++ {
		code := fmt.Sprintf("write_set_target_missing:target-%03d", index)
		t.Run(fmt.Sprintf("missing-write-set-target-%03d", index), func(t *testing.T) {
			result := AutoprogrammingReviewGateResultFromIssuesV0([]AutoprogrammingReviewGateIssueV0{{
				Code:  code,
				Field: "write_set",
			}})
			if !result.Accepted {
				t.Fatalf("code=%q accepted=false issues=%+v", code, result.Issues)
			}
			assertAutoprogrammingReviewGateIssueV0(t, result, code)
			assertAutoprogrammingReviewGateFollowupV0(t, result)
		})
	}
}
