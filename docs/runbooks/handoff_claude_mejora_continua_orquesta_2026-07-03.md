# Handoff para Claude - mejora continua Orquesta 2026-07-03

## Proposito

Este documento deja el estado para revision posterior de Claude tras la tanda de
mejora continua dirigida con Orquesta y reparaciones puntuales integradas por
Codex. No es un cierre total de la app ni de todos los conectores: es el corte
verificable de la ola actual y de las incidencias observadas.

## Estado corto

- Fecha de corte: 2026-07-03.
- Worktree: con cambios sin commit.
- Procesos: tras la limpieza no quedaban procesos `orquesta-server run`,
  `codebase-memory-mcp` ni `codex app-server --listen`.
- Verificacion global ejecutada: verde con `go test -count=1 ./...`,
  `go build ./...` y `git diff --check`.
- Excepcion operativa: MEJ-206 se lanzo por Orquesta, pero el piloto quedo sin
  progreso util con consumo alto de tokens y checkpoint invalido. Se paro el
  runtime y Codex integro una reparacion acotada, dejando bugs documentados.

## Frentes integrados

### T285 - despertar por resultado materializado

Estado: integrado y probado.

Archivos principales:

- `modulos/orquesta-app-codex-stack/goal_materialized_result_watcher_v0.go`
- `modulos/orquesta-app-codex-stack/goal_materialized_result_watcher_v0_test.go`
- `modulos/orquesta-server/runtime_background_worker_v0.go`
- Wiring en `cmd/orquesta-server/stack.go` y paquetes `modulos/orquesta-server`.

Resultado funcional: el stack observa resultados materializados y despierta el
runtime/background worker cuando hay evidencia nueva.

### MEJ-202 - property tests y falsos verdes

Estado: integrado y probado.

Archivos principales:

- `modulos/orquesta-run-coordinator/external_work_reconciliation_v0.go`
- `modulos/orquesta-run-coordinator/external_work_reconciliation_properties_v0_test.go`
- `modulos/orquesta-runtime-codex-appserver/estado_backend_v0.go`
- `modulos/orquesta-runtime-codex-appserver/estado_backend_properties_v0_test.go`
- `modulos/orquesta-estado-vivo/proyeccion_properties_v0_test.go`
- `modulos/orquesta-estado-vivo/testdeps/rapid`
- `go.mod`

Nota para Claude: `pgregory.net/rapid` se resuelve mediante reemplazo local de
test en `modulos/orquesta-estado-vivo/testdeps/rapid`; revisar sin convertirlo
en dependencia productiva.

### MEJ-203 - piloto de mutation testing

Estado: integrado y probado.

Archivos principales:

- `scripts/orquesta_mutation_pilot.sh`
- `scripts/test_orquesta_mutation_pilot.sh`
- `docs/runbooks/mutation_testing_piloto_2026-07-04.md`
- Evidencias bajo `scripts/docs/`.

Resultado funcional: harness acotado de mutacion con runbook y smoke local.

### MEJ-205 - golden evals

Estado: integrado y probado.

Archivos principales:

- `scripts/orquesta_golden_evals.sh`
- `docs/evals/`
- `docs/runbooks/orquesta_golden_evals_2026-07-04.md`

Resultado funcional: harness de evaluaciones doradas para detectar regresiones
en casos compactos.

### MEJ-206 - habilidades curadas para autoprogramacion

Estado: integrado por reparacion acotada tras piloto Orquesta bloqueado.

Archivos principales:

- `modulos/orquesta-autoprogramming/curated_skills_v0.go`
- `modulos/orquesta-autoprogramming/curated_skills_v0_test.go`
- `cmd/orquesta-server/idle_self_improvement_skills_v0.go`
- `cmd/orquesta-server/idle_self_improvement_skills_v0_test.go`
- `cmd/orquesta-server/idle_self_improvement_stack_v0.go`
- `cmd/orquesta-server/stack.go`
- `modulos/orquesta-server/supervisor_idle_goal_first_v0.go`
- `modulos/orquesta-server/supervisor_loop_v0_test.go`

Resultado funcional:

- Carga metadata compacta desde `skills/*/SKILL.md` solo desde composicion/cmd.
- Filtra entradas inseguras o con rutas absolutas/secretos.
- Enriquecimiento de peticiones idle con `SkillRefs` y `context_refs` de tipo
  `skill_ref`.
- No genera ni comitea automaticamente `SKILL.md`; solo puede proponer
  destilacion revisable.

Nota para Claude: no clasificar MEJ-206 como cierre autonomo de Orquesta. El
runtime se uso, pero quedo bloqueado; la integracion final es una excepcion de
reparacion local documentada.

## Bugs documentados

Inventario actualizado en `docs/inventario_bugs_orquesta_2026-06-30.md` con:

- `BUG-ORQ-20260703-154`: MEJ-206 consumio muchos tokens sin progreso util y
  dejo checkpoint invalido.
- `BUG-ORQ-20260703-155`: MEJ-206 mezclo criterios de habilidades curadas con
  criterios idle/APG no relacionados.
- `BUG-ORQ-20260703-156`: pilotos previos dejaron procesos `orquesta-server`
  y `codex app-server` vivos tras la entrega.
- `BUG-ORQ-20260703-157`: el launcher background/nohup del piloto podia quedar
  matado por el wrapper mientras el estado parecia vivo.

Estos bugs quedan abiertos para analisis estructural posterior; no deben
tratarse como anecdotas aisladas.

## Pruebas ejecutadas

Verificaciones focales:

```bash
go test -count=1 ./modulos/orquesta-autoprogramming
go test -count=1 ./modulos/orquesta-server -run 'TestRuntimeV0IdleSelfImprovementGoalFirstPropagaSkillRefsDeRequestV0'
go test -count=1 ./cmd/orquesta-server -run 'TestServerStackSupervisorV0FiltroAnadeSkillCuradaCasadaV0|TestServerCuratedSkillsFromProjectV0'
go test -count=1 ./modulos/orquesta-autoprogramming ./cmd/orquesta-server ./modulos/orquesta-server
go test -count=1 ./
```

Verificaciones globales:

```bash
go build ./...
git diff --check
go test -count=1 ./...
```

Tras documentar bugs y este handoff se debe repetir al menos:

```bash
git diff --check
```

## Limpieza operativa

Se pararon los pilotos y backends temporales asociados a los workdirs bajo
`/tmp/claude-1000/.../scratchpad/pilot-*`.

Comprobacion final usada:

```bash
pgrep -af 'orquesta-server run|codebase-memory-mcp|codex app-server --listen'
```

Resultado observado: sin salida.

## Advertencias para revision de Claude

- No integrar truncados de `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
  generados en worktrees piloto. En main no se integro ese cambio.
- No reabrir T285, MEJ-202, MEJ-203, MEJ-205 ni MEJ-206 salvo regresion
  demostrada por prueba o diff concreto.
- No usar `codebase-memory-mcp` para esta revision salvo consulta de grafo muy
  concreta; para este handoff bastan `rg`, `git diff` y tests.
- Revisar especialmente MEJ-206: aislamiento de composicion, ausencia de rutas
  absolutas/secretos en metadata compacta, y propagacion de `SkillRefs` a
  `GoalWorkSpec`.
- Verificar que las evidencias de pilotos no se confunden con codigo productivo.

## Pendiente aproximado

Ola actual cerrada en terminos de integracion verificable:

- T285.
- MEJ-202.
- MEJ-203.
- MEJ-205.
- MEJ-206, con excepcion documentada.

Siguiente ola recomendada:

- MEJ-201.
- MEJ-204.
- MEJ-104, para presupuesto/no-progreso y parada cooperativa.

Ola posterior:

- MEJ-102.
- MEJ-105.
- MEJ-207.

Condicionales, si siguen vigentes tras revision de backlog:

- MEJ-101.
- MEJ-103.
- MEJ-106.

## Checklist de Claude

1. Revisar `git status --short` y separar cambios de cada frente.
2. Revisar diff de MEJ-206 con foco en contratos y frontera core/composicion.
3. Confirmar que `docs/inventario_bugs_orquesta_2026-06-30.md` contiene los
   bugs 154-157.
4. Ejecutar `git diff --check`, `go test -count=1 ./...` y `go build ./...`.
5. Confirmar que no quedan procesos vivos con el `pgrep` indicado arriba.
6. Si todo sigue verde, preparar commit o siguiente ola sin relanzar los pilotos
   cerrados.
