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

## Replays r4/r5 y BUG-ORQ-20260711-231

El replay `r4` no lanzo un goal: el pipeline completo rechazo correctamente el
piloto porque solo declaraba comprobaciones manuales y ninguna atestacion
independiente. No fue un fallo de Orquesta. En `r5` se anadio el test focal
requerido y el primer tick persistio el baseline, pero los siguientes fallaron
con `worktree_baseline_store_failed` antes de lanzar un agente.

La comparacion diagnostica retenida en
`/tmp/orquesta-bug226-t9104-r5-runtime/compare_snapshot.go` confirmo 5.091
ficheros, paths y digests identicos. La unica diferencia era semantica de JSON:
el snapshot recapturado tenia `OmittedPaths=[]`, mientras que el reabierto tenia
`OmittedPaths=nil`. `reflect.DeepEqual` convertia esa representacion opcional en
un conflicto de contenido y rompia la idempotencia del prepare-run residente.

Cierre local de `BUG-ORQ-20260711-231`:

- el store normaliza a `nil` las listas opcionales vacias antes de comparar;
- contenido real divergente sigue devolviendo conflicto;
- una regresion recrea el store tras la serializacion JSON y repite el record;
- `/tmp/orquesta-bug226-t9104-r5-runtime/shutdown-bug231.json` acredita
  `shutdown_ready=true`, cero trabajo en vuelo y salida cooperativa del proceso.

El cierre empirico de `BUG-226` sigue pendiente de un replay completamente
nuevo que alcance lanzamiento, progreso material, atestacion y cierre terminal.

## Replay r6 y BUG-ORQ-20260711-232

`r6` acredito que los fixes anteriores alcanzan el runtime real:

- goal externo `019f5072-08d7-78e0-ba01-2a5c2e530ade` lanzado una sola vez;
- baseline durable ligado a la spec y binder independiente activo;
- progreso material persistido: primero `none` y despues `diff`, reiniciando a
  cero los tokens sin material;
- entrega terminal con el artefacto esperado y test autodeclarado verde;
- el atestador externo ejecuto realmente el test y rechazo ese verde.

El rechazo descubrio `BUG-ORQ-20260711-232`: el adaptador configuraba
`GOMODCACHE` directamente sobre el snapshot de dependencias de solo lectura.
Go siempre intenta crear metadata/locks bajo `cache/download`; sin ese indice
fallaba al crear el directorio y con el indice fallaba al abrir el `.lock`.
Por tanto, ningun `go test` con modulo externo podia superar la atestacion
hermetica. Evidencia retenida:

- `/tmp/orquesta-bug226-t9104-r6-runtime/attestation/evidence/commands/591d63e280932913f89929f1.log`;
- recibo independiente `goal-required-test-attestation-ref-049a83c81f0d83ef7f4c54c9fdfd6cb69cb8fd53f7b48df9db9106d6d12c5c46`,
  `status=failed`, `exit_code=1`;
- Orquesta no acepto el cierre y lanzo automaticamente el rework causal
  `019f5073-c8a2-72b3-a4f7-a7a1cdaf9815`, conservando el artefacto anterior.

Cierre local de `BUG-232`:

- el snapshot canonico se revalida antes y despues de copiar;
- la copia se valida contra el mismo hash mientras aun es read-only;
- solo la copia privada y unica por ejecucion se hace escribible y se expone
  como `GOMODCACHE`;
- preflight y test usan workdirs temporales distintos, eliminados al terminar;
- outputs y recibos durables permanecen fuera del workdir efimero;
- regresiones prueban fuente inmutable, copia escribible, replay sin residuos,
  deteccion de mutacion y limpieza de preflight.

El rework de `r6` se detuvo por control tipado al confirmar que el fixture no
podia pasar; shutdown elimino el app-server con `shutdown_ready=true`. El
cierre integrado de `BUG-226/232` requiere un replay nuevo con snapshot offline
completo y binario que contenga este arreglo.

## Replay r7 y BUG-ORQ-20260711-233

`r7` arranco desde `6ae099cd3` con snapshot offline completo. El preflight uso
su copia privada y limpio el workdir; el goal
`019f507d-58e7-7dc1-897e-127fb9851444` materializo un informe, y el gobernador
registro `warning` a 33.015 tokens seguido de progreso `diff` a 37.512 tokens.

El resultado final escribio por error
`goal-ref-task-autoprogramming-a2ae86a2ae2e-g01` en vez de
`goal-ref-task-autoprogramming-c0ae86a2ae2e-g01`. El thread externo y todos los
demas refs estaban ligados al goal correcto, pero el adaptador descarto el
marker completo y dejo `attempt_blocked` sin atestacion. Es una forma
recuperable que la regla canonica obliga a normalizar, no una ref imposible.

Cierre local de `BUG-ORQ-20260711-233`:

- solo se recuperan refs `goal-ref-*` de igual longitud con exactamente una
  sustitucion, dentro del thread externo ya ligado por la observacion;
- la ref se reemplaza por la esperada antes del guard de write-set y del merge;
- el recibo conserva evidencia explicita
  `evidence-ref-codex-app-server-goal-result-goal-ref-normalized`;
- refs con dos cambios, otra longitud/prefijo o external ref divergente siguen
  rechazadas;
- una regresion app-server reproduce el typo y exige resultado terminal unido a
  la ref esperada.

Se reiniciara el mismo estado `r7` con el binario corregido para reobservar el
thread ya terminal sin gastar otro goal y cerrar la atestacion pendiente.

## Replay r8 y BUG-ORQ-20260711-234

La reobservacion de `r7` no pudo recuperar el marker despues de limpiar su
backend y quedo bloqueada honestamente como `goal_backend_gone_without_result`.
No se fabrico un cierre manual. `r8` arranco limpio desde `96f1fe70c`, completo
T9104 en el goal `019f5085-67a1-7863-9664-28c9f0f36dc5`, materializo un
resultado durable correcto y alcanzo la atestacion independiente.

El comando independiente paso realmente:

```text
status=passed
ok orquesta/cmd/orquesta-server 0.007s
```

Sin embargo, el claim durable termino
`goal_required_test_attestor_infrastructure_failed` sin receipt. La causa fue
posterior al test: Go vuelve a marcar directorios de su module cache como
read-only; el `defer os.RemoveAll(runDir)` no podia atravesarlos y convertia el
verde en fallo de limpieza.

Cierre local de `BUG-ORQ-20260711-234`:

- el entorno Go aislado fija `GOFLAGS=-modcacherw`;
- la limpieza recorre unicamente el workdir efimero, restaura `0700` en sus
  directorios y despues ejecuta `RemoveAll`;
- un fallo de limpieza sigue siendo infraestructura roja, nunca falso verde;
- una regresion ejecuta un comando que crea deliberadamente un subdirectorio
  read-only dentro de `GOMODCACHE` y exige que no quede ningun workdir;
- evidencia real retenida en
  `/tmp/orquesta-bug226-t9104-r8-runtime/attestation/evidence/commands/17d07c63b092cee037a6ade1.log`.

El claim fallido de `r8` es inmutable por diseño; el cierre empirico necesita
estado nuevo con el arreglo de limpieza.
