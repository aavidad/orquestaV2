<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# OP-090 — Protocolo de Sondeo (Probing) y Nudge de Agentes

## Objetivo
Implementar un mecanismo de "tanteo" para agentes que parecen estar bloqueados pero que técnicamente siguen en ejecución (estado `pensando`). Permitir que Orquesta les consulte su estado en caliente sin interrumpir el flujo principal de la tarea.

## El Problema del "Pensamiento Profundo"
Con los nuevos modelos de razonamiento (O1, Claude 3.7 con Thinking, etc.), un agente puede estar en silencio durante 2 o 3 minutos mientras "piensa". 
- Si Orquesta lo mata por falta de heartbeat, perdemos el trabajo.
- Si Orquesta no hace nada, el usuario (Alberto) se desespera al no saber si la IA ha muerto o sigue trabajando.

## Protocolo Propuesto: El "Nudge" (Empujoncito)

### 1. Umbral de Silencio (T_sondeo)
Orquesta definirá un tiempo de silencio razonable (ej. 60s). Si el agente no ha emitido ningún token ni actualización de estado en ese tiempo, Orquesta activa el sondeo.

### 2. Inyección de Consulta (Out-of-band)
Orquesta enviará un mensaje de tipo `nudge` a través del `RuntimeAdapter`:
- **En PTY:** Si el agente lo soporta, se inyecta una secuencia de control (ej: `CTRL+C` capturado como estado, o un mensaje de sistema inyectado en la entrada estándar si el agente es un wrapper).
- **En MCP:** Orquesta actualiza un recurso de sistema `orquesta://config/status_request` que el agente debe estar monitorizando.

### 3. Respuesta de Diagnóstico del Agente
El agente, al detectar el `nudge`, debe estar programado para responder con un "Informe de Latido" breve:
> *"Sigo vivo. Estoy analizando el fichero `db/schema.go` para encontrar la causa del bug X. Llevo el 60% del análisis."*

### 4. Acciones tras el Sondeo
- **Éxito:** Si el agente responde, Orquesta reinicia el cronómetro del Watchdog y actualiza la UI para informar a Alberto.
- **Fallo:** Si tras 2 nudges sigue sin haber respuesta, Orquesta lo declara definitivamente `atascado` y procede al **Handoff** (OP-089).

## Preguntas de Voto

### A. ¿Debe ser el Nudge intrusivo?
Opciones:
- `A1`: No intrusivo. Solo mirar métricas de CPU/Memoria sin mandar mensajes.
- `A2`: Intrusivo. Mandar un mensaje real al flujo del agente para forzar respuesta.

Recomendación Antigravity: `A2` (con agentes modernos que soportan interrupciones o flujos asíncronos).

## Impacto
- Mayor confianza de Alberto en el sistema (menos "incertidumbre").
- Menos reinicios innecesarios por falsos positivos de cuelgue.
- Trazabilidad del razonamiento en tiempo real.

## Recomendación Antigravity
Posición: **ACUERDO**. Es la pieza que permite distinguir el "Silencio de Trabajo" del "Silencio de Muerte".
