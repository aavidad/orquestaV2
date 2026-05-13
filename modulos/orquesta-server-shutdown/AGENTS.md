# Contexto local: orquesta-server-shutdown

Lee este fichero antes de tocar el modulo.

## Responsabilidad

Caso de uso hexagonal para apagado controlado de Orquesta:

```text
listar runs -> pedir stop por RunControl -> drenar por supervisor -> verificar cierre
```

## Reglas

- No importa Codex, MCP, web, DB, filesystem, runtime ni `cmd`.
- No para procesos directamente.
- Usa `RunControl`, `RunQueue`, `RunSupervisor` y stats por puertos.
- Mantiene DTOs compactos y sin rutas, HOME, OAuth, proveedor, prompts ni transcripts.
- Ficheros pequenos y pruebas por comportamiento.
