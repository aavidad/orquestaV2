# T9104: progreso material desacoplado y shutdown stale

Fecha: 2026-07-11

Estado: `BUG-ORQ-20260711-227` cerrado empiricamente;
`BUG-ORQ-20260711-228` cerrado localmente y wiring real confirmado;
`BUG-ORQ-20260711-229` cerrado localmente, replay real pendiente.

## Alcance y evidencia retenida

La prueba empirica autorizada para cerrar `BUG-226` se ejecuto con un unico
goal util `T9104`, `MAX_REQUESTS=1`, worktree temporal y runtime aislado. No se
toco remoto, OPES ni ninguna aplicacion productiva.

- runtime: `/tmp/orquesta-bug226-t9104-runtime`
- worktree: `/tmp/orquesta-bug226-t9104-worktree`
- status inicial: `status-001.json`
- paradas HTTP: `shutdown-unbound.json` a `shutdown-unbound-4.json`
- control forzado confirmado: `run-control-unbound.json`
- goal externo abortado: `019f5040-faee-78c3-9941-6d3d2fd6d82e`

Ese goal no se reutiliza. Tras confirmar la parada por `runs/control`, el
servidor temporal se cerro cooperativamente con `SIGINT` porque el segundo bug
impedia alcanzar `shutdown_ready`.

## BUG-ORQ-20260711-228: el decorador ocultaba el puerto material

El servidor vivo publico `material_progress_state_reader_unbound`. El store
file-based implementaba `MaterialProgressStateStorePortV0`, pero
`serverWakeupGoalStateStoreV0` solo reexponia los contratos Goal. El stack
obtiene el reader mediante type assertion sobre el store ya decorado, por lo
que el governor, MCP y `runs/control` perdian la misma capacidad al activar el
relay residente.

Cierre local:

- el decorador delega lectura y CAS al puerto material de su `inner`;
- no se amplia el contrato Goal ni se obliga a los fakes legacy a implementar
  persistencia material;
- el test de wiring con relay real exige reader en MCP y `runs/control`.

## BUG-ORQ-20260711-227: accion de shutdown anterior bloqueaba un snapshot nuevo

Tras la parada confirmada del backend, el shutdown de aplicacion devolvia un
estado actual sin trabajo. El wrapper HTTP seguia fusionando
`cleanup_required` de la llamada anterior porque `MarkShutdownSnapshotV0`
reemplazaba refs y contador, pero no `ShutdownGoalActions`. El resultado era un
`stop_pending` indefinido con todos los contadores a cero.

Cierre local:

- cada snapshot nuevo invalida las acciones derivadas del anterior;
- la respuesta downstream vuelve a persistir las acciones actuales si siguen
  siendo necesarias;
- un resultado `ready` fresco no hereda `ShutdownAsyncWorkActive` de un intento
  anterior y limpia `ShutdownStopTimeoutAt`; los estados no terminales siguen
  conservando el contador async, porque aun puede representar drain real;
- una regresion secuencial demuestra `backend_still_running -> snapshot vacio
  -> ready`, sin relajar el guard que conserva trabajo realmente observado.

La auditoria paralela encontro dos decoradores adicionales que estrechan sus
interfaces: `serverWakeupRunStoreV0` no reexpone lectores de eventos y
`ackRuntimeCleanupEventSinkV0` tampoco. No se registran como bug operativo en
este corte: los consumidores actuales usan `EventReader` explicito o el
`EventSink` original. Quedan como hallazgo de limpieza estructural para revisar
sin bloquear el replay de nucleo.

## Verificacion local

```text
git diff --check
go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server
ok orquesta/cmd/orquesta-server 60.628s
ok orquesta/modulos/orquesta-server 0.444s
```

## Criterio de cierre empirico

1. Compilar un binario nuevo desde un commit limpio.
2. Crear worktree, estado, runtime y goal refs nuevos.
3. Ejecutar de nuevo `T9104` con un solo request.
4. Confirmar que no aparece `material_progress_state_reader_unbound`, que el
   estado material se persiste/proyecta y que el cierre usa tests independientes.
5. Confirmar que el shutdown termina en `shutdown_ready=true` sin señal manual.
6. Conservar recibos y actualizar `BUG-226/227/228`; no declarar cerrado el
   nucleo solo por los tests focales.

## Replay r2 y BUG-ORQ-20260711-229

El replay `r2` arranco desde `1018f8c31`, con binario limpio, worktree/estado
nuevos y `ORQUESTA_SERVER_WORKTREE` canonico. El primer preflight degradado por
omitir esa variable no lanzo goals y se cerro antes de repetir.

Evidencia retenida en `/tmp/orquesta-bug226-t9104-r2-runtime`:

- goal nuevo `019f5052-2813-7943-a2f6-3e7137c37cb0`;
- ningun diagnostico `material_progress_state_reader_unbound`;
- uso tipado observado hasta 41.526 tokens y un informe materializado;
- `run-control-bug229-2.json`: parada tipada confirmada;
- `shutdown-bug229.json`: `shutdown_ready=true`, `exit_pending=true`, backend
  `cleanup_completed`; el proceso salio sin senal manual.

La ausencia de estado material revelo `BUG-ORQ-20260711-229`: el servidor solo
inyectaba `GoalFirstSnapshotStore` cuando la promocion Git estaba activada, y
el prepare-run solo grababa el baseline en ese caso. Con promocion desactivada
por seguridad, el clasificador devolvia `material_progress_diff_verifier_unavailable`
y el governor omitia la decision.

Cierre local:

- la composicion server inyecta siempre su state-file como snapshot store;
- prepare-run graba baseline siempre que exista ese puerto, con independencia
  de `promotion.Enabled`;
- `promotion.Enabled` sigue gobernando exclusivamente promocion/commit;
- focales prueban baseline durable con promotion desactivada y wiring server.

El goal `r2` tampoco se reutiliza. El siguiente replay debe usar refs, estado y
worktree nuevos, y demostrar el fichero `material_progress_states/*.json`.

## Replay r3 y BUG-ORQ-20260711-230

El replay `r3`, goal `019f5059-3866-7912-953f-83f63615b6a7`, arranco limpio
desde `6a0cbe9a7` y fue detenido en la primera observacion al comprobar que su
spec seguia sin `worktree_baseline`. Evidencia retenida en
`/tmp/orquesta-bug226-t9104-r3-runtime`, incluidos `run-control-bug230.json` y
`shutdown-bug230.json`; la parada tipada quedo confirmada y shutdown termino
`ready` sin procesos residuales.

La causa no era un fallo del enlace por task ref. El scheduler residente, al
ver un launcher Goal, elegia `launchIdleSelfImprovementGoalsV0` y evitaba
`PrepareIdleSelfImprovementV0`, incluso cuando la composicion server ofrecia
ambos. Ese shortcut compila otra spec y no captura el baseline durable del
prepare-run.

Cierre local:

- nuevo puerto de capacidad `GoalFirstIdleSelfImprovementPreparerPortV0`;
- el stack server declara que su preparador soporta goal-first completo;
- el scheduler prefiere ese puerto y conserva launch directo solo para
  composiciones que no dispongan de preparacion completa;
- no cae a un preparador legacy por inferencia;
- regresion con ambos puertos exige una preparacion y cero launches directos.

La discrepancia declarada por los agentes `r1/r2` sobre el hash del backlog no
demuestra mutacion: compararon el SHA del fichero completo con una ref de scan
acotada por linea/seccion. Se conserva como ambiguedad contractual a revisar en
limpieza; no se usa como evidencia de cierre ni como causa de `BUG-230`.
