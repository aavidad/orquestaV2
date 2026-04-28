package runtimeagente

import "testing"

func TestConectorPorDefectoAgentePorFamilia(t *testing.T) {
	tests := []struct {
		agente   string
		esperado string
	}{
		{agente: "Codex1", esperado: "codex-cli"},
		{agente: "Claude1", esperado: "claude-code"},
		{agente: "Gemini1", esperado: "gemini-cli"},
		{agente: "Gemma1", esperado: "ollama-cli"},
		{agente: "QwenDev", esperado: "ollama-cli"},
		{agente: "Desconocido1", esperado: "codex-cli"},
	}
	for _, tc := range tests {
		if got := ConectorPorDefectoAgente(tc.agente); got != tc.esperado {
			t.Fatalf("ConectorPorDefectoAgente(%q)=%q; want %q", tc.agente, got, tc.esperado)
		}
	}
}

func TestFamiliaAgenteReconocePremiumYOllama(t *testing.T) {
	tests := []struct {
		agente   string
		esperado string
	}{
		{agente: "Codex1", esperado: "codex"},
		{agente: "Claude1", esperado: "claude"},
		{agente: "Gemini1", esperado: "gemini"},
		{agente: "Gemma4", esperado: "ollama"},
		{agente: "Llama3", esperado: "ollama"},
		{agente: "Qwen3", esperado: "ollama"},
		{agente: "otro", esperado: ""},
	}
	for _, tc := range tests {
		if got := familiaAgente(tc.agente); got != tc.esperado {
			t.Fatalf("familiaAgente(%q)=%q; want %q", tc.agente, got, tc.esperado)
		}
	}
}

func TestConectorCompatibleConAgenteIgnoraHerenciaIncompatible(t *testing.T) {
	tests := []struct {
		agente  string
		slug    string
		comando string
		want    bool
	}{
		{agente: "Claude1", slug: "ollama_pool_local", want: false},
		{agente: "Gemini1", slug: "ollama-cli", comando: "ollama", want: false},
		{agente: "Codex1", slug: "codex-cli", comando: "codex", want: true},
		{agente: "Claude1", slug: "claude-code", comando: "claude-code", want: true},
		{agente: "Gemma1", slug: "ollama_pool_local", comando: "ollama", want: true},
	}
	for _, tc := range tests {
		if got := ConectorCompatibleConAgente(tc.agente, tc.slug, tc.comando); got != tc.want {
			t.Fatalf("ConectorCompatibleConAgente(%q,%q,%q)=%v; want %v", tc.agente, tc.slug, tc.comando, got, tc.want)
		}
	}
}

func TestModeloPorDefectoAgenteLocal(t *testing.T) {
	tests := []struct {
		agente string
		want   string
	}{
		{agente: "Gemma1", want: "gemma4:26b"},
		{agente: "QwenCoder1", want: "qwen2.5-coder:7b"},
		{agente: "Codex1", want: ""},
	}
	for _, tc := range tests {
		if got := ModeloPorDefectoAgente(tc.agente); got != tc.want {
			t.Fatalf("ModeloPorDefectoAgente(%q)=%q; want %q", tc.agente, got, tc.want)
		}
	}
}

func TestModeloAfinAgente(t *testing.T) {
	tests := []struct {
		agente string
		modelo string
		want   bool
	}{
		{agente: "Gemma1", modelo: "gemma4:26b", want: true},
		{agente: "Gemma1", modelo: "qwen2.5-coder:7b", want: false},
		{agente: "QwenCoder1", modelo: "qwen2.5-coder:7b", want: true},
		{agente: "QwenCoder1", modelo: "gpt-5.4", want: false},
	}
	for _, tc := range tests {
		if got := ModeloAfinAgente(tc.agente, tc.modelo); got != tc.want {
			t.Fatalf("ModeloAfinAgente(%q,%q)=%v; want %v", tc.agente, tc.modelo, got, tc.want)
		}
	}
}
