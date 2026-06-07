# Pruebas locales: orquesta-context

## CTX-P004 contexto required ref_only en T15

Tipo: contract | regression

Comando: `go test -count=1 ./modulos/orquesta-rails ./modulos/orquesta-core-workflow ./modulos/orquesta-context ./modulos/orquesta-director-agent ./cmd/orquesta-server`

Evidencia esperada:

- cada entrada required `ref_only` conserva razon y accion requerida;
- ACK completed incluye evidencia explicita cuando `required_ref_action` es
  `ack_evidence_required`;
- refs opacas y vocabulario operativo no se tratan como dato sensible;
- vocabulario operativo o local no bloquea el bundle por strings sueltos;
- valores sensibles efectivos o material crudo siguen bloqueando/saneandose;
- sanitizacion de metadata no persiste valores sensibles efectivos en refs de
  evidencia.

Ultima ejecucion documentada: 2026-05-27, ACKs cerrados de T15 con suite
requerida pasada.

Riesgos: la prueba no habilita nuevos scopes de detalle; solo valida el cierre
documental y el contrato de contexto.

## CTX-P001 builder de contexto pequeno

Tipo: unit_contract

Comando: `go test -count=1 ./modulos/orquesta-context`

Evidencia esperada:

- programacion incluye reglas comunes, docs locales minimos, `docs/pruebas.md`, read-set, write-set y contratos;
- brainstorming incluye `docs/decisiones.md`;
- refs cruzadas quedan en `contract_context`;
- si falta contexto externo, la politica obliga a `CONSULTA AL DIRECTOR`;
- vocabulario HOME/OAuth/proveedor/motor/secreto no reactiva veto automatico
  por palabra suelta;
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
- clasifica cada `required ref_only` con razon y accion esperada;
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
  con accion `ask_director` y el bundle exige `CONSULTA_AL_DIRECTOR`.

Riesgos: el sanitizador real es adaptador de composicion; este modulo solo fija
el puerto y el comportamiento neutral del bundle.
