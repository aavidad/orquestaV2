# Handoff: Parent Autonomy Program DAG durable — 2026-07-10

Estado final: `ready_for_operator_autonomy_dag`.

## Corte realizado

Se agrega `modulos/orquesta-autonomy-program`, un contrato P0 genérico y puro
para que un **programa padre** sea el agregado durable de goals/tareas. No se
interpreta el primer goal como run global. El scope obligatorio es
`project_ref + root_ref + program_ref`; las refs de goals, runners y contextos
son opacas.

Cada nodo declara `depends_on`, `write_set` y `required_tests`. El scheduler
`PrepareAutonomyProgramFrontierV0` sólo materializa la frontera cuyas
dependencias están `accepted`; conserva en espera los solapes de write-set y
devuelve juntos los conjuntos disjuntos. Marca esa frontera `launched` antes
de entregar los comandos, por lo que persistir el retorno antes de actuar hace
el replay/restart idempotente.

`OperatorTaskV0` cambia un nodo lanzado a `wait_external`. No es un estado
documental terminal: `OperatorReceiptV0` causal y dentro del mismo
programa/proyecto/root lo devuelve a la frontera. El puerto
`AutonomyProgramActuatorPortV0` es la frontera para runner/operador; el contrato
no usa ni sustituye esa acción por `runs/control` legacy.

El cierre de un nodo requiere un receipt causal y atestaciones para todos sus
tests requeridos; `accepted` exige cada prueba `passed`. `blocked` conserva
atestaciones causales. `rework` vuelve a la frontera y está acotado a una
reentrada; el agregado sólo queda `completed`/`blocked` cuando todos sus nodos
han cerrado causalmente, nunca por autodeclaración.

`modulos/orquesta-state-file` implementa el puerto con JSON atómico bajo un
índice hash de `project_ref/root_ref/program_ref`. Tras recrear el store se
recuperan refs, estados y frontera, y un scope distinto no puede leer el
programa. `modulos/orquesta-autoprogramming` compila el trabajo aceptado a este
agregado, conservando su DAG, write-sets y tests, sin lanzar runtime.

## Relación con hallazgos

Ataca directamente F4 del análisis estructural del 2026-07-10: el write-set
ahora gobierna la frontera de scheduling durable, no sólo una prevalidación.
También evita el error de F1 de confundir proyecciones/runs con el estado
causal: programa, transiciones, receipts y pruebas son un contrato durable
separado. No resuelve F2/F3/F5/F6: no hay procesos, control plane, API,
deploy, proveedor ni app dentro de este corte.

## Evidencia focal

Las pruebas cubren:

- DAG de tres nodos y paralelismo de dos write-sets disjuntos;
- serialización de solape padre/hijo de ruta;
- replay sin relanzamiento y restart después del primer cierre, con refs
  estables;
- exactamente un rework;
- `wait_external` + receipt/resume y rechazo de receipt fuera de scope;
- `OperatorTaskV0` durable: reinicio entre wait y receipt, seguido de resume
  scoped sobre el mismo nodo;
- cierre causal con tests atestados y aislamiento por proyecto/root;
- arquitectura: el nuevo módulo no importa runtime, goal, state-file, SQL ni
  procesos.

Comando focal ejecutado con cachés fuera del repo:

```sh
mkdir -p /srv/orquesta-self/runtime/tmp-deploy/autonomy-dag-{tmp,gocache,gotmp}
TMPDIR=/srv/orquesta-self/runtime/tmp-deploy/autonomy-dag-tmp \
GOCACHE=/srv/orquesta-self/runtime/tmp-deploy/autonomy-dag-gocache \
GOTMPDIR=/srv/orquesta-self/runtime/tmp-deploy/autonomy-dag-gotmp \
go test -count=1 ./modulos/orquesta-autonomy-program \
  ./modulos/orquesta-state-file ./modulos/orquesta-autoprogramming
```

Revalidación posterior tras liberar `/tmp`: las tres suites focales pasaron con
`-race`; también pasó la frontera raíz
`TestNeutralOrchestrationPackagesDoNotImportProductAdapters`. Finalmente pasó
la batería amplia completa:

```sh
TMPDIR=/tmp/orquesta-autonomy-dag/tmp \
GOCACHE=/tmp/orquesta-autonomy-dag/gocache \
GOTMPDIR=/tmp/orquesta-autonomy-dag/gotmp \
go test -count=1 ./...
```

Los temporales y la caché se aislaron en `/tmp/orquesta-autonomy-dag` y se
eliminan al cerrar la sesión.

## Frontera siguiente

Una composición puede persistir el retorno del scheduler y llamar al actuador
por su puerto, guardando después receipts/cierres. Ese wiring debe ser una
tarea aparte, con write-set de la composición y evidencia propia; este corte no
autoriza `cmd/orquesta-server`, runtime providers, API real, deploy ni otras
apps.
