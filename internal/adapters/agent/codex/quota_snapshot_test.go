package codex

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/ports"
)

func TestTraduccionCuotaCodexEsConservadora(t *testing.T) {
	instante := time.Unix(1_000, 0).UTC()
	colocacion, _ := ports.NewAgentPlacementRef("placement:uno")
	const primaria = `{"usedPercent":25,"windowDurationMins":300,"resetsAt":2000}`
	const secundaria = `{"usedPercent":40,"windowDurationMins":10080,"resetsAt":3000}`
	respuesta := func(id, primera, segunda, alcanzado, gasto string) []byte {
		return []byte(fmt.Sprintf(`{"rateLimits":{"credits":{"balance":"saldo-secreto"},"individualLimit":{"used":"/ruta/perfil"},"limitId":%s,"limitName":"perfil-secreto","planType":"plus","primary":%s,"rateLimitReachedType":%s,"secondary":%s,"spendControlReached":%s},"rateLimitResetCredits":null,"rateLimitsByLimitId":{}}`, id, primera, alcanzado, segunda, gasto))
	}
	casos := []struct {
		nombre, id, primera, segunda, alcanzado, gasto string
		estado                                         application.AgentQuotaObservationStatus
	}{
		{"primaria y secundaria", `"codex"`, primaria, secundaria, "null", "null", application.AgentQuotaAvailable},
		{"id vacío y solo secundaria", `""`, "null", secundaria, "null", "null", application.AgentQuotaAvailable},
		{"id nulo", "null", primaria, secundaria, "null", "null", application.AgentQuotaAvailable},
		{"porcentaje cien", `"codex"`, strings.Replace(primaria, "25", "100", 1), secundaria, "null", "null", application.AgentQuotaExhausted},
		{"tipo alcanzado", `"codex"`, primaria, secundaria, `"rate_limit_reached"`, "null", application.AgentQuotaExhausted},
		{"control de gasto", `"codex"`, primaria, secundaria, "null", "true", application.AgentQuotaExhausted},
		{"ventanas nulas", `"codex"`, "null", "null", "null", "null", application.AgentQuotaUnknown},
		{"porcentaje nulo", `"codex"`, strings.Replace(primaria, "25", "null", 1), secundaria, "null", "false", application.AgentQuotaUnknown},
		{"porcentaje fuera de rango", `"codex"`, strings.Replace(primaria, "25", "-1", 1), secundaria, "null", "false", application.AgentQuotaUnknown},
		{"reinicio incoherente", `"codex"`, strings.Replace(primaria, "2000", "999", 1), secundaria, "null", "false", application.AgentQuotaUnknown},
		{"tipo desconocido", `"codex"`, primaria, secundaria, `"inventado"`, "false", application.AgentQuotaUnknown},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			resultado, err := traducirLecturaCuotaCodex(respuesta(caso.id, caso.primera, caso.segunda, caso.alcanzado, caso.gasto), colocacion, time.Minute, func() time.Time { return instante })
			if err != nil || resultado.Observacion.Status != caso.estado || resultado.Observacion.Quality != "measured" {
				t.Fatalf("traducción = %s/%s, %v", resultado.Observacion.Status, resultado.Observacion.Quality, err)
			}
			if (caso.estado == application.AgentQuotaExhausted) != !resultado.Observacion.RetryAt.IsZero() {
				t.Fatalf("reintento incoherente: %s", resultado.Observacion.RetryAt)
			}
		})
	}
	base, _ := traducirLecturaCuotaCodex(respuesta(`"codex"`, primaria, secundaria, "null", "null"), colocacion, time.Minute, func() time.Time { return instante })
	porcentaje, _ := traducirLecturaCuotaCodex(respuesta(`"codex"`, strings.Replace(primaria, "25", "80", 1), secundaria, "null", "null"), colocacion, time.Minute, func() time.Time { return instante })
	rotada, _ := traducirLecturaCuotaCodex(respuesta(`"codex"`, strings.Replace(primaria, "2000", "4000", 1), secundaria, "null", "null"), colocacion, time.Minute, func() time.Time { return instante })
	if base.Observacion.ObservedAt != instante || base.Observacion.ExpiresAt != instante.Add(time.Minute) ||
		base.Observacion.ResetAt != time.Unix(2_000, 0).UTC() || base.Observacion.WindowRef != porcentaje.Observacion.WindowRef ||
		base.Observacion.WindowRef == rotada.Observacion.WindowRef {
		t.Fatalf("reloj, reinicio o identidad inestables: %+v", base.Observacion)
	}
	evidencia := string(base.Evidencia)
	if !strings.Contains(evidencia, `"usedPercent":25`) || strings.Contains(evidencia, "saldo-secreto") ||
		strings.Contains(evidencia, "/ruta/perfil") || strings.Contains(evidencia, "perfil-secreto") {
		t.Fatalf("evidencia no saneada: %s", evidencia)
	}
	extendida := strings.Replace(string(respuesta(`"codex"`, primaria, secundaria, "null", "null")), `"rateLimits":`, `"accountId":"opaco","rateLimitUpsell":null,"rateLimits":`, 1)
	traducida, err := traducirLecturaCuotaCodex([]byte(extendida), colocacion, time.Minute, func() time.Time { return instante })
	if err != nil || traducida.Observacion.Status != application.AgentQuotaAvailable ||
		strings.Contains(string(traducida.Evidencia), "opaco") {
		t.Fatalf("extensión oficial contaminó la observación: %+v, %v", traducida, err)
	}
}
