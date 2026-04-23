<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Politica de supervisor residente y recuperacion local-first

## Estado

Este documento consolida una politica operativa ya reflejada de forma dispersa en:

- `docs/BIBLIA_APP_ORQUESTA.md`
- `docs/op_087_autogestion_supervisada_agentes.md`
- `docs/runbook_control_plane_agentes.md`
- tests de soporte del control plane y `autonomiapolicy`

No introduce una arquitectura nueva de produccion. Sirve como contrato de lectura rapida para documentacion, soporte y futuros tests.

## Politica canonica

El modelo objetivo de autonomia persistente en Orquesta es:

- `supervisor residente` ligero, reservado y estable por proyecto
- `control plane event-driven` como arbitro de trabajo, continuidad y recuperacion
- `workers reales` como cupo ejecutor separado de supervisor/reviewer
- `repair-helper local-first` como primera respuesta ante atasco real
- `prime only on escalation`: los modelos caros solo entran con evidencia operativa

Semantica visible asociada:

- `workersConectados` mide workers visibles conectados del cupo ejecutor real
- `workersTrabajando` mide workers visibles con trabajo real activo
- `supervisoresActivos` mide gobierno vivo y no debe mezclarse con capacidad ejecutora

## Invariantes

### 1. Supervisor residente

- Cada proyecto autonomo debe conservar un supervisor ligero, vivo y trazable.
- El supervisor no debe reciclarse como worker generico mientras siga reservado para gobernanza, supervision o review.
- `supervisor` y `reviewer` no consumen `max_workers`; ese cupo mide solo workers reales/ejecutores.
- Un tick periodico no debe reinyectar guidance de supervision si el supervisor ya sigue operativo; la supervision rica debe venir de eventos y senales reales.

Senal operativa estable:

- la foto sana de continuidad se documenta como `autonomyPending=0`

### 2. Control plane event-driven

- El control plane decide por eventos observables, no por pulsos ciegos.
- Las decisiones deben apoyarse en `runtime_orders`, `runtime_handles`, `runtime_mailbox`, `checkpoints`, `heartbeat`, `last_event_at` y evidencia de transcript.
- Reiniciar el daemon no puede equivaler a reinstruir toda la flota ni a repetir bootstrap pasivo.
- Si la flota esta sana y sin incidencia, el comportamiento correcto es `standby event-driven`: no-op, observacion y espera de señal relevante.
- La foto sana actual del loop debe converger a `ready`, `autonomyContinuing=0` y `autonomyPending=0`.

### 3. Recuperacion local-first

Ante un worker atascado, la escalera recomendada es:

1. observar el runtime y confirmar que el atasco es real
2. abrir `repair-helper` corto, local y barato
3. si no resuelve o reincide, relevar a otro worker no-prime util
4. solo despues escalar a `prime` o a un modelo caro

Reglas:

- el `repair-helper` debe ser quirurgico: reproducir, aislar causa, aplicar parche minimo o devolver diagnostico de escalado
- el `repair-helper` debe arrancar con briefing comprimido y salida minima suficiente
- la tarea origen no debe moverse por reflejo mientras exista una reparacion local plausible en curso

### 4. Prime solo por escalado

- `prime` no es el default operativo
- `prime` no se usa por preferencia del operador ni por comodidad
- `prime` requiere evidencia de atasco real, review fallida repetida, criticidad alta o riesgo tecnico demostrado
- si existe una alternativa local no-prime equivalente para continuidad normal, esa alternativa debe ganar

### 5. `work_confirmed` exige salud operativa actual

- `work_confirmed` no puede sostenerse solo con evidencia vieja, transcript superficial o mailbox absorbida si el worker ya no conserva salud operativa suficiente
- `runtime/handle` degradados, junto con heartbeat/progreso caducados, deben pesar mas que una confirmacion historica
- `tmux` reciente o actividad operativa debil pueden servir para permitir autocuracion, pero no para maquillar indefinidamente un frente roto como sano
- cuando salud operativa actual y `work_confirmed` divergen, la politica correcta es degradar a estado recuperable para disparar `repair-helper`, reinicio coordinado o relevo barato

## Regla compacta de decision

Orden canonico:

1. supervisor residente observa
2. control plane decide si hace falta actuar
3. si hay atasco, `repair-helper` local primero
4. si sigue roto, relevo no-prime
5. `prime` solo si la evidencia obliga

## Lectura operativa corta

En una frase:

> mantener un supervisor barato siempre online, dejar que el control plane reaccione a eventos reales, intentar primero una reparacion local barata y reservar `prime` para escalados justificados.

## Consecuencias para documentacion y tests

- la documentacion nueva debe hablar de `tmux + manifest/status/heartbeat + session_resume` como doctrina viva de runtime local
- los tests de soporte deben blindar preferencia por supervisor activo y sesgo anti-`prime` en continuidad normal
- los tests del control plane deben seguir tratando `repair-helper` como primera remediacion antes de handoff caro o promocion a `prime`

## Fuera de alcance

Este documento no cambia:

- la implementacion productiva actual del control plane
- el orden interno exacto de cada batch del daemon
- la semantica de compatibilidad legacy usada solo para rescate o tests
