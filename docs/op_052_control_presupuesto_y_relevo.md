<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# OP-052 — Control de presupuesto de sesión y relevo preventivo

## Objetivo

Decidir cómo debe medir Orquesta el presupuesto restante de un agente y cuándo debe forzar checkpoint, handoff o relevo para no dejar trabajo a medias.

Esta decisión afecta a:

- sesiones manuales y automáticas
- pools de capacidad por proveedor
- continuidad entre agentes del mismo proyecto
- prevención de cortes por fin de ventana, fin de cuota semanal o degradación del runtime

## Problema real

Los runtimes no exponen el presupuesto igual:

- unos muestran tiempo restante aproximado
- otros muestran cuota semanal
- otros muestran mensajes, créditos o tokens
- otros solo permiten inferencia operativa o configuración manual

Además, la misma cifra no significa lo mismo según la ventana:

- `6%` de una ventana de `5h` puede implicar relevo inmediato
- `6%` de una ventana semanal puede seguir permitiendo trabajo corto

Por tanto, Orquesta no debe usar una sola métrica ciega.

## Preguntas de voto

### A. ¿Qué métrica debe mandar en Orquesta?

Opciones:

- `A1`
  Porcentaje restante normalizado.
  Simple, comparable entre proveedores, pero puede ocultar la duración real de la ventana.

- `A2`
  Tiempo absoluto restante.
  Claro en ventanas cortas, pero no sirve bien cuando el runtime solo expone cuota semanal o créditos.

- `A3`
  Política compuesta:
  porcentaje restante + tiempo absoluto + tipo de ventana + criticidad de tarea.

Recomendación inicial: `A3`.

### B. ¿Cómo debe modelarse la ventana?

Opciones:

- `B1`
  Solo `reset_at`.

- `B2`
  Solo `window_kind`.

- `B3`
  `window_kind` + `reset_at` + `window_seconds` cuando se conozca.

Recomendación inicial: `B3`.

### C. ¿Qué umbral debe disparar handoff preventivo?

Opciones:

- `C1`
  Umbral por porcentaje.
  Ejemplo: `<=10%`.

- `C2`
  Umbral por tiempo.
  Ejemplo: `<=30 min`.

- `C3`
  Umbral combinado:
  `<=10%` o `<=30 min`, lo que ocurra antes.

Recomendación inicial: `C3`.

### D. ¿Qué debe hacer Orquesta al entrar en zona amarilla?

Opciones:

- `D1`
  Solo avisar.

- `D2`
  Avisar y bloquear tareas nuevas largas.

- `D3`
  Avisar, bloquear tareas nuevas largas y exigir checkpoint próximo.

Recomendación inicial: `D3`.

### E. ¿Qué debe hacer Orquesta al entrar en zona roja?

Opciones:

- `E1`
  Solo sugerir relevo.

- `E2`
  Forzar checkpoint y handoff si hay otro agente disponible.

- `E3`
  Forzar checkpoint, bloquear cambios largos y pausar si no hay relevo viable.

Recomendación inicial: `E3`.

### F. ¿Qué hacer cuando la telemetría es débil o indirecta?

Opciones:

- `F1`
  No automatizar nada.

- `F2`
  Usar solo configuración manual por proveedor y plan.

- `F3`
  Mezclar configuración manual, captura desde CLI/API e inferencia con nivel de confianza.

Recomendación inicial: `F3`.

## Estado recomendado

Orquesta debería calcular un estado operativo por sesión:

- `normal`
  El agente puede seguir.

- `amarillo`
  Debe evitar tareas largas y preparar resumen de continuidad.

- `rojo`
  Debe forzar checkpoint, guardar contexto y transferir o cerrar con orden.

- `desconocido`
  Sin telemetría fiable; usar política conservadora.

## Datos mínimos a guardar

- `window_kind`
- `window_seconds`
- `reset_at`
- `remaining_ratio`
- `remaining_seconds`
- `remaining_tokens`
- `remaining_messages`
- `remaining_credits`
- `budget_source`
- `telemetry_confidence`
- `observed_at`
- `raw_snapshot_json`

## Regla recomendada para nuestro caso

La opción más robusta para Orquesta parece ser:

1. Medir primero `remaining_ratio`.
2. Si existe, cruzarlo con `remaining_seconds`.
3. Ajustar por `window_kind`:
   - `session_5h`
   - `daily`
   - `weekly`
   - `credits`
   - `unknown`
4. Aplicar un multiplicador por criticidad de la tarea:
   - `script`
   - `implementacion`
   - `orquestacion`
   - `revision`
   - `handoff`
5. Si cae en `amarillo`, no iniciar cambios largos.
6. Si cae en `rojo`, checkpoint y relevo preventivo.

## Casos que deben soportarse

- `Codex` con ventanas cortas y reinicio periódico.
- `Codex` gratuito o plan equivalente con ventana semanal o muy distinta.
- `Claude` con límites por uso/plan y reinicios periódicos.
- proveedores futuros como `Grok` con otra unidad de cuota.
- runtimes sin API oficial, solo con salida textual o interacción manual.

## Recomendación inicial de Codex1

- `A3`
- `B3`
- `C3`
- `D3`
- `E3`
- `F3`

## Impacto esperado si se aprueba

- no depender de una única métrica frágil
- hacer handoff antes de que el agente se corte
- permitir continuidad entre agentes del mismo proyecto
- soportar proveedores y planes heterogéneos sin reescribir el núcleo
