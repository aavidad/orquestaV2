# TAREA OPES Orquesta: no adoptar autoprogramación stale en sesiones OPES

Fecha: 2026-06-25.

## Contexto

Durante el arranque de Orquesta para el temario OPES `operario` en:

`/home/alberto/Trabajo/OPES/opes-salidas/diputacion_granada/cursos_opes_2026/administracion_especial/operario`

el servidor lanzó y re-adoptó varias veces la run:

`request-ref-autoprogramming-backlog-scanner-15eeecb9`

La run era automejora de Orquesta, no trabajo OPES del temario.

## Evidencia

- Con `ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED=false` apareció un scanner de autoprogramación.
- El nombre antiguo `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER=0` no desactiva nada y no avisa; el binario usa `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS=0`.
- Con `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS=0`, `/api/v0/server/status` ya mostraba `idle_self_improvement_reason=disabled`, pero el arranque seguía adoptando una run stale previa:
  `startup_message="runs activos asumidos=1 sin purga"`.
- La cola solo quedó limpia tras archivar `.orquesta-runtime`, archivar estados sucios y arrancar con:
  `ORQUESTA_STARTUP_CLEANUP_MODE=forced_stop`.

## Problema

En una sesión OPES, la automejora de Orquesta no debe consumir agentes ni cuota si no hay opt-in explícito. Aunque la automejora esté desactivada, una run stale de autoprogramación puede reaparecer por adopción de estado anterior.

Además, si el operador usa una variable antigua o incompleta, el servidor no publica un error accionable; simplemente mantiene defaults.

## Cambio necesario en la app

1. Si `ORQUESTA_OPES_PROJECT_WORKDIR` o `ORQUESTA_CODEX_PROJECT_WORKDIR` apuntan a una salida OPES y `idle_self_improvement` está desactivado, no adoptar runs `request-ref-autoprogramming-*` como activas.
2. Mover esas runs a estado `stale_suppressed_by_domain_session` o equivalente, conservando evidencia y sin borrarlas.
3. Publicar en `server/status` y `autoprogramming/status` una razón clara:
   `idle_self_improvement_suppressed_by_domain_session`.
4. Añadir diagnóstico para variables parecidas no reconocidas, especialmente:
   `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER` frente a
   `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS`.
5. El endpoint de shutdown/control debe poder parar una run de autoprogramación stale sin esperar checkpoint de un agente que no pertenece al trabajo OPES.

## Criterio de cierre

- Arrancar un servidor OPES con `.orquesta-runtime` que contenga un `request-ref-autoprogramming-backlog-scanner-*` antiguo y con `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS=0`.
- Resultado esperado: cola OPES visible limpia, cero procesos Codex de autoprogramación y diagnóstico público de supresión.
- Con opt-in explícito de automejora, el scanner puede arrancar, pero debe aparecer separado de la cola OPES del curso.

## Avance 2026-06-25

- `cmd/orquesta-server` detecta sesión OPES también cuando `ORQUESTA_CODEX_PROJECT_WORKDIR` apunta a `OPES` o a `opes-salidas`.
- En arranque `diagnose`, si `idle_self_improvement` está desactivado y la sesión es OPES/dominio, las runs `request-ref-autoprogramming-*` ejecutables no se adoptan como activas: se marcan `stopped` con `rescue_reason=idle_self_improvement_suppressed_by_domain_session` y evidencia `evidence-ref-idle-self-improvement-domain-session`.
- `server/status` publica la razón en `startup_message` y `startup_evidence_refs`.
- `/api/v0/autoprogramming/status` pide también terminales de cola y publica diagnóstico `idle_self_improvement_suppressed_by_domain_session` cuando encuentra la run suprimida.
- La variable legacy `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER` ya queda diagnosticada como alias/ignorada frente a `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS`.

Evidencia local:

```bash
go test -count=1 ./modulos/orquesta-mcp ./cmd/orquesta-server ./modulos/orquesta-web ./modulos/orquesta-server
git diff --check
```

## Relacionado

- `modulos/orquesta-server/docs/tareas.md`, `SRV-TASK-027`.
- `TAREA_OPES_RESIDENT_DIRECTOR_REWORK_LOOP_2026-06-23.md`.
- `TAREA_OPES_ORQUESTA_RESET_COLA_STALE_ACTIVE_STEP_2026-06-25.md`.
