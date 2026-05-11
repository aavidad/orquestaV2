# orquesta-run-coordinator

Lee este archivo y `README.md` antes de tocar el modulo.

Reglas locales:

- Write-set del modulo: `modulos/orquesta-run-coordinator/**`.
- Paquete puro de coordinacion: sin adaptadores, persistencia, transporte ni procesos.
- Usar `orquesta-run-queue` para leer y rankear candidatos.
- Usar `orquesta-run-control` para decidir si una run se puede despachar.
- Ejecutar solo mediante el puerto inyectado `RunDrainerPortV0`.
- Mantener archivos pequenos y contratos versionados `V0`.

Validacion local:

```bash
go test -count=1 ./modulos/orquesta-run-coordinator
```
