<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Diseño técnico — Pools de capacidad

## 1. Objetivo

Este documento aterriza la decisión aprobada en `OP-050`.

El objetivo es modelar:

- pools de capacidad por licencia o runtime
- modelos habilitados dentro de cada pool
- presupuesto de sesión por sesión activa o reanudable
- reglas mínimas para handoff preventivo

No implementa todavía la lógica completa de reparto.
Define el mínimo estructural para introducirla sin romper el núcleo ni la BD existente.

## 2. Principios de diseño

1. Las migraciones deben ser aditivas e idempotentes.
2. El núcleo decide reparto y handoff; SQLite solo persiste estado.
3. Un agente pertenece a un pool, pero la asignación de trabajo sigue siendo por proyecto y tarea.
4. El presupuesto puede venir incompleto.
5. La telemetría real y la configuración manual deben convivir.

## 3. Entidades mínimas

### 3.1. `pools_capacidad`

Representa una bolsa de capacidad operativa agrupada por proveedor/runtime/plan.

Campos recomendados:

- `id`
- `slug`
- `nombre`
- `proveedor`
- `runtime`
- `plan`
- `es_de_pago`
- `capacidad_total`
- `capacidad_reservada`
- `permite_hijos`
- `permite_modelos_multiples`
- `politica_handoff`
- `fuente_telemetria`
- `metadata_json`
- `activo`
- `created_at`
- `updated_at`

Notas:

- `capacidad_disponible` no debe persistirse como fuente de verdad.
- Se calcula como `capacidad_total - capacidad_reservada - sesiones_activas_del_pool`.
- `politica_handoff` debe ser un enum pequeño o texto controlado.

### 3.2. `pool_modelos`

Modelos habilitados o preferidos dentro de un pool.

Campos recomendados:

- `id`
- `pool_id`
- `model_slug`
- `activo`
- `prioridad`
- `coste_relativo`
- `limite_conocido_json`
- `created_at`
- `updated_at`

Notas:

- Un pool puede tener uno o varios modelos.
- `coste_relativo` permite decidir entre modelos cuando el proveedor lo permita.

### 3.3. `presupuestos_sesion`

Snapshot de presupuesto operativo asociado a una sesión concreta.

Campos recomendados:

- `id`
- `sesion_id`
- `pool_id`
- `model_slug`
- `window_kind`
- `window_started_at`
- `reset_at`
- `remaining_seconds`
- `remaining_messages`
- `remaining_tokens`
- `remaining_credits`
- `source`
- `raw_snapshot_json`
- `checked_at`
- `created_at`
- `updated_at`

Notas:

- Debe admitirse null en casi todos los campos cuantitativos.
- La fuente de verdad es el último snapshot válido, no una predicción eterna.
- `source` debería distinguir al menos:
  - `cli_status`
  - `provider_api`
  - `manual`
  - `inferencia`

## 4. Relaciones

Relaciones mínimas recomendadas:

- `sesiones.pool_id -> pools_capacidad.id`
- `agentes.pool_id -> pools_capacidad.id` o tabla intermedia de membresía
- `presupuestos_sesion.sesion_id -> sesiones.id`
- `presupuestos_sesion.pool_id -> pools_capacidad.id`
- `pool_modelos.pool_id -> pools_capacidad.id`

Decisión recomendada:

- añadir `pool_id` a `sesiones`
- no añadir todavía `pool_id` a `tareas`
- para `agentes`, preferir `pool_id` nullable cuando la pertenencia sea estable

Si la pertenencia de un agente a pool puede cambiar frecuentemente, mejor una tabla `agente_pool_historial`.
Para el primer bloque, basta con `agentes.pool_id`.

## 5. Reglas operativas mínimas

### 5.1. Reserva de capacidad

- un pool no debe abrir más sesiones hijas que su capacidad efectiva
- el supervisor puede reservar slots para proyectos prioritarios
- la reserva debe ser explícita, no inferida

### 5.2. Handoff preventivo

Disparadores recomendados:

- `remaining_seconds` por debajo de umbral
- `reset_at` inminente
- snapshot de proveedor indicando límite cercano
- falta de telemetría combinada con antigüedad excesiva de la sesión

Efectos mínimos:

- no arrancar trabajo largo nuevo
- exigir `resumen_continuidad`
- exigir `external_session_id`
- proponer relevo del mismo proyecto
- preferir el mismo pool; si no hay capacidad, escalar al supervisor global

### 5.3. Degradación segura

Si no hay telemetría fiable:

- el pool sigue existiendo
- se usa política manual
- el sistema marca el presupuesto como incierto
- el supervisor evita decisiones agresivas basadas en datos incompletos

## 6. Cambios mínimos sobre esquema actual

### Fase A

- crear `pools_capacidad`
- crear `pool_modelos`
- crear `presupuestos_sesion`
- añadir `pool_id` nullable a `sesiones`

### Fase B

- añadir `pool_id` nullable a `agentes`
- sembrar pools iniciales conocidos:
  - `codex`
  - `claude`
  - `android`

### Fase C

- exponer pools y presupuestos por CLI, API web y MCP
- mostrar capacidad reservada, usada y disponible

## 7. Semilla inicial recomendada

Pools de arranque:

- `codex`
  - `proveedor=OpenAI`
  - `runtime=codex`
  - `capacidad_total=4`
  - `permite_hijos=1`

- `claude`
  - `proveedor=Anthropic`
  - `runtime=claude`
  - `capacidad_total=1`
  - `permite_hijos=1`

- `android`
  - `proveedor=Android`
  - `runtime=android`
  - `capacidad_total=1`
  - `permite_hijos=0`

## 8. Riesgos a vigilar

- duplicar lógica de reparto en SQLite y en conectores
- persistir campos derivados como si fueran verdad primaria
- acoplar demasiado el modelo a proveedores actuales
- tomar decisiones de handoff con telemetría incierta sin marcar nivel de confianza
- mezclar la capacidad global del pool con el presupuesto concreto de una sesión

## 9. Orden recomendado de implementación

1. migraciones aditivas de `pools_capacidad`, `pool_modelos`, `presupuestos_sesion`
2. extensión mínima de `sesiones` con `pool_id`
3. seed inicial de pools conocidos
4. lectura por CLI/API/MCP
5. captura manual de telemetría
6. políticas de handoff preventivo
7. reparto automático entre pools

## 10. Nota de consolidación

Existe duplicidad entre `OP-050` y `OP-051`.

La decisión aprobada debe consolidarse sobre `OP-050`.
`OP-051` debería cerrarse o marcarse como duplicada/backlog por Alberto para no bifurcar la arquitectura.
