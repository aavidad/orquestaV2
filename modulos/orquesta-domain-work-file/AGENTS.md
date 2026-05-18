# AGENTS: orquesta-domain-work-file

Este modulo es un adaptador filesystem para `orquesta-domain-work`.

Reglas:

- implementar `DomainWorkJobRecordStorePortV0` sin mezclar comando y lectura;
- usar solo contratos neutrales de `orquesta-domain-work`;
- no importar OPES, Codex, runtime, MCP, web, `cmd`, `db`, HTTP ni drivers SQL;
- validar y normalizar antes de escribir;
- preparar el snapshot siguiente, escribirlo y solo despues mutar el estado en
  memoria;
- escribir JSON con temp file, `fsync`, `rename` y sync de directorio;
- tratar conflicto de idempotencia como `DomainWorkJobV0{status=invalid}`, no
  como error Go;
- mantener la clave idempotente como `domain_ref + idempotency_key`;
- conservar requests normalizados, jobs aceptados y fingerprint en el snapshot;
- cualquier lectura debe filtrar por contratos neutrales, ordenar de forma
  estable y devolver copias defensivas;
- ejecutar `orquesta-domain-work/contracttest` desde tests del adaptador cuando
  cambie `DomainWorkJobRecordStorePortV0`;
- devolver copias defensivas.

No conviertas este paquete en bundle productivo completo. Un conector DB debe
vivir en otro adaptador o extraer antes un contrato comun de ledger.
