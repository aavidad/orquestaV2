<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Runbook operativo del control plane de agentes

## Objetivo

Definir el procedimiento operativo mínimo para supervisar y operar el control plane de agentes en Orquesta sin recurrir a acceso directo a la base de datos ni a atajos fuera de la app.

Este runbook se apoya en:

- `docs/operacion_agentes_manuales.md`
- `docs/diseno_control_activo_agentes.md`
- `docs/diseno_observabilidad_pasiva_runtimes_es.md`
- la CLI real disponible en `orquesta agente` y `orquesta runtime`

## Alcance

Este documento cubre:

- verificación de estado del control plane
- inspección de runtimes, handles, órdenes y checkpoints
- preparación de agentes
- handoff operativo
- pautas de diagnóstico cuando un runtime o una orden se degrada

No cubre:

- edición manual de tablas
- recuperación forense avanzada de SQLite
- despliegue systemd o Docker del servicio

## Principios operativos

- Orquesta es la fuente de verdad para sesiones, órdenes, checkpoints y continuidad.
- La web, la CLI y futuros clientes deben operar contra Orquesta, no contra la persistencia directa.
- La observabilidad primaria debe ser pasiva.
- El handoff y la continuidad deben dejar trazabilidad explícita.
- Si una operación no existe por CLI, API o web, no debe improvisarse por acceso lateral a la BD.
- Workers: salida mínima y evidencia mínima suficiente.
- Si el flujo lo permite, preferir `caveman`/`compact` o compresión equivalente antes que subir modelo.
- Escalar a `prime` solo con evidencia operativa, no por intuición.

Politica consolidada de operacion:

- mantener `supervisor residente` por proyecto como capacidad de gobierno, no como worker generico
- dejar que el `control plane` opere de forma `event-driven`; si no hay incidencia real, debe tender a `standby`, no a churn
- intentar primero `repair-helper` local y estrecho
- relevar a otro worker barato antes de escalar
- subir a `prime` solo con evidencia operativa clara

## Preflight de operador

Antes de tocar agentes vivos:

1. comprobar que el servicio de Orquesta responde
2. identificar el proyecto y el agente afectados
3. confirmar si hay tarea viva asociada
4. revisar si existe continuidad o checkpoint reciente
5. verificar si el problema es de runtime, de orden o de sesión

Comandos útiles:

```bash
./orquesta server status
./orquesta server doctor
./orquesta status
./orquesta runtime listar
./orquesta runtime handles
./orquesta runtime ordenes
./orquesta runtime checkpoints --agente Codex1
```

Paneles web útiles:

- `/runtimes`
- `/time-travel`

## Arranque operativo normal

Flujo recomendado:

1. registrar o confirmar agente, proyecto y tarea
2. dejar que el daemon procese `runtime_orders` y gestione el ciclo de vida
3. usar `orquesta agente preparar` solo para inspección, compatibilidad o rescate
4. observar órdenes, runtimes y checkpoints desde CLI/web/API
5. recurrir a scripts manuales solo si el camino oficial daemon/API/control plane no está disponible

Ejemplos:

```bash
./orquesta server status
./orquesta status
./orquesta runtime ordenes
./orquesta agente preparar Codex1 --proyecto orquestador --perfil implementacion --json
./orquesta runtime listar
```

## Estado actual del daemon

Contrato operativo vigente:

- la via oficial para dejar Orquesta residente es `./orquesta server start`; `./orquesta server run` queda como ejecucion foreground
- un daemon sano no se considera listo solo por arrancar proceso: debe responder `healthz`, publicar `statefile` coherente y devolver `/api/status` decodificable
- `./orquesta server status` y `./orquesta server doctor` ya pueden recuperar el daemon vivo por `healthz` cuando el `statefile` falta o esta stale, siempre sobre el mismo `scope` y el mismo storage
- el control plane real drena `runtime_orders` desde el daemon; el modo local queda solo para recuperacion explicita

Comprobacion minima recomendada:

```bash
./orquesta server status
./orquesta server doctor
./orquesta status
```

Interpretacion:

- si `server doctor` muestra `Health RPC (daemon): OK` y `API server: OK`, el plano server-first esta vivo
- si `server status` informa recuperacion por `healthz`, el daemon sigue siendo valido pero conviene revisar por que el `statefile` no estaba utilizable
- si `status` responde pero `server doctor` no llega a `API server: OK`, no des por buena la operacion continua del daemon

## Telemetria visible en `status`

`status` y `/api/status` publican ya una foto compacta de capacidad visible:

- `workersConectados`: workers visibles conectados y disponibles; descuenta supervisores reservados para no inflar el cupo ejecutor
- `workersTrabajando`: workers visibles con trabajo real activo; tambien descuenta supervisores reservados
- `supervisoresActivos`: supervisores visibles en rol de supervision activa, publicados aparte del cupo de workers

Reglas de lectura:

- `agentesActivos` sigue siendo la lista detallada; `workersConectados` es su resumen operativo para workers reales
- `agentesTrabajando` sigue siendo la lista detallada; `workersTrabajando` resume cuantos workers estan ejecutando ahora
- `supervisoresActivos` no se suma a workers: es capacidad de gobierno, no capacidad ejecutora
- estos contadores son la semantica operativa visible canónica y deben reutilizarse tal cual en CLI, web, MCP y briefing de supervisor
- no se deben reinterpretar como simple longitud de `agentesActivos` o `agentesTrabajando`

Lectura operativa rapida:

- `workersConectados > workersTrabajando` indica capacidad visible libre
- `workersTrabajando = 0` con `supervisoresActivos > 0` indica supervision presente sin ejecucion visible
- si `workersConectados = 0`, no asumas que no hay daemon; primero valida `server doctor` y revisa si la flota esta bloqueada por cuota, enfriamiento o degradacion

Lectura de politica:

- `supervisoresActivos > 0` con `workersConectados = 0` no es necesariamente error; puede significar supervisor residente en espera de trabajo o en fase de gobierno
- `workersConectados > 0` con `workersTrabajando = 0` describe capacidad libre
- `workersTrabajando > 0` con `autonomyPending = 0` y `autonomyContinuing = 0` describe ejecucion ya absorbida y confirmada

## Inspección diaria

Qué mirar primero:

- runtimes visibles y su estado lógico
- handles activos por agente
- órdenes pendientes o fallidas
- último checkpoint por agente
- divergencias entre sesión esperada y runtime observado

CLI útil:

```bash
./orquesta runtime listar
./orquesta runtime ver <runtime-id>
./orquesta runtime handles
./orquesta runtime ordenes
./orquesta runtime checkpoints --agente Codex1
```

Señales sanas:

- el agente aparece en `runtime listar`
- existe handle activo coherente con el proyecto
- las órdenes avanzan de `pendiente` a `completada`
- hay checkpoints recientes cuando el flujo exige continuidad

Señales de degradación:

## Control total y estadisticas

Objetivo operativo:

- poder responder sin shell manual que ha hecho un agente en una ventana temporal
- poder ver por proyecto que ramas, worktrees, diffs y merges estan activos
- poder correlacionar tarea, agente, runtime, worktree y evidencia Git

Estado real hoy:

- ya existe control total por proyecto en `/api/proyectos/{slug}/cockpit`
- ya existe control total por agente con `orquesta agente actividad <agente> --desde <ventana>`
- ya existen `GET /api/audit` y `GET /api/runtime-transcript` para reconstruccion server-first de actividad
- ya existe base de estadisticas Git por repo/cwd con ficheros tocados y lineas añadidas/borradas
- supervisor y reviewer ya se tratan como reservas; los workers reales son el cupo ejecutor visible
- la foto operativa estable que debe esperarse hoy es `autonomyPending=0`
- lo pendiente queda en la capa final unificada de timeline/estadisticas globales de `OP 096`

Contrato deseado de observabilidad:

- por agente: tarea actual, tiempo activo, handoffs, repairs, archivos tocados, lineas añadidas/borradas, presupuesto y ultima evidencia real
- por proyecto: fase, porcentaje, timeline, hotspots Git, flota, bloqueos y coste agregado
- global: capacidad real, throughput, bloqueos recurrentes y consumo total

Regla:

- si para responder `que ha hecho Codex1 en la ultima hora` hace falta shell manual, la app aun no esta cerrada
- la respuesta correcta debe salir del cockpit, timeline y estadisticas canónicas de Orquesta

Comandos ya utiles en el estado actual:

```bash
./orquesta agente actividad Codex1 --desde 1h
./orquesta agente actividad Codex1 --proyecto orquestador --desde 24h --json
./orquesta runtime transcript --agente Codex1 --proyecto orquestador
./orquesta logs --agente Codex1 --desde 1h
```

Politica operativa de coste en supervision:

- preferir salida compacta y evidencia corta
- si el runtime lo permite, usar `caveman`/`compact` o equivalente
- probar `repair-helper` local o relevo barato antes de promover un frente normal a `prime`
- reservar `prime` para bloqueo real, reinicio reincidente, review repetidamente fallida o criticidad alta

- runtime ausente pero sesión supuestamente activa
- órdenes acumuladas sin progreso
- checkpoint inexistente antes de un handoff
- agente con heartbeat desactualizado y sin evento de cierre

## Handoff operativo

Usar handoff cuando:

- la sesión actual está degradada
- el presupuesto o contexto recomiendan relevo
- el agente origen no debe continuar con trabajo útil

Comando base:

```bash
./orquesta agente handoff Codex1 Codex2 --tarea 340 --motivo "relevo operativo" --resumen "continuar por API de runtimes"
```

Reglas:

- incluir `--tarea` si la reasignación afecta a una tarea viva
- incluir `--resumen` siempre que haya contexto no trivial
- incluir `--external-session-id` si ya se conoce y es relevante para la reanudación

Validaciones posteriores:

1. revisar que la tarea cambió de agente si aplicaba
2. verificar orden o rastro de continuidad en runtime
3. comprobar checkpoint asociado al relevo
4. verificar que el agente destino puede preparar bundle coherente

## Uso de checkpoints

Los checkpoints son la base mínima para continuidad segura.

Mirarlos cuando:

- un agente va a ser relevado
- una orden parece bloqueada
- una sesión murió sin cierre limpio
- hace falta reconstruir contexto operativo reciente

Comandos:

```bash
./orquesta runtime checkpoints --agente Codex1
./orquesta runtime ver <runtime-id>
```

Regla práctica:

- si no hay checkpoint reciente y el caso exige relevo, no asumir continuidad implícita

## Diagnóstico de órdenes

Las `runtime_orders` son la cola operativa del control plane.

Qué hacer si una orden no progresa:

1. listar órdenes y localizar tipo y estado
2. comprobar si existe runtime o handle activo para ese agente
3. revisar último checkpoint y último heartbeat
4. decidir si basta con reintentar desde la app o si hace falta handoff

Comando:

```bash
./orquesta runtime ordenes
```

Criterios:

- si hay runtime sano y la orden sigue atascada, investigar el dispatcher o el transporte
- si no hay runtime sano, la orden bloqueada es un síntoma y no la causa
- si el agente quedó sin continuidad suficiente, priorizar checkpoint y relevo

## Incidencias típicas

### 1. Sesión viva sin runtime visible

Posibles causas:

- cierre anómalo del proceso
- handle desactualizado
- sesión arrancada fuera del flujo recomendado

Acción:

- comprobar `runtime handles`
- revisar último checkpoint
- decidir entre reanudar manualmente o hacer handoff

### 2. Orden pendiente durante demasiado tiempo

Posibles causas:

- dispatcher parado
- transporte no resoluble
- runtime ya no existe

Acción:

- revisar `runtime ordenes`
- revisar `runtime listar`
- si no hay runtime sano, no seguir encolando órdenes del mismo tipo sin diagnóstico

### 3. Handoff sin continuidad útil

Posibles causas:

- resumen demasiado pobre
- checkpoint ausente
- external session id no guardado

Acción:

- regenerar continuidad desde la información disponible
- dejar resumen explícito antes del nuevo relevo
- evitar cadenas de handoff sin checkpoint intermedio

## Qué no hacer

- no usar `sqlite3` para mutar estado operativo normal
- no asumir que `resume --last` del runtime equivale a continuidad correcta
- no hablar directamente con el proceso o PTY desde la web o un cliente fino
- no iniciar handoff sin revisar tarea viva y continuidad
- no tratar una orden atascada como incidente aislado si el runtime está caído

## Checklist de cierre de incidencia

Antes de dar una incidencia por cerrada:

1. el runtime vuelve a aparecer o el relevo quedó hecho
2. la tarea viva quedó asignada al agente correcto
3. existe checkpoint o resumen útil para el siguiente relevo
4. no quedan órdenes pendientes engañosas del incidente
5. la trazabilidad en Orquesta refleja lo ocurrido

## Evolución esperada

Cuando la cobertura CLI/API siga creciendo, este runbook deberá ampliarse con:

- operaciones explícitas sobre mailbox
- políticas automáticas de watchdog
- procedimientos de stale order recovery
- operación daemon/systemd de servicio único
