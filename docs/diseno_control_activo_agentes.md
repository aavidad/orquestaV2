<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Diseño mínimo — Control activo de agentes vivos

## Problema

Orquesta ya puede:

- registrar agentes
- asignar proyectos
- crear tareas
- preparar bundles de arranque
- guardar sesiones

Pero todavía no puede:

- enviar una instrucción a un agente ya abierto
- pausarlo
- reanudarlo
- hacer handoff real desde la app

## Objetivo

Que la app pueda gobernar agentes vivos sin que el usuario haga de puente manual entre Orquesta y la consola.

## Capacidades mínimas

### 1. Registro de runtime vivo

Tabla sugerida:

```sql
CREATE TABLE IF NOT EXISTS runtime_handles (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    agente           TEXT NOT NULL REFERENCES agentes(nombre),
    sesion_id        INTEGER REFERENCES sesiones(id) ON DELETE SET NULL,
    proyecto_id      INTEGER REFERENCES proyectos(id) ON DELETE SET NULL,
    transporte       TEXT NOT NULL,
    handle_kind      TEXT NOT NULL,
    handle_ref       TEXT NOT NULL,
    estado           TEXT NOT NULL DEFAULT 'activo',
    metadata_json    TEXT NOT NULL DEFAULT '{}',
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

Ejemplos de `handle_kind`:

- `pty`
- `process`
- `mcp_session`
- `remote_api`

### 2. Cola de órdenes

Tabla sugerida:

```sql
CREATE TABLE IF NOT EXISTS runtime_orders (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    agente           TEXT NOT NULL REFERENCES agentes(nombre),
    proyecto_id      INTEGER REFERENCES proyectos(id) ON DELETE SET NULL,
    tipo             TEXT NOT NULL,
    payload_json     TEXT NOT NULL DEFAULT '{}',
    estado           TEXT NOT NULL DEFAULT 'pendiente',
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at       DATETIME,
    finished_at      DATETIME
);
```

Tipos mínimos:

- `enviar_instruccion`
- `pausar`
- `continuar`
- `handoff`

### 3. Comandos/API mínimos

- `orquesta agente arrancar <agente>`
- `orquesta agente enviar <agente> --mensaje "..."`
- `orquesta agente pausar <agente>`
- `orquesta agente continuar <agente>`
- `orquesta agente handoff <agente-origen> <agente-destino>`

## Flujo recomendado

### Envío de instrucción

1. La web crea una `runtime_order`.
2. El supervisor de runtime recoge la orden.
3. Resuelve el `runtime_handle` activo del agente.
4. Inyecta el mensaje por el transporte adecuado.
5. Marca la orden como completada o fallida.

### Handoff

1. Se detecta que la sesión actual debe relevarse.
2. Orquesta fuerza checkpoint.
3. Guarda continuidad y `external_session_id`.
4. Crea orden de `handoff`.
5. Arranca o reanuda el agente destino.
6. Le envía la continuidad.

## Transportes previstos

### `pty/process`

Para consolas locales como Terminator o procesos lanzados por Orquesta.

### `mcp_session`

Para runtimes conectados por MCP.

### `remote_api`

Para agentes que expongan API de control remoto.

## Primera implementación útil

No intentar resolver todos los transportes a la vez.

Primera fase:

- órdenes en BD
- API para crearlas
- implementación real para `pty/process`

## Reparto sugerido

- `Codex2`
  envío autónomo de instrucciones y cola de órdenes

- `Codex3`
  handoff y reasignación sobre agentes vivos

- `Codex1`
  consolidación arquitectónica y contratos
