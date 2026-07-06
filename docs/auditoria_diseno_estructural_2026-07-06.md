# Auditoria de diseno estructural - 2026-07-06

Autor: Claude (director/revisor). Rol acordado con el operador: observar
errores de diseno y estructura; el codigo lo programa Codex.

Base de evidencia: las averias reales de 2026-07-05/06 (supervisor congelado,
regresion del carril progress, historial inflado de T137) y lectura dirigida
del codigo. Cada patron nombra donde se ha visto y que regla de diseno
deberia fijarse para que no se repita.

## P1. Presupuestos y limites desalineados entre capas (GRAVE, visto 2 veces)

Sintoma real: `LoadRunEventsV0` pedia una pagina de `MaxRunEvents=20000` y el
clamp de pagina la recortaba a 1000, convirtiendo un limite de transporte en
presupuesto total (T137 con 2669 eventos "excedia presupuesto" de 20000).
Antes, el snapshot del tick crecia sin limite hasta chocar con los 256 KiB
del scheduler.

Defecto de diseno: cada capa define sus constantes de presupuesto en privado
(`maxEventLogPageLimitV0`, `runEventsFullReadMaxV0=10000`,
`operationalRunEventsMaxV0=10000`, `defaultEventLogMaxRunEventsV0=20000`,
`maxSchedulerTickPayloadBytesV0=262144`) sin contrato comun ni tests de
coherencia. Notese que ya hay TRES "maximos de eventos por run" distintos
(1000 efectivo, 10000, 20000) en tres modulos.

Regla propuesta: un unico contrato de presupuestos (modulo o fichero de
constantes compartidas) con test que verifique las invariantes entre capas
(pagina <= lectura total <= presupuesto de store; snapshot compactado <=
payload del scheduler). Tarea candidata para Codex.

## P2. Fallo de un elemento congela el sistema global (GRAVE, visto 3 veces)

Sintomas reales: (a) tick del supervisor abortado 4500+ veces por el payload
de T137; (b) tick abortado por events.budget de T137 en la fase de
preparacion (`prepareRunCoordinatorTickV0` hace `return err` dentro de bucles
por-candidato); (c) run `accepted` invisible que no progresa ni informa.

Defecto de diseno: los bucles por-candidato de la preparacion del tick
(reconcile orphan/stopped/running-stale/domain-recovery) propagan el error
del candidato como error del tick entero, mientras que el bucle principal del
coordinador si sabe aislar (skips, ContinueOnDrainError, rotacion). La
proteccion existe en una capa y falta en la anterior.

Regla propuesta: "aislamiento de fallo por candidato" como invariante de
arquitectura: en cualquier bucle sobre la cola, un error atribuible a UN run
degrada a skip/park con evidencia y el resto avanza; solo errores de
infraestructura (store caido, contexto cancelado) abortan el tick. Fix 3 en
curso por Codex (`run_oversized`); mi variante equivalente quedo de respaldo
en la rama `claude/fix3-parking-runs-envenenados` (commit cac171a3d) por si
sirve de contraste. Falta ademas cubrir la clase (c): un `accepted` debe ser
siempre observable o reconciliable.

## P3. Emisores de eventos sin idempotencia semantica (GRAVE, causa raiz de T137)

Sintoma real: 1332 parejas identicas `AgentWorkAssessed` +
`DirectorQuestionRaised` (5 MB) en un run atascado: cada tick re-evaluaba y
re-preguntaba lo mismo con refs nuevos.

Defecto de diseno: en event sourcing, la idempotencia por `event_id` no basta
si el emisor genera ids nuevos para contenido semanticamente identico. La
clave debe derivar del contenido causal (run+agente+estado+fase), no del
reporte/tick.

Estado: corregido en la reconciliacion del drain (clave semantica,
`ec07bf300`). Pendiente auditar los DEMAS emisores del director que generan
assessments/preguntas/replan en bucles de supervision: mismo antipatron
posible. Candidato a regla de revision: todo emisor invocado desde un bucle
de tick debe declarar su clave de dedupe semantica.

## P4. Compactaciones destructivas sin gate de tamano (MEDIO, regresion real)

Sintoma real: `compactTickInputSnapshotForProgressV0` anulaba familias
enteras del snapshot siempre, y rompio la supervision progresiva
(`wait_external` -> `quiescent`). Arreglado con gate de 64 KiB (`3c323763c`).

Riesgo estructural pendiente: las otras lanes hacen lo mismo SIN gate:
`compactTickInputSnapshotForDeliveriesV0`, `...ForPhaseArtifactsV0`,
`...ForReviewGateV0` filtran/anulan incondicionalmente
(`tick_input_active_lane_snapshot_v0.go`, `tick_input_active_lane_v0.go`).
Llevan mas tiempo en produccion y sus tests pasan, pero el episodio del
carril progress demuestra que el patron es fragil: cualquier consumidor
transitivo que necesite un campo anulado fallara en silencio con el veredicto
equivocado.

Regla propuesta: compactar solo bajo presion de tamano (gate comun) o, si la
lane exige poda semantica, que el contrato del scheduler declare que campos
son prescindibles por lane y un test lo fije por cada consumidor.

## P5. Bateria de tests requerida no derivada de dependencias (MEDIO, dejo pasar la regresion)

Sintoma real: la validacion del goal que toco `orquesta-director-tick-input`
no incluia `orquesta-orchestration-core` (dependiente transitivo) y la
regresion se colo hasta que la caze a mano con bisect.

Regla propuesta: el launcher de goals debe calcular los paquetes afectados
por dependencia inversa (`go list -deps` invertido sobre el write-set) y
exigirlos en `required_test_results`, en vez de listas manuales. Mientras
tanto, regla manual: tocar `orquesta-director-*` exige correr
`orquesta-orchestration-core`.

## P6. Contratos de rutas durables generables invalidas (MEDIO, ya en curso)

Sintoma real: BUG-ORQ-20260705-195/198, resultado durable pedido bajo
`<fichero>/docs/...`. El commit `24e7b4e09` ("runtime: evitar resultados
durables bajo ficheros") ataca esta clase. Regla: el generador de contratos
debe validar que la raiz del artefacto durable es un directorio real del
write-set antes de lanzar el goal (validacion en el productor del contrato,
no en el consumidor).

## P7. Estados operativos sin contrato de redaccion (LEVE, molesto y repetido)

Sintoma real: placeholders de progreso con `status=blocked` y redacciones
variables ("started", "checkpoint_started; implementacion pendiente",
"in_progress_checkpoint_materialized") que confundieron 3 veces a los
watchers; y `startup_message` con contadores embebidos en texto libre.

Regla propuesta: los estados intermedios durables usan reason codes de
catalogo (como ya hacen los guards: `worktree_not_isolated`,
`missing_required_settings`) y nunca texto libre como campo discriminante.

## P8. Invariantes de despliegue sin runbook ejecutable (LEVE, costo real hoy)

Sintoma real: la tanda del 2026-07-05 arranco el servidor como root; el
estado quedo con dueno root y el arranque siguiente como berserk fallo con
`codex_receipt_descriptor_file_store: read_failed` (opaco). Ademas hay 4
binarios/backup sueltos en `/srv/orquesta-self/runtime` con duenos mixtos.

Regla propuesta: runbook unico de arranque/parada (usuario fijo berserk,
umask, rutas), y que el error de arranque distinga "permiso denegado" de
"fichero corrupto" en el reason code.

## Prioridad sugerida al operador

1. P2 (cierre completo con el fix de Codex + clase accepted-invisible).
2. P1 y P5 (contrato de presupuestos; bateria derivada de dependencias):
   baratos y evitan clases enteras.
3. P3 (auditoria de emisores) y P4 (gate comun de compactacion).
4. P6 ya en curso; P7/P8 cuando haya hueco.
