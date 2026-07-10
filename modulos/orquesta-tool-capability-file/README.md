# orquesta-tool-capability-file

Adaptador durable de referencia para `ToolOperationReceiptStorePortV0`.

Persiste plan, receipt, indices y leases de destino en un estado JSON sustituido
atomicamente. Usa `flock` para serializar clientes de procesos distintos,
`O_NOFOLLOW` para estado/lock, permisos privados y `fsync` de fichero/directorio.
No resuelve ni materializa snapshots de bundles.

```sh
go test -race -count=1 ./modulos/orquesta-tool-capability-file
```
