# Handoff: atestación independiente 208H

Fecha: 2026-07-10. Estado: `ready_for_operator_attestation_208h`.

## Resultado

El cierre goal-first con política
`require_independent_required_test_attestation=true` ya no acepta
`required_test_results=passed` informado por el implementador. Ese campo queda
como diagnóstico no autoritativo. Para cada prueba requerida debe existir un
receipt durable, independiente y `passed`.

El contrato neutral reside en `modulos/orquesta-goal`:

- `GoalWorkSpecV0` congela `revision_ref`, `implementer_agent_ref`, write-set y
  los refs/hashes de cada required test (`command_ref`, `command_sha256`,
  `definition_sha256`).
- `GoalRequiredTestAttestationV0` exige agente y credencial del atestador,
  hashes antes/después, exit code, timestamps RFC3339, entorno aislado y refs
  de evidencia. El agente atestador no puede ser el implementador.
- `GoalRequiredTestAttestorPortV0` y
  `GoalRequiredTestAttestationStorePortV0` son puertos neutrales; no conocen
  shell, Codex, proveedores ni filesystem.
- `IndependentGoalRequiredTestAttestationClosureValidatorV0` solo acepta
  receipts que correspondan exactamente a run, goal, revisión, implementador,
  test, command ref y hashes congelados. `failed`, ausente, identidad igual,
  hash distinto o revisión stale devuelve `blocked` con código tipado y
  `needs_rework=true`.

`orquesta-state-file` persiste receipts por run de forma inmutable e
idempotente. Reutilizar la misma ref con payload distinto es conflicto. También
impide cambiar la `GoalWorkSpecV0` de un run existente, para no sustituir tests,
write-set o revisión después del lanzamiento.

La composición Codex solo cablea el atestador si el operador inyecta
`AppGoalRequiredTestAttestor` y un
`GoalRequiredTestAttestationStore`. `LocalGoalRequiredTestAttestorV0` es un
adaptador opt-in que delega a un executor local inyectado; no escoge ni ejecuta
comandos por sí mismo y no hay fallback al implementador.

## Replay y frontera operativa

Antes de pedir una atestación el lifecycle consulta receipts de la misma
`run_ref/goal_ref/revision_ref`. Si ya existen, no reinvoca al atestador; el
store hace la segunda barrera idempotente. Un receipt de otra revisión queda
fuera de la query y no puede cerrar la revisión actual.

No se activó runtime, proceso, API, deploy, proveedor ni aplicación. Para un
smoke futuro, el operador debe inyectar un executor aislado que emita el receipt
completo; la atestación solo se considera válida al persistirse y validarse por
el contrato, no por texto libre.

## Regresiones cubiertas

- implementador dice `passed`, atestador independiente falla;
- misma identidad implementador/atestador;
- hash de test mutado;
- `passed` independiente cierra;
- replay no vuelve a ejecutar atestador;
- varios tests y receipt de revisión stale;
- persistencia tras recrear `state-file`, conflicto de receipt y spec congelada;
- wiring opt-in y adaptador local inyectado sin fallback.

Comandos focales ejecutados con temporales y cache de compilación fuera del
worktree (`/tmp/attestation-208h-20260710`):

```bash
go test -count=1 ./modulos/orquesta-goal ./modulos/orquesta-state-file
go test -count=1 ./modulos/orquesta-app-director-service
go test -count=1 ./modulos/orquesta-app-codex-stack \
  -run 'Test(LocalGoalRequiredTestAttestorV0EsInyectadoYNoHaceFallback|BuildDirectorPortsV0CableaAttestorIndependienteOptIn)$'
```
