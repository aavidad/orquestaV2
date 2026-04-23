# Propuesta OMX para Orquesta

## Objetivo

Copiar de `oh-my-codex` solo los contratos operativos que ya sabemos que funcionan bien para ejecución de agentes, sin sustituir el control plane actual ni competir con la línea `ADK contracts`.

Esto no propone adoptar `oh-my-codex` como framework.
Propone reforzar la capa de ejecución con:

- hooks de ciclo de vida
- equipos por rol
- `worktree + tmux + resume`
- perfiles de ejecución

## Qué copiar

### 1. Hooks de ciclo de vida

Puntos canónicos:

- `before_tool`
- `after_tool`
- `on_worker_fail`
- `on_handoff`
- `on_stop`
- `on_recovery`

Regla:

- cada hook debe ser explícito, pequeño y observable
- no debe mutar estado durable por canales laterales
- si cambia estado contractual, debe pasar por el control plane
- Orquesta ya tiene una base parcial en `db/hooks.go`; la tarea correcta es extender y conectar esa base, no crear un segundo sistema de hooks

### 2. Equipos por rol

Roles canónicos iniciales:

- `supervisor`
- `executor`
- `reviewer`
- `fixer`

Regla:

- el supervisor gobierna
- el executor implementa
- el reviewer valida
- el fixer corrige slices rechazadas

Eso debe convivir con el supervisor residente ya existente y no mezclar gobernanza con ejecución.

### 3. `worktree + tmux + resume`

Contrato operativo:

- un frente de trabajo usa una `worktree` propia
- el runtime canónico interactivo vive en `tmux`
- la continuidad se reanuda con `resume` seguro
- el handoff se hace sobre frente acotado y con evidencia, no sobre contexto difuso
- Orquesta ya recorre parte de este camino; el objetivo no es duplicarlo, sino convertirlo en contrato canónico y uniforme

### 4. Perfiles de ejecución

Perfiles mínimos:

- `barato`
- `paralelo`
- `persistente`
- `qa-heavy`

Regla:

- el perfil es política operativa, no prompt decorativo
- debe decidir coste, rigor, paralelismo y tipo de verificación

## Qué no copiar

- otra CLI completa
- otra capa de sesiones paralela
- otra verdad de estado
- decisiones de producto acopladas al runtime de otro proyecto

## Complementariedad con ADK

La línea `oh-my-codex style runtime` y la línea `ADK contracts` no son alternativas.

Se reparten el problema:

- `ADK contracts` refuerza:
  - trazabilidad
  - reversibilidad
  - estado durable
  - artifacts
- `OMX style runtime` refuerza:
  - workflow de agente
  - disciplina de ejecución
  - roles
  - reanudación

Regla doctrinal:

- Orquesta puede implementar ambas
- siempre que sigan siendo complementarias
- y siempre que el control plane canónico siga siendo uno solo
- si una mejora OMX obliga a crear otra verdad de sesión, de runtime o de handoff, debe rechazarse

## Roadmap mínimo

### Fase 1

- hooks de ciclo de vida canónicos

### Fase 2

- pipeline de equipo por rol

### Fase 3

- `worktree + tmux + resume` como contrato canónico, no como patrón informal

### Fase 4

- perfiles de ejecución integrados con coste, review y autonomía persistente

## Criterio de aceptación

Esta línea merece la pena solo si consigue:

- menos caos operativo de workers
- slices más acotadas
- handoffs más seguros
- resume más fiable
- mejor throughput sin subir el coste por defecto
