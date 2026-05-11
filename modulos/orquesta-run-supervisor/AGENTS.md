# orquesta-run-supervisor

Lee este archivo y `README.md` antes de tocar el modulo.

Reglas locales:

- Write-set del modulo: `modulos/orquesta-run-supervisor/**`.
- Paquete puro de supervision: sin adaptadores, persistencia, transporte ni procesos.
- Ejecutar ticks solo mediante `RunSupervisorTickPortV0`.
- No usar sleeps, goroutines ni bucles sin presupuesto.
- Mantener archivos pequenos y contratos versionados `V0`.

Validacion local:

```bash
go test -count=1 ./modulos/orquesta-run-supervisor
```
