# Gates de aceptación del rebuild

`v02_authority_rules_test.go` ejecuta `AC-V02-AUTHORITY-RULES` sin arrancar,
importar ni compilar el runtime antiguo. Solo lee dos superficies congeladas y
analiza el producto nuevo:

- `modulos/**` completo;
- `cmd/**`, salvo el binario nuevo `cmd/orquesta/**`;
- `internal/**` y `cmd/orquesta/**` mediante AST y resolución local de tipos.

El fixture `fixtures/v02_authority_rules.json` sella cada fichero legacy en
orden de ruta. El hash enmarca ruta, bit ejecutable, tamaño y contenido, por lo
que añadir, quitar, renombrar o modificar un fichero cambia el digest.

No se actualiza el fixture para silenciar una regresión. Solo V34 puede cambiar
una superficie congelada, con equivalencia, cutover y retirada acreditados en
su propio write-set. Un temporal dentro de estas rutas también rompe el gate y
debe limpiarse o justificarse, no incorporarse al baseline.

La regla de ejecuciones de V02 es arquitectónica: `ExecutionRecord` es un
intento/proyección idempotente fuera del agregado `Goal` y el provider entra por
`AgentLauncher`. Este gate no acredita todavía rework ni múltiples intentos por
`WorkItem`; esas conductas requieren sus verticales y pruebas causales.

Las capacidades V02 solo cuentan como `accredited` mientras
`product/evidence/v02_authority_rules.json` sea válido. El receipt enumera los
ficheros exactos del candidato, excluye el propio receipt y sella contenido y
ruta con framing. El candidato estable contiene `AGENTS.md` y este paquete de
aceptación —guía, soporte común, fixture y test—. El gate recalcula sus hashes; cambiar
autoridad, gate, fixture o guía invalida la acreditación.

`product/roadmap.json` queda fuera deliberadamente: su gate causal valida por
separado contrato, receipt y `evidence_refs`, y el progreso de V03+ no debe
invalidar una V02 ya acreditada.

Comando focal seguro:

```bash
go test -mod=vendor -count=1 ./acceptance \
  -run '^TestAcceptanceV02AuthorityRules$'
```
