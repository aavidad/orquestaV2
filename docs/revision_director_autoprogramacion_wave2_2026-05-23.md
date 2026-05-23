# Revision Director autoprogramacion wave2 - 2026-05-23

## Alcance revisado

Revision acotada de la ola `run-autoprog2-*` bajo el runtime de control de
autoprogramacion del 2026-05-23. Esta revision no toca codigo: solo consolida
evidencia disponible para el Director dentro del write-set `docs`.

Refs opacas que deben conservarse en la entrega actual:

- `worktree_ref=worktree-ref-run-autoprog2-director-review-wave2`
- `branch_ref=branch-ref-run-autoprog2-director-review-wave2`

Estas refs son identificadores opacos. No se interpretan como ruta de filesystem
ni como nombre Git.

## Resultado ejecutivo

La ola wave2 todavia no esta lista para cierre automatico global. Ya existen
ACK completados para `t04`, `t05`, `t07` y `t08`, pero siguen sin ACK durable
`t01`, `t02`, `t03` y `t06`. El primer informe del Director
`run-autoprog2-director-review-wave2` dejo ACK `failed`: el informe se creo,
`git diff --check` paso, pero el comando obligatorio amplio fallo en
`orquesta-app-codex-stack` por `TestDirectorTaskV0IncluyeContratoDeDecisionesEjecutables`
con objetivo demasiado grande.

El estado observable es mixto:

- Hay avance tecnico real en preservacion de `worktree_ref` y `branch_ref` como
  refs opacas.
- Hay cambios visibles de producto y docs repartidos entre los write-sets de la
  ola.
- Solo los runs con ACK pueden alimentar cierre causal; logs sin ACK no bastan
  para declarar `files`, `tests` ni `status`.
- La rama de revision debe conservar los identificadores opacos, sin convertir
  `worktree_ref` en path ni `branch_ref` en rama Git.

## ACKs observados

| Run | ACK | Estado | Tests declarados |
| --- | --- | --- | --- |
| `run-autoprog2-director-review-wave2` | Si | `failed` | `git diff --check` |
| `run-autoprog2-t01-cierre-autonomo-cola` | No | Sin entrega contractual | Ninguno declarable |
| `run-autoprog2-t02-cola-terminal-async` | No | Sin entrega contractual | Ninguno declarable |
| `run-autoprog2-t03-contrato-tareas-explicitas` | No | Sin entrega contractual | Ninguno declarable |
| `run-autoprog2-t04-reparacion-replan-laxo` | Si | `completed` | `go test -count=1 ./modulos/orquesta-orchestration-core ./modulos/orquesta-core-workflow ./modulos/orquesta-app-director-service ./modulos/orquesta-app-codex-stack ./modulos/orquesta-governance` |
| `run-autoprog2-t05-worktree-aislada-real` | Si | `completed` | `go test -count=1 ./modulos/orquesta-runtime-worktree ./modulos/orquesta-runtime ./modulos/orquesta-runtime-codex ./modulos/orquesta-runtime-codex-delivery ./cmd/orquesta-server` |
| `run-autoprog2-t06-observabilidad-operador` | No | Sin entrega contractual | Ninguno declarable |
| `run-autoprog2-t07-smoke-desatendido-real` | Si | `completed` | `go test -count=1 ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server ./modulos/orquesta-run-file ./modulos/orquesta-state-file` |
| `run-autoprog2-t08-opes-e2e-consumidor` | Si | `completed` | `go test -count=1 ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-connector ./modulos/orquesta-document-plan-expander ./modulos/orquesta-domain-work` |

## Cambios visibles a revisar

El worktree contiene cambios visibles relacionados con la ola. La lista no es
una aceptacion, solo inventario para review causal:

- `cmd/orquesta-server/codex_director_wave_command_v0.go`
- `cmd/orquesta-server/codex_director_wave_command_v0_test.go`
- `cmd/orquesta-server/codex_director_worktree_v0.go`
- `cmd/orquesta-server/startup_check.go`
- `cmd/orquesta-server/startup_check_candidates.go`
- `cmd/orquesta-server/startup_check_queue.go`
- `cmd/orquesta-server/startup_check_types.go`
- `modulos/orquesta-app-codex-stack/autoprogramming_scope_v0.go`
- `modulos/orquesta-app-codex-stack/autoprogramming_scope_v0_test.go`
- `modulos/orquesta-app-codex-stack/docs/contratos.md`
- `modulos/orquesta-app-codex-stack/spec_task_planner_bridge_v0_test.go`
- `modulos/orquesta-app-codex-stack/spec_task_programming_v0_test.go`
- `modulos/orquesta-autoprogramming/autoprogramming_programmable_work_v0.go`
- `modulos/orquesta-autoprogramming/autoprogramming_programmable_work_v0_test.go`
- `modulos/orquesta-autoprogramming/autoprogramming_task_contract_v0.go`
- `modulos/orquesta-autoprogramming/autoprogramming_task_group_v0.go`
- `modulos/orquesta-autoprogramming/autoprogramming_task_group_v0_test.go`
- `modulos/orquesta-autoprogramming/docs/contratos.md`
- `modulos/orquesta-autoprogramming/docs/pruebas.md`
- `modulos/orquesta-cli/*autoprogramming*.go`
- `modulos/orquesta-cli/command_runner_v0.go`
- `modulos/orquesta-cli/command_runner_v0_test.go`
- `modulos/orquesta-mcp/autoprogramming_*_v0.go`
- `modulos/orquesta-mcp/autoprogramming_*_v0_test.go`
- `modulos/orquesta-mcp/docs/contratos.md`
- `modulos/orquesta-opes-bridge/docs/contratos.md`
- `modulos/orquesta-opes-bridge/docs/pruebas.md`
- `modulos/orquesta-opes-bridge/mapper_v0.go`
- `modulos/orquesta-opes-bridge/mapper_autoprogramming_v0_test.go`
- `modulos/orquesta-opes-bridge/mapper_refs_v0.go`
- `modulos/orquesta-run-queue/*`
- `modulos/orquesta-runtime-worktree/docs/contratos.md`
- `modulos/orquesta-server/runtime_v0.go`
- `modulos/orquesta-server/supervisor_loop_v0.go`
- `modulos/orquesta-server/supervisor_loop_v0_test.go`
- `modulos/orquesta-web/autoprogramming_prepare_run_client_v0.go`
- `modulos/orquesta-web/autoprogramming_prepare_run_client_v0_test.go`
- `modulos/orquesta-web/director_stats_*_v0.go`
- `modulos/orquesta-web/director_stats_*_v0_test.go`

Los ficheros nuevos no trackeados observados deben incorporarse, descartarse o
devolver a rework con decision explicita. No deben quedar como evidencia
implicita sin ACK.

## Revision por run

| Run | Evidencia observada | Estado de revision |
| --- | --- | --- |
| `t01 cierre autonomo cola` | Log con cambios en cierre/supervision y ejecucion iniciada de `go test -count=1 ./modulos/orquesta-app-director-service ./modulos/orquesta-orchestration-core ./modulos/orquesta-state-file ./modulos/orquesta-app-codex-stack`; no hay ACK ni resultado final contractual. | Pendiente de ACK o rework. No cerrar. |
| `t02 cola terminal async` | Log con cambios en cola/supervisor, estado terminal y tick async; no hay ACK. | Pendiente de ACK y pruebas obligatorias. |
| `t03 contrato tareas explicitas` | Worktree muestra transporte de titulo, objetivo, contexto, criterios, tests y reglas compactas en autoprogramacion/MCP/CLI/web; log muestra conflicto de `apply_patch` al final y no hay ACK. | Pendiente de ACK o reparacion. Revisar coherencia de cambios concurrentes. |
| `t04 reparacion replan laxo` | ACK `completed`; declara cambios en `orquesta-app-codex-stack` para preservar `worktree_ref` y `branch_ref` en criterios de cierre y pruebas focales pasadas. | Aceptable para review causal si el diff coincide con el ACK. |
| `t05 worktree aislada real` | ACK `completed`; declara aislamiento opt-in en adaptador, branch opaca y rechazo de `branch_ref` con forma de ruta. | Aceptable para review causal si el diff coincide con el ACK. |
| `t06 observabilidad operador` | Log muestra cambios en MCP/CLI/web y pruebas focales de esos paquetes, pero no hay ACK. | Sin entrega durable; pedir ACK o rework. |
| `t07 smoke desatendido real` | ACK `completed`; declara test amplio de stack/cmd/run-file/state-file y test de preservacion de refs opacas. | Aceptable para review causal si el diff coincide con el ACK. |
| `t08 OPES consumidor` | ACK `completed`; declara mapper OPES con refs opacas y test obligatorio de bridge/conector/expander/domain-work. | Aceptable para review causal si el diff coincide con el ACK. |

## Hallazgos

1. Cierre global bloqueado por entregas sin ACK.
   `t01`, `t02`, `t03` y `t06` no aportan `agent_ack.json`; Orquesta no debe
   inferir entrega desde logs.

2. El informe anterior del Director fallo por prueba amplia.
   Hay evidencia documental y `git diff --check`, pero el comando obligatorio
   amplio no quedo verde por un test de objetivo demasiado grande. No debe
   declararse como prueba pasada.

3. Refs opacas preservadas en los ACK completados.
   `t04`, `t05`, `t07` y `t08` declaran explicitamente que `worktree_ref` y
   `branch_ref` se conservan como opacas y no se convierten en paths o ramas
   Git. Esa condicion debe mantenerse en review.

4. Hay cambios visibles sin cierre causal completo.
   Algunos ficheros modificados o nuevos corresponden a runs sin ACK. El
   Director debe asociarlos a ACK posterior, rework o descarte explicito.

5. OPES sigue como consumidor.
   La evidencia de `t08` queda en adaptador OPES/DomainWork; no mete OPES en el
   nucleo y conserva conector opt-in.

## Recomendacion al Director

No cerrar wave2 como `completed` global todavia.

Pasos recomendados:

1. Solicitar ACK durable o rework para `t01`, `t02`, `t03` y `t06`.
2. Reejecutar las pruebas obligatorias de cada paquete antes de aceptar cambios.
3. Corregir o acotar el fallo de
   `TestDirectorTaskV0IncluyeContratoDeDecisionesEjecutables` antes de aceptar
   el informe de cierre automatico.
4. Resolver todos los ficheros nuevos no trackeados con decision explicita.
5. Mantener `worktree_ref` y `branch_ref` como refs opacas durante cierre,
   review, rework y posible merge.
