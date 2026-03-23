<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Coordinación rápida — revisión de OP-086 y OP-087

Fecha: 2026-03-23

## Objetivo

Dejar una guía breve para que `Codex2`, `Codex3` y `antigravity` voten con foco y para que Alberto pueda cerrar propuestas abiertas que ya están técnicamente resueltas.

## Propuestas que requieren voto ahora

### OP-086

Título:

`Refactorizar integracion Terminator y daemon para alinearla con arquitectura hexagonal`

Pregunta práctica:

- si mantenemos la funcionalidad ya conseguida
- pero obligamos a encapsularla detrás de casos de uso, API/CLI y adaptadores finos
- sin seguir metiendo lógica de negocio en scripts, `cmd/` o accesos directos de persistencia

Criterio recomendado de voto:

- `acuerdo` si se acepta preservar capacidad operativa pero rehacer la integración bajo el control plane y la arquitectura hexagonal
- `desacuerdo` si se considera que Terminator y wrappers deben seguir siendo el eje principal del sistema

### OP-087

Título:

`Autogestión supervisada de agentes y resolución autónoma de bloqueos`

Pregunta práctica:

- si la autonomía debe vivir en el plano de control de Orquesta
- con `runtime_handles`, `runtime_orders`, `runtime_mailbox` y `runtime_checkpoints`
- dejando `pty/process` como driver y no como arquitectura central

Criterio recomendado de voto:

- `acuerdo` si se quiere que los agentes resuelvan primero por sí mismos, consulten a pares y escalen tarde
- `desacuerdo` si se prefiere control humano temprano o lógica repartida en wrappers locales

## Resumen sugerido para `Codex2`

- evaluar `OP-086` desde la arquitectura de daemon, web y clientes finos
- evaluar `OP-087` desde la gobernanza de runtime vivo, handoff y control plane
- comprobar que ninguna propuesta rompe `OP-083`, `OP-084` ni la línea de desacople de persistencia

## Resumen sugerido para `Codex3`

- evaluar `OP-086` desde contratos, casos de uso y separación entre núcleo y adaptadores
- evaluar `OP-087` desde checkpoint, continuidad, handoff y supervisión automática
- confirmar si el orden incremental propuesto es técnicamente viable sin romper sesiones actuales

## Resumen sugerido para `antigravity`

- revisar `OP-086` y `OP-087` desde claridad operativa y documentación de uso real
- verificar si el lenguaje de ambas propuestas es suficientemente concreto para documentación futura y manuales
- confirmar si la política resultante reduce dependencia de intervención manual de Alberto

## Propuestas abiertas que ya requieren cierre humano

## Preparaciones seguras ya hechas

### OP-094

Estado técnico:

- ya existe preparación aislada del contrato A2UI en `internal/a2ui`
- ya existe envelope versionado para `runtime_mailbox`
- no está activado todavía en dashboard ni en flujo operativo

Lectura operativa:

- esta OP sigue sin estar lista para uso real
- pero ya no parte de cero: el contrato y la validación estricta están preparados para una integración futura más segura

Referencia:

- ver `docs/op_094_ui_declarativa_agentes.md`

### OP-085

Estado real:

- `1` acuerdo
- `3` desacuerdos
- `0` pendientes

Lectura operativa:

- hay rechazo técnico claro a hacer de Docker/Compose la base obligatoria del sistema
- puede mantenerse como opción de empaquetado futura, pero no como decisión arquitectónica principal

Recomendación:

- cierre manual por Alberto como `rechazada` o reformulación posterior más acotada

### OP-039

Estado real:

- `4` desacuerdos
- `0` pendientes

Lectura operativa:

- se considera duplicada de `OP-036`

Recomendación:

- cierre manual por Alberto como `rechazada`

### OP-037

Estado real:

- `4` desacuerdos
- `0` pendientes

Lectura operativa:

- se considera duplicada de `OP-036`

Recomendación:

- cierre manual por Alberto como `rechazada`
