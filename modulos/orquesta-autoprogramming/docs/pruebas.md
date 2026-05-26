# Pruebas: orquesta-autoprogramming

## Local

```sh
go test -count=1 ./modulos/orquesta-autoprogramming
```

Cobertura actual:

- acepta solicitud pequena con worktree aislada;
- acepta ola de 10 tareas padre dentro de los limites vigentes;
- rechaza presupuestos de delegacion fuera de rango y preserva limites
  explicitos validos en `WorkflowTaskV0`;
- rechaza branch/ref/isolation incompletos;
- rechaza alcance demasiado amplio;
- rechaza `write_set` vacio o inseguro;
- agrupa tareas por area normalizada;
- preserva contratos explicitos por tarea: objetivo, contexto, context refs,
  criterios, tests y reglas compactas;
- compacta textos largos de tareas antes de `WorkflowTaskV0` y mantiene el
  payload durable por debajo del limite del core;
- transforma solicitudes validas en `WorkProfileV0`/`WorkflowTaskV0` con refs
  opacas y pruebas requeridas preservadas;
- particiona `write_set` por area, normaliza aliases y rechaza rutas no
  asignables;
- secuencia rutas que coinciden con varias areas compatibles;
- pospone tareas cuyo `write_set` solapa trabajos vivos y conserva `depends_on`
  causal;
- rechaza trabajos vivos con refs imposibles o rutas inseguras;
- review gate acepta ACK completado con tests verdes;
- review gate rechaza ACK ausente, tests fallidos y tests ausentes;
- review gate acepta como rail blando reutilizable una entrega con tests verdes,
  ficheros grandes, ficheros fuera del `write_set` o destinos faltantes;
- review gate mantiene ACK ausente o tests fallidos como bloqueo de cierre con
  evidencia compacta;
- politica de capacidad usa tokens operativos para OPES/riesgo alto y no escala
  a `xhigh` por subcadenas inocuas como `scopes`;
- arquitectura impide importar core, runtime, DB, `cmd` o adaptadores.

## Integracion focal

```sh
go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-mcp ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-orchestration-core
```
