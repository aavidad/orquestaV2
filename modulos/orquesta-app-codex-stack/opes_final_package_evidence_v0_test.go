package orquestaappcodexstack

import (
	"encoding/json"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
)

func TestCodexStackOPESFinalPackageEvidenceIssueRefV0BloqueaSinTutorV0(t *testing.T) {
	fields := opesFinalPackageManifestFieldsForTestV0(opesFinalPackageManifestOptionsForTestV0{
		OmitTutorEvidence: true,
	})

	if got := codexStackOPESFinalPackageEvidenceIssueRefV0(fields, nil, nil, nil); got != codexStackOPESFinalPackageEvidenceIncompleteIssueV0 {
		t.Fatalf("issue=%q want %q", got, codexStackOPESFinalPackageEvidenceIncompleteIssueV0)
	}
}

func TestCodexStackOPESFinalPackageEvidenceIssueRefV0BloqueaQABancoYTutorV0(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts opesFinalPackageManifestOptionsForTestV0
		want string
	}{
		{
			name: "question-bank-pass",
			opts: opesFinalPackageManifestOptionsForTestV0{OmitQuestionBankPass: true},
			want: codexStackOPESFinalPackageQuestionBankQAMissingIssueV0,
		},
		{
			name: "question-bank-report",
			opts: opesFinalPackageManifestOptionsForTestV0{OmitQuestionBankReport: true},
			want: codexStackOPESFinalPackageQuestionBankQAMissingIssueV0,
		},
		{
			name: "tutor-pass",
			opts: opesFinalPackageManifestOptionsForTestV0{OmitTutorPass: true},
			want: codexStackOPESFinalPackageTutorAssetsQAMissingIssueV0,
		},
		{
			name: "tutor-report",
			opts: opesFinalPackageManifestOptionsForTestV0{OmitTutorReport: true},
			want: codexStackOPESFinalPackageTutorAssetsQAMissingIssueV0,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fields := opesFinalPackageManifestFieldsForTestV0(tc.opts)

			if got := codexStackOPESFinalPackageEvidenceIssueRefV0(fields, nil, nil, nil); got != tc.want {
				t.Fatalf("issue=%q want %q", got, tc.want)
			}
		})
	}
}

func TestCodexStackOPESFinalPackageEvidenceIssueRefV0AceptaTutorBancoYQAEstrictaV0(t *testing.T) {
	fields := opesFinalPackageManifestFieldsForTestV0(opesFinalPackageManifestOptionsForTestV0{})

	if got := codexStackOPESFinalPackageEvidenceIssueRefV0(fields, nil, nil, nil); got != "" {
		t.Fatalf("issue=%q want empty", got)
	}
}

type opesFinalPackageManifestOptionsForTestV0 struct {
	OmitTutorEvidence      bool
	OmitQuestionBankPass   bool
	OmitQuestionBankReport bool
	OmitTutorPass          bool
	OmitTutorReport        bool
}

func opesFinalPackageManifestFieldsForTestV0(
	opts opesFinalPackageManifestOptionsForTestV0,
) []orquestadomainwork.DomainWorkFieldV0 {
	required := map[string]any{
		"html":   "opes-final-evidence:html:test",
		"rag":    "opes-final-evidence:rag:test",
		"audio":  "opes-final-evidence:audio:test",
		"tests":  "opes-final-evidence:tests:test",
		"tutor":  "opes-final-evidence:tutor:test",
		"visual": "opes-final-evidence:visual:test",
		"qa":     "opes-final-evidence:qa:test",
	}
	if opts.OmitTutorEvidence {
		delete(required, "tutor")
	}
	qaPasses := map[string]any{
		"extension_pass":           true,
		"official_text_qa_pass":    true,
		"strict_editorial_qa_pass": true,
		"question_bank_publicable": true,
		"tutor_assets_publicable":  true,
	}
	if opts.OmitQuestionBankPass {
		delete(qaPasses, "question_bank_publicable")
	}
	if opts.OmitTutorPass {
		delete(qaPasses, "tutor_assets_publicable")
	}
	qaReports := map[string]any{
		"extension":                "09_validacion/informe_extension_temario.json",
		"official_text":            "09_validacion/informe_texto_publico_sin_notas_autor.json",
		"strict_editorial":         "09_validacion/informe_texto_publico_sin_andamiaje_interno.json",
		"question_bank_publicable": "09_validacion/informe_question_bank_publicable.json",
		"tutor_assets_publicable":  "09_validacion/informe_tutor_assets_publicable.json",
	}
	if opts.OmitQuestionBankReport {
		delete(qaReports, "question_bank_publicable")
	}
	if opts.OmitTutorReport {
		delete(qaReports, "tutor_assets_publicable")
	}
	raw, _ := json.Marshal(map[string]any{
		"schema_version":                     "opes_final_package_evidence_manifest.v0",
		"package_ref":                        "package-ref-opes-final-test",
		"manifest_ref":                       "manifest-cierre-ref-opes-final-test",
		"checksum_refs":                      []string{"checksum-ref-opes-final-test"},
		"validation_report_ref":              "validation-report-ref-opes-final-test",
		"review_matrix_ref":                  "review-matrix-ref-opes-final-test",
		"required_evidence_refs":             required,
		"qa_passes":                          qaPasses,
		"qa_report_refs":                     qaReports,
		"topic_quality_contract_result_refs": map[string]string{"tema_001": "topic-quality-contract-result-ref-test"},
	})
	return []orquestadomainwork.DomainWorkFieldV0{{
		Name:      "manifest_cierre",
		ValueJSON: raw,
	}}
}
