# Incidencia: gates OPES por work_kind antes de settlement

## Sintoma

Los avances de BUG-058, BUG-066 y BUG-075 ya proyectaban calidad textual,
lifecycle goal-first y artefactos parciales recuperables, pero la cobertura no
era uniforme para todos los derivados de la secuencia OPES. Algunos work_kind
podian quedar solo con el required test generico de submit_artifact, sin una
evidencia minima especifica de HTML, visual, juegos, manual, revision, fuentes
o derivados.

## Cambio

- `orquesta-opes-bridge` genera required tests especificos por
  `work_kind/artifact_type` para toda la secuencia OPES principal.
- `orquesta-opes-director` incorpora gates de evidencia minima al registro de
  tema: si un derivado aceptado no aporta su evidencia publica minima, se
  anade `pending_refs=required-evidence-*`, el settlement queda `not_settled`
  y se crea rework causal.
- `topic_registry_update` no se realimenta a si mismo: el bridge lo exige en el
  contrato de lanzamiento, pero una entrega de registro ya materializada no
  genera followup por falta de su propia evidencia.

## Evidencia

```bash
go test -count=1 ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-director ./modulos/orquesta-opes-topic-registry ./modulos/orquesta-app-codex-stack
```

No toca OPES productivo ni cursos reales.
