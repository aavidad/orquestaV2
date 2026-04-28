# Inventario De Pendientes Para Orquestación Autónoma

Fecha: `2026-03-24`
Responsable de esta pasada: `Codex1`
Actualización de estado real: `2026-04-28`

## Alcance

Este inventario resume lo que sigue pendiente para que Orquesta gobierne agentes vivos de forma autónoma, sin apoyarse en accesos directos a persistencia ni en flujos manuales como camino operativo principal.

No reemplaza la arquitectura ni las tareas existentes. Sirve como mapa operativo de cierre para repartir trabajo sin solapes.

## Ya Cubierto

- Cliente fino en `cmd/` para la mayor parte de lecturas y mutaciones de alta frecuencia.
- Control plane persistente con `runtime_orders`, `runtime_mailbox`, `runtime_checkpoints` y `handoff`.
- Adaptador de proceso con control real de `start`, `pause`, `resume`, `stop` y `send_instruction`.
- Observabilidad pasiva y diagnóstico de runtimes.
- Single-writer reforzado cuando hay servidor activo.
- `finish_app` persistente ya operativo, con policy durable de proyecto, supervisor residente, reviewer reservado y workers acotados.
- Auto-creación de trabajo útil en supervision autonoma cuando el proyecto se queda sin frente activo.
- `repair-helper` barato, compactacion de frentes premium y drenaje de `prime` antes de escalado caro.
- Control total por proyecto y actividad/control total por agente ya visibles por superficies server-first.
- `autonomyPending=0` como foto estable de continuidad drenada.
- Mitigaciones server-first para retiro/fuera de orquestación:
  - no reactivar agentes retirados
  - consumir mailbox/autonomía residual dirigida a retirados
  - bloquear tareas huérfanas sin perder trazabilidad del último agente
- Degradación `workers_stuck` ya tratada por el control plane con restart coordinado, continuidad local por `session_resume`, `repair-helper` y cierre del helper cuando deja de ser útil

## Estado Real A 2026-04-28

### Cerrado o muy avanzado

- Bug de retiro bloqueante:
  - ya no debe tratarse como hueco estructural base
  - el repo ya contiene cobertura para retirada sin reactivación y para tarea huérfana por agente fuera de orquestación
  - la deuda restante es de proyección visible, no de ausencia de mecanismo

- Degradación `workers_stuck`:
  - ya existe como flujo real del control plane
  - el frente pendiente aquí es tuning, observabilidad y consistencia de lectura global

### Parcial

- Control total y estadísticas:
  - ya está operativo por agente y por proyecto
  - sigue parcial a escala global de workspace

### Abierto y de alto impacto

- Clasificación de transcript:
  - sigue siendo la deuda más visible del plano de observabilidad pasiva
  - el problema real no es una clase persistida `sin_clasificar`, sino muchas filas con `classification=''`
  - la mayor parte del volumen histórico parece ruido TUI/bootstrap no filtrado

## Pendiente Crítico

### 1. Camino oficial único de daemon

Tareas relacionadas:
- `#253`
- `#361`

Pendiente real:
- decidir y consolidar un único camino entre `serve`, `server` y el runner del plano de control
- evitar semánticas duplicadas de arranque, descubrimiento y operación continua
- dejar clara la vía oficial para producción y para `systemd`

Riesgo:
- mientras convivan varios caminos, el control plane puede comportarse distinto según cómo se arranque la app

### 1.b. Extender el control total observable al plano global

Pendiente real:
- consolidar una proyeccion canonica unica global a partir del control total ya operativo por agente/proyecto
- poder responder tambien a nivel workspace `que ha hecho X`, `que queda`, `cuanto se ha tocado` y `por que se ha escalado` sin recomposicion manual

Riesgo:
- sin esa capa global, la autonomia ya gobernable por agente/proyecto sigue siendo mas dificil de auditar a escala de workspace

### 1.c. Reducir `sin_clasificar` real en transcript y endurecer progreso semántico

Tareas relacionadas:
- `#33`
- `#35`

Pendiente real:
- bajar el volumen de `classification=''` en `runtime_transcript`
- distinguir mejor ruido TUI/bootstrap frente a progreso útil o evidencia de herramienta
- hacer más durable la señal de progreso semántico útil

Estado real actual:
- el clasificador ya reconoce señales fuertes de bloqueo, revisión, aprobación, replan, pánico y parte de `tool_*`
- la deuda dominante sigue en transcript pasivo ruidoso
- la fotografía histórica disponible en `backups/legacy-sqlite-20260422/orquesta.db` muestra `203500` filas de transcript y `203474` con `classification` vacío

Riesgo:
- sin este cierre, el control total por agente sigue contaminado y el supervisor depende demasiado de texto débil

### 2. Sustituir scripts/manualidades como vía operativa principal

Tareas relacionadas:
- `#256`
- `#362`

Pendiente real:
- migrar `runtime_connector` y `agente_console` a cliente fino contra API/servidor
- dejar los scripts como compatibilidad o bootstrap, no como lógica de negocio principal

Riesgo:
- si el runtime vivo depende de scripts con conocimiento propio de sesiones/runtimes, Orquesta no gobierna de forma realmente centralizada

### 3. Pruebas extremo a extremo con runtime vivo

Tareas relacionadas:
- `#363`

Pendiente real:
- validar el ciclo completo `orden -> proceso -> heartbeat -> watchdog -> handoff`
- cubrir recuperación, pausas, reanudación e instrucciones sobre runtime real

Riesgo:
- hoy la mayor parte del núcleo está probada por paquete, pero falta más humo real de operación continua

## Pendiente Importante

### 3.b. Cerrar del todo la lectura operativa del retiro en superficies globales

Pendiente real:
- reflejar de forma canónica en resúmenes globales cuando una tarea queda huérfana por agente retirado o fuera de orquestación
- evitar que reaparezca como “worker parado pero tarea en progreso” por una proyección incompleta

Riesgo:
- bajo a nivel de mecanismo base; medio a nivel de visibilidad y auditoría

### 4. Cierre de web cliente-fino

Tareas relacionadas:
- `#244`
- `#357`

Pendiente real:
- asegurar que la web deja de abrir persistencia local para consultas o mutaciones que ya están en API
- cerrar el frente i18n/web sin mezclarlo con la unificación del daemon

Riesgo:
- duplicar lógica entre web y CLI rompe el modelo de cliente fino

### 5. Cobertura CLI/API residual de observabilidad y administración

Tareas relacionadas:
- `#224`
- `#227`
- `#364`

Pendiente real:
- rematar los últimos comandos residuales de estado/diagnóstico
- seguir sustituyendo consultas manuales por rutas trazables en la app

Riesgo:
- bajo en comparación con daemon y runtimes, pero aún afecta a la disciplina de single-writer

## Orden Recomendado

1. `#33`: reducir `sin_clasificar` real del transcript y endurecer progreso semántico.
2. `#34`: cerrar la agregación global de control total y estadísticas.
3. `#361`: unificar daemon oficial y runner operativo.
4. `#363`: humo E2E de runtime vivo y revalidación de degradaciones.
5. `#357` y `#244`: cierre web/i18n sin pisar daemon.
6. `#364` y cortes restantes de `#227`: remates residuales.

## Criterio De Cierre

Se podrá considerar Orquesta autónoma sin matices cuando:

- toda operación de gobierno pase por daemon/API oficial
- los agentes vivos obedezcan órdenes desde el control plane, no desde lógica dispersa
- no haga falta acceso directo a BD como flujo normal
- los scripts de consola no sean la fuente de verdad operativa
- exista validación E2E con runtime vivo real
- exista observabilidad temporal y Git/coste suficiente por agente/proyecto y agregacion global para gobernar el sistema sin shell manual
- retiro/fuera de orquestación, `workers_stuck` y transcript ruidoso queden reflejados de forma canónica en las superficies de control
