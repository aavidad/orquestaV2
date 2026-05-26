# Runbook: codex-director-wave con guardas estrictas

Fecha: 2026-05-25.

## Alcance

Este runbook cubre el lanzamiento de olas Codex gobernadas por el Director
Operativo desde `cmd/orquesta-server codex-director-wave`.

## Default seguro

En ejecucion real, la ola usa guardas estrictas por defecto. El comando no debe
rellenar implicitamente `write_set=.` ni `operator-validation-required`. Si
faltan `branch_ref`, `worktree_ref`, `write_set` o `required_tests`, el summary
devuelve issues y no materializa agentes.

`--dry-run` conserva compatibilidad para diagnostico local y puede rellenar
rails minimos si no se activa `--strict-director-guards`.

## Control de Orquesta

`codex-launch-wave` y `codex-launch-director-wave` no deben usarse para trabajo
real fuera del servidor residente. Por defecto, cualquier ejecucion real queda
bloqueada con `unmanaged_launch_blocked` antes de materializar agentes. El
camino operativo es servidor/cola/Director, para que `/ops`, shutdown,
prioridades, auditoria y recuperacion vean todos los agentes.

`--dry-run` sigue permitido. El breakglass `--allow-unmanaged-launch` exige
`--unmanaged-launch-reason` y `--confirm-unmanaged-launch=<wave-ref>` y queda
reservado a tests/smokes de bajo nivel con runtime controlado.

## Opt-in auditado

Un operador puede permitir `write_set=.` o tests placeholder solo con opt-in
explicito:

```bash
go run ./cmd/orquesta-server codex-director-wave \
  --allow-global-write-set \
  --allow-placeholder-tests \
  --guard-override-reason "operador autoriza auditoria completa" \
  --guard-override-evidence-ref "evidence-ref-guard-override-001" \
  --write-set "." \
  --required-tests "operator-validation-required" \
  --branch-ref "branch-ref-opaca" \
  --worktree-ref "worktree-ref-opaca" \
  --objective "objetivo acotado"
```

El prompt mantiene las guardas de no borrar sin revision, no salir del proyecto
y no ampliar alcance por texto libre. El shard asignado es ownership inicial; el
alcance total autorizado sigue siendo el write-set global.

## Ola complementaria 10x6

Para completar una ola de autoprogramacion con 10 agentes padre y hasta 6
subagentes por padre, la capacidad debe declararse en el comando o entorno de
operador. No se debe cambiar el core ni los defaults seguros para conseguir ese
paralelismo.

Ejemplo opt-in:

```bash
ORQUESTA_CODEX_DIRECTOR_WAVE_AGENTS=10 \
ORQUESTA_CODEX_DIRECTOR_MAX_SUBAGENTS_PER_AGENT=6 \
ORQUESTA_CODEX_MAX_CONCURRENCY=10 \
ORQUESTA_CODEX_MAX_BATCH_READY=10 \
go run ./cmd/orquesta-server codex-director-wave \
  --agents 10 \
  --max-subagents-per-agent 6 \
  --branch-ref "branch-ref-opaca" \
  --worktree-ref "worktree-ref-opaca" \
  --write-set "modulos/orquesta-app-codex-stack" \
  --write-set "docs" \
  --required-tests "go test -count=1 ./modulos/orquesta-app-codex-stack" \
  --objective "autoprogramacion acotada con ola complementaria"
```

Supuestos practicos:

- `ORQUESTA_CODEX_MAX_CONCURRENCY=10` limita procesos Codex vivos globales; no
  significa 10 padres cerrados ni 60 hijos arrancados a la vez.
- Cada padre conserva ownership inicial y puede usar hasta 6 subagentes solo si
  Orquesta materializa esa ola hija con `parent_agent_ref`, `wave_ref` y
  presupuesto.
- Los hijos no lanzan mas hijos cuando `max_delegation_depth=1`; entregan ACK
  compacto con rutas tocadas, pruebas y bloqueos.
- `WaitAgentRefs`/cohorte/ola siguen acotando espera e ingesta; no se vuelve a
  esperar "todos los agentes vivos del run".
- `CODEX-WAVE-REAL` y `CODEX-RECURSION-REAL` ya son evidencia cerrada salvo
  regresion; esta ola complementaria debe aportar evidencia propia si descubre
  un blocker nuevo.

Errores frecuentes:

- Subir solo `--agents` sin subir `ORQUESTA_CODEX_MAX_CONCURRENCY` deja outbox
  pendiente por capacidad, no por fallo de Director.
- Usar `write_set=.` sin override auditado bloquea por guardas estrictas.
- Omitir `required_tests` impide cerrar con evidencia durable; un resumen del
  agente no sustituye `RequiredTestEvidenceV0`.
- Relanzar hijos manualmente desde un subagente rompe linaje y presupuesto; el
  replan debe pedirlo al Director.
- Confundir `tasks=[]` temporal con ausencia de progreso puede cortar una run
  mientras `LastSequence`, ACKs, descriptors u outbox siguen avanzando.

## Validacion focal

```bash
go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-director-operativo ./modulos/orquesta-app-codex-stack
```
