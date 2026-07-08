# Incidencia: automejora remota goal-first desreconcilia receipts y estado

Fecha: 2026-07-01

Entorno observado:

- Remoto aislado: `berserk@uso.dipgra.cloud`.
- Worktree: `/srv/orquesta-self/runtime/audit-225a6752-next`.
- HEAD cargado en servidor aislado: `483a7a9b39`.
- Servidor: `127.0.0.1:18787`, sin tocar produccion ni puertos externos.

## Sintomas

Tras reiniciar el servidor aislado con el worktree actualizado, Orquesta activo
automejora residente goal-first. `autoprogramming/status` paso a publicar
`running_live=7` y accion `observe_active_goals:goals`.

Durante esa automejora:

- APG-004 quedo `state.status=complete` en
  `state/orchestration-state/app_director_goal_states/...json`.
- El `last_result.artifact_refs` de APG-004 conserva
  `modulos/orquesta-autoprogramming/docs/orquesta_goal_result_v0.json`.
- El mismo fichero quedo borrado del worktree:
  `git diff --name-status` mostraba
  `M modulos/orquesta-autoprogramming/docs/orquesta_goal_result_v0.json` como
  deleted.
- `status` dejo de contar APG-004 como vivo, pero no lo proyecto en
  `resolved_runs`; quedaron 6 goals vivos.

Tambien se observo que una primera llamada manual a
`POST /api/v0/autoprogramming/goal/observe` para T260 devolvio HTTP 400 con
`codex_goal_observation_rejected`; llamadas posteriores devolvieron HTTP 200
con `goal_status=running` y `recommended_action=observe_later`.

## Hipotesis arquitectonica

El lifecycle goal-first puede cerrar un goal por estado durable y conservar
artefactos en `last_result`, mientras el worktree queda con artefactos
terminales eliminados o no materializados. Falta una reconciliacion atomica
entre:

- estado del goal (`complete/running`);
- artifact refs terminales;
- existencia actual de los artefactos en el write-set;
- proyeccion publica `resolved_runs`;
- observacion activa/background.

Esto se relaciona con los bugs abiertos de goal-first sobre artefactos parciales
y receipts terminales, especialmente BUG-ORQ-20260701-072 y
BUG-ORQ-20260701-078. La diferencia aqui es que el caso se observo en
automejora de Orquesta contra el propio repo remoto, no en OPES.

## Evidencia operacional

Comandos usados:

```bash
POST /api/v0/autoprogramming/status
POST /api/v0/autoprogramming/goal/observe
POST /api/v0/autoprogramming/goals/observe-active
git diff --name-status
jq '.state | {status,last_result}' state/orchestration-state/app_director_goal_states/...json
```

Evidencia concreta:

- `observe-active` devolvio HTTP 202 con
  `autoprogramming_observe_active_goals_background_accepted`.
- `status` tras `observe-active` devolvio `running_live=6`.
- `APG-004` tenia `state.status=complete` y `last_result.artifact_refs`
  incluyendo el receipt eliminado.

## Accion esperada

- No marcar un goal como terminal completo si artifact refs terminales del
  write-set faltan o fueron eliminadas sin recibo de reconciliacion.
- Si el goal ya esta terminal pero falta un receipt versionado, publicar
  `missing_terminal_artifact_after_goal_complete` o accion equivalente, y no
  ocultarlo fuera de `resolved_runs`.
- `observe-active`/`status` deben mostrar terminales recientes con problemas de
  artefactos, no solo los vivos.
- La automejora no debe borrar receipts versionados sin crear reemplazo
  verificable o sin dejar rework explicito.

## Actualizacion 2026-07-08: cierre de codigo local

Se cierra la parte local verificable de la incidencia con el reason code
`terminal_artifact_missing_after_goal_complete`.

Cambio aplicado:

- `modulos/orquesta-app-codex-stack` detecta `GoalWorkState` terminal
  `complete` con `LastResult.ArtifactPaths` declarados dentro del write-set que
  ya no existen en disco.
- La deteccion no lo mezcla con `artifact_paths_omitted_materialized`: aqui el
  recibo si declaro la ruta, pero el artefacto desaparecio despues o no quedo
  materializado.
- `modulos/orquesta-mcp` propaga la senal a `observe_goal`,
  `director/stats` y `autoprogramming/status`; la accion recomendada es
  `replan`, no `repair_receipt`, porque reparar solo el recibo no recupera el
  fichero ausente.

Pruebas verdes:

- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestStackGoalMaterializedRefsSourceV0(DetectaArtifactPathDeclaradoPeroBorrado|DetectaArtifactPathsOmitidos|DetectaRequiredTestEvidence)'`
- `go test -count=1 ./modulos/orquesta-mcp -run 'Test(EnrichMCPObserveAppDirectorGoalWithMaterializedRefsV0QAFailedPublicTextRunningNoEspera|MCPDirectorStatsToolExecutorV0GoalFirstProyectaArtifactPathDeclaradoPeroBorrado|MCPAutoprogrammingStatusExecutorV0ArtifactPathDeclaradoPeroBorradoPideReplan)'`
- `git diff --check`

Estado: cerrado en codigo local para la clase "goal terminal con artefacto
declarado y ausente". Pendiente solo verificacion remota cuando el servidor
sincronice este commit y exista proveedor/cuota suficiente para repetir
automejora residente.

Inventario: enlazado como
`BUG-ORQ-20260701-RECEIPT-DESRECONCILIADO` en
`docs/inventario_bugs_orquesta_2026-06-30.md`.
