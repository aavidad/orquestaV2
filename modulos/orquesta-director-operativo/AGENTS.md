# Contexto para agentes: orquesta-director-operativo

Lee este archivo y `README.md` antes de editar el modulo.

## Reglas

- Mantener el modulo puro: contratos, validacion y planes compactos.
- No importar `cmd`, DB, filesystem productivo, red, web, MCP, runtime real,
  Codex, OPES, proveedores, modelos, HOME, OAuth ni credenciales.
- Programacion es caso de primer nivel, no una excepcion OPES.
- La delegacion recursiva es opt-in y gobernada por el director: profundidad,
  fanout, presupuesto y revision obligatoria.
- Si el contexto de dominio es insuficiente, el plan debe pedir contexto y no
  lanzar subagentes que inventen contenido.
- Si el trabajo de programacion no trae write-set, rama, worktree aislada y
  tests, el plan debe rechazarlo.
- No borrar pasos, campos o documentos porque parezcan provisionales sin revisar
  materializacion en `modulos/orquesta-orchestration-core` y referencias en docs.

## Frontera

Este modulo no ejecuta el bucle. El primer materializador real vive en
`modulos/orquesta-orchestration-core/operational_director_materializer_v0.go` y
convierte solo items `launch_subagents` listos a `WorkflowTaskV0` y comandos
`CreateMicrotask`.

Pendientes locales a preservar en contratos:

- refs de cohorte/ola para que `WaitAgentRefs` no sea el unico mecanismo de
  espera;
- estados y evidencias de `wait_subagents`, `review_deliveries`,
  `run_required_tests` y `replan_or_close`;
- parent/child refs y limites de recursion para Codex y dominios externos;
- bloqueo explicito cuando OPES/domain-work no trae contexto suficiente.
