# orquesta-core-concurrency

Responsabilidad: politica pura de paralelismo logico y conflictos de read/write-set.

Este microproyecto decide si varias microtareas/agentes pueden trabajar a la vez sin pisarse. No observa Git ni filesystem real; usa contratos declarados y refs opacas.

Incluye:

- dependencias entre tareas;
- read_set y write_set compactos;
- conflictos entre agentes;
- locks logicos por scope;
- recomendacion de orden o consulta al director.

No incluye runtime real, Git real, filesystem productivo, DB, provider, modelo, HOME ni OAuth.

## Arranque

```bash
./arrancar_codex.sh "microtarea concreta"
```
