# Handoff: terminar Orquesta al 100% — problemas y plan accionable

`doc_estado=vigente`. Fecha: 2026-06-19. Destinatario: agente Codex (vía orquestador
o directo) que vaya a cerrar los frentes abiertos para dejar Orquesta operativa y
autónoma de extremo a extremo.

Subordinado a `AGENTS.md`, `README.md`, `docs/estado_actual_2026-05-17.md` y
`docs/orquesta_director_y_entregas_2026-06-18.md` (este último detalla la filosofía y
los cambios ya hechos; aquí se consolida la lista de PROBLEMAS pendientes).

**Regla rectora (del operador):** los bugs que destapa una app de prueba se arreglan en
**Orquesta en general**, no parcheando para la app concreta. Conservar trabajo
recuperable y avisar para revisión, nunca colgarse por nimiedades. Solo cortes duros por
seguridad real, causalidad rota, refs imposibles, datos sensibles efectivos o efectos
externos no autorizados.

## Estado actual (lo que YA funciona, verificado con Codex real)

- Orquesta acepta un spec de app por JSON en `/api/v0/autoprogramming/prepare-run` con
  `write_set` y `depends_on` por tarea (sin harness por app).
- El director **respeta dependencias al lanzar**: lanza solo las tareas sin deps; las
  dependientes esperan.
- Los agentes Codex programan código real de calidad (probado: dominio hexagonal de la
  app Bolsa, compila, con máquina de estados y tests ejecutados).
- Smoke real Bolsa cerrado el 2026-06-19:
  - `bolsa-nucleo-001`: 8/8 tareas entregadas/cerradas, 8 reviews aceptadas, 1
    validacion y 1 cierre.
  - `bolsa-tests-001`: 4/4 tareas de tests entregadas/cerradas por supervisor residente,
    sin `runs/supervise` manual, con 4 reviews aceptadas, validacion final y cierre.
  - Verificacion externa: `go test -count=1 ./...` en
    `~/Trabajo/Bolsa_Diputacion_app` ejecuta paquetes `ok` reales; solo
    `internal/candidate/ports` queda como interfaz sin tests.
- Recuperación de artefacto-sin-ACK, streaming por sub-ola, validaciones soft, default
  reasoning `high`, guard de límite de replan: implementados y en verde (commits
  `053098e6`, `9823e450`).

## Reproducción de referencia

Spec real de app multi-tarea con dependencias hexagonales:
`~/Trabajo/Bolsa_Diputacion_app/orquesta_spec_nucleo.json` (8 tareas:
base→{baremo,ports,auth,i18n}→{usecase,repo}→http).

Arranque del servidor con Codex real (OAuth en `~/.codex`, gestionado por Orquesta):
```bash
ORQUESTA_SERVER_ADDR=127.0.0.1:8799 \
ORQUESTA_SERVER_STATE_DIR=<dir-estado> \
ORQUESTA_CODEX_PROJECT_WORKDIR=~/Trabajo/Bolsa_Diputacion_app \
ORQUESTA_CODEX_RUNTIME_WORKDIR=<dir-runtime> \
ORQUESTA_CODEX_HOME=$HOME ORQUESTA_CODEX_CODE_HOME=$HOME/.codex \
ORQUESTA_CODEX_COMMAND=$(command -v codex) ORQUESTA_CODEX_PATH=$PATH \
ORQUESTA_CODEX_MODEL=gpt-5.5 ORQUESTA_CODEX_APPROVAL_POLICY=never \
ORQUESTA_CODEX_SANDBOX=workspace-write ORQUESTA_OPES_BASE_URL= OPES_BASE_URL= \
  ./orquesta-server run
# luego: POST el spec a /api/v0/autoprogramming/prepare-run
# el supervisor residente del servidor debe drenar la cola hasta cierre.
```

---

# ESTADO DE CIERRE Y PENDIENTES (orden de prioridad)

## P1 — CERRADO real: frontera de tareas con dependencias

El bloqueo observado con Bolsa no se cerró cambiando `WaitAgentRefs`. La causa principal
era que `AutoprogrammingTaskGroupCandidateV0.depends_on` llegaba en refs fuente del spec
(`bolsa-base`, etc.), pero el scheduler satisface dependencias contra refs
`WorkflowTaskV0.TaskID` ya materializadas. Ahora `orquesta-autoprogramming` remapea refs
fuente a `task-autoprogramming-...`, conserva refs opacas desconocidas y evita
autodependencias cuando varias tareas acaban agrupadas.

El stack Codex también calcula `WaitAgentRefs` como frontera viva del run:
- run nuevo: solo tareas sin dependencias satisfechas;
- replay/existing run: tareas no terminales cuyas dependencias aparecen en
  `DeliveredTasks` o `ClosedTasks`;
- tareas entregadas/cerradas: fuera de `WaitAgentRefs`.

Esto conserva la regla vigente: `WaitAgentRefs` acota pending/wait/ingesta y no debe
rellenarse con agentes futuros no materializados.

**Evidencia 2026-06-19:**
- `TestBuildAutoprogrammingProgrammableWorkV0HonraWriteSetYDependsOnPorTareaV0`.
- `TestCodexStackAutoprogrammingRunGlobalTickV0RelanzaFronteraDependienteTrasACKV0`.
- `TestCodexStackAutoprogrammingPrepareRunAPIV0TickGlobalTrasACKLanzaFronteraDependienteV0`.
- `TestAutoprogrammingResidentModeV0RelanzaFronteraDependienteTrasACKV0`.
- `go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-app-codex-stack
  ./modulos/orquesta-app-director-service ./modulos/orquesta-orchestration-core`.

**Evidencia real 2026-06-19:**
- `bolsa-nucleo-001` sobre `~/Trabajo/Bolsa_Diputacion_app/orquesta_spec_nucleo.json`
  cerró 8/8 tareas con Codex real (`status=cerrada`, `phase=cierre`, 8 reviews
  aceptadas, 1 validacion, 1 closure).
- `bolsa-tests-001` cerró 4/4 tareas adicionales de tests y demostró la frontera viva:
  arrancó `g01+g03`, después `g02` tras ACK de `g01`, y finalmente `g04` tras ACK de
  `g02+g03`, sin relanzar tareas ya entregadas/cerradas.

Si falla un caso futuro, abrir incidencia nueva con evidencia del run; no reabrir
`queuedOperationalDirectorWaitAgentRefsV0` salvo regresión demostrada.

## P2 — CERRADO real acotado: autonomía residente en autoprogramación

El run `bolsa-tests-001` se preparó por `/api/v0/autoprogramming/prepare-run` y se dejó
en manos del servidor. El `SupervisorLoopV0` drenó la cola global, lanzó las fronteras
dependientes, esperó ACK/deliveries, abrió revisión, ejecutó validación final y cerró el
run sin `POST /api/v0/runs/supervise` manual.

Queda recomendable convertir esa evidencia en script/runbook repetible para regresión
operativa, pero el bloqueo funcional de autonomía residente observado en Bolsa queda
cerrado para este camino.

## P3 — Incidencia A2: bucle del supervisor sobre runs ya completados

Documentado en `docs/incidencia_opes_supervise_ack_reconciliation_a2_2026-06-13.md`. Un
`active_step` que no progresa sobre un run ya entregado se eleva a `tick_error`
permanente y el supervisor entra en bucle; `runs/supervise` puede devolver 500; shutdown
atascado. El arreglo correcto vive en la capa **supervisor/composición**, no en el
contrato del plan-state (intentos previos rompían tests del contrato). Hacer
`runs/supervise` idempotente sobre runs ya entregados y reconciliar a cerrado en vez de
`tick_error`.

## P4 — CERRADO mínimo: required-tests no aceptan `go test` vacío

Observado en Bolsa: el agente ejecutó los `RequiredTests` (`go test ./...` con receipt
`passed`) pero NO escribió ficheros `*_test.go` propios del dominio, aunque el spec lo
pedía en `acceptance_criteria`. El gate de Orquesta funciona (exige que los required
tests pasen), pero no verifica acceptance_criteria semánticos ("incluye tests del
dominio").

Mitigación genérica 2026-06-19: `LocalCommandExecutorV0` marca como `failed` un
`go test` que no ejecuta ningún paquete con tests reales y solo produce marcadores
`[no test files]`/`[no tests to run]`. Esto evita que Orquesta cierre una app Go con
evidencia vacía. Bolsa se reparó por Orquesta con el run `bolsa-tests-001`, que añadió
tests reales en dominio, auth, i18n, repositorio, casos de uso y handler HTTP.

Pendiente de calidad no bloqueante: validar acceptance criteria semánticos más allá del
runner de comandos, preferiblemente mediante required-tests más específicos por tarea o
review de aceptación por agente.

## P5 — Frentes abiertos heredados (de docs/estado_actual_2026-05-17.md)

- Seguridad operativa, multi-tenant, TLS/mTLS, RBAC, auditoría fuerte (antes de
  despliegue amplio).
- OPES temporal real de derivados/cierre quedo cerrado por el runbook real del
  2026-06-28; conservar como frente abierto solo residuales editoriales, coste,
  automatizacion larga o regresiones demostradas.
- Condición sospechosa de yield de progreso a replan
  (`schedulerProgressCandidateCanYieldToReplanV0`, `scheduler_tick_progress_candidate_v0.go`):
  `started && !failed && stopped` parece contradictoria; revisar si la supervisión de
  progreso cede a replan cuando debe.
- Revisar capacidad sin tope duro explícito (sin circuit breaker global de paralelismo).

---

## Orden recomendado para el agente Codex

1. **P3 (bucle A2)** — robustez del supervisor sobre runs ya terminales.
2. **Script/runbook de regresión Bolsa** — convertir la evidencia real en smoke
   repetible opt-in.
3. **P4 semántico, P5** — calidad de acceptance criteria y frentes heredados.

Cada arreglo: reproducir con test que falla → arreglar genérico → `go test ./...` en
verde → no romper OPES/recursión/olas. Tras Bolsa real y P2, el propio Orquesta puede
orquestar el resto de forma residente; Codex directo queda para observar y reparar
tapones genéricos.
