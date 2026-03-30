<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# OP-089 — Relevo de Agentes en Caliente por Agotamiento de Tokens (Handoff)

## Objetivo
Definir el mecanismo técnico mediante el cual Orquesta expulsa a un agente de una sesión viva (debido a cuotas de API, tokens agotados o bloqueos persistentes) y reanuda el trabajo exactamente en el mismo punto de estado con un agente distinto, sin intervención humana.

## Problema Rectificado
Actualmente, si un agente como `claude` agota su presupuesto o cuota horaria en medio de una Tarea Dificil, la sesión "muere" o se queda colgada (`zombie`). Si abrimos un `codex` nuevo, este arranca desde cero (contexto en blanco) y tiene que adivinar qué ficheros tocó el agente anterior guiándose por los diffs sueltos en el sistema de archivos.

## Arquitectura de Relevo (El flujo de "Eviction & Resume")

Para resolver esto usando los elementos ya aprobados en **OP-087** y **OP-088**, Orquesta ejecutará automáticamente un flujo de 4 pasos:

### 1. Detección (Fallo o Prevención)
- **Modo reactivo:** El `RuntimeAdapter` intercepta en los logs o a nivel de API un error de proveedor (ej: `HTTP 429 Too Many Requests` o fallos de saldo).
- **Modo preventivo:** El `Vigilante` de Orquesta examina el `heartbeat` y los metadatos de cuota, dándose cuenta de que el agente le quedan <5% de tokens.

### 2. Eviction (Desalojo) y Checkpoint
- Orquesta cambia inmediatamente el estado de la tarea de `en_progreso` a `handoff_en_curso`.
- Se envía una señal `SIGTERM` o equivalente al `RuntimeHandle` del Agente A (se cierra el PTY o la conexión MCP).
- **El paso crítico:** El daemon de Orquesta captura el estado del `worktree` actual (ficheros modificados sin commitear, rama activa, y la última instrucción que recibió) y escribe un registro completo en `runtime_checkpoints` a través del adaptador de persistencia activo de Orquesta.

### 3. Reasignación Automática
- Orquesta consulta su base de datos de `agentes` buscando otro agente que esté libre y tenga capacidad (tokens disponibles o distinta licencia).
- Se crea un nuevo `RuntimeHandle` asignado al Agente B.

### 4. Reanudación con "Inyección de Contexto"
- Orquesta arranca la sesión del Agente B.
- *Inyección:* En el mensaje "System Prompt" inicial, Orquesta no solo le manda las reglas (OP-088), sino que le adjunta el bloque de `Checkpoint` que dice:
  > *"Atención Agente B: Estás relevando al Agente A por agotamiento de tokens. Tu directorio de trabajo ya tiene cambios a medias. El Agente A estaba haciendo X. El último error que vio fue Y. Aquí tienes el `git diff` actual. Continúa el trabajo donde lo dejó y termina la tarea."*

## Ventajas del modelo
- **Resiliencia Pura:** Un trabajo masivo de refactorización (que consumiría la cuota diaria de un solo LLM) ahora se puede repartir entre múltiples agentes de forma encadenada ("Carrera de relevos").
- **Coste Cero de Contexto Humano:** Alberto no tiene que "explicarle" al Agente B por dónde iba el Agente A. La persistencia gobernada por Orquesta serializa y transfiere la mente de la sesión automáticamente.

## Votación y Tareas
- Esta propuesta requiere crear el motor lógico de `Handoff Manager` dentro de `core/`.
- Posición Antigravity: **ACUERDO**. Esta es una de las "Killer Features" que diferencian a un Orquestador maduro de un simple script de consola.
