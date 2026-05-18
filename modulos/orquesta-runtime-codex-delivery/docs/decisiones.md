# Decisiones: orquesta-runtime-codex-delivery

```text
Fecha: 2026-05-15
Decision: `stalled` corto es telemetria, no decision automatica.
Motivo: en pruebas reales con agentes `xhigh`, algunos procesos pasan varios
ticks sin escribir ACK ni artefactos aunque sigan pensando. Si cada `stalled`
genera pregunta al director, la web y el director reciben ruido y Orquesta
parece rota antes de que haya evidencia de fallo real.
Alternativas:
  - Subir mucho `stalled_ticks`: descartado porque oculta senal temprana para
    estadisticas.
  - Mantener pregunta para todo stalled: descartado porque reintroduce bucles de
    supervision.
  - Separar telemetria de decision: aceptado.
Impacto: `AgentStalledV0` se emite para estadisticas, pero solo exige decision
si viene con `DecisionRequired` por presupuesto/contexto, si escala a
`loop_detected`, si el proceso se observa parado o si hay no-actividad real
fuera de presupuesto. El nucleo puede mostrar stalled sin tratarlo como alerta
fuerte.
Estado: aceptada.
```

```text
Fecha: 2026-05-15
Decision: Separar actividad fisica de logs y accion semantica repetida.
Motivo: en una prueba real el director genero los manuales pedidos, pero quedo
vivo repitiendo el mismo diff sin escribir `agent_ack.json`. El progreso por
tamano/modtime de logs lo clasificaba como activo aunque no producia trabajo
nuevo. Esto reproduce el patron de v1/v2: gastar cuota en bucles que parecen
actividad.
Alternativas:
  - Bajar timeouts globales: descartado porque corta agentes lentos que si
    estan trabajando.
  - Tratar cualquier log largo como progreso valido: descartado porque oculta
    bucles de diff/salida repetida.
  - Anadir firma compacta de accion en el conector Codex: aceptado.
Impacto: `CodexProgressSampleV0` conserva `Signature` para actividad real de
ficheros, y anade `ActionSignature` para detectar la misma accion compacta
repetida mientras crece el log. Si se repite hasta el umbral, el reporte pasa a
`loop_detected` y el nucleo puede parar el agente. No se exponen rutas,
transcripts ni detalles de proveedor/modelo; la firma es hash compacto.
Estado: aceptada.
```

```text
Fecha: 2026-05-12
Decision: Los descriptors ya reflejados se omiten antes de leer ACK o verificar
worktree.
Motivo: una entrega ya registrada puede ser revisada mas tarde y provocar
cambios legitimos de rework en el mismo worktree. Si el source de deliveries
vuelve a verificar descriptors historicos antes de saltarlos, esos cambios
posteriores parecen violaciones del write-set de agentes que ya cerraron.
Alternativas:
  - Relajar el verificador de worktree: descartado porque perderia deteccion de
    interferencias reales en entregas nuevas.
  - Borrar descriptors al registrar delivery: descartado porque review gate y
    auditoria necesitan el descriptor.
  - Saltar por `ack_ref` ya reflejado antes de parsear/verificar: aceptado.
Impacto: `CodexDeliveryObservationSourceV0` usa el `ack_ref` del descriptor
para omitir deliveries/phase artifacts ya reflejados sin tocar filesystem ni
releer ACK. Las entregas nuevas siguen verificando write-set estricto.
Estado: aceptada.
```

```text
Fecha: 2026-05-15
Decision: `over_budget_no_activity` no queda suprimido por un assessment
`stalled` previo.
Motivo: el deduplicador de progreso evita repetir la misma pregunta al
director, pero esa proteccion no puede tapar una escalada posterior de
presupuesto. Si un agente ya fue marcado como `stalled` y despues supera el
limite sin actividad, Orquesta debe publicar una nueva observacion para que el
director pueda emitir `timeout/stop_agent`.
Alternativas: borrar todos los assessments previos; ignorar deduplicacion;
subir limites de actividad. Borrar historia pierde trazabilidad, ignorar
deduplicacion genera ruido, y subir limites oculta agentes perdidos.
Impacto: `codexProgressReportAlreadyHandledV0` solo considera tratado un
`over_budget_no_activity` si el agente ya esta parado, confirmado, fallido o
perdido. El conector mantiene refs compactas y no expone logs, HOME, proveedor,
modelo ni rutas.
Estado: aceptada.
```


```text
Fecha: 2026-05-12
Decision: Un ACK Codex con `status=failed` es no-entrega terminal, no error
fatal del source de deliveries.
Motivo: en trabajo multiagente real un agente puede fallar por su propio
write-set mientras otros agentes ya tienen ACK completados. Si el source
propaga el ACK failed como error global, Orquesta bloquea entregas validas y
vuelve al patron de v1/v2 de atascar todo el run por un fallo parcial.
Alternativas:
  - Aceptar el ACK failed como entrega: descartado porque permitiria registrar
    trabajo fallido como delivery valida.
  - Fallar todo el drenaje: descartado porque impide procesar ACKs completados
    independientes.
  - Omitir el ACK failed en delivery source y dejarlo a progreso/revision:
    aceptado.
Impacto: `CodexDeliveryObservationSourceV0` omite ACKs no completados por
status y sigue leyendo el resto de descriptors. El fallo queda disponible para
supervision, assessment, replan o rework por otros puertos, pero no se acepta
como entrega.
Estado: aceptada.
```

```text
Fecha: 2026-05-11
Decision: La observacion de review gate permanece viva hasta una proyeccion terminal.
Motivo: el scheduler aplica el review gate en pasos progresivos. Primero pide
revision, despues registra resultado y, si hay cambios solicitados, pide
rework. Si el source deja de emitir al ver `RequestReview`, el flujo queda
atascado con una revision pendiente y nunca llega a `RequestRework`.
Alternativas:
  - Subir limites de drenaje: descartado porque no corrige la ausencia de
    candidatos.
  - Crear comandos manuales en el stack: descartado porque haria que el
    adaptador suplante al scheduler.
  - Mantener la observacion hasta aceptar o pedir rework: aceptado.
Impacto: `CodexReviewGateObservationSourceV0` solo omite una entrega cuando ya
existe `AcceptedReviews` para accepted o `ReworkRequests` para
changes_requested/rejected. La logica de proyeccion queda separada en fichero
pequeno para mantener el modulo depurable.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: El smoke de app pequena completa nace desde StartAppDirectorV0 y no desde tareas preparadas por el test.
Motivo: una prueba de app completa debe demostrar que Orquesta arranca el director, consume decisiones ejecutables, lanza agentes de programacion y registra artefactos/entregas sin que el test haga de orquestador.
Alternativas:
  - Mantener solo microtareas preparadas en memoria: descartado porque no valida el puente director -> decisiones -> programacion.
  - Recuperar la app como una unica tarea: descartado por duracion, ACK fragil y falta de paralelismo.
Impacto: `TestProgrammingTeamCodexRealOptInV0` arranca director real, consume `director_decisions.json`, usa batch de hasta 2 agentes para API/web/docs y valida API REST, tests, web i18n, README y limites de lineas.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: La revision de entregas Codex usa un source propio y no reutiliza el filtro de delivery pendiente.
Motivo: el store de descriptors descarta deliveries ya registradas para evitar duplicados en `DeliveryRegistered`; el review gate necesita justo lo contrario: revisar entregas ya reflejadas en el run. Reutilizar el filtro ocultaba las entregas y dejaba la revision vacia.
Alternativas:
  - Relajar el store global: descartado porque reintroduce duplicados en delivery.
  - Duplicar descriptors en otro store: descartado por complejidad y estado extra.
Impacto: `CodexReviewGateObservationSourceV0` lista descriptors por run/agente sin `deliveries` ni `phase_artifacts`, y luego filtra por `run.Deliveries` y review no proyectado.
Estado: aceptada.
```

```text
Fecha: 2026-05-12
Decision: El review gate puede revisar entregas de agentes ya cerrados.
Motivo: `StoppedAgents` y `ConfirmedStoppedAgents` describen el ciclo de vida
del proceso externo, no la validez de una entrega ya registrada. Si la entrega
esta en `Run.Deliveries`, el review gate debe evaluarla aunque el proceso se
haya parado correctamente despues del ACK.
Alternativas:
  - Reutilizar la elegibilidad de delivery pendiente: descartado porque oculta
    entregas registradas y bloquea review/rework.
  - Mantener agentes vivos hasta terminar revision: descartado porque acopla
    revision a runtime y gasta cuota sin necesidad.
Impacto: `CodexReviewGateObservationSourceV0` valida agente conocido,
arrancado y no fallido, pero no filtra por `StoppedAgents` para revisar
deliveries ya reflejadas.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: El limite de tamano de ficheros se calcula desde evidencia real del proyecto.
Motivo: aceptar solo lo que declara el ACK permite que un agente entregue trabajo inmanejable o inexistente. La comprobacion debe vivir en un adaptador externo para no meter filesystem en el nucleo.
Alternativas:
  - Confiar en el ACK: descartado por evidencia insuficiente.
  - Leer ficheros desde el core: descartado por romper hexagonal.
Impacto: `CodexReviewGateProjectFileEvidenceV0` cuenta lineas reales, marca ficheros ausentes/no legibles y entrega solo incidencias compactas al review gate.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Mantener el lector de ACK Codex fuera del nucleo y fuera del conector Codex puro.
Motivo: el nucleo no debe conocer rutas ni Codex; el conector Codex puro no debe depender de la app de orquestacion. Este adaptador exterior une ambos contratos.
Alternativas:
  - Importar Codex desde orquestacionnucleoapp: descartado por acoplamiento.
  - Hacer que orquesta-runtime-codex implemente directamente el puerto del nucleo: descartado por dependencia inversa.
Impacto: se puede sustituir el store o el proveedor sin tocar el core; la prueba real puede inyectar este source por puerto.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: Anadir store filesystem explicito para descriptors de ACK.
Motivo: los smokes paralelos usaban memoria; si Orquesta reinicia despues de
arrancar agentes, pierde donde leer sus ACKs. El descriptor es estado
operacional del conector y debe ser durable sin meter DB ni filesystem en el
nucleo.
Alternativas:
  - Mantener solo memoria: descartado para apps grandes/reanudacion.
  - Guardarlo en el core: descartado porque incluye ack_path/project_work_dir.
  - Elegir una DB concreta: descartado; la persistencia concreta es conector.
Impacto: `FileCodexReceiptDescriptorStoreV0` persiste JSON versionado en un
directorio absoluto configurado por operador, recupera al recrear instancia y no
filtra rutas en errores.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: Verificar el diff real del worktree antes de aceptar un ACK Codex.
Motivo: un agente puede declarar en `agent_ack.json` solo ficheros permitidos y
haber tocado otros paths. Para apps grandes esto rompe el paralelismo y vuelve a
los bucles de v1/v2.
Alternativas:
  - Confiar solo en el ACK: descartado por evidencia incompleta.
  - Usar Git desde el nucleo: descartado por romper hexagonal y acoplar a VCS.
Impacto: `CodexReceiptRecordingSpecResolverV0` puede capturar baseline externo
antes del launch; `CodexDeliveryObservationSourceV0` puede inyectar
`CodexReceiptWorktreeVerifierV0` y rechazar cambios reales fuera del write-set
sin filtrar paths absolutos al nucleo.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: No asociar una supervision de progreso previa al ACK con `delivery_ref`.
Motivo: antes del ACK no existe entrega durable. Si el adaptador marca
`delivery_ref=ack_ref`, el core rechaza correctamente `AssessAgentWork` porque
no puede evaluar una entrega que aun no esta registrada.
Alternativas:
  - Relajar `AssessAgentWork` para aceptar deliveries inexistentes: descartado
    porque rompe invariantes de reviews, rework y cierre.
  - Registrar una delivery ficticia de timeout: descartado porque mezcla
    supervision con entrega de trabajo.
Impacto: `CodexProgressObservationSourceV0` evalua agente/tarea sin
`delivery_ref` hasta que el ACK este listo. La entrega real sigue entrando por
`CodexDeliveryObservationSourceV0`.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: No escalar una firma Codex sin cambios a `loop_detected` por si sola.
Motivo: una firma de logs/ACK sin cambios solo demuestra ausencia de evidencia
externa nueva; un Codex real puede estar pensando sin escribir durante decenas
de segundos. Tratar eso como bucle terminal produce falsos cortes y desperdicia
cuota arreglando supervisores demasiado agresivos.
Alternativas:
  - Cortar por timeout fijo: descartado porque no distingue trabajo lento de
    proceso sin avance real.
  - Cortar procesos por nombre: descartado porque podria afectar sesiones
    ajenas o al director.
  - Mantener la escalada anterior: descartado tras prueba multiagente real,
    porque podia parar agentes validos antes de ACK.
Impacto: el estado de progreso conserva conteo acumulado de no-progreso para
emitir `stalled`, pero deja `RepeatedActionCount=0` hasta que exista una senal
real de accion repetida. La parada automatica por `loop_detected` sigue
soportada por el nucleo si otro conector aporta esa evidencia.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: Filtrar ACKs de agentes no elegibles antes de proponer artefactos.
Motivo: el core exige que el agente este en `Agents`, en `StartedAgents` y que
no este fallido/parado antes de aceptar `PhaseArtifactRegistered`. El adaptador
no debe fabricar comandos que sabe que violaran esa invariante.
Alternativas:
  - Relajar el core: descartado porque permitiria entregas de agentes cerrados.
  - Dejar fallar el drenaje: descartado porque convierte un estado esperado en
    error operacional y tapa ACKs validos de otros agentes.
Impacto: `CodexDeliveryObservationSourceV0` solo emite observaciones de agentes
pedidos, arrancados y no cerrados. Si un agente fue parado, su ACK tardio queda
fuera del drenaje.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Filtrar ACKs de artefactos de fase ya registrados antes de construir nuevos candidatos.
Motivo: el scheduler procesa el primer candidato de artefacto por tick. Si el source vuelve a entregar un ACK ya registrado, ese candidato no produce comando y bloquea los ACKs pendientes posteriores.
Alternativas:
  - Hacer que el scheduler escanee todos los candidatos: aplazado, porque aumenta la responsabilidad del scheduler y no evita lecturas repetidas.
  - Subir los limites del loop: descartado porque no resuelve la cola bloqueada por un candidato ya consumido.
Impacto: `CodexReceiptDescriptorRequestV0` incluye `phase_artifacts`; el store en memoria los filtra y `CodexDeliveryObservationSourceV0` mantiene un filtro defensivo por `artifact_ref`.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Los smokes reales de programacion deben usar microtareas cerradas, no UI o API amplias.
Motivo: una tarea web inicialmente pequena crecio hasta agotar 360s sin ACK, y una tarea API supero temporalmente el limite de lineas. Los agentes reales tienden a seguir refinando si el objetivo permite interpretacion amplia.
Alternativas:
  - Subir timeout: descartado porque oculta bucles de refinado.
  - Aceptar entregas sin ACK: descartado porque rompe control de Orquesta.
  - Acotar write-set, objetivo y lineas por fichero: aceptada.
Impacto: `TestProgrammingTeamCodexRealOptInV0` valida programacion real con dos agentes, write-sets disjuntos, ACK por agente, `DeliveryRegistered` y test de la app generada.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: El smoke de formulario MCP con director real valida ACK/observacion, no `DeliveryRegistered`.
Motivo: El director trabaja en `brainstorming_arquitectura`; `DeliveryRegistered` del core exige fase `programacion` y microtarea ya reflejada. Forzar ese evento para brainstorming mezclaria semanticas y ocultaria un hueco real de modelado.
Alternativas:
  - Relajar `DeliveryRegistered` para cualquier fase: descartado en este corte porque puede romper invariantes de reviews/rework.
  - Registrar los documentos del director como delivery de programacion: descartado por falso positivo.
  - Crear un evento nuevo de artefacto de director: aceptado despues como `RegisterPhaseArtifact`/`PhaseArtifactRegistered` en `orquesta-core-workflow`.
Impacto: `TestMCPFormularioDirectorCodexRealOptInV0` demostro el hueco de modelado: ACK valido sin evento durable correcto para brainstorming. Queda como decision historica.
Estado: superada por `RegisterPhaseArtifact`.
```

```text
Fecha: 2026-05-09
Decision: El ACK de fases no-programacion se registra como `PhaseArtifactRegistered`.
Motivo: Un director de brainstorming/documentacion no entrega una microtarea de codigo; entrega artefactos compactos de fase. Reutilizar `DeliveryRegistered` romperia invariantes de programacion.
Alternativas:
  - Seguir solo con ACK en filesystem: descartado porque Orquesta perderia memoria durable del trabajo.
  - Relajar `DeliveryRegistered`: descartado porque mezcla fase, task y delivery de programacion.
  - Crear ruta paralela de artefactos de fase: aceptada.
Impacto: `CodexDeliveryObservationSourceV0` expone `artifact_ref`; `DeliveryCandidateProviderV0` decide por fase; scheduler emite `RegisterPhaseArtifact`; el smoke real valida `PhaseArtifactRegistered` y ausencia de `DeliveryRegistered`.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: No usar una app completa como una sola tarea de agente.
Motivo: la prueba con Codex real de una mini app Go paso, pero tardo 222.247s;
una variante mas ambiciosa agoto el presupuesto sin ACK. Eso confirma que el
problema no se arregla subiendo timeout, sino dividiendo trabajo.
Alternativas:
  - Aumentar timeout: descartado porque oculta agentes sin progreso y gasta
    cuota sin control.
  - Relajar validacion de ACK: descartado porque permitiria entregas basura.
Impacto: las pruebas reales de app completa deben modelarse como microtareas
independientes, con write-set acotado, ACK por corte y supervision de progreso.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Tratar ACK ausente como entrega no lista, no como fallo del ciclo.
Motivo: un agente real se arranca de forma asincrona; el recibo aparece despues
del launch. Si el source falla por un ACK aun no escrito, Orquesta rompe el
flujo normal de polling y obliga a parchear tiempos manualmente.
Alternativas:
  - Ignorar cualquier error de lectura: descartado porque ocultaria permisos
    rotos, JSON corrupto o paths mal configurados.
  - Bloquear el loop hasta que aparezca el ACK: descartado porque ata el nucleo
    al runtime y puede gastar cuota/tiempo sin progreso.
Impacto: `orquesta-runtime-codex` marca `ack_not_ready` como retryable y este
adaptador lo omite hasta el siguiente tick. Los ACK invalidos siguen fallando.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Mantener el smoke con Codex real como prueba opt-in.
Motivo: lanzar Codex real consume cuota y depende de credenciales locales, pero
necesitamos demostrar que Orquesta, no el operador humano, arranca el agente,
espera el recibo y registra la entrega.
Alternativas:
  - Ejecutarlo siempre: descartado por coste y dependencia del entorno.
  - Probar solo con runtime simulado: descartado porque no valida el conector
    Codex real ni el retardo natural entre launch y ACK.
Impacto: `ORQUESTA_CODEX_SMOKE=1` activa la prueba real con timeout explicito y
logs acotados. Modelo, HOME, PATH y politica de aprobacion entran por variables
del conector, no por el nucleo.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: Registrar el descriptor del ACK durante la resolucion del spec, antes de AgentStarted.
Motivo: si el agente arranca pero no queda registrado donde leer su receipt, el nucleo podria quedarse sin entrega observable y entrar en esperas/manualidad.
Alternativas:
  - Reconstruir el path desde refs de evidencia: descartado porque las evidencias son opacas.
  - Escanear directorios de runtime: descartado por acoplamiento y riesgo de paths.
  - Guardarlo dentro del core: descartado por filtrar detalles operacionales.
Impacto: `CodexReceiptRecordingSpecResolverV0` decora el spec resolver del launcher y exige recorder + path resolver inyectados.
Estado: aceptada.
```

```text
Fecha: 2026-05-09
Decision: La deteccion de agentes sin avance vive en el adaptador Codex delivery, no en el nucleo.
Motivo: el nucleo necesita informes neutrales de progreso, pero no debe conocer
paths, logs, runtime workdir, nombres de ficheros ni detalles Codex. El conector
si puede observar esos detalles y condensarlos en heartbeats compactos.
Alternativas:
  - Vigilar procesos por nombre: descartado porque podria cortar sesiones ajenas
    o el director activo.
  - Meter polling de paths en el core: descartado por romper hexagonal.
  - Esperar solo al ACK con timeout externo: descartado porque reproduce el
    fallo de esperas largas sin diagnostico.
Impacto: `CodexProgressObservationSourceV0` usa descriptors externos, registry
de procesos y `CodexProgressStateStorePortV0`; solo emite `AgentProgressReportV0`
cuando hay estancamiento real y solo silencia avisos ya materializados en el
run.
Estado: aceptada.
```

```text
Fecha: 2026-05-10
Decision: Anadir store filesystem explicito para estado de progreso Codex.
Motivo: la supervision anti-bucle pierde fuerza si un reinicio borra firmas,
contadores y avisos ya emitidos. Para apps grandes necesitamos que stalled y
loop_detected sobrevivan al proceso de Orquesta.
Alternativas:
  - Mantener solo memoria: descartado para ejecuciones largas.
  - Guardar progreso en el core: descartado porque es estado operacional del
    conector.
  - Elegir una DB concreta: descartado; la persistencia concreta es conector.
Impacto: `FileCodexProgressStateStoreV0` persiste JSON versionado en directorio
absoluto configurado por operador, recupera al recrear instancia y mantiene
errores compactos sin filtrar rutas.
Estado: aceptada.
```

```text
Fecha: 2026-05-11
Decision: Una lectura de progreso no consume una decision si el run no la ha
materializado.
Motivo: web/MCP/director pueden consultar progreso para estadisticas antes de
que el scheduler procese el candidato. Si el source marca el aviso como emitido
y luego lo silencia sin mirar el estado del run, una observacion pasiva puede
dejar un agente real esperando indefinidamente.
Alternativas:
  - No deduplicar nunca: descartado porque puede generar ruido despues de que
    el run ya refleje assessment, parada o confirmacion.
  - Deduplicar solo por firma local: descartado porque el estado operacional del
    conector no demuestra que el core haya aplicado la decision.
Impacto: `CodexProgressObservationSourceV0` reemite stalled/loop/stopped hasta
que `OrchestrationRunV0` contiene la evaluacion, parada, confirmacion o cierre
equivalente. Una escalada posterior de stalled a loop no queda tapada por una
pregunta previa al director.
Estado: aceptada.
```

```text
Fecha: 2026-05-11
Decision: Un proceso Codex parado sin ACK se trata como decision de supervision
inmediata.
Motivo: un proceso que termina sin `agent_ack.json` no puede contarse como
trabajo pendiente normal. En pruebas reales ocurrio por cuota externa agotada;
Orquesta debe cerrar el agente logico o consultar al director, no seguir
esperando ciclos largos.
Alternativas:
  - Esperar hasta timeout global: descartado porque reproduce el bucle de v1/v2.
  - Convertirlo en entrega fallida generica: descartado porque mezcla fallo de
    runtime con calidad del trabajo.
Impacto: el reporte `AgentStoppedV0` incluye contexto compacto si detecta cuota
en stderr, sin filtrar proveedor/modelo/HOME. El director decide `stop_agent`
si la parada es segura; los agentes de direccion protegidos pueden preguntar al
director, pero un proceso ya parado no queda protegido artificialmente.
Estado: aceptada.
```

```text
Fecha: 2026-05-12
Decision: Una interrupcion Codex sin ACK se publica como parada gobernable y no
como capacidad salvo evidencia explicita.
Motivo: una salida terminal con `turn interrupted` o `tokens used` y sin
`agent_ack.json` no es una entrega ni un simple pendiente. Antes el contexto de
fallo solo especializaba capacidad/cuota desde stderr; si la senal venia en
stdout o `codex_last_message.txt`, el scheduler recibia una parada generica sin
evidencia no-ACK compacta.
Alternativas:
  - Propagar stdout/stderr al nucleo: descartado porque filtraria HOME, paths,
    proveedor, modelo o transcripts.
  - Tratar cualquier interrupcion como `capacity_limited`: descartado porque
    mezcla fallo terminal sin ACK con saturacion externa recuperable.
  - Mantener solo `stopped` generico: descartado porque oculta causa gobernable
    al director/scheduler.
Impacto: `CodexProgressObservationSourceV0` mantiene `status=stopped` para el
proceso parado sin ACK, anade evidencia opaca de no-ACK interrumpido y prioriza
`capacity_limited` si cualquiera de las salidas controladas indica capacidad.
El adaptador lee solo logs acotados del runtime Codex y no expone texto crudo,
HOME, rutas, proveedor ni modelo.
Estado: aceptada.
```

```text
Fecha: 2026-05-12
Decision: Distinguir capacidad externa limitada de cuota agotada y de trabajo
basura.
Motivo: en smoke real un agente termino sin ACK con error de runtime indicando
capacidad externa saturada. Eso no prueba mala calidad del agente ni bucle; es
una senal operativa para cerrar el agente logico y pedir relevo/capacidad
alternativa sin dejar el run esperando indefinidamente.
Alternativas:
  - Reutilizar over_budget: descartado porque mezcla cuota agotada con modelo o
    runtime temporalmente no disponible.
  - Marcar garbage: descartado porque castigaria una entrega no producida por
    una condicion externa.
  - Esperar al timeout global: descartado porque reproduce bucles de v1/v2.
Impacto: el adaptador convierte stderr de capacidad limitada en
`AgentProgressReportV0` con `budget_status=capacity_limited`, resumen neutral,
evidencia compacta y sin filtrar proveedor, modelo, HOME, rutas ni stderr. La
decision posterior queda para director/scheduler mediante contratos publicos.
Estado: aceptada.
```

```text
Fecha: 2026-05-14
Decision: La clasificacion delivery vs artefacto no depende solo de la fase.
Motivo: Orquesta ya puede tener microtareas durables en documentacion,
integracion o revision, pero los agentes directores tambien llevan `task_ref`
en su paquete sin que eso sea una microtarea del run. Decidir solo por fase o
solo por `task_id` mezcla dos contratos distintos.
Impacto: `CodexDeliveryObservationSourceV0` sigue exponiendo `delivery_ref`,
`artifact_ref`, `task_id` y `phase_id` compactos; el provider del nucleo
registra `DeliveryRegistered` solo si `task_id` existe en `run.Tasks`, y
`PhaseArtifactRegistered` si el ACK pertenece a una fase sin microtarea
durable.
Estado: aceptada.
```

```text
Fecha: 2026-05-12
Decision: `director_decisions.json` es sidecar causal del ACK del agente que lo
produjo.
Motivo: en smoke real el director escribio decisiones que abrian `programacion`
antes de que su ACK de `brainstorming_arquitectura` estuviera reflejado en el
run. Consumir ese sidecar primero rompia la fase actual y hacia fallar el ACK
tardio con `payload.phase_id`.
Alternativas:
  - Ignorar `transicion_invalida`: descartado porque el core protege una
    invariante correcta.
  - Reordenar fases a mano en el stack: descartado por parche puntual y por
    acoplarse a nombres de fase.
  - Meter espera/sleep: descartado porque reproduce los bloqueos de v1/v2.
Impacto: `CodexReceiptDirectorDecisionFileDescriptorProviderV0` solo expone
ficheros de decisiones cuando el `ack_ref` del descriptor ya aparece en
`PhaseArtifacts` o `Deliveries` del run. La decision file sigue siendo externa;
el conector solo aplica causalidad con refs compactas, sin leer core interno,
proveedor, modelo, HOME ni rutas fuera del descriptor.
Estado: aceptada.
```
