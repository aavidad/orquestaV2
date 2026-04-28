package db

import (
	"os"
	"path/filepath"
	"testing"
)

func openTestScoreDB(t *testing.T) {
	t.Helper()
	Close()
	path := filepath.Join(t.TempDir(), "orquesta.db")
	prev := os.Getenv("ORQUESTA_DB")
	if err := os.Setenv("ORQUESTA_DB", path); err != nil {
		t.Fatalf("setenv ORQUESTA_DB: %v", err)
	}
	t.Cleanup(func() {
		if prev == "" {
			_ = os.Unsetenv("ORQUESTA_DB")
		} else {
			_ = os.Setenv("ORQUESTA_DB", prev)
		}
		Close()
	})
	if err := Open(); err != nil {
		t.Fatalf("open: %v", err)
	}
}

func TestGarantizarScoresLocalesAgenteInicializaMateriasNeutrales(t *testing.T) {
	openTestScoreDB(t)

	if err := GarantizarScoresLocalesAgente("Gemma1", ""); err != nil {
		t.Fatalf("GarantizarScoresLocalesAgente: %v", err)
	}

	agente := "Gemma1"
	items, err := ListarAgenteScoresLocales(&agente, nil)
	if err != nil {
		t.Fatalf("ListarAgenteScoresLocales: %v", err)
	}
	if len(items) != len(MateriasScoreAgenteCanonicas()) {
		t.Fatalf("numero de materias inesperado: got=%d want=%d", len(items), len(MateriasScoreAgenteCanonicas()))
	}
	for _, item := range items {
		if item == nil {
			t.Fatalf("score nil")
		}
		if item.ScoreBase != agenteScoreNeutral || item.ScoreTotal != agenteScoreNeutral {
			t.Fatalf("score neutral inesperado para %s: %+v", item.Materia, item)
		}
		if item.Muestras != 0 || item.Benchmarks != 0 || item.Exitos != 0 {
			t.Fatalf("contadores iniciales inesperados para %s: %+v", item.Materia, item)
		}
	}
}

func TestRegistrarObservacionScoreAgenteLocalEsMutable(t *testing.T) {
	openTestScoreDB(t)

	const agente = "Gemma1"
	const materia = "codigo"

	for i := 0; i < 4; i++ {
		if _, err := RegistrarObservacionScoreAgenteLocal(agente, "", materia, 1.0, false, "entrega floja"); err != nil {
			t.Fatalf("RegistrarObservacionScoreAgenteLocal mala #%d: %v", i+1, err)
		}
	}
	item, err := GetAgenteScoreLocal(agente, "", materia)
	if err != nil {
		t.Fatalf("GetAgenteScoreLocal tras malas: %v", err)
	}
	if item == nil || item.ScoreTotal >= 5.0 {
		t.Fatalf("el score deberia caer por debajo de neutral tras malas entregas: %+v", item)
	}
	scoreTrasMalas := item.ScoreTotal

	for i := 0; i < 8; i++ {
		if _, err := RegistrarObservacionScoreAgenteLocal(agente, "", materia, 9.0, true, "entrega buena"); err != nil {
			t.Fatalf("RegistrarObservacionScoreAgenteLocal buena #%d: %v", i+1, err)
		}
	}
	item, err = GetAgenteScoreLocal(agente, "", materia)
	if err != nil {
		t.Fatalf("GetAgenteScoreLocal tras buenas: %v", err)
	}
	if item == nil || item.ScoreTotal <= scoreTrasMalas {
		t.Fatalf("el score deberia recuperarse tras buenas entregas: %+v", item)
	}
	if item.ScoreTotal <= 5.0 {
		t.Fatalf("el score deberia poder subir por encima de neutral: %+v", item)
	}
}

func TestFitnessAgenteLocalRespetaPesoPorMateria(t *testing.T) {
	openTestScoreDB(t)

	if _, err := RegistrarBenchmarkScoreAgenteLocal("Gemma1", "", "brainstorming", 9.0, "muy buena ideacion"); err != nil {
		t.Fatalf("benchmark Gemma brainstorming: %v", err)
	}
	if _, err := RegistrarBenchmarkScoreAgenteLocal("Gemma1", "", "infraestructura", 3.0, "floja en infra"); err != nil {
		t.Fatalf("benchmark Gemma infraestructura: %v", err)
	}
	if _, err := RegistrarBenchmarkScoreAgenteLocal("QwenCoder1", "", "brainstorming", 3.0, "flojo en ideacion"); err != nil {
		t.Fatalf("benchmark Qwen brainstorming: %v", err)
	}
	if _, err := RegistrarBenchmarkScoreAgenteLocal("QwenCoder1", "", "infraestructura", 9.0, "fuerte en infra"); err != nil {
		t.Fatalf("benchmark Qwen infraestructura: %v", err)
	}

	fitnessGemmaInfra, err := FitnessAgenteLocal("Gemma1", "", map[string]float64{
		"brainstorming":   0.25,
		"infraestructura": 0.75,
	})
	if err != nil {
		t.Fatalf("FitnessAgenteLocal Gemma infra: %v", err)
	}
	fitnessQwenInfra, err := FitnessAgenteLocal("QwenCoder1", "", map[string]float64{
		"brainstorming":   0.25,
		"infraestructura": 0.75,
	})
	if err != nil {
		t.Fatalf("FitnessAgenteLocal Qwen infra: %v", err)
	}
	if fitnessQwenInfra <= fitnessGemmaInfra {
		t.Fatalf("infraestructura deberia pesar mas que brainstorming en este perfil: gemma=%.1f qwen=%.1f", fitnessGemmaInfra, fitnessQwenInfra)
	}

	fitnessGemmaBrainstorm, err := FitnessAgenteLocal("Gemma1", "", map[string]float64{
		"brainstorming":   0.80,
		"infraestructura": 0.20,
	})
	if err != nil {
		t.Fatalf("FitnessAgenteLocal Gemma brainstorming: %v", err)
	}
	fitnessQwenBrainstorm, err := FitnessAgenteLocal("QwenCoder1", "", map[string]float64{
		"brainstorming":   0.80,
		"infraestructura": 0.20,
	})
	if err != nil {
		t.Fatalf("FitnessAgenteLocal Qwen brainstorming: %v", err)
	}
	if fitnessGemmaBrainstorm <= fitnessQwenBrainstorm {
		t.Fatalf("brainstorming deberia inclinar el fitness hacia Gemma: gemma=%.1f qwen=%.1f", fitnessGemmaBrainstorm, fitnessQwenBrainstorm)
	}
}
