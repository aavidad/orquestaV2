# Autoprogramacion review/rework - 2026-05-23

## Alcance

Runbook para repetir el smoke real opt-in de review/rework del stack Codex sin
tocar OPES ni el nucleo puro.

El caso cubierto es:

1. Orquesta arranca una app multiagente con Codex real.
2. El stack espera una cohorte estable y registra entregas por ACK.
3. El smoke dana una evidencia real dentro del write-set.
4. El review gate emite `changes_requested`.
5. `ReviewReworkReplanSourceV0` crea un retry causal con `task_ref` del
   descriptor original y `agent_request_id` corto/estable.
6. El dispatcher lanza el agente de rework y registra su descriptor ACK.

## Contrato externo

- Solo refs opacas salen al nucleo: task, delivery, review, rework, replan,
  capacidad y agente.
- El stack Codex puede leer descriptors y ficheros reales porque es adaptador
  exterior.
- El agente de rework reutiliza la task original y conserva el write-set
  declarado; no abre scope global.
- Las refs de agente para rework no dependen del texto completo de
  `rework_request_ref`; se derivan de `task_ref` mas hash causal para evitar
  rutas de runtime largas.
- El rework conserva manga ancha, pero no bucle infinito: como maximo se crean
  seis agentes de retry por `task_ref`; a partir de ahi el director debe
  escalar/replanificar con otra estrategia.
- La evidencia ambiental de runtime, proveedor, HOME, DB, tokens o filesystem
  no se propaga como evidencia neutral de replan.

## Validacion rapida

```bash
go test -count=1 ./modulos/orquesta-app-codex-stack \
  -run 'TestReviewReworkReplanSourceV0|TestCodexStackRealSmokeReworkDescriptorV0|TestCodexStackV0ExternalWorkRunSupervisorConsumeDeliverySinExpirarWaitV0'
```

## Smoke real opt-in

```bash
ORQUESTA_CODEX_STACK_REVIEW_REWORK_SMOKE=1 \
ORQUESTA_CODEX_COMMAND="$(command -v codex)" \
ORQUESTA_CODEX_HOME="$HOME" \
ORQUESTA_CODEX_CODE_HOME="${CODEX_HOME:-$HOME/.codex}" \
ORQUESTA_CODEX_PATH="$PATH" \
ORQUESTA_CODEX_APPROVAL_POLICY=never \
ORQUESTA_CODEX_SANDBOX=workspace-write \
ORQUESTA_CODEX_MODEL=gpt-5.5 \
ORQUESTA_CODEX_SMOKE_TIMEOUT_SECONDS=900 \
ORQUESTA_CODEX_PROJECT_WORKDIR=/tmp/orquesta-smokes/review-rework-20260523/project \
ORQUESTA_CODEX_RUNTIME_WORKDIR=/tmp/orquesta-smokes/review-rework-20260523/project/.orquesta-codex-runtime \
go test ./modulos/orquesta-app-codex-stack \
  -run TestNuevaAppWebCodexStackRealReviewReworkOptInV0 \
  -count=1 -timeout 1100s -v
```

Requiere confirmacion explicita del operador porque consume cuota real y lanza
procesos Codex.
