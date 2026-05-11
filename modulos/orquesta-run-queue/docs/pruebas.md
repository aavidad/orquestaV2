# Pruebas

Comando local:

```sh
go test -count=1 ./modulos/orquesta-run-queue
```

Cobertura v0:

- filtrado de `paused`, `canceled`, `stopped` y `closed`;
- prioridad antes de aging;
- aging como desempate entre misma prioridad;
- `updated_at` ascendente con estabilidad en empates exactos;
- pureza basica: no mutar referencias de evidencia del input;
- gate de arquitectura contra imports/terminos de adaptadores y persistencia.
