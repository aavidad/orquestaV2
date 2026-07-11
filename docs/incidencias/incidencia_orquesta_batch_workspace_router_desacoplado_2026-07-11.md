# Incidencia: batch con worktrees provisionados pero Codex en checkout canonico

Fecha: 2026-07-11  
Bug: `BUG-ORQ-20260711-255`  
Estado: abierto; piloto detenido y evidencia conservada

## Resumen

Un `POST /api/v0/autoprogramming/prepare-run` real preparo un batch de dos
goals con write-sets disjuntos y dos `GoalWorkspaceV0` fisicos. Sin embargo,
los dos threads Codex app-server nacieron con el checkout canonico como `cwd`.
Los worktrees provisionados quedaron limpios y el cambio de uno de los goals
aparecio directamente en el checkout canonico.

El aislamiento estaba declarado y persistido, pero no gobernaba el proceso que
ejecutaba el goal. El guard de write-set de g01 observo entonces el cambio
valido de g02 como escritura ajena y bloqueo el goal con
`codex_app_server_runtime_write_set_violation`.

## Evidencia del piloto

- request: `request-ref-live-bug245-cleanup-20260711`
- batch: `batch-ref-autoprogramming-6e1fa480b73d8393c503d15bbacbb9506890564c23be2bd6dba107147721d8c4`
- base congelada: `79a82182e373600bcb8453d13d55e96d56b58010`
- miembros: dos workspaces distintos, `7d76a7044d132df48cb56f7e1e2909d5`
  y `c59c59661e88092a6d571656cd49cea8`
- ambos worktrees fisicos quedaron sin diff
- el `session_meta` y `thread_settings_applied` de ambos Codex registraron el
  checkout canonico como `cwd`
- g01 termino bloqueado al detectar como path externo el fichero autorizado
  exclusivamente para g02
- g02 produjo el cambio solicitado en el checkout canonico, pero su cierre
  quedo divergente por identidad y requirio rework
- el agregado batch permanecio en `goals_running`, sin integrar ni ejecutar el
  gate global; no hubo falso cierre del batch
- la parada cooperativa `/api/v0/server/shutdown` devolvio `shutdown_ready=true`
  y limpio el backend tmux antes de salir

La evidencia runtime completa se mantuvo durante el diagnostico bajo un root
temporal local. No se versionan transcripts ni bases de datos del proveedor;
este documento conserva las refs y hechos necesarios para reproducir y auditar.

## Causa estructural

`serverCodexGoalBackendsFromEnvV0` construye dos backends:

- `AppGoal` sin `physicalGoalWorkspaces`;
- `IdleGoal` con `physicalGoalWorkspaces=true` y `WorkspaceRouter`.

La superficie publica `prepare-run` usa el launcher/observer de `AppGoal`,
mientras el coordinador batch provisiona worktrees fisicos por separado. Por
tanto, provision, ejecucion, observacion, atestacion e integracion no comparten
la misma identidad de workspace.

Los unitarios del router solo demostraban que un backend que ya lo tuviera
inyectado enviaba el CWD correcto. No habia prueba de composicion que demostrase
que el launcher real de autoprogramacion recibia ese backend.

## Riesgo

- contaminacion cruzada entre goals paralelos;
- atribucion falsa de cambios fuera de write-set;
- cambios directos en la rama canonica antes del gate batch;
- atestaciones ejecutadas sobre una revision distinta de la workspace declarada;
- rework incapaz de reparar un contrato cuya identidad de ejecucion ya divergio.

## Criterio de cierre

1. La composicion selecciona para todo goal batch de autoprogramacion el mismo
   `GoalWorkspaceV0` en launch, observe, fingerprint y atestacion.
2. Los goals normales de apps conservan su semantica; no se aislan sin un camino
   causal de integracion definido.
3. Una prueba de wiring falla si `prepare-run` batch termina usando el CWD
   canonico o si el workspace declarado no coincide con el ejecutado.
4. Replay real de dos goals disjuntos: cada diff nace solo en su worktree, el
   checkout canonico permanece limpio hasta la integracion gobernada, se crean
   dos commits encadenados, el gate global corre una sola vez y el batch cierra.
5. Repetir la misma request no relanza goals, no duplica commits ni repite el
   gate.
6. Cualquier divergencia de identidad bloquea antes de editar, no despues de
   atribuir el cambio a otro goal.

## Relacion con otros bugs

- profundiza `BUG-ORQ-20260711-244`: la provision fisica existe, pero el wiring
  real no la usaba;
- impidio validar `BUG-ORQ-20260711-245` con proveedor real;
- confirma que `BUG-ORQ-20260711-246/247/249/250` necesitan identidad de
  ejecucion probada de extremo a extremo, no solo un batch/store correcto.
