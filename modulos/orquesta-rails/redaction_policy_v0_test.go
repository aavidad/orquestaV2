package orquestarails

import (
	"strings"
	"testing"
)

func TestRedactOperationalTextForFieldV0SanitizaSecretosEfectivosConRailsOffline(t *testing.T) {
	t.Setenv(RailsModeEnvV0, RailsModeEnforcedV0)
	t.Setenv(DetailProhibitedRailsEnvV0, "on")
	t.Setenv(DetailProhibitedRailsScopeEnvV0, "codex_wave_tail.*")

	input := strings.Join([]string{
		`access_token=abc123`,
		`Authorization: Bearer token-real`,
		`client_secret="valor con espacios"`,
		`{"password":"super secreto"}`,
		`postgres://user:pass-real@db.local/app`,
		`prompt=texto completo no publicable`,
		`/home/alberto/proyecto/secreto`,
		`-----BEGIN PRIVATE KEY-----`,
		`linea-real`,
		`-----END PRIVATE KEY-----`,
	}, "\n")
	got, changed := RedactOperationalTextForFieldV0("codex_wave_tail", "stdout", input)
	if !changed {
		t.Fatalf("redaccion no bloqueante no sanitizo secretos efectivos")
	}
	for _, forbidden := range []string{"abc123", "token-real", "valor con espacios", "super secreto", "pass-real", "texto completo", "/home/alberto", "linea-real"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("valor sensible %q filtrado en %q", forbidden, got)
		}
	}
	for _, want := range []string{"access_token=<redacted>", "Authorization: Bearer <redacted>", `{"password":"<redacted>"}`, "postgres://user:<redacted>@db.local/app", "prompt=<redacted>", "<home-path-redacted>", "<private-material-redacted>"} {
		if !strings.Contains(got, want) {
			t.Fatalf("redaccion esperada %q ausente en %q", want, got)
		}
	}
	generic := "provider model token budget home prompt policy sin valor efectivo"
	if TextContainsOperationalDetailMarkerV0(generic) ||
		TextContainsOperationalRawDetailForFieldV0("codex_wave_tail", "stdout", generic) {
		t.Fatalf("redaccion no debe reactivar deteccion/bloqueo de rail blando generico")
	}
}

func TestRedactOperationalTextForFieldV0NoUsaMarcadoresBlandosGenericos(t *testing.T) {
	t.Setenv(RailsModeEnvV0, RailsModeEnforcedV0)
	t.Setenv(DetailProhibitedRailsEnvV0, "on")
	t.Setenv(DetailProhibitedRailsScopeEnvV0, "*")

	input := strings.Join([]string{
		`provider=model model=gpt token budget secret policy sin valor`,
		`-----BEGIN CERTIFICATE-----`,
		`cert-data`,
		`-----END CERTIFICATE-----`,
	}, "\n")
	got, changed := RedactOperationalTextForFieldV0("codex_wave_tail", "stdout", input)
	if changed {
		t.Fatalf("marcadores blandos genericos no deben disparar redaccion: %q", got)
	}
	if got != input {
		t.Fatalf("texto operativo debe preservarse: %q", got)
	}
}
