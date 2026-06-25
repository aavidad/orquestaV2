# orquesta-server

Servidor residente de Orquesta.

Este modulo permite ejecutar Orquesta como proceso independiente de la consola
que lo lanza. Expone `/healthz` como liveness del proceso,
`/api/v0/server/readiness` como readiness operativa,
`/api/v0/server/status` como estado publico y delega el resto del trafico al
handler de aplicacion inyectado. `/api/status` queda solo como alias legacy con
headers de deprecacion/canonical.

El supervisor global se ejecuta por pulsos acotados mediante un puerto. Si el
usuario cierra la sesion de Codex, el daemon sigue vivo y el operador puede
reengancharse leyendo el statefile y consultando la API.

## Uso local y supervision

Para trabajo local, arranca el residente desde la raiz con
`go run ./cmd/orquesta-server run`. `/healthz` solo confirma socket/proceso
vivo; antes de lanzar Codex, OPES, `domain_work` o automejora usa
`GET /api/v0/server/readiness` y exige `ready=true`. La supervision operativa
debe hacerse contra las APIs publicas del servidor, por ejemplo
`GET /api/v0/server/status`, `POST /api/v0/autoprogramming/status` y
`POST /api/v0/autoprogramming/supervise`; el supervisor avanza por pulsos
acotados y no debe sustituirse por acceso directo a stores, runtime, filesystem
o procesos internos.

La composicion productiva configura presupuesto de progreso para agentes Codex
por entorno:

- `ORQUESTA_CODEX_MAX_EXPECTED_SECONDS`: tiempo esperado antes de pedir decision.
- `ORQUESTA_CODEX_NO_ACTIVITY_SECONDS`: ventana sin actividad antes de marcar
  riesgo operativo.
- `ORQUESTA_CODEX_STALLED_TICKS`: muestras sin cambio compacto antes de pedir
  atencion del director. Por defecto son 300 ticks con intervalo de 2s.
- `ORQUESTA_CODEX_LOOP_TICKS`: muestras repetidas antes de marcar posible
  bucle. Por defecto son 300 ticks con intervalo de 2s.

Estos valores alimentan estadisticas de director y no pertenecen al nucleo.

La proyeccion viva de progreso cerrada por T210 se transporta desde
`DirectorRunStatsV0`: entregas, agentes iniciados y proceso registrado pueden
elevar `percent_complete`, pero `TasksClosed` permanece ligado a cierre/review
aceptada. El servidor residente no debe rellenar 0% por defecto si la fuente
trae una senal viva ni recalcular esa semantica desde runtime o filesystem.

## Reconciliacion de backlog residente

La automejora residente puede cerrar patrones ya resueltos mediante evidencia
documental causal, sin relanzar otra implementacion. Para SRV-TASK-015, el
productor causal OPES queda reconciliado por las docs locales y el runbook del
2026-06-13: el servidor solo cablea puertos y wakeups; la logica OPES vive en
adaptadores de dominio y el nucleo sigue neutral. Un nuevo intento solo procede
si aporta regresion causal con refs de job, receipt o artifact.
El rework de revision `874937b97f16-g01-162db2d326380eeab85c029bbfcfe285`
mantiene esa frontera: contexto `ref_only` se resuelve por evidencia documental
y no por otra implementacion residente.
El rework sobre rework
`7b57471b0af67dc475be23b72a6c25c7` conserva la misma frontera: no relanza padre
de codigo, no amplia write-set y solo cierra con la prueba focal del servidor.
El rework de revision materializado como
`8f75b93913fef84c105ce29cf9734bca` conserva ese no-op documental: resuelve el
contexto `ref_only` por evidencia local, no toca codigo y exige el mismo test
focal antes del ACK.
El agente de reemplazo
`0b2-f385b5e4533d62b5c7d2a64b70950490` mantiene ese cierre: corrige solo la
entrega documental rechazada, conserva la evidencia valida y no relanza otro
padre ni abre implementacion sin regresion causal nueva.
El agente externo
`30d589fce14c-g01-8bb7a3491960a73cba493ed199be20e1` aplica el mismo cierre:
resuelve el contexto `ref_only` por evidencia local y limita la correccion al
rastro documental y a la prueba focal obligatoria.
La correccion
`dacb62e5ec1ae158a640baa674e72940` conserva esa decision: no relanza padre,
no toca codigo y cierra solo si la prueba focal obligatoria del servidor pasa.
La correccion externa
`abea33163b68-g01-04bebd80bd7c5afa1ff0cde8543ec8ec` mantiene el mismo cierre:
sin regresion causal nueva, resuelve `ref_only` por evidencia local, conserva
la entrega documental valida y exige la prueba focal obligatoria.
La correccion externa
`99f93b5dadeb-g01-46ac951607ee8f489f914ee25e92cb88` no cambia esa frontera:
queda corregida como rastro de SRV-TASK-024, no como nueva evidencia de
SRV-TASK-015.
El assessment externo
`de83dfc8cce6e142d5f9b4a9f2a47b24` queda igualmente asociado a SRV-TASK-024.
La correccion
`dc31be1188c5569b01263b5f388f0788` conserva lo valido de esas entregas: corrige
la asociacion causal, no relanza padre ni abre codigo dentro de un write-set
documental, y mantiene SRV-TASK-024 abierto hasta dispatch real o bloqueo
causal publico con smoke acotado.
El assessment externo
`agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-autoprogr-dc3-0ff5f84c9556672a698237df3988111c`
revalida esa frontera con contexto `ref_only`: solo sincroniza evidencia local,
ejecuta la prueba focal obligatoria y no sustituye el cierre tecnico pendiente.
El assessment externo de reemplazo
`agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-autoprogr-dc3-36f24a2209e93ca38f4443ce22f527cf`
mantiene el mismo alcance: corrige solo el rastro documental de la entrega
rechazada, resuelve `ref_only` por paquete y docs locales, y deja SRV-TASK-024
abierto hasta dispatch real o bloqueo causal publico probado.
La correccion de entrega
`agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-autoprogr-dc3-a5a32e27fa8bf23a4b29ee07490e9a10`
conserva ese alcance: no relanza padre, no toca codigo, resuelve `ref_only` por
paquete y docs locales, y solo cierra su propia entrega tras la prueba focal
obligatoria.
La correccion de burst 003
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-autoprogr-72998ee36295059452b54a9eb4e875ee`
conserva ese alcance: completa la evidencia de revision, resuelve `ref_only`
por paquete y docs locales, no abre codigo ni relanza padre, y mantiene
SRV-TASK-024 abierto hasta dispatch real o bloqueo causal publico probado.
El assessment externo
`agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-autoprogr-729-9fc3f13d44dbaff94cfe5adff747a3e9`
conserva esa misma frontera: valida la entrega documental, resuelve `ref_only`
por paquete y fuentes locales, no toca codigo ni relanza padre, y mantiene
SRV-TASK-024 abierto hasta dispatch real o bloqueo causal publico probado.
La correccion externa
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-cfd9c13a92a5cd21fd6c622e6f405e46`
conserva esa frontera: corrige solo el rastro documental de la entrega
rechazada, resuelve `ref_only` por paquete y docs locales, no toca codigo ni
relanza padre, y mantiene SRV-TASK-024 abierto hasta dispatch real o bloqueo
causal publico probado.
El assessment externo
`agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-cfd-c006963b24043636d557f12d4659b4a3`
mantiene ese cierre acotado: corrige solo el rastro documental de su entrega,
resuelve `ref_only` por paquete y docs locales, no toca codigo ni relanza padre,
y conserva SRV-TASK-024 abierto hasta dispatch real o bloqueo causal publico
probado.
El assessment externo de reemplazo
`agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-cfd-791ce4584f28597b364194874b2875a4`
mantiene la misma frontera: conserva el rastro documental valido, resuelve
`ref_only` por paquete y fuentes locales, no toca codigo ni relanza padre, y
conserva SRV-TASK-024 abierto hasta dispatch real o bloqueo causal publico
probado.
La correccion externa
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-99773dd28e4df93406b2912127997d04`
mantiene esa frontera: completa solo el rastro documental de su entrega,
resuelve `ref_only` por paquete y docs locales, no toca codigo ni relanza
padre, y conserva SRV-TASK-024 abierto hasta dispatch real o bloqueo causal
publico probado.
La correccion externa
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-30ffd078ad6ee9febe4c50ac0bd29891`
mantiene la misma frontera: completa solo el rastro documental de esta entrega
nueva, resuelve `ref_only` por paquete, AGENTS y docs locales, no toca codigo
ni relanza padre, y conserva SRV-TASK-024 abierto hasta dispatch real o bloqueo
causal publico probado.
El assessment externo
`agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-30f-c4501a4d6b170b706b55f9a3dd4a44a6`
conserva esa frontera: corrige solo el rastro documental de su evaluacion,
resuelve `ref_only` por paquete, AGENTS y docs locales, no toca codigo ni
relanza padre, y conserva SRV-TASK-024 abierto hasta dispatch real o bloqueo
causal publico probado.
La correccion externa
`agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-30f-aad45cf56f890e04e96c6b86b7c9bc7b`
conserva esa frontera: corrige solo el rastro documental del replan por
evaluacion, resuelve `ref_only` por paquete, AGENTS y docs locales, no toca
codigo ni relanza padre, y conserva SRV-TASK-024 abierto hasta dispatch real o
bloqueo causal publico probado.
El assessment externo
`agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-30f-6842075f2c746477bb06b52d2c34e0c5`
mantiene ese mismo alcance: completa solo el rastro documental de esta
correccion tras revision, resuelve `ref_only` por paquete, AGENTS y docs
locales, no toca codigo ni relanza padre, y conserva SRV-TASK-024 abierto hasta
dispatch real o bloqueo causal publico probado.
La correccion externa
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-9ba51f68ffa908088c91ca219cb73b84`
mantiene ese mismo alcance: completa solo el rastro documental de esta entrega,
resuelve `ref_only` por paquete, AGENTS y docs locales, no toca codigo ni
relanza padre, y conserva SRV-TASK-024 abierto hasta dispatch real o bloqueo
causal publico probado.
La correccion externa
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-5f1990c9bda61f11e4c4572d7d8ec5f9`
mantiene ese alcance: completa solo el rastro documental de esta entrega nueva,
resuelve `ref_only` por paquete, AGENTS raiz, README/AGENTS locales y docs
locales, no toca codigo ni relanza padre, y conserva SRV-TASK-024 abierto hasta
dispatch real o bloqueo causal publico probado.
La correccion externa
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-2f664d21881795ee2d8392327887480f`
mantiene ese alcance: completa solo el rastro documental de esta entrega nueva,
resuelve `ref_only` por paquete, AGENTS raiz, README/AGENTS locales y docs
locales, no toca codigo ni relanza padre, y conserva SRV-TASK-024 abierto hasta
dispatch real o bloqueo causal publico probado.
El assessment externo
`agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-2f6-4a88d678723f47ef86524e438127a1e1`
mantiene ese alcance: completa solo el rastro documental de la evaluacion,
resuelve `ref_only` por paquete, AGENTS raiz, README/AGENTS locales y docs
locales, no toca codigo ni relanza padre, y conserva SRV-TASK-024 abierto hasta
dispatch real o bloqueo causal publico probado.
La correccion externa
`agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-2f6-d07cfaec65f49155f8328d0512377fd7`
mantiene la misma frontera: corrige solo la entrega rechazada de esa evaluacion,
resuelve `ref_only` por paquete, AGENTS raiz, README/AGENTS locales y docs
locales, no toca codigo ni relanza padre, y conserva SRV-TASK-024 abierto hasta
dispatch real o bloqueo causal publico probado.
El assessment externo de reemplazo
`agent-ref-assessment-task-ref-review-rework-task-ref-review-rework-task-ref-revie-2f6-20f9ab529a94f082f863a30e7a9275dd`
mantiene esa frontera: corrige la evaluacion anterior sin ACK, resuelve
`ref_only` por paquete, AGENTS raiz, README/AGENTS locales y docs locales, no
toca codigo ni relanza padre, y conserva SRV-TASK-024 abierto hasta dispatch
real o bloqueo causal publico probado.
La correccion externa
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-d20f765b96bef254281f4683e6ab3480`
mantiene ese alcance: completa solo el rastro documental de esta entrega nueva,
resuelve `ref_only` por paquete, AGENTS raiz, README/AGENTS locales, foto
vigente y docs locales, no toca codigo ni relanza padre, y conserva
SRV-TASK-024 abierto hasta dispatch real o bloqueo causal publico probado.
La correccion externa
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-9e9d33b777c6565a38a9b5a6eb7529a2`
mantiene ese alcance: completa solo el rastro documental de esta entrega nueva,
resuelve `ref_only` por paquete, AGENTS raiz, README/AGENTS locales, foto
vigente y docs locales, no toca codigo ni relanza padre, y conserva
SRV-TASK-024 abierto hasta dispatch real o bloqueo causal publico probado.
La correccion externa
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-2b5521cd1c1dbd4767cebb3d9c1684d7`
mantiene ese alcance: completa solo el rastro documental de esta entrega nueva,
resuelve `ref_only` por paquete, AGENTS raiz, README/AGENTS locales, foto
vigente y docs locales, no toca codigo ni relanza padre, y conserva
SRV-TASK-024 abierto hasta dispatch real o bloqueo causal publico probado.
La correccion externa
`agent-ref-task-ref-review-rework-task-ref-review-rework-task-ref-revie-15e93ad10820843acec443a42cbac3c7`
mantiene la misma frontera: corrige solo el rastro documental de esta entrega
nueva, resuelve `ref_only` por paquete, AGENTS raiz, README/AGENTS locales, foto
vigente y docs locales, no toca codigo ni relanza padre, y conserva
SRV-TASK-024 abierto hasta dispatch real o bloqueo causal publico probado.

## Bridge OPES residente

`cmd/orquesta-server run` puede arrancar el bridge OPES por opt-in:

```sh
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:18080 \
ORQUESTA_OPES_BRIDGE_ENABLED=1 \
ORQUESTA_OPES_BRIDGE_CONFIRM=1 \
ORQUESTA_OPES_BRIDGE_LIMIT=1 \
ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE=draft_content_block,generate_visual_asset,review_legal,review_pedagogical,review_quality,validate_topic,assemble_topic,generate_audio_asset \
go run ./cmd/orquesta-server run
```

La secuencia de tipos es composicion OPES, no contrato del runtime residente.
Cada tick usa el loop generico y drena solo el primer tipo que siga pendiente.

## Guardian de autoprogramacion

La promocion residente puede exigir guardian externo por opt-in. En ese modo el
servidor consume `orquesta_guardian_result.v0`, trata bloqueos del guardian como
retryables y no sustituye el binario vivo si el candidato no queda promovido.
T208 queda reconciliado como umbrella historico; nuevos huecos deben abrir owner
focal.
