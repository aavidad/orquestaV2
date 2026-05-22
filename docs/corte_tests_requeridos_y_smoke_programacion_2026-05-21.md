# Corte tests requeridos y smoke de programacion - 2026-05-21

## Objetivo

Avanzar Orquesta para que el Director Operativo pueda programar nucleo/director
con tests requeridos durables, sin meter runtime real dentro del nucleo y sin
mezclar OPES como producto base.

## Cambios subidos

- `6339115 Integrar runner de tests requeridos del director`
  - `orquesta-orchestration-core` incorpora `RequiredTestRunnerV0`.
  - El runner ejecuta por puerto `RequiredTestCommandExecutorPortV0`.
  - Guarda `RequiredTestEvidenceV0` causal e idempotente por
    `RequiredTestEvidenceWriterPortV0`.
  - `app-director-service` lo invoca desde `run_required_tests` si no existen
    evidencias causales ya persistidas.

- `7eaf574 Anadir adaptador local de tests requeridos`
  - Nuevo modulo `modulos/orquesta-runtime-required-test`.
  - Ejecuta comandos reales solo por allowlist inyectada.
  - No ejecuta shell, no hereda entorno y devuelve refs relativas de artefactos.
  - Queda fuera del nucleo y fuera de `orquesta-runtime` base para evitar ciclos
    de imports.

- `125b0ee Cablear tests requeridos opt-in en el stack`
  - `orquesta-app-codex-stack` acepta `RequiredTestRunner` inyectado.
  - `cmd/orquesta-server` lo cablea solo con
    `ORQUESTA_REQUIRED_TEST_RUNNER_ENABLED=1`.
  - Requiere allowlist mediante `ORQUESTA_REQUIRED_TEST_GO_COMMAND` o
    `ORQUESTA_REQUIRED_TEST_ALLOWED_COMMANDS`.
  - Apagado por defecto.

## Verificacion de codigo

Ejecutado antes de los commits:

```bash
git diff --check
go test -count=1 ./...
```

Resultado: verde.

## Smoke real de programacion con agentes y subagentes

Directorio temporal purgado antes de ejecutar:

```text
/tmp/orquesta-real-smoke-programacion-subagentes-20260521
```

Comando principal:

```bash
go run ./cmd/orquesta-server codex-launch-director-wave \
  --wave-ref real-programming-subagents-20260521 \
  --project-dir /tmp/orquesta-real-smoke-programacion-subagentes-20260521/project \
  --runtime-dir /tmp/orquesta-real-smoke-programacion-subagentes-20260521/runtime/real-programming-subagents-20260521 \
  --command "$(command -v codex)" \
  --agents 2 \
  --allow-recursive-delegation \
  --max-delegation-depth 1 \
  --max-subagents-per-agent 2 \
  --write-set "." \
  --required-tests "go test ./..." \
  --reasoning-effort high \
  --sandbox danger-full-access \
  --approval-policy never \
  --purge-runtime \
  --objective-file /tmp/orquesta-real-smoke-programacion-subagentes-20260521/objective.md
```

Resultado observado:

- 2 agentes padre lanzados.
- 4 subagentes hijos lanzados.
- 6 procesos Codex reales terminaron solos.
- No se corto ningun proceso.
- El repo Orquesta quedo limpio.
- El proyecto temporal genero codigo Go bajo `internal/smoke`.
- Se generaron 6 artefactos markdown bajo `smoke_outputs/`.

Validacion del proyecto temporal:

```bash
cd /tmp/orquesta-real-smoke-programacion-subagentes-20260521/project
go test -count=1 ./...
```

Resultado:

```text
ok smoke.local/orquesta-real-smoke/internal/smoke 0.002s
```

## Pendiente

- El smoke directo `codex-launch-director-wave` demuestra lanzamiento real de
  agentes/subagentes, pero no es todavia el ciclo durable completo
  `wait -> review -> run_required_tests -> replan_or_close`.
- El smoke de servidor/director acotado donde el `RequiredTestRunner` opt-in
  genera `RequiredTestEvidenceV0` dentro del `PlanState` real quedo cerrado el
  2026-05-22 como `CODEX-REQTEST-REAL-E2E`. No cubre ola/cohorte amplia ni
  recursion real.
- Falta smoke real de replan negativo con runner opt-in; el replan automatico
  offline ante `required-tests-failed` de un unico task ya queda cubierto por
  `docs/corte_required_tests_failed_replan_2026-05-21.md`.
- Quedan duplicidades/historico en comandos directos de wave que conviene
  clasificar, no borrar.

## Smoke real completo de servidor

Directorio temporal purgado antes de ejecutar:

```text
/tmp/orquesta-real-complete-fixed-20260521
```

Flujo probado:

- `orquesta-server run` con estado, runtime y proyecto temporales.
- `POST /api/v0/apps/director` con `request_kind=crear_app_completa`,
  `execution_mode=normal`, app Go API y arquitectura hexagonal.
- Supervisor residente + drenaje final con `POST /api/v0/runs/supervise`.
- Codex real con `sandbox=danger-full-access`,
  `approval_policy=never`, `reasoning_effort=high`.
- `RequiredTestRunner` opt-in cableado por entorno.

Resultado observado:

- Arranque limpio: `startup_ready=true`, sin cola sucia.
- 4 agentes iniciales lanzados: director, api, i18n y calidad.
- El director genero una `WorkflowTaskV0` de programacion con
  `required_tests=["go test ./..."]`.
- Orquesta lanzo un quinto agente de programacion:
  `agent-ref-workflow-task-ref-programacion-biblioteca-go-v0`.
- 5 ACKs reales escritos y validados.
- El supervisor ya no fallo al ingerir el ACK de programacion y respondio
  `stop_reason=done`.
- Se registro 1 delivery de programacion en la run.

Proyecto generado:

```text
cmd/server/main.go
go.mod
internal/application/*
internal/domain/*
internal/http/*
internal/i18n/*
internal/memory/*
internal/ports/*
docs/*.md
```

Validacion externa del proyecto temporal:

```bash
cd /tmp/orquesta-real-complete-fixed-20260521/project
go test -count=1 ./...
```

Resultado: verde.

Defecto encontrado y corregido durante el smoke:

- El ACK de programacion declaraba artefactos de test bajo un write-set con
  `**/*_test.go`.
- `orquesta-runtime-codex` validaba globs con `path.Match`, que no entiende
  `**` como globstar recursivo.
- Efecto: `ReadCodexDeliveryObservationFileV0` devolvia
  `missing_required_artifact` y `/api/v0/runs/supervise` acababa en 500.
- Correccion: soporte explicito de globstar recursivo en
  `codexAckPathMatchesWriteSetEntryV0` y prueba
  `TestCodexAgentAckReceiptV0AceptaWriteSetConGlobstarRecursivo`.

Limitacion confirmada:

- El smoke de servidor encontro que el flujo app/director con `WorkflowTaskV0`
  registraba la entrega de programacion, pero no cerraba la task ni ejecutaba
  `RequiredTestRunner` como evidencia durable: `tasks_delivered=1`,
  `tasks_closed=0`, `required_test_evidence` vacio.
- Corte posterior 2026-05-22: queda cubierta por regresion integrada offline en
  `app-director-service`. `ContinueAppDirectorV0` ya puede tomar una microtarea
  nacida de `director_decisions`, reconocer su scope resuelto aunque el run
  global siga con otros agentes vivos, avanzar
  `wait -> review -> run_required_tests -> replan_or_close`, ejecutar el runner
  inyectado y cerrar task/run con evidencias causales.
- Corte posterior 2026-05-22: queda cubierta tambien la composicion de
  `cmd/orquesta-server`. `buildStackFromEnvV0` con
  `ORQUESTA_REQUIRED_TEST_RUNNER_ENABLED=1` ejecuta un `go test ./...` real sobre
  un modulo temporal, persiste `RequiredTestEvidenceV0` en `state-file`, avanza
  `PlanState` a `replan_or_close` y cierra el run sin filtrar rutas locales en
  los artefactos publicados.
- Corte posterior 2026-05-22: ese camino quedo validado en un caso real acotado
  por `CODEX-REQTEST-REAL-E2E`: un agente Codex vivo entrega, pasa por review
  causal aceptada, ejecuta `RequiredTestRunner`, persiste
  `RequiredTestEvidenceV0` y cierra plan/run con estado persistido. Sigue
  pendiente repetirlo para ola/cohorte amplia, recursion real y replan negativo
  con agente real.
- Corte posterior 2026-05-22: el bloqueo de shutdown ordenado con run activa
  pero sin agentes vivos queda corregido en `orquesta-server-shutdown`. Si
  `agents_in_flight=0` y no hay checkpoint pendiente, shutdown queda `ready`
  aunque existan refs de stop no confirmadas; con agentes vivos sigue
  `waiting_drain`.
- Corte posterior 2026-05-22: `cmd/orquesta-server` ya permite bajar la
  capacidad de razonamiento usada por el dispatcher de capacidad sin tocar el
  nucleo. `ORQUESTA_CODEX_REASONING_EFFORT=medium` alimenta tambien
  `Capacity.ReasoningEffort`; `ORQUESTA_CAPACITY_REASONING_EFFORT` y
  `ORQUESTA_CAPACITY_TIER` permiten sobrescribir esa politica desde la
  composicion.

## Reintento de smoke 2026-05-22 tras continuidad review

Cambio probado offline antes del reintento:

- `ContinueAppDirectorV0` abre `revision` con `OpenPhase` idempotente cuando el
  `PlanState` entra o reentra en `review_deliveries` y el run sigue en
  `programacion`.
- Si el loop acaba de consumir `wait_subagents`, el servicio ejecuta un pase
  acotado adicional para aplicar review gate sobre entregas ya disponibles.
- Prueba focal:
  `go test -count=1 ./modulos/orquesta-app-director-service -run TestContinueAppDirectorV0AvanzaDeWaitAReviewAbriendoRevision`.
- Suite completa local pasada: `go test -count=1 ./...`.

Smoke real intentado con director Codex real, `reasoning_effort=medium`,
`sandbox=danger-full-access`, `approval_policy=never` y
`ORQUESTA_REQUIRED_TEST_RUNNER_ENABLED=1`.

Directorio temporal purgado antes de ejecutar:

```text
/tmp/orquesta-real-review-autofollow-20260522
```

Resultado:

- Servidor temporal levantado en `http://127.0.0.1:35081`.
- `POST /api/v0/apps/director` devolvio 200 y creo
  `run-spec-inventario-review-autofollow-req-inventario-review-autofollow-18db032d`.
- Codex no llego a entregar ACK del director por limite de cuota antes de crear
  microtareas: el stderr del agente contiene `You've hit your usage limit` y
  pide reintentar a las 14:33.
- Stats al parar: fase `brainstorming_arquitectura`, `agents_started=1`,
  `agents_in_flight=0`, `tasks_total=0`, `tasks_closed=0`,
  `required_test_evidence=0`.
- Shutdown ordenado por `/api/v0/server/shutdown` devolvio 200 con
  `status=ready`, `shutdown_ready=true`, `agents_in_flight=0` y
  `checkpoints_pending=0`.

Este reintento no valida ni invalida el tramo nuevo con runtime real: quedo
bloqueado por cuota antes de que existiera una entrega a revisar. La evidencia
util de ese cambio queda en pruebas offline; el smoke real de app completa,
ola/cohorte amplia o recursion debe repetirse cuando haya cuota.

## Cierre acotado posterior 2026-05-22

Despues del reintento bloqueado por cuota se cerro el caso acotado
`CODEX-REQTEST-REAL-E2E` con `./scripts/smoke_codex_real_required_test_runner.sh`.

Resultado documentado:

- `TestCodexStackRealRequiredTestRunnerEndToEndOptInV0` paso con Codex real en
  64.49s;
- arranco un unico agente Codex real bajo `WaitAgentRefs`;
- hubo ACK/entrega, review causal aceptada, ejecucion del
  `RequiredTestRunner`, persistencia de `RequiredTestEvidenceV0` y cierre de
  plan/run;
- no toco OPES ni core.

Pendiente real tras este cierre:

- ola/cohorte amplia con varios agentes Codex vivos y scope formal del Director
  Operativo;
- recursion real con parent/child refs, profundidad/fanout, presupuesto y review
  causal;
- replan negativo con runner y agente real tras fallo de tests;
- cierre vivo de composiciones no-OPES y OPES reales con sus fuentes/validadores
  de dominio.
