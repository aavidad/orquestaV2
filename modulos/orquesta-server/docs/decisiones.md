# Decisiones

## SRV-001: server-first residente

El trabajo real no debe depender de `go test`, de una sesion de Codex ni de una
terminal interactiva. Orquesta debe arrancar como servidor residente y aceptar
ordenes por API/MCP.

## SRV-002: statefile de reenganche

El servidor publica un statefile con PID, direccion HTTP, hora de arranque,
estado y ultima supervision. Al reconectar, el operador consulta ese fichero y
despues valida `/api/v0/server/readiness` o `/api/v0/server/status`; `/api/status`
queda solo como alias legacy.

Con automejora goal-first, el statefile puede guardar spec, receipt, resultado
y validacion de cierre completos para restaurar el proceso residente sin perder
causalidad. Esa informacion es estado local de reenganche, no superficie
publica: readiness, status y operational-status publican solo refs, estados y
contadores compactos.

## SRV-003: supervisor por puerto

El daemon no contiene logica del nucleo de orquestacion. Solo llama a un puerto
`RunGlobalSupervisorV0` con una orden acotada. El stack concreto decide que
runs drenar.

## SRV-004: estado operativo durable por conectores

El statefile del proceso no basta para autoprogramacion larga. Runs, eventos,
outbox, tareas, cambios solicitados, cola, control y registro de procesos deben
entrar por puertos persistentes reemplazables.

La implementacion local inicial puede ser file-based con JSON atomico. No se
autoriza acoplar el servidor a SQLite, Postgres ni otra base concreta. Si en el
futuro se usa una base de datos, sera otro conector con los mismos contratos.

## SRV-005: umbrales reales para agentes frontera

El servidor productivo no debe marcar como perdido a un agente de razonamiento
alto solo porque pase unos minutos pensando sin tocar ficheros. En la prueba
real OPES -> Orquesta -> Codex `gpt-5.5 xhigh`, el agente entrego resultado
valido despues de varios minutos, pero los defaults antiguos del comando eran
8 ticks de progreso con intervalo de 2s.

La composicion productiva usa ahora defaults conservadores: 300 ticks para
`stalled`, 300 ticks para posible bucle, 10 minutos sin actividad y 20 minutos
como presupuesto esperado. El nucleo no conoce esos numeros; siguen entrando
por configuracion y pueden ajustarse por entorno.

## SRV-006: backend file opt-in para domain_work generico

`cmd/orquesta-server` puede inyectar `orquesta-domain-work-file` como backend
de `DomainWorkJobCreatorPortV0` cuando se configura
`ORQUESTA_DOMAIN_WORK_FILE_ENABLED=1` o `ORQUESTA_DOMAIN_WORK_FILE_DIR`.

Esto cablea `/api/v0/domain-work` para `create_job` sin depender de OPES y sin
importar filesystem desde paquetes puros. No habilita `submit_artifact` ni el
bridge de entrega, porque el adaptador file solo crea jobs. Si tambien existe
`ORQUESTA_OPES_BASE_URL`, el servidor falla por backend ambiguo.

## SRV-008: backend HTTP neutral opt-in para domain_work generico

`cmd/orquesta-server` puede inyectar `orquesta-domain-work-http` cuando se
configura `ORQUESTA_DOMAIN_WORK_HTTP_BASE_URL`. El adaptador envia
`DomainWorkJobRequestV0` y `DomainWorkArtifactSubmissionV0` como JSON neutral a
la app externa propietaria y espera `DomainWorkJobV0`/`DomainWorkArtifactReceiptV0`.

Esto habilita `create_job`, `submit_artifact` y `DomainDelivery` sin OPES. El
backend HTTP es excluyente con OPES y con el backend file para evitar mezclar
propietarios de jobs. La app externa conserva persistencia, validacion y
ensamblado; Orquesta solo conserva refs opacas y evidencia causal.

## SRV-007: autodiagnostico antes de supervisor

El servidor residente ejecuta un `StartupCheckPortV0` opcional antes de escuchar
HTTP y antes de arrancar el supervisor. Si el check no devuelve `ready`, el
servidor publica `startup_blocked` en el statefile y no empieza a drenar runs.

La purga concreta no vive en `orquesta-server`: una composicion puede decidir
si diagnostica, solicita parada logica, limpia procesos de su runtime o bloquea
el arranque. El servidor solo persiste estado, mensaje para el Director y refs
de evidencia compactas.

## SRV-009: el supervisor residente no espera agentes largos en linea

El servidor residente debe lanzar u observar trabajo y volver pronto al bucle
global. La manga ancha de los agentes se gestiona por runtime, heartbeats,
progreso, leases y reentradas posteriores, no dejando bloqueado el tick del
supervisor durante minutos.

Por defecto `ORQUESTA_SERVER_DRAIN_MAX_EXTERNAL_WAITS` usa una espera externa
en linea. Esto permite materializar el arranque y una observacion inicial sin
impedir que otros runs entren en ticks posteriores. El modo residente acepta
subir ese margen hasta el maximo publicado de `70` cuando la tanda real lo
necesita; valores por encima siguen bloqueados para detectar configuraciones
descontroladas en vez de dejarlas ambiguas.

El pulso residente se ejecuta de forma asincrona con una guarda atomica de
actividad: un segundo tick no se solapa con el primero, pero la siguiente
iteracion puede avanzar en cuanto el pulso anterior libera el slot. La
preparacion de automejora se lanza aparte para no mezclar trabajo secundario con
el tick principal.

## SRV-010: stats T210 son contrato de transporte

El servidor no recalcula la semantica de progreso vivo ni cierre. Transporta la
proyeccion de `DirectorRunStatsV0` y conserva `include_agent_progress` como
opcion publica. Si una senal de progreso llega degradada, se publica con reason
code; no se rellena con 0% silencioso ni se convierte entrega en task cerrada.

## SRV-011: T208 reconciliado por resultado estructurado

El servidor no debe inferir exito del guardian por exit code ni tratar cualquier
fallo break-glass como cierre de promocion. La frontera vigente consume
`orquesta_guardian_result.v0`, reason codes y evidence refs compactas; solo
`candidate_promoted` cierra el efecto cuando la promocion real esta habilitada.

T208 queda como umbrella historico porque los huecos concretos se cerraron en
owners focales. Nuevos blockers deben entrar como tareas separadas, manteniendo
guardian, Codex, filesystem, proveedor y configuracion operacional fuera del
modulo residente puro.

## SRV-012: Director residente opt-in por puerto

El servidor puede alojar un pulso residente del Director, pero no debe construir
briefings ni conocer Codex, OPES, MCP, modelos, HOME o persistencia concreta.
Por eso el contrato es `ResidentDirectorPortV0` y se activa solo con
`ConfigV0.ResidentDirectorEnabled=true`.

El pulso usa el mismo patron operativo que el supervisor: tick inicial, ticker,
wakeup no bloqueante por progreso durable, guarda atomica de anti-solape,
coalescing de un pulso pendiente, panic convertido en error durable y estado
publico `resident_director_*`. El self-watchdog trata actividad y progreso del
Director residente como causa operativa observable.

Composicion cerrada: `cmd/orquesta-server` inyecta un adaptador real opt-in que
usa `RunResidentDirectorBriefingLoopV0` con fuente de briefing reentrable desde
stores vivos. El modulo `orquesta-server` sigue puro: no construye briefings ni
conoce Codex, OPES, MCP, modelos, HOME ni persistencia concreta.

## SRV-013: reconciliar backlog cerrado antes de reabrir implementacion

Cuando automejora detecte un patron ya cerrado en docs locales, el servidor debe
preferir reconciliacion documental con evidencia causal antes de lanzar otro
padre de codigo. Para SRV-TASK-015, la evidencia vigente es:

- `modulos/orquesta-server/docs/tareas.md` declara el productor causal OPES como
  hecho local y lista pruebas focales;
- `docs/runbooks/opes_productor_causal_autonomo_2026-06-13.md` fija frontera:
  logica en `orquesta-opes-director`, servidor solo cablea puertos/wakeups y el
  nucleo no importa OPES;
- un intento posterior cerrado por apagado no toco archivos ni ejecuto pruebas,
  por lo que no invalida el cierre local ni justifica relanzar implementacion.

Si aparece una regresion causal, debe abrirse rework focal con refs concretas de
job, receipt o artifact. Si solo reaparece texto historico del patron, se cierra
como no-op documental y se deja ACK con `contexto_ref_only_resuelto`.

El rework de revision
`agent-ref-task-ref-review-rework-task-autoprogramming-874937b97f16-g01-162db2d326380eeab85c029bbfcfe285`
aplica esta decision: conserva la entrega documental ya valida, resuelve el
contexto obligatorio por refs y limita la validacion a
`go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server`.

El rework de revision sobre rework
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-autoprogr-7b57471b0af67dc475be23b72a6c25c7`
no cambia la decision ni reabre implementacion: solo completa el rastro causal
de la correccion rechazada, conserva el no-op documental y exige el mismo test
focal antes de ACK `completed`.

La correccion posterior
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-8f75b93913fef84c105ce29cf9734bca`
mantiene la misma decision. Su unico alcance es sincronizar la evidencia local
de esta revision con el backlog y docs del servidor; cualquier cambio de codigo
requiere regresion causal nueva con refs concretas de job, receipt o artifact.

El agente de reemplazo
`agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-0b2-f385b5e4533d62b5c7d2a64b70950490`
no cambia la decision: conserva la entrega documental valida, resuelve el
`ref_only` por evidencia local disponible y no relanza padre de codigo sobre la
tarea original.

El agente externo
`agent-ref-task-ref-review-rework-task-autoprogramming-30d589fce14c-g01-8bb7a3491960a73cba493ed199be20e1`
mantiene la misma decision: el paquete solo reobserva SRV-TASK-015 con contexto
obligatorio `ref_only` y write-set cerrado al servidor, por lo que se cierra
como no-op documental salvo regresion causal nueva con refs concretas.

La correccion
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-dacb62e5ec1ae158a640baa674e72940`
mantiene esa decision: evidencia `ref_only` local suficiente, sin relanzar otro
padre ni programar codigo, y cierre condicionado al test focal obligatorio.

La correccion externa
`agent-ref-task-ref-review-rework-task-autoprogramming-abea33163b68-g01-04bebd80bd7c5afa1ff0cde8543ec8ec`
mantiene esa decision: el contrato solo exige corregir la entrega rechazada,
conservar lo valido y resolver `ref_only` por evidencia local; no reabre
implementacion ni relanza otro padre sin regresion causal nueva con refs
concretas.

La correccion externa
`agent-ref-task-ref-review-rework-task-autoprogramming-99f93b5dadeb-g01-46ac951607ee8f489f914ee25e92cb88`
queda corregida como ref de SRV-TASK-024. No modifica la decision de SRV-TASK-015
ni se usa como evidencia de cierre de ese owner.

El assessment externo
`agent-ref-assessment-task-ref-review-rework-task-autoprogramming-99f93b5dadeb-g01-46a-de83dfc8cce6e142d5f9b4a9f2a47b24`
queda corregido como assessment de SRV-TASK-024. No cambia la frontera
documental de SRV-TASK-015.

## SRV-014: SRV-TASK-024 no se cierra por rastro documental

La request
`request-ref-autoprogramming-backlog-srv-task-024-90719eb4-reconcile-c83a27e7`
reconcilia un backlog real sobre runs OPES en `resident_director_pending` sin
agente visible. Las entregas documentales pueden conservar evidencia y
normalizar refs, pero no sustituyen el cierre tecnico: debe existir codigo en
composicion/servidor que despache o publique bloqueo causal y un smoke OPES
acotado que lo demuestre.

## SRV-015: app-server goal-first es composicion

`cmd/orquesta-server` puede usar `codex app-server proxy` como backend
goal-first opt-in. Ese codigo queda en composition root: arranca threads,
configura goals, inicia turns, observa `thread/goal/get` y lee `thread/read`
para extraer `ORQUESTA_GOAL_RESULT_V0`. El modulo servidor y el nucleo neutral
siguen viendo solo puertos y refs opacas; un `complete` de Codex no cierra nada
sin `GoalWorkClosureValidatorV0`.

Si el transporte app-server no esta disponible, la composicion no debe volver al
loop legacy como si nada. El preflight crea un backend goal degradado que falla
con reason codes compactos y accionables, manteniendo la frontera opt-in:
goal-first configurado significa goal-first o bloqueo explicito.

La correccion
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-autoprogr-dc31be1188c5569b01263b5f388f0788`
aplica esta decision: corrige la asociacion causal de las refs previas a
SRV-TASK-024, resuelve `ref_only` por evidencia local y no relanza otro padre ni
abre implementacion fuera del write-set documental.

El assessment externo
`agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-autoprogr-dc3-0ff5f84c9556672a698237df3988111c`
no cambia la decision: conserva la evidencia valida, resuelve el nuevo
`ref_only` por paquete y docs locales y deja el cierre tecnico en SRV-TASK-024.

El assessment externo de reemplazo
`agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-autoprogr-dc3-36f24a2209e93ca38f4443ce22f527cf`
aplica la misma decision: no relanza otro padre, no abre implementacion dentro
de un write-set documental y solo puede cerrar su entrega si la prueba focal
obligatoria pasa; SRV-TASK-024 sigue abierto hasta evidencia tecnica causal.

La correccion de entrega
`agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-autoprogr-dc3-a5a32e27fa8bf23a4b29ee07490e9a10`
mantiene esa decision. El `ref_only` se resuelve por paquete y docs locales; no
hay regresion causal nueva ni permiso de codigo en el write-set, por lo que el
cierre tecnico de SRV-TASK-024 sigue dependiendo de dispatch real o bloqueo
causal publico con smoke OPES acotado.

La correccion de burst 003
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-autoprogr-72998ee36295059452b54a9eb4e875ee`
mantiene la misma decision: conserva la evidencia documental valida, resuelve
`ref_only` por paquete y docs locales, no reabre implementacion ni relanza
padre; SRV-TASK-024 sigue abierto hasta dispatch real o bloqueo causal publico
probado por smoke OPES acotado.

El assessment externo
`agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-autoprogr-729-9fc3f13d44dbaff94cfe5adff747a3e9`
mantiene esa decision: la correccion revisada se conserva como evidencia
documental valida, el `ref_only` se resuelve por paquete y fuentes locales, y
no se relanza padre ni se abre codigo dentro del write-set. El cierre tecnico
de SRV-TASK-024 sigue dependiendo de dispatch real o bloqueo causal publico
probado por smoke OPES acotado.

La correccion externa
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-cfd9c13a92a5cd21fd6c622e6f405e46`
aplica la misma decision: completa solo el rastro documental de la entrega
rechazada, conserva lo valido, resuelve `ref_only` por paquete y docs locales,
no relanza padre ni abre codigo, y no cierra SRV-TASK-024 sin dispatch real o
bloqueo causal publico probado por smoke OPES acotado.

El assessment externo
`agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-cfd-c006963b24043636d557f12d4659b4a3`
mantiene la misma decision: corrige solo la entrega documental de su evaluacion,
resuelve `ref_only` por paquete y docs locales, no relanza padre ni abre codigo
dentro del write-set. El cierre tecnico de SRV-TASK-024 sigue dependiendo de
dispatch real o bloqueo causal publico probado por smoke OPES acotado.

El assessment externo de reemplazo
`agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-cfd-791ce4584f28597b364194874b2875a4`
mantiene esa decision: conserva la entrega documental valida, resuelve
`ref_only` por paquete de control y fuentes locales, no relanza padre ni abre
codigo dentro del write-set documental. El cierre tecnico de SRV-TASK-024 sigue
dependiendo de dispatch real o bloqueo causal publico probado por smoke OPES
acotado.

La correccion externa
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-99773dd28e4df93406b2912127997d04`
mantiene la misma decision: completa solo el rastro documental de la entrega
rechazada, resuelve `ref_only` por paquete de control y docs locales, no
relanza padre ni abre codigo dentro del write-set documental. El cierre tecnico
de SRV-TASK-024 sigue dependiendo de dispatch real o bloqueo causal publico
probado por smoke OPES acotado.

La correccion externa
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-30ffd078ad6ee9febe4c50ac0bd29891`
mantiene esa decision: completa solo el rastro documental de la entrega nueva,
resuelve `ref_only` por paquete de control, AGENTS raiz, README/AGENTS locales
del servidor y docs locales, no relanza padre ni abre codigo dentro del
write-set documental. El cierre tecnico de SRV-TASK-024 sigue dependiendo de
dispatch real o bloqueo causal publico probado por smoke OPES acotado.

El assessment externo
`agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-30f-c4501a4d6b170b706b55f9a3dd4a44a6`
mantiene esa decision: corrige solo el rastro documental de la evaluacion
nueva, resuelve `ref_only` por paquete de control, AGENTS raiz, README/AGENTS
locales del servidor y docs locales, no relanza padre ni abre codigo dentro del
write-set documental. El cierre tecnico de SRV-TASK-024 sigue dependiendo de
dispatch real o bloqueo causal publico probado por smoke OPES acotado.

La correccion externa
`agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-30f-aad45cf56f890e04e96c6b86b7c9bc7b`
mantiene esa decision: corrige solo el rastro documental del replan por
evaluacion, resuelve `ref_only` por paquete de control, AGENTS raiz,
README/AGENTS locales del servidor y docs locales, no relanza padre ni abre
codigo dentro del write-set documental. El cierre tecnico de SRV-TASK-024 sigue
dependiendo de dispatch real o bloqueo causal publico probado por smoke OPES
acotado.

El assessment externo
`agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-30f-6842075f2c746477bb06b52d2c34e0c5`
mantiene la misma decision: completa solo el rastro documental de esta
correccion tras revision, resuelve `ref_only` por paquete de control, AGENTS
raiz, README/AGENTS locales del servidor y docs locales, no relanza padre ni
abre codigo dentro del write-set documental. El cierre tecnico de SRV-TASK-024
sigue dependiendo de dispatch real o bloqueo causal publico probado por smoke
OPES acotado.

La correccion externa
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-9ba51f68ffa908088c91ca219cb73b84`
mantiene la misma decision: completa solo el rastro documental de esta entrega,
resuelve `ref_only` por paquete de control, AGENTS raiz, README/AGENTS locales
del servidor y docs locales, no relanza padre ni abre codigo dentro del
write-set documental. El cierre tecnico de SRV-TASK-024 sigue dependiendo de
dispatch real o bloqueo causal publico probado por smoke OPES acotado.

La correccion externa
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-5f1990c9bda61f11e4c4572d7d8ec5f9`
mantiene la misma decision: completa solo el rastro documental de esta entrega
nueva, resuelve `ref_only` por paquete de control, AGENTS raiz, README/AGENTS
locales del servidor y docs locales, no relanza padre ni abre codigo dentro del
write-set documental. El cierre tecnico de SRV-TASK-024 sigue dependiendo de
dispatch real o bloqueo causal publico probado por smoke OPES acotado.

La correccion externa
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-2f664d21881795ee2d8392327887480f`
mantiene la misma decision: completa solo el rastro documental de esta entrega
nueva, resuelve `ref_only` por paquete de control, AGENTS raiz, README/AGENTS
locales del servidor y docs locales, no relanza padre ni abre codigo dentro del
write-set documental. El cierre tecnico de SRV-TASK-024 sigue dependiendo de
dispatch real o bloqueo causal publico probado por smoke OPES acotado.

El assessment externo
`agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-2f6-4a88d678723f47ef86524e438127a1e1`
mantiene la misma decision: completa solo el rastro documental de la evaluacion
nueva, resuelve `ref_only` por paquete de control, AGENTS raiz, README/AGENTS
locales del servidor y docs locales, no relanza padre ni abre codigo dentro del
write-set documental. El cierre tecnico de SRV-TASK-024 sigue dependiendo de
dispatch real o bloqueo causal publico probado por smoke OPES acotado.

La correccion externa
`agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-2f6-d07cfaec65f49155f8328d0512377fd7`
mantiene esa decision: corrige solo la entrega rechazada de la evaluacion,
resuelve `ref_only` por paquete de control, AGENTS raiz, README/AGENTS locales
del servidor y docs locales, no relanza padre ni abre codigo dentro del
write-set documental. El cierre tecnico de SRV-TASK-024 sigue dependiendo de
dispatch real o bloqueo causal publico probado por smoke OPES acotado.

El assessment externo de reemplazo
`agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-2f6-20f9ab529a94f082f863a30e7a9275dd`
mantiene esa decision: corrige solo la evaluacion previa sin ACK, resuelve
`ref_only` por paquete de control, AGENTS raiz, README/AGENTS locales del
servidor y docs locales, no relanza padre ni abre codigo dentro del write-set
documental. El cierre tecnico de SRV-TASK-024 sigue dependiendo de dispatch
real o bloqueo causal publico probado por smoke OPES acotado.

La correccion externa
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-d20f765b96bef254281f4683e6ab3480`
mantiene esa decision: completa solo el rastro documental de la entrega nueva,
resuelve `ref_only` por paquete de control, AGENTS raiz, README/AGENTS locales,
foto vigente y docs locales, no relanza padre ni abre codigo dentro del
write-set documental. El cierre tecnico de SRV-TASK-024 sigue dependiendo de
dispatch real o bloqueo causal publico probado por smoke OPES acotado.

La correccion externa
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-9e9d33b777c6565a38a9b5a6eb7529a2`
mantiene esa decision: completa solo el rastro documental de esta entrega nueva,
resuelve `ref_only` por paquete de control, AGENTS raiz, README/AGENTS locales,
foto vigente y docs locales, no relanza padre ni abre codigo dentro del
write-set documental. El cierre tecnico de SRV-TASK-024 sigue dependiendo de
dispatch real o bloqueo causal publico probado por smoke OPES acotado.

La correccion externa
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-2b5521cd1c1dbd4767cebb3d9c1684d7`
mantiene esa decision: completa solo el rastro documental de esta entrega nueva,
resuelve `ref_only` por paquete de control, AGENTS raiz, README/AGENTS locales,
foto vigente y docs locales, no relanza padre ni abre codigo dentro del
write-set documental. El cierre tecnico de SRV-TASK-024 sigue dependiendo de
dispatch real o bloqueo causal publico probado por smoke OPES acotado.

La correccion externa
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-15e93ad10820843acec443a42cbac3c7`
mantiene esa decision: completa solo el rastro documental de esta entrega nueva,
resuelve `ref_only` por paquete de control, AGENTS raiz, README/AGENTS locales,
foto vigente y docs locales, no relanza padre ni abre codigo dentro del
write-set documental. El cierre tecnico de SRV-TASK-024 sigue dependiendo de
dispatch real o bloqueo causal publico probado por smoke OPES acotado.

La correccion externa
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-0f0bdd6085651f057155f6a385ccea82`
mantiene esa decision: completa solo el rastro documental de esta entrega
nueva, resuelve `ref_only` por paquete de control, AGENTS raiz, README/AGENTS
locales, foto vigente y docs locales, no relanza padre ni abre codigo dentro del
write-set documental. El cierre tecnico de SRV-TASK-024 sigue dependiendo de
dispatch real o bloqueo causal publico probado por smoke OPES acotado.

La correccion externa
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-d4aec6d93452fd3b4f10a49c0bca437c`
mantiene esa decision: corrige solo esta entrega documental, resuelve
`ref_only` por paquete de control, AGENTS raiz, README/AGENTS locales, foto
vigente y docs locales, no relanza padre ni abre codigo dentro del write-set
documental. El cierre tecnico de SRV-TASK-024 sigue dependiendo de dispatch
real o bloqueo causal publico probado por smoke OPES acotado.

La correccion externa
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-403a707ea7e18c0c29793d3df9266ede`
mantiene esa decision: completa solo el rastro documental de esta entrega,
resuelve `ref_only` por paquete de control, AGENTS raiz, README/AGENTS locales,
foto vigente y docs locales, no relanza padre ni abre codigo dentro del
write-set documental. El cierre tecnico de SRV-TASK-024 sigue dependiendo de
dispatch real o bloqueo causal publico probado por smoke OPES acotado.

La correccion externa
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-026aefcaa6977288909f6693516c13c1`
mantiene esa decision: completa solo el rastro documental de esta entrega
nueva, resuelve `ref_only` por paquete de control, AGENTS raiz, README/AGENTS
locales, foto vigente y docs locales, no relanza padre ni abre codigo dentro
del write-set documental. El cierre tecnico de SRV-TASK-024 sigue dependiendo
de dispatch real o bloqueo causal publico probado por smoke OPES acotado.

## SRV-016: configuracion efectiva de automejora idle

`ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER_SECONDS` es la variable canonica
para el disparador idle. El alias historico
`ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_AFTER` se acepta solo por compatibilidad
cuando la canonica no existe; el estado efectivo publica un diagnostico de
migracion. Si ambas existen, gana la canonica y el alias queda ignorado con
diagnostico publico.
