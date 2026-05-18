# AGENTS: orquesta-domain-work-memory

Este modulo es un adaptador neutral de memoria para `orquesta-domain-work`.

Reglas:

- mantener la API centrada en `DomainWorkJobRecordStorePortV0` sin meter IO;
- no importar adaptadores de producto, runtime, DB, HTTP ni filesystem;
- no introducir semantica OPES, programacion, Codex o app concreta;
- validar antes de mutar estado;
- conservar reintentos idempotentes: misma clave y mismo request normalizado
  devuelve el mismo `job_ref`;
- rechazar conflictos de idempotencia sin sobrescribir el job previo;
- devolver copias defensivas de slices y `json.RawMessage`;
- si se anade lectura, debe filtrar por contrato neutral y no listar internals;
- ejecutar `orquesta-domain-work/contracttest` desde tests del adaptador cuando
  cambie `DomainWorkJobRecordStorePortV0`;
- documentar cambios de contrato en `docs/contratos.md` y evidencias en
  `docs/pruebas.md`.

Si necesitas persistencia durable, crea otro adaptador externo. No metas SQL,
rutas o escritura de ficheros en este paquete.
