<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# OP-087 — Autogestión supervisada de agentes y resolución autónoma de bloqueos

> Cuarentena T124 2026-05-26
> Esta OP conserva decisiones historicas sobre autogestion, `pty/process`,
> Terminator y wrappers manuales. No es fuente viva para abrir runtimes,
> sesiones, worktrees ni procesos. La autogestion vigente debe pasar por
> Director, runtime neutral, conectores opt-in, ACK/checkpoint y refs opacas. Si
> se usa Terminator, `tmux` o wrapper manual, queda como recuperacion asistida o
> adaptador legacy con decision y pruebas propias.

> Nota de vigencia 2026-04-05
> La doctrina viva de runtime local interactivo ya no es `pty/process` como primer driver. El criterio vigente está en `docs/BIBLIA_APP_ORQUESTA.md`: `tmux` + `manifest/status/heartbeat` + continuidad por `session_resume`. Las referencias de esta OP a `pty/process` se conservan como contexto histórico y no deben abrir tareas nuevas PTY-first.

## Objetivo

Decidir cómo debe evolucionar Orquesta para que los agentes no dependan de supervisión humana constante y puedan resolver problemas operativos por sí mismos antes de escalar.

Esta decisión afecta a:

- sesiones manuales y automáticas
- control activo de agentes vivos
- continuidad entre sesiones y handoff
- observabilidad pasiva y supervisión automática
- reparto de trabajo entre agentes del mismo proyecto

## Problema real

Hoy Orquesta ya sabe:

- registrar agentes, tareas, propuestas y sesiones
- persistir continuidad básica
- preparar arranques por conector
- observar parte del estado runtime

Pero todavía no gobierna bien la **autogestión** del agente cuando aparece un bloqueo real. En la práctica falta definir:

- cuándo debe reintentar un agente por sí mismo
- cuándo debe consultar a otro agente antes que al humano
- cuándo debe hacer checkpoint
- cuándo debe hacer handoff
- cuándo debe pausar o reiniciarse
- qué datos mínimos debe adjuntar al escalar

Sin estas reglas, el humano sigue actuando como coordinador manual, memoria viva del sistema y resolvedor de incidencias de primer nivel.

## Principio rector propuesto

Orquesta debe adoptar un modelo de:

**autogestión supervisada**

Esto significa:

- máxima autonomía operativa normal
- guardrails duros para acciones peligrosas
- escalación tardía al humano
- trazabilidad completa de intentos, consultas y relevos

## Flujo objetivo de resolución autónoma

Ante un bloqueo, el orden esperado debe ser:

1. autodiagnóstico local
2. reintento o remediación segura
3. consulta a otro agente si sigue bloqueado
4. checkpoint de continuidad
5. handoff, relevo o reinicio si aplica
6. escalación humana con evidencia si todo lo anterior falla

## Preguntas de voto

### A. ¿Qué principio debe gobernar la operación de los agentes?

Opciones:

- `A1`
  Autonomía libre.
  El agente decide casi todo y solo se registra el resultado final.

- `A2`
  Control central estricto.
  Casi todas las acciones relevantes requieren orden explícita desde Orquesta.

- `A3`
  Autogestión supervisada.
  El agente resuelve solo lo normal, consulta a pares cuando se atasca y solo escala al humano al final.

Recomendación inicial: `A3`.

### B. ¿Qué primitive debe ser obligatoria antes de escalar a humano?

Opciones:

- `B1`
  Ninguna. Basta con marcar el bloqueo.

- `B2`
  Resumen textual de continuidad.

- `B3`
  Checkpoint estructurado con contexto, evidencia, último intento y siguiente acción propuesta.

Recomendación inicial: `B3`.

### C. ¿Cómo debe hacerse la consulta entre agentes?

Opciones:

- `C1`
  Manual y externa a Orquesta.

- `C2`
  Mediante comentarios en tarea o propuesta.

- `C3`
  Mediante mailbox persistente entre agentes y sesiones, con tipos como `consulta`, `respuesta`, `handoff`, `nudge` y `escalacion`.

Recomendación inicial: `C3`.

### D. ¿Dónde debe vivir la lógica de autogestión?

Opciones:

- `D1`
  Dentro de cada wrapper de terminal.

- `D2`
  Repartida entre CLI, web y scripts locales.

- `D3`
  Autogestión supervisada con Watchdog activo.
  En un plano de control único de Orquesta con `runtime_orders`, `runtime_handles`, `runtime_mailbox` y un supervisor de salud capaz de detectar cuelgues (ausencia de heartbeat) y bucles sin fin (repitición de patrones o herramientas).

Recomendación inicial: `D3`.

### E. ¿Qué transporte debe ser el modelo central?

Opciones:

- `E1`
  `pty` como centro del sistema.

- `E2`
  API remota como centro del sistema.

- `E3`
  Plano de control abstracto con adaptadores; `pty/process` como primer driver, no como arquitectura central.

Recomendación inicial: `E3`.

### F. ¿Qué política de escalación debería usarse por defecto?

Opciones:

- `F1`
  Escalación inmediata en el primer bloqueo.

- `F2`
  Un reintento local y luego escalación humana.

- `F3`
  Dos reintentos locales seguros, una consulta a otro agente, checkpoint obligatorio y solo entonces escalación humana.

Recomendación inicial: `F3`.

## Datos mínimos a introducir

### `runtime_handles`

- identidad del handle vivo
- tipo de transporte
- referencia al proceso, PTY, sesión MCP o API remota
- capacidades
- lease y heartbeat (latido cardíaco de salud)
- métricas de consumo de tokens estimadas/reales

### `runtime_orders`

- orden persistente
- estado de ejecución
- payload y resultado
- tiempos de creación, inicio y fin

Tipos mínimos:

- `start`
- `send_instruction`
- `pause`
- `resume`
- `checkpoint`
- `handoff_prepare`
- `handoff_commit`
- `restart`
- `stop`
- `sync_status`

### `runtime_mailbox`

- `consulta`
- `respuesta`
- `nudge`
- `handoff`
- `escalacion`
- `mensaje`

### `runtime_checkpoints`

- agente
- sesión origen
- runtime origen
- branch
- cwd
- resumen estructurado
- evidencia
- siguiente acción propuesta
- estrategia de reanudación

## Estados operativos recomendados

- `disponible`
- `pensando` (latido activo)
- `ejecutando_herramienta`
- `esperando_io`
- `atascado` (unresponsive o bloqueado por falta de contexto)
- `reintentando`
- `consultando_par`
- `bucle_detectado` (circuital break por repetición de patrón)
- `handoff`
- `pausado`
- `escalado_humano`
- `cerrado`
- `zombie` (ausencia de heartbeat prolongada)

## Regla recomendada para nuestro caso

La opción más robusta para Orquesta parece ser:

1. adoptar `autogestión supervisada` como política base
2. centralizar la lógica en Orquesta, no en Terminator ni en wrappers
3. modelar `runtime_orders`, `runtime_handles`, `runtime_mailbox` y `runtime_checkpoints`
4. mantener `pty/process` como primer adaptador real
5. exigir checkpoint antes de handoff o escalación humana
6. priorizar consulta entre agentes antes que intervención del humano

## Dependencias con decisiones previas

Esta OP complementa y aterriza:

- `OP-079`
  estrategia híbrida de control de agentes
- `OP-082`
  observabilidad pasiva de runtimes
- `OP-083`
  clientes finos sobre Orquesta
- `OP-084`
  daemon persistente y single-writer
- `OP-086`
  refactor de integración Terminator y daemon

## Impacto esperado si se aprueba

- menos dependencia operativa de Alberto
- menos bloqueos sin contexto
- handoff y recuperación más fiables
- reparto de trabajo entre agentes más natural
- menor acoplamiento a `pty` o al emulador de terminal
- base clara para control activo, web y futura app de escritorio

## Recomendación inicial de Codex

- `A3`
- `B3`
- `C3`
- `D3`
- `E3`
- `F3`
