# Incidencia: goal-first debe exponer alto consumo activo

Fecha: 2026-07-02

## Sintoma

En ejecuciones OPES goal-first, un goal podia seguir `active` con mas de
100.000 tokens consumidos y solo checkpoint o artefactos insuficientes. Las
superficies operativas no siempre diferenciaban ese estado de un `running`
normal.

## Riesgo

El operador puede seguir esperando o ejecutar shutdown sin una causa publica
compacta que indique consumo alto. Eso degrada cuota, observabilidad y decision
de replan/corte.

## Avance aplicado

- El backend Codex app-server proyecta alto consumo cuando `thread/goal/get`
  devuelve un goal `running` con `tokensUsed >= 100000`.
- La observacion conserva `status=running`; no bloquea ni descarta el trabajo.
- El `summary` incluye `tokens_used`, `time_used_seconds` y `token_budget` si
  estan disponibles.
- Se anade evidencia
  `evidence-ref-codex-app-server-goal-high-token-usage`.

## Evidencia

Tests:

```bash
go test -count=1 ./cmd/orquesta-server -run 'TestServerCodexAppServerGoalBackendV0(ExponeAltoConsumoActivo|BloqueaGoalActivoPorTimeout|ObservaUsageLimitedConCausaOperable)'
git diff --check
```

## Residual

Sigue pendiente detectar `checkpoint_only_high_consumption` con artefactos del
write-set, proponer replan/corte seguro y garantizar que `shutdown forced=true`
no publique `ready` si el backend app-server propio sigue vivo.
