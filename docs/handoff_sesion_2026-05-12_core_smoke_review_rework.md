# Handoff sesion 2026-05-12 - core smoke review/rework

Estado al cortar:

- No quedan procesos vivos de `arrancar_codex.sh`, `go test ./modulos/orquesta-app-codex-stack` ni `codex exec` de la smoke.
- No hay commit creado en esta sesion.
- Hay cambios pendientes en worktree; no revertirlos.
- Tests focales pasados despues del ultimo ajuste.
- Falta repetir `go test -count=1 ./...` y la smoke real completa.

## Que paso

La smoke real de review/rework ya no fallo por el problema anterior de agentes
parados. Orquesta arranco 4 agentes Codex reales en paralelo:

- director;
- api;
- web;
- persistencia.

Los 4 entregaron ACK de brainstorming/documentacion. Despues Orquesta abrio
`programacion`, pero la smoke fallo porque no habia tareas materializadas:

```text
phase=programacion tasks=[] delivered=[] artifacts=[...4 ACKs de brainstorming...]
```

La causa raiz fue el `director_decisions.json` real:

- el contenido del plan era bueno;
- incluia contrato funcional;
- incluia 5 microtareas de programacion;
- pero las decisiones `create_microtask` venian con
  `decision.phase_id=programacion`;
- para el plan inicial la fase canonica de la decision debe ser
  `planificacion_microtareas`;
- la fase objetivo de ejecucion debe estar en
  `create_microtask.task.phase_id=programacion`.

## Cambios aplicados al final

- `modulos/orquesta-director-agent-file-source/normalize_v0.go`
  normaliza el caso inequivoco de fichero externo:
  si `create_microtask.decision.phase_id` viene igual que
  `create_microtask.task.phase_id`, pasa a `planificacion_microtareas`.
- `modulos/orquesta-director-agent-file-source/source_v0_test.go`
  agrega test de normalizacion para ese caso real.
- `modulos/orquesta-app-codex-stack/director_decision_contract_v0.go`
  aclara el prompt del director:
  `decision.phase_id=planificacion_microtareas` y
  `task.phase_id=programacion` en la primera entrega.
- `modulos/orquesta-app-codex-stack/drain_director_decisions_batch_v0_test.go`
  agrega prueba de stack con lote realista que venia con
  `decision.phase_id=programacion`; ahora `DrainRunV0` materializa tareas.
- Se mantuvo permitido `create_microtask` en `programacion` para cambios en
  caliente, porque `app-change` lo necesita y el core lo permite.
- Docs actualizadas en:
  - `modulos/orquesta-director-agent-file-source/docs/decisiones.md`;
  - `modulos/orquesta-director-agent-file-source/docs/pruebas.md`;
  - `modulos/orquesta-director-agent/docs/contratos.md`;
  - `modulos/orquesta-app-codex-stack/docs/decisiones.md`;
  - `modulos/orquesta-app-codex-stack/docs/pruebas.md`.

## Tests pasados justo antes de cortar

```bash
go test -count=1 \
  ./modulos/orquesta-director-agent \
  ./modulos/orquesta-director-agent-file-source \
  ./modulos/orquesta-app-change-director-source \
  ./modulos/orquesta-app-codex-stack \
  ./modulos/orquesta-app-director-service \
  ./modulos/orquesta-director-agent-workflow \
  -run 'TestValidateDirectorAgentDecisionV0(AceptaMicrotareaCompacta|AceptaMicrotareaDuranteProgramacion)|TestDirectorAgentDecisionFileSourceV0Normaliza(CreateMicrotaskPhaseObjetivo|FaseDesdePayload)|TestDirectorTaskV0IncluyeContratoDeDecisionesEjecutables|TestDrainRunV0NormalizaCreateMicrotaskPhaseObjetivoDelDirectorReal|TestDrainRunV0ConsumeDecisionFileConLoteProgramacionGrande|TestDrainRunV0AppChangeConProgramacionPendienteArrancaCambio|TestCodexStackV0FlujoProgresivoAPIMCPProgramaYCambiaSinIntervencionManual' -v
```

Resultado: `ok`.

## Cambios anteriores de esta tanda que ya estaban aplicados

- Review gate ya revisa deliveries de agentes cerrados.
- `StoppedAgents`/`ConfirmedStoppedAgents` no ocultan una entrega registrada.
- Silencio o ACK sin cambios ya no se cuenta como loop repetido.
- `progressSourceV0` usa `WaitInterval` como ventana minima.
- El drain considera pendientes los agentes con stop solicitado pero no
  confirmado.
- El director protegido solo se puede parar ante loop duro, no por simple
  stalled.
- El tick input se compacta por carril activo y no mete todo el contexto.
- Review gate emite una sola etapa siguiente: request, result, accept o rework.
- Candidate providers limitan ruido: una delivery/artifact y una observacion
  de progreso accionable por tick.
- Prompt Codex exige modo compacto/caveman si existe y ACK final de una linea.

## Siguiente paso al retomar

1. Ejecutar:

```bash
git diff --check
go test -count=1 ./...
```

2. Si pasa, repetir la smoke real:

```bash
tmp=/tmp/orquesta-smokes/app-codex-stack-review-rework-real-$(date +%Y%m%d%H%M%S)
mkdir -p "$tmp/project/.orquesta-runtime"
ORQUESTA_CODEX_STACK_OPT_IN=1 \
ORQUESTA_CODEX_STACK_COMMAND='ORQUESTA_CODEX_STACK_REVIEW_REWORK_SMOKE=1 go test ./modulos/orquesta-app-codex-stack -run TestNuevaAppWebCodexStackRealReviewReworkOptInV0 -count=1 -timeout 1800s -v' \
ORQUESTA_CODEX_COMMAND="$(command -v codex)" \
ORQUESTA_CODEX_HOME="$HOME" \
ORQUESTA_CODEX_CODE_HOME="${CODEX_HOME:-$HOME/.codex}" \
ORQUESTA_CODEX_PATH="$PATH" \
ORQUESTA_CODEX_APPROVAL_POLICY=never \
ORQUESTA_CODEX_SANDBOX=workspace-write \
ORQUESTA_CODEX_MODEL=gpt-5.5 \
ORQUESTA_CODEX_REASONING_EFFORT=high \
ORQUESTA_CODEX_SMOKE_TIMEOUT_MS=1200000 \
ORQUESTA_CODEX_PROJECT_WORKDIR="$tmp/project" \
ORQUESTA_CODEX_RUNTIME_WORKDIR="$tmp/project/.orquesta-runtime" \
timeout 25m ./modulos/orquesta-app-codex-stack/arrancar_codex.sh
```

3. Validacion esperada de la smoke:

- Orquesta arranca director/api/web/persistencia en paralelo;
- director entrega `director_decisions.json`;
- Orquesta materializa microtareas antes de abrir programacion;
- Orquesta lanza agentes de programacion reales;
- al menos una entrega de programacion se registra;
- se fuerza fichero grande;
- review gate pide cambios;
- Orquesta lanza agente de rework real;
- rework entrega ACK;
- review gate acepta rework.

## Regla para retomar

Si vuelve a fallar, no subir timeouts como primera reaccion. Mirar la transicion
exacta que falta en el run y crear un test rojo minimo antes de tocar el codigo.
