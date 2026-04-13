package smokehelper_gemini

import (
	"reflect"
	"testing"
)

func TestNormalizarTokens(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{
			input: "Hello, World!",
			want:  []string{"hello", "world"},
		},
		{
			input: "NormalizarIdentificadorTecnico",
			want:  []string{"normalizaridentificadortecnico"},
		},
		{
			input: "uno, dos: tres-cuatro",
			want:  []string{"uno", "dos", "tres", "cuatro"},
		},
		{
			input: "  ESPACIOS   ",
			want:  []string{"espacios"},
		},
		{
			input: "ASCII-only 123!",
			want:  []string{"ascii", "only", "123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := NormalizarTokens(tt.input); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NormalizarTokens() = %v, want %v", got, tt.want)
			}
		})
	}
}
