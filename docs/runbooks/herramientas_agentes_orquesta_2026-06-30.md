# Herramientas persistentes para agentes Orquesta - 2026-06-30

Objetivo: que cada instalacion de Orquesta para agentes deje las mismas reglas
de uso de herramientas, no dependientes de memoria de sesion.

## Norma

- El camino canonico en Orquesta es el broker central de contexto de codigo:
  `rg` por defecto y `codebase-memory-mcp` solo como adaptador opt-in detras del
  broker cuando hagan falta simbolos, llamadas, rutas, clusters o arquitectura.
- `rg`/lectura local acotada sigue siendo canonico para strings exactos,
  Markdown, incidencias, configs, scripts shell, logs y busquedas no cubiertas
  por el grafo.
- La comunicacion de agentes debe ser compacta. Si existe `caveman`, usarla o
  pedir estilo equivalente: hecho, tests, riesgos/bloqueos y siguiente accion.
- En OPES, `codebase-memory-mcp` no es indice de contenido pedagogico. Si se
  usa, debe ser solo para codigo y validadores, preferentemente via broker. Para
  no leer todos los temas, usar RAG/documentos canonicos OPES con refs
  compactas: `rag/corpus/chunks.jsonl`, `rag/corpus/summary.json` y
  `rag/manifest.json`.
- No abrir UI ni puertos externos por defecto. En remoto aislado, cualquier
  indice o cache vive bajo usuario `berserk` o `/srv/orquesta-self`, nunca en
  produccion.

## Bootstrap

Ejecutar desde el repo:

```bash
scripts/bootstrap_agent_tooling.sh --repo "$(pwd)"
```

Efectos:

- escribe norma persistente en `~/.codex/AGENTS.md`;
- guarda ledger en `~/.codex/log/orquesta-agent-tooling.env`;
- no instala `codebase-memory-mcp` como MCP directo en `CODEX_HOME` por defecto.

Para preparar el proveedor opt-in y reindexar sin configurar MCP directo:

```bash
scripts/bootstrap_agent_tooling.sh --repo "$(pwd)" --enable-codebase --index
```

Efectos adicionales:

- ejecuta `codebase-memory-mcp update -y` si existe y no se pasa `--no-update`;
- escribe norma persistente en `~/.codex/AGENTS.md`;
- guarda version en `~/.codex/log/orquesta-agent-tooling.env`;
- con `--index`, indexa el worktree actual en modo `fast`.
- si se trabaja con subagentes, no se debe pasar este contexto como permiso para
  que cada subagente arranque su propio MCP o indexador.

`codebase-memory-mcp install -y --ui=false` solo se ejecuta con
`--install-direct-mcp`. Ese modo requiere opt-in operativo explicito porque
puede dejar MCP directo disponible para sesiones o subagentes fuera del broker
central de Orquesta.

Si el binario no esta en `PATH`, definir:

```bash
CODEBASE_MEMORY_MCP_BIN=/ruta/codebase-memory-mcp \
  scripts/bootstrap_agent_tooling.sh --repo "$(pwd)" --enable-codebase --index
```

## Status no destructivo

Antes de una sesion larga o al investigar procesos duplicados:

```bash
scripts/bootstrap_agent_tooling.sh --repo "$(pwd)" --status
```

El status informa `config_codebase_memory_mcp`, `direct_mcp`,
`broker_state_dir` y `live_codebase_memory_mcp_processes`. No para procesos; si
hay MCP directo no autorizado o procesos vivos, devuelve
`estado=attention_required`.

## Evidencia local

El MCP local respondio con proyecto `home-alberto-Trabajo-orquesta`, estado
`ready`, alrededor de 38k nodos y 157k relaciones. Busquedas de prueba
devolvieron simbolos reales de `running_stale` y diagnosticos del supervisor.

## Evidencia remota aislada

En `berserk@uso.dipgra.cloud` se instalo el binario estatico
`~/.local/bin/codebase-memory-mcp` version `0.8.1`, checksum:

```text
48e1f5b086dc2dff4a320085e400464630583e1d8eaaf143568041ca6d992cd9
```

Historico: `codebase-memory-mcp install -y --ui=false` configuro Codex remoto en
`~/.codex/config.toml` y `~/.codex/AGENTS.md`. Desde el ajuste de gobernanza del
2026-07-02, ese paso queda detras de `--install-direct-mcp` y no forma parte del
bootstrap normal. El worktree aislado
`/srv/orquesta-self/runtime/audit-14ab41f1-next` quedo indexado como
`srv-orquesta-self-runtime-audit-14ab41f1-next`, con busqueda verificada sobre
`app_server_tmux shutdown pane pid`.

## Desactivar MCP directo heredado

Si `scripts/bootstrap_agent_tooling.sh --status` devuelve
`config_codebase_memory_mcp=1` con `direct_mcp=false`, el entorno tiene una
configuracion heredada fuera del broker. Hay que retirar el bloque
`[mcp_servers.codebase-memory-mcp]` y el hook `codebase-memory-mcp SessionStart`
de `CODEX_HOME/config.toml`, dejando copia de seguridad antes de editar.

Despues de retirarlo, ejecutar:

```bash
scripts/bootstrap_agent_tooling.sh --repo "$(pwd)" --no-update
scripts/bootstrap_agent_tooling.sh --repo "$(pwd)" --status
```

El estado esperado es `estado=ok`, `config_codebase_memory_mcp=0`,
`direct_mcp=false` y `live_codebase_memory_mcp_processes=0`.

## Mantenimiento

- Reejecutar `scripts/bootstrap_agent_tooling.sh --enable-codebase --index` solo
  cuando haga falta preparar el proveedor opt-in en un nuevo contenedor, usuario,
  worktree remoto o sesion larga.
- Reindexar tras cambios grandes de arquitectura o imports.
- Si `codebase-memory-mcp update` falla por red o entorno aislado, no bloquear
  trabajo: registrar aviso y continuar con la version instalada.
- Si una herramienta nueva demuestra ahorro real, anadirla a `AGENTS.md`, este
  runbook y el bootstrap antes de convertirla en uso normal.
