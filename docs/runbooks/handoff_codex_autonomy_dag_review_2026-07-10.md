# Handoff de auditoria adversarial: DAG durable — 2026-07-10

## Alcance y veredicto

Auditoria limitada al write-set autorizado del programa de autonomia,
compilador de autoprogramacion y store file-based. No se tocaron composiciones,
`uso-app`, otras apps, runtime, proveedor, deploy ni procesos residentes. No se
uso `codebase-memory-mcp`.

No quedan hallazgos altos abiertos dentro de este alcance. El handoff previo
`handoff_codex_autonomy_dag_2026-07-10.md` describe el corte anterior a esta
revision; se conserva sin editar y este documento gobierna las conclusiones de
la auditoria adversarial.

## Hallazgos demostrados y cierre

1. **Alto — falso paralelismo tras replay.** La seleccion comparaba candidatos
   solo con la frontera nueva. Un segundo tick podia lanzar una ruta hija de un
   write-set que seguia `launched`. La frontera compara ahora tambien con todos
   los nodos activos; se prueba padre/hijo y frontera de segmento (`scripts`
   no colisiona con `scripts2`).
2. **Alto — lost update y doble claim.** El store usaba un mutex por instancia
   y `Save` sobrescribia estados completos. Dos stores podian leer el mismo
   snapshot, cruzar ambos la frontera y pisarse. `Save` queda create-or-identical;
   las transiciones usan CAS del agregado completo, validacion monotona y lock
   de fichero por scope. `ClaimAutonomyProgramFrontierV0` solo devuelve launches
   despues de ganar el CAS.
3. **Alto — snapshot esperado mutado por aliasing.** Las funciones puras
   modificaban slices compartidos pese a recibir el agregado por valor. Eso
   alteraba el `expected` del CAS y rompia replay/contencion. La normalizacion
   clona profundamente nodos, pruebas, receipts y operator tasks antes de
   modificar.
4. **Alto — normalizacion de paths insegura.** El contrato quitaba `/` inicial
   antes de validar y podia aceptar `/tmp/x` como `tmp/x`. Ahora rechaza raiz,
   paths POSIX/Windows/UNC absolutos, `~`, URI, NUL y cualquier segmento `..`;
   normaliza separadores, `.` y slash redundante antes de comparar solapes.
5. **Alto — `wait_external` relanzaba el goal.** Un receipt devolvia el nodo a
   `pending` y reemitia el mismo lanzamiento. Ahora continua el launch durable,
   conserva el receipt completo y scoped, exige causalidad con operator task y
   launch, y no crea otra frontera. Las dependencias externas de `live_works`
   tienen `ExternalDependencyReceiptV0` durable y resoluble sin mutar el DAG.
6. **Alto — causalidad de cierre insuficiente.** Bastaban strings no vacios en
   tests/receipts. El cierre exige el `launch_ref` actual, status de prueba
   `passed|failed`, cobertura exacta sin duplicados, evidence/causal refs y
   pertenencia de cada prueba a la cadena del cierre. `accepted` exige todos los
   tests `passed`; refs de launch, goal, closure y receipts no pueden colisionar.
7. **Alto — crash entre persistencia y actuacion sin proyeccion de recovery.**
   `RecoverAutonomyProgramActionsV0` reconstruye el mismo launch o resume con
   las mismas refs. La entrega externa es deliberadamente at-least-once y el
   puerto exige deduplicacion por `launch_ref`/`receipt_ref`; no se declara un
   exactly-once externo ficticio.
8. **Medio — estados bloqueados no detenian dependientes.** Un nodo `blocked`
   podia dejar el agregado `active` con descendientes pendientes. El programa
   deriva `blocked` de forma fail-fast, no abre nuevas fronteras ni recupera
   acciones de un agregado terminal, aunque aun puede registrar observaciones
   tardias sobre trabajo que ya estaba en vuelo.
9. **Medio — DAG/refs incompletos.** Se anadio rechazo adversarial de ciclos de
   varios nodos, dependencias ausentes o duplicadas, ambiguedad interno/externo,
   goal refs duplicadas o iguales al parent goal, scopes vacios y receipts/launch
   refs reutilizados. El programa padre sigue siendo el agregado; ningun primer
   goal se promueve a programa global ni se introduce fallback legacy.

## Arquitectura conservada

- `orquesta-autonomy-program` solo contiene contratos, validacion, scheduler,
  transiciones, recovery y puertos; no importa filesystem, runtime, Goal,
  proveedor, procesos, SQL ni adaptadores.
- `orquesta-state-file` es el unico owner de JSON atomico, lock de fichero y
  CAS concreto.
- `orquesta-autoprogramming` solo compila trabajo ya aceptado al programa padre;
  no lanza runtime y conserva dependencias externas como refs opacas.
- No se usa `runs/control`, no se convierte el primer goal en run global y no
  se inventa ruta legacy.

## Pruebas ejecutadas

Con `TMPDIR`, `GOCACHE` y `GOTMPDIR` aislados bajo `/tmp/orquesta-dag-*`:

```sh
go test -count=1 ./modulos/orquesta-autonomy-program \
  ./modulos/orquesta-state-file ./modulos/orquesta-autoprogramming

go test -race -count=1 ./modulos/orquesta-autonomy-program \
  ./modulos/orquesta-state-file ./modulos/orquesta-autoprogramming

go test -count=1 . \
  -run '^TestNeutralOrchestrationPackagesDoNotImportProductAdapters$'

git diff --check
```

Resultados: las tres suites focales pasan, tambien con `-race`; la frontera
neutral raiz pasa. El test de contencion usa dos instancias independientes de
`StoreV0` sobre el mismo directorio y demuestra exactamente un claim durable,
CAS stale sin swap, rechazo de rollback y rechazo de overwrite ciego.

## Residuales no altos

- No hay wiring de composicion ni actuador real en el write-set. La futura
  composicion debe usar `Claim...`, persistir cada transicion con CAS y hacer
  sus operaciones idempotentes por las refs durables. Eso requiere evidencia
  propia en el modulo consumidor y no autoriza tocar apps desde este corte.
- `syscall.Flock` da exclusion entre procesos para el store local Linux actual.
  Un store distribuido o filesystem sin semantica flock debe implementar el
  mismo puerto con CAS nativo y su propia prueba de contencion.
- No se actualizo el inventario global de bugs porque queda fuera del write-set
  expreso. Los hallazgos y su evidencia durable quedan registrados aqui para
  que el operador los enlace si amplia el alcance.
- La deduplicacion evita efectos duplicados solo si el actuador cumple el
  contrato de idempotencia. Sin un adaptador concreto no se afirma exactly-once
  externo; el core si conserva refs estables y replay recuperable.

estado_final: ready_for_operator_dag_review

## Seguimiento local posterior

La rama se rebasó sobre `6a8cb3e066` sin incorporar wiring de composición.
La auditoría local cerró una brecha adicional de path seguro: tanto el contrato
del DAG como el compilador de autoprogramación rechazan ahora URIs sin
autoridad, como `file:/tmp`, y rutas Windows drive-relative, como
`C:relative`. El guard arquitectónico del core también bloquea imports de
filesystem, syscall y HTTP además de runtime/proveedor/adaptadores ya
prohibidos.

Evidencia posterior al ajuste, con cachés temporales aisladas:

```sh
go test -race -count=1 ./modulos/orquesta-autonomy-program \
  ./modulos/orquesta-state-file ./modulos/orquesta-autoprogramming
go test -count=50 \
  -run '^TestStoreV0AutonomyProgramCASClaimsFrontierOnceAcrossStoreInstancesV0$' \
  ./modulos/orquesta-state-file
go test -count=50 \
  -run '^TestNeutralOrchestrationPackagesDoNotImportProductAdapters$' .
```

Las tres órdenes pasaron. Sigue fuera de este write-set cualquier composición
que persista transiciones CAS y entregue `LaunchAutonomyProgramNodeV0` o
`ResumeAutonomyProgramNodeV0`: debe deduplicar por `launch_ref`/`receipt_ref`
y aportar su propia evidencia. No se añadió runtime, proveedor, proceso, API,
deploy ni wiring de servidor.

La batería raíz `go test -count=1 ./...` no queda verde por una regresión
preexistente y ajena a este DAG: `TestEnvVarsBudgetMEJ106V0` mide 521 variables
`ORQUESTA_*` frente al límite 513. El test y el DAG rebasado no modifican esa
superficie. Su corrección o la justificación documental del presupuesto exige
un write-set específico de configuración global; no se alteró desde este corte.
