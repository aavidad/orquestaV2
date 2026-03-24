# Inventario De Pendientes Para Orquestación Autónoma

Fecha: `2026-03-24`
Responsable de esta pasada: `Codex1`

## Alcance

Este inventario resume lo que sigue pendiente para que Orquesta gobierne agentes vivos de forma autónoma, sin apoyarse en accesos directos a persistencia ni en flujos manuales como camino operativo principal.

No reemplaza la arquitectura ni las tareas existentes. Sirve como mapa operativo de cierre para repartir trabajo sin solapes.

## Ya Cubierto

- Cliente fino en `cmd/` para la mayor parte de lecturas y mutaciones de alta frecuencia.
- Control plane persistente con `runtime_orders`, `runtime_mailbox`, `runtime_checkpoints` y `handoff`.
- Adaptador de proceso con control real de `start`, `pause`, `resume`, `stop` y `send_instruction`.
- Observabilidad pasiva y diagnóstico de runtimes.
- Single-writer reforzado cuando hay servidor activo.

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

1. `#361`: unificar daemon oficial y runner operativo.
2. `#362`: sacar scripts del camino principal y pasarlos a cliente fino.
3. `#363`: humo E2E de runtime vivo.
4. `#357` y `#244`: cierre web/i18n sin pisar daemon.
5. `#364` y cortes restantes de `#227`: remates residuales.

## Criterio De Cierre

Se podrá considerar Orquesta autónoma sin matices cuando:

- toda operación de gobierno pase por daemon/API oficial
- los agentes vivos obedezcan órdenes desde el control plane, no desde lógica dispersa
- no haga falta acceso directo a BD como flujo normal
- los scripts de consola no sean la fuente de verdad operativa
- exista validación E2E con runtime vivo real
