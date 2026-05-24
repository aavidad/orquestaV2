# Pruebas locales: orquesta-context

## CTX-P001 builder de contexto pequeno

Tipo: unit_contract

Comando: `go test -count=1 ./modulos/orquesta-context`

Evidencia esperada:

- programacion incluye reglas comunes, docs locales minimos, `docs/pruebas.md`, read-set, write-set y contratos;
- brainstorming incluye `docs/decisiones.md`;
- refs cruzadas quedan en `contract_context`;
- si falta contexto externo, la politica obliga a `CONSULTA AL DIRECTOR`;
- detalles de HOME/OAuth/proveedor/motor/secreto se rechazan;
- exceso de entradas falla con error publico.

Ultima ejecucion: 2026-05-05, ejecutada correctamente.

Riesgos: no prueba lectura real de archivos ni arranque runtime; solo fija el contrato puro del manifiesto.

## CTX-P002 materializacion explicita por filesystem

Tipo: unit_contract

Comando: `go test -count=1 ./modulos/orquesta-context`

Evidencia esperada:

- `FileContextRefStoreV0` exige root explicito;
- bloquea parent traversal;
- reporta ref no encontrada;
- materializa `doc_ref` y `read_ref` como contenido;
- mantiene `write_ref`, `contract_ref` y `evidence_ref` como `ref_only`;
- trunca entradas grandes;
- mantiene `total_bytes` por debajo de `max_total_bytes`;
- propaga la pista `CONSULTA_AL_DIRECTOR`.

Ultima ejecucion: 2026-05-05, ejecutada correctamente.

Riesgos: usa filesystem temporal de test; no genera prompt final ni arranca agente real.

## CTX-P003 saneamiento local de contexto materializado

Tipo: unit_contract

Comando: `go test -count=1 ./modulos/orquesta-context`

Evidencia esperada:

- `MaterializeContextBundleWithSanitizerV0` permite inyectar
  `ContextSanitizerPortV0`;
- conserva `ContextSanitizationEvidenceV0` sin persistir token, secreto ni HOME;
- si el sanitizador devuelve `review_required`, la entrada queda como `ref_only`
  y el bundle exige `CONSULTA_AL_DIRECTOR`.

Riesgos: el sanitizador real es adaptador de composicion; este modulo solo fija
el puerto y el comportamiento neutral del bundle.
