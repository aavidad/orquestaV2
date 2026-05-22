# Pruebas: orquesta-autoprogramming

## Local

```sh
go test -count=1 ./modulos/orquesta-autoprogramming
```

Cobertura actual:

- acepta solicitud pequena con worktree aislada;
- rechaza branch/ref/isolation incompletos;
- rechaza alcance demasiado amplio;
- rechaza `write_set` vacio o inseguro;
- agrupa tareas por area normalizada;
- transforma solicitudes validas en `WorkProfileV0`/`WorkflowTaskV0` con refs
  opacas y pruebas requeridas preservadas;
- particiona `write_set` por area y rechaza rutas no asignables o solapadas;
- review gate acepta ACK completado con tests verdes;
- review gate rechaza ACK ausente, tests fallidos, tests ausentes, ficheros
  grandes y ficheros fuera del `write_set`;
- arquitectura impide importar core, runtime, DB, `cmd` o adaptadores.

## Integracion focal

```sh
go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-mcp ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-orchestration-core
```
