# Pruebas: orquesta-agent-process-registry-memory

```sh
go test -count=1 ./modulos/orquesta-agent-process-registry-memory
```

Cobertura:

- record + resolve;
- refs inseguras rechazadas;
- replay idempotente;
- conflicto sin sobrescritura;
- lookup invalido;
- missing record;
- cancelacion de contexto;
- arquitectura sin dependencias prohibidas.
