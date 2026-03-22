<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Diseño mínimo — Pools de capacidad y presupuestos de sesión

## Objetivo

Definir el siguiente bloque mínimo para que `Codex2` y `Codex3` implementen la capa de:

- pools de capacidad por proveedor/runtime/plan
- modelos habilitados por pool
- presupuestos de sesión

Sin tocar todavía integraciones externas complejas.

## Principio

No modelar “licencias” rígidas.
Modelar `pools` de capacidad ampliables y reducibles.

## 1. Tablas mínimas

### `pools_capacidad`

```sql
CREATE TABLE IF NOT EXISTS pools_capacidad (
    id                   INTEGER PRIMARY KEY AUTOINCREMENT,
    slug                 TEXT NOT NULL UNIQUE,
    proveedor            TEXT NOT NULL,
    runtime              TEXT NOT NULL,
    plan                 TEXT NOT NULL DEFAULT '',
    es_de_pago           INTEGER NOT NULL DEFAULT 0,
    capacidad_total      INTEGER NOT NULL DEFAULT 1,
    capacidad_reservada  INTEGER NOT NULL DEFAULT 0,
    permite_hijos        INTEGER NOT NULL DEFAULT 1,
    permite_modelos_multi INTEGER NOT NULL DEFAULT 1,
    permite_sobrecoste   INTEGER NOT NULL DEFAULT 0,
    politica_handoff     TEXT NOT NULL DEFAULT 'preventivo',
    fuente_telemetria    TEXT NOT NULL DEFAULT 'manual',
    metadata_json        TEXT NOT NULL DEFAULT '{}',
    activo               INTEGER NOT NULL DEFAULT 1,
    created_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### `pool_modelos`

```sql
CREATE TABLE IF NOT EXISTS pool_modelos (
    id                   INTEGER PRIMARY KEY AUTOINCREMENT,
    pool_id              INTEGER NOT NULL REFERENCES pools_capacidad(id) ON DELETE CASCADE,
    model_slug           TEXT NOT NULL,
    activo               INTEGER NOT NULL DEFAULT 1,
    prioridad            INTEGER NOT NULL DEFAULT 100,
    coste_relativo       REAL NOT NULL DEFAULT 1.0,
    limite_conocido_json TEXT NOT NULL DEFAULT '{}',
    created_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(pool_id, model_slug)
);
```

### `presupuestos_sesion`

```sql
CREATE TABLE IF NOT EXISTS presupuestos_sesion (
    id                     INTEGER PRIMARY KEY AUTOINCREMENT,
    sesion_id              INTEGER NOT NULL REFERENCES sesiones(id) ON DELETE CASCADE,
    pool_id                INTEGER REFERENCES pools_capacidad(id) ON DELETE SET NULL,
    model_slug             TEXT NOT NULL DEFAULT '',
    window_kind            TEXT NOT NULL DEFAULT 'unknown',
    window_started_at      DATETIME,
    reset_at               DATETIME,
    remaining_seconds      INTEGER,
    remaining_messages     INTEGER,
    remaining_tokens       INTEGER,
    remaining_credits      REAL,
    budget_source          TEXT NOT NULL DEFAULT 'manual',
    raw_snapshot_json      TEXT NOT NULL DEFAULT '{}',
    checked_at             DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at             DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

## 2. Tipos recomendados en Go

### Pool

```go
type PoolCapacidad struct {
    ID                  int64
    Slug                string
    Proveedor           string
    Runtime             string
    Plan                string
    EsDePago            bool
    CapacidadTotal      int
    CapacidadReservada  int
    PermiteHijos        bool
    PermiteModelosMulti bool
    PermiteSobrecoste   bool
    PoliticaHandoff     string
    FuenteTelemetria    string
    MetadataJSON        string
    Activo              bool
}
```

### Modelo del pool

```go
type PoolModelo struct {
    ID               int64
    PoolID           int64
    ModelSlug        string
    Activo           bool
    Prioridad        int
    CosteRelativo    float64
    LimiteConocidoJSON string
}
```

### Presupuesto de sesión

```go
type PresupuestoSesion struct {
    ID                int64
    SesionID          int64
    PoolID            *int64
    ModelSlug         string
    WindowKind        string
    WindowStartedAt   *time.Time
    ResetAt           *time.Time
    RemainingSeconds  *int64
    RemainingMessages *int64
    RemainingTokens   *int64
    RemainingCredits  *float64
    BudgetSource      string
    RawSnapshotJSON   string
    CheckedAt         time.Time
}
```

## 3. Configuración inicial sugerida

```text
pool_handoff_threshold_seconds=1800
pool_handoff_threshold_ratio=0.10
pool_default_budget_source=manual
model_policy_default_profile=implementacion
model_policy_default_reasoning=high
```

## 4. Casos de uso mínimos

### Codex2

- alta y edición de pools
- alta y edición de modelos por pool
- listado de capacidad disponible
- API para consultar pools y modelos

### Codex3

- registrar presupuesto de una sesión
- consultar último presupuesto de una sesión
- calcular si una sesión debe entrar en handoff preventivo
- API para consultar presupuestos y alertas

## 5. No hacer todavía

- scraping complejo de proveedores
- integración real con billing externo
- estimadores avanzados de coste
- automatización completa de handoff entre pools

## 6. Criterio de éxito del bloque

- se pueden registrar pools configurables
- se pueden registrar modelos por pool
- una sesión puede guardar su presupuesto observado
- la app puede saber si conviene relevo preventivo
