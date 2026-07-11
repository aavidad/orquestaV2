# Incidencia: goals paralelos comparten worktree y se bloquean entre si

Fecha: 2026-07-11
Estado: en correccion; aislamiento y routing implementados, integracion/batch pendientes
Area: autoprogramacion goal-first / paralelismo / worktree / atestacion

## Contexto

Orquesta recibio una ola real de limpieza con dos tareas y write-sets
disjuntos:

1. registro canonico de `cmd/orquesta-guardian`;
2. renombrado 100% de un helper de tests en `cmd/orquesta-server`.

La particion creo dos goals y los lanzo en paralelo, pero ambos Codex usaron el
mismo `ProjectWorkDir` fisico. El contrato declaraba `worktree_isolated=true`,
aunque esa aislamiento era solo una ref opaca y no una separacion efectiva por
goal.

## BUG-ORQ-20260711-244: violacion cruzada entre goals hermanos

- request: `request-ref-cleanup-wave-20260711-001`;
- goal Guardian/run 01: `goal-ref-task-autoprogramming-7e21686d5967-g01` /
  `request-ref-cleanup-wave-20260711-001-goal-01`;
- goal rename/run 02: `goal-ref-task-autoprogramming-7e21686d5967-g02` /
  `request-ref-cleanup-wave-20260711-001-goal-02`.

Mientras g01 leia Guardian, g02 renombro:

```text
cmd/orquesta-server/startup_check_test_helpers.go
-> cmd/orquesta-server/startup_check_helpers_test.go
```

Al observar g01, su baseline global detecto esos dos paths fuera de su
write-set Guardian y publico `codex_app_server_runtime_write_set_violation`.
El resultado de g01 incluyo como `artifact_paths` los ficheros producidos por
g02. No hubo escritura indebida del agente Guardian: fue contaminacion entre
dos goals autorizados por la misma ola.

### Lectura estructural

No es seguro permitir simplemente todos los write-sets hermanos en el guard:
entonces g01 podria escribir en el scope de g02 sin atribucion. La solucion es
una frontera externa de worktree por goal:

- materializar worktree/branch efimero desde un baseline comun;
- resolver `CWD` y snapshot/attestor por `goal_ref`;
- persistir ownership y lease del worktree;
- ejecutar goals disjuntos en paralelo real;
- tras closure y atestacion, integrar commits en orden gobernado sobre una unica
  rama canonica, con recibo y deteccion de conflicto;
- retener la rama ante conflicto y limpiar solo despues de integracion durable.

Hasta que esa ruta exista, compartir un worktree obliga a serializar por
seguridad; no puede declararse paralelismo funcional.

## BUG-ORQ-20260711-245: renombrado autorizado no cuenta como progreso

G02 tenia ambas rutas en su write-set y el objetivo exigia literalmente el
movimiento Git. El contenido era identico, el test Guardian paso y el focal
Startup paso. Aun asi, `VerifyWorktreeWriteSetV0` devuelve siempre
`renamed_or_moved_path`; el governor lo redujo a `material_class=none` y el
runtime no dispone de un contrato tipado para autorizar ese cambio destructivo
exacto.

El movimiento valido se verifico byte a byte y se integro manualmente como
`269de5ad2`, sin cambios de contenido. Esta integracion no cierra BUG-245: una
tarea autonoma de limpieza sigue sin poder declarar un rename/delete previsto.

### Direccion de cierre

Transportar autorizaciones destructivas exactas y tipadas desde request/tarea
hasta Goal y `WorktreeVerifyRequestV0`, al menos:

- `rename`: previous_path + current_path;
- `remove`: path;
- `truncate`/`replace_large_delta`: path solo con autorizacion explicita y
  review causal.

El verificador conserva evidencia advisory de todo cambio destructivo, pero
solo bloquea si no coincide con una autorizacion exacta. No se deben inferir
permisos desde palabras del objetivo.

## Atestacion observada

Los required tests globales del request se fusionaron en ambos goals. G02
obtuvo un receipt independiente `passed` para uno; un segundo claim quedo
`goal_required_test_attestor_infrastructure_failed` durante observacion
concurrente HTTP/residente. Se enlaza como evidencia adicional de 208AC y del
scheduler de atestaciones pendiente; no se abre identificador duplicado.

## BUG-ORQ-20260711-246: tests globales replicados por goal

El compilador fusionaba `request.required_tests` dentro de cada grupo. En una
ola de dos goals, ambos ejecutaban los tests globales sobre el mismo arbol
mutable, ademas de sus tests focales. Esto duplicaba coste y permitia resultados
dependientes del estado transitorio del hermano.

`569e16287` separa `BatchRequiredTests` de los tests focales. En multi-goal,
cada goal recibe solo tests de tarea, acceptance checks y guards contractuales;
un grupo sin test focal se rechaza antes del launch. Sigue pendiente ejecutar
los tests de batch una sola vez sobre la revision integrada antes del cierre
definitivo.

## BUG-ORQ-20260711-247: commit aislado publicado como integrado

`autoprogrammingPromotionEffectWithIntegrationStatusV0` inferia
`integration_status=integrated` cuando el puerto devolvia `promoted|clean` sin
estado explicito. Esa inferencia era tolerable mientras el puerto operaba sobre
el checkout canonico, pero se convierte en falso verde cuando el commit vive en
una worktree fisica del goal.

La correccion exige que el adaptador produzca un recibo durable de integracion:
commit del goal verificado, lock multiproceso, cherry-pick/merge gobernado sobre
el checkout de integracion, conflicto con abort y retencion, y replay que
demuestre que el commit integrado sigue en la historia. Un commit aislado sin
ese recibo queda `pending_integration`, nunca `integrated`.

## Avance implementado

- `0dce780b3`: autorizaciones destructivas exactas en el verificador; falta
  completar su transporte desde tarea hasta packet/runtime.
- `ba33d04b3`: provisionador Git idempotente de worktree fisica por goal.
- `c4bdedf0c`: app-server resuelve el CWD por goal en start/observe/fingerprint.
- `ca16cbfdd`: el stack captura cada baseline en su workspace y rechaza un
  multi-goal sin provisionador fisico.
- `023cdc486` y `d2ce9f80c`: binding durable reiniciable y wiring exclusivo del
  backend de autoprogramacion; apps externas no cambian de workspace.
- `872eed05a`: `goal_ref` viaja por el contrato causal de promocion.
- `b88c7428a`: progreso material y write-set se verifican en el workspace del
  goal, no en el checkout compartido.

Estos commits cierran la contaminacion durante ejecucion, pero BUG-244 no se
declara cerrado hasta integrar dos commits paralelos, ejecutar el gate global y
limpiar solo workspaces ya integrados.

## Criterios de cierre

1. Dos goals con write-sets disjuntos ejecutan en worktrees fisicos distintos.
2. Ningun goal observa paths del hermano ni puede escribirlos.
3. Ambos commits verdes se integran una sola vez en la rama canonica con recibo.
4. Un conflicto de integracion bloquea y conserva ramas/evidencia, sin perder
   trabajo ni falsear cierre.
5. Rename exacto autorizado cuenta como `material_class=diff`, conserva
   evidencia destructiva y puede atestarse.
6. Rename/remove no autorizado sigue bloqueado.
7. Shutdown limpia procesos y worktrees ya integrados, pero no evidencia ni
   ramas bloqueadas.
8. Los tests globales se ejecutan una vez sobre la revision integrada y su
   atestacion no pertenece a ningun implementador individual.

El shutdown de la ola devolvio `shutdown_ready=true`, retiro backend/tmux y el
servidor salio con codigo cero.
