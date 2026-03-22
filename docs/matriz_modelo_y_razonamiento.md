<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Matriz de decisión — Modelo y razonamiento

## Objetivo

Que Orquesta decida automáticamente:

- qué pool usar
- qué modelo usar
- qué nivel de razonamiento usar
- cuándo merece la pena un modelo caro o un razonamiento alto

## Principio

No elegir modelo solo por proveedor disponible.
Elegirlo por:

- tipo de tarea
- riesgo
- coste
- presupuesto restante
- fase del proyecto

## Campos mínimos a decidir por tarea

- `pool_slug`
- `model_slug`
- `reasoning_effort`
- `task_profile`
- `max_session_budget_ratio`
- `handoff_threshold_ratio`

## Perfiles iniciales recomendados

### `orquestacion`

- uso: arquitectura, coordinación, decisiones estructurales
- modelo: frontier del pool
- razonamiento: `xhigh`
- coste permitido: alto

### `analisis`

- uso: investigación, lectura grande, comparación de opciones
- modelo: contexto amplio
- razonamiento: `high` o `xhigh`
- coste permitido: medio/alto

### `implementacion`

- uso: features normales, refactor local, cambios de código habituales
- modelo: principal del pool
- razonamiento: `high`
- coste permitido: medio

### `script`

- uso: shell, utilidades, glue code, tareas pequeñas
- modelo: económico del pool
- razonamiento: `medium`
- coste permitido: bajo

### `revision`

- uso: revisión, validación, contraste, scoring
- modelo: revisor del pool
- razonamiento: `high`
- coste permitido: medio

### `handoff`

- uso: cierre de sesión, resumen, relevo
- modelo: económico del pool
- razonamiento: `medium`
- coste permitido: bajo

## Reglas recomendadas

1. Si la tarea es de `orquestacion`, usar el mejor modelo disponible del pool con `xhigh`.
2. Si la tarea es de `script`, evitar `xhigh` salvo excepción explícita.
3. Si el presupuesto restante del pool baja de umbral, degradar primero el razonamiento antes de cambiar de pool.
4. Si el presupuesto restante no permite terminar bien, forzar `handoff`.
5. Las revisiones importantes deben evitar el mismo perfil exacto que el autor cuando sea posible.

## Ejemplos

### Orquesta núcleo

- tarea: diseñar pools de capacidad
- perfil: `orquestacion`
- modelo: frontier
- razonamiento: `xhigh`

### Script de Terminator

- tarea: ajustar wrapper bash
- perfil: `script`
- modelo: económico
- razonamiento: `medium`

### MCP server

- tarea: diseñar resources/prompts/tools
- perfil: `analisis` o `orquestacion`
- modelo: frontier
- razonamiento: `high` o `xhigh`

### Refactor local

- tarea: simplificar un adaptador SQLite
- perfil: `implementacion`
- modelo: principal
- razonamiento: `high`

## Diseño mínimo para implementar

### Tabla sugerida `politicas_modelo`

```sql
CREATE TABLE IF NOT EXISTS politicas_modelo (
    id                    INTEGER PRIMARY KEY AUTOINCREMENT,
    scope_tipo            TEXT NOT NULL,
    scope_ref             TEXT NOT NULL DEFAULT '',
    perfil_tarea          TEXT NOT NULL,
    pool_slug             TEXT NOT NULL DEFAULT '',
    model_slug            TEXT NOT NULL DEFAULT '',
    reasoning_effort      TEXT NOT NULL DEFAULT '',
    prioridad             INTEGER NOT NULL DEFAULT 100,
    activa                INTEGER NOT NULL DEFAULT 1,
    metadata_json         TEXT NOT NULL DEFAULT '{}',
    created_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### Resolución recomendada

1. política por tarea concreta
2. política por fase
3. política por proyecto
4. política por perfil
5. default global

## Siguiente paso para implementación

- `Codex2`: resolver pool/modelo por política
- `Codex3`: cruzar política con presupuesto restante y handoff

## Configuración inicial sugerida

```text
model_policy_default_profile=implementacion
model_policy_default_reasoning=high
pool_handoff_threshold_seconds=1800
pool_handoff_threshold_ratio=0.10
```
