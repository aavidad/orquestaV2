package orquestaopesdirector

import "testing"

func TestValidateOPESTopicQualityContractV0BloqueaNivelBPorDebajoMinimoV0(t *testing.T) {
	result := ValidateOPESTopicQualityContractV0(OPESTopicQualityContractRequestV0{
		TopicRef:      "tema-012",
		Level:         "b",
		WordCountText: "El texto ampliado limpio alcanza 9.301 palabras, supera el minimo de nivel B de 10.800 palabras.",
		EvidenceRefs:  []string{"evidence-ref-opes-topic-word-count"},
	})

	if result.Status != OPESTopicQualityStatusNeedsReworkV0 ||
		result.WordCount != 9301 ||
		result.MinWords != OPESTopicQualityMinWordsLevelBV0 ||
		!opesTopicQualityIssueCodeInSetV0(result.Issues, ErrOPESTopicQualityMinWordsNotMetV0) {
		t.Fatalf("result=%+v", result)
	}
}

func TestValidateOPESTopicQualityContractV0DetectaMetacomentariosPublicosV0(t *testing.T) {
	result := ValidateOPESTopicQualityContractV0(OPESTopicQualityContractRequestV0{
		TopicRef:  "tema-018",
		Level:     OPESTopicQualityLevelBV0,
		WordCount: 11407,
		Text:      "En examen conviene recordar este apartado. Para este tema basta con conocer la estructura.",
	})

	if result.Status != OPESTopicQualityStatusNeedsReworkV0 ||
		!opesTopicQualityIssueCodeInSetV0(result.Issues, ErrOPESTopicQualityPublicMetacommentV0) {
		t.Fatalf("result=%+v", result)
	}
}

func TestValidateOPESTopicQualityContractV0NoAceptaRasterDecorativoComoDidacticoV0(t *testing.T) {
	result := ValidateOPESTopicQualityContractV0(OPESTopicQualityContractRequestV0{
		TopicRef:              "tema-010",
		Level:                 OPESTopicQualityLevelBV0,
		WordCount:             11000,
		RequireDidacticVisual: true,
		Visuals: []OPESTopicQualityVisualEvidenceV0{{
			VisualRef: "visual-ref-reunion-generica",
			Raster:    true,
		}},
	})

	if result.Status != OPESTopicQualityStatusNeedsReworkV0 ||
		!opesTopicQualityIssueCodeInSetV0(result.Issues, ErrOPESTopicQualityDidacticVisualRequiredV0) {
		t.Fatalf("result=%+v", result)
	}
}

func TestValidateOPESTopicQualityContractV0CompletaConMinimosYVisualDidacticoV0(t *testing.T) {
	result := ValidateOPESTopicQualityContractV0(OPESTopicQualityContractRequestV0{
		TopicRef:              "tema-010",
		Level:                 OPESTopicQualityLevelBV0,
		CanonicalWordCount:    11000,
		Text:                  "Contenido didactico publico sin notas internas.",
		RequireDidacticVisual: true,
		Visuals: []OPESTopicQualityVisualEvidenceV0{{
			VisualRef:        "visual-ref-estructura-normativa",
			Raster:           true,
			AnchorRef:        "apartado-ref-transversalidad",
			DidacticFunction: "mapa de obligaciones, planes y conceptos evaluables",
			EvidenceRefs:     []string{"evidence-ref-visual-didactic"},
		}},
	})

	if result.Status != OPESTopicQualityStatusCompleteV0 ||
		len(result.Issues) != 0 ||
		result.WordCount != 11000 ||
		len(result.Visuals) != 1 ||
		result.Visuals[0].AnchorRef == "" {
		t.Fatalf("result=%+v", result)
	}
}

func TestValidateOPESTopicQualityContractV0PriorizaContadorCanonicoSobreWCV0(t *testing.T) {
	result := ValidateOPESTopicQualityContractV0(OPESTopicQualityContractRequestV0{
		TopicRef:           "tema-038",
		Level:              OPESTopicQualityLevelBV0,
		WordCount:          10800,
		WordCountSource:    "wc -w",
		CanonicalWordCount: 10336,
	})

	if result.Status != OPESTopicQualityStatusNeedsReworkV0 ||
		result.WordCount != 10336 ||
		result.WordCountSource != "canonical" ||
		!opesTopicQualityIssueCodeInSetV0(result.Issues, ErrOPESTopicQualityMinWordsNotMetV0) {
		t.Fatalf("result=%+v", result)
	}
}

func TestValidateOPESTopicQualityContractV0NoAceptaWCSinCanonicoV0(t *testing.T) {
	result := ValidateOPESTopicQualityContractV0(OPESTopicQualityContractRequestV0{
		TopicRef:        "tema-041",
		Level:           OPESTopicQualityLevelBV0,
		WordCount:       10983,
		WordCountSource: "wc",
	})

	if result.Status != OPESTopicQualityStatusNeedsReworkV0 ||
		result.WordCount != 10983 ||
		!opesTopicQualityIssueCodeInSetV0(result.Issues, ErrOPESTopicQualityCanonicalWordsRequiredV0) {
		t.Fatalf("result=%+v", result)
	}
}

func TestValidateOPESTopicQualityContractV0DetectaInformeContradictorioV0(t *testing.T) {
	result := ValidateOPESTopicQualityContractV0(OPESTopicQualityContractRequestV0{
		TopicRef:           "tema-041",
		Level:              OPESTopicQualityLevelBV0,
		CanonicalWordCount: 10531,
		ReportStatus:       "fail",
		ReportText:         "El informe indica que supera 10.800 palabras por conteo interno.",
	})

	if result.Status != OPESTopicQualityStatusNeedsReworkV0 ||
		!opesTopicQualityIssueCodeInSetV0(result.Issues, ErrOPESTopicQualityInvalidReportContractV0) ||
		!opesTopicQualityIssueCodeInSetV0(result.Issues, ErrOPESTopicQualityMinWordsNotMetV0) {
		t.Fatalf("result=%+v", result)
	}
}

func opesTopicQualityIssueCodeInSetV0(issues []OPESTopicQualityIssueV0, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
