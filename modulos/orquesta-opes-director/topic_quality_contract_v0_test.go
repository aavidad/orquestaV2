package orquestaopesdirector

import (
	"strings"
	"testing"
)

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

func TestValidateOPESTopicQualityContractV0DetectaMojibakePreAudioV0(t *testing.T) {
	result := ValidateOPESTopicQualityContractV0(OPESTopicQualityContractRequestV0{
		TopicRef:           "tema-integrador-social-001",
		Level:              OPESTopicQualityLevelBV0,
		CanonicalWordCount: 11200,
		Text: strings.Join([]string{
			"El alumnado debe comprender la designaci\u00c3\u00b3n del apoyo.",
			"El apartado p\u00c3\u00bablico contiene m\u00c3\u00a1s ejemplos y una marca \u00c2 visible.",
			"Un guion de audio no puede llegar a TTS con el caracter \ufffd ni con comillas \u00e2\u20ac.",
		}, "\n"),
	})

	if result.Status != OPESTopicQualityStatusNeedsReworkV0 ||
		!opesTopicQualityIssueCodeInSetV0(result.Issues, ErrOPESTopicQualityPublicMojibakeV0) {
		t.Fatalf("result=%+v", result)
	}
}

func TestValidateOPESTopicQualityContractV0DetectaAndamiajeInternoConTildesYMayusculasV0(t *testing.T) {
	result := ValidateOPESTopicQualityContractV0(OPESTopicQualityContractRequestV0{
		TopicRef:           "tema-029",
		Level:              OPESTopicQualityLevelBV0,
		CanonicalWordCount: 11025,
		Text: strings.Join([]string{
			"Preguntas De Recuperaci\u00f3n",
			"Repaso Espaciado",
			"D\u00eda 0: reconstruye sin mirar el mapa mental.",
			"Una respuesta fuerte empieza por delimitar el concepto.",
		}, "\n"),
	})

	if result.Status != OPESTopicQualityStatusNeedsReworkV0 ||
		!opesTopicQualityIssueCodeInSetV0(result.Issues, ErrOPESTopicQualityStudyScaffoldingV0) {
		t.Fatalf("result=%+v", result)
	}
}

func TestValidateOPESTopicQualityContractV0DetectaContaminacionEstructuralV0(t *testing.T) {
	result := ValidateOPESTopicQualityContractV0(OPESTopicQualityContractRequestV0{
		TopicRef:           "tema-046",
		Level:              OPESTopicQualityLevelBV0,
		CanonicalWordCount: 11120,
		Text: strings.Join([]string{
			"El apartado incorpora un bloque ajeno procedente del tema_017.",
			"Se conserva una referencia interna a canon maestro y derivaciones futuras.",
			"El texto visible apunta a 02_temas/tema_014/02_markdown/tema_ampliado.md y a una figura .svg.",
		}, "\n"),
	})

	if result.Status != OPESTopicQualityStatusNeedsReworkV0 ||
		!opesTopicQualityIssueCodeInSetV0(result.Issues, ErrOPESTopicQualityStructuralContaminationV0) {
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
