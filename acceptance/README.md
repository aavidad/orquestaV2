# Gates de aceptación del rebuild

## V01: integración de fuentes

`AC-V01-SOURCE-INTEGRATION` ejecuta todos los tests `TestProductRoadmap.*` y
acredita el catálogo causal solo mediante receipt V2. `V01` identifica la
vertical; `V2` identifica la versión del formato de evidencia.

El candidato exacto contiene el soporte común de receipts, el fixture y test
V01, `product/roadmap.json` y `product_roadmap_test.go`. Receipt y salida quedan
fuera del candidato. Como el roadmap es estado vivo, cualquier cambio suyo
invalida el digest y obliga a ejecutar de nuevo el gate y reemitir ambos
artefactos:

- `product/evidence/v01_source_integration.output.txt`;
- `product/evidence/v01_source_integration.json`.

Comando acreditado:

```bash
go test -mod=vendor -count=1 . -run '^TestProductRoadmap.*$'
```

La verificación posterior usa
`^TestAcceptanceV01SourceIntegrationReceipt$`; nunca se acredita V01 por su
posición, por tener contrato ejecutable ni por una excepción de nombre.
El gate estructural exige la ruta canónica del receipt, pero no que el fichero
exista antes de ejecutar V01: así la primera emisión no necesita placeholder.
La existencia, hashes y metadata se exigen únicamente en la verificación
posterior, que sí debe quedar verde para declarar el corte acreditado.

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

## V03: trazabilidad canónica

`v03_canonical_ledgers_test.go` acredita el censo no circular del legado y la
partición exacta de tareas, exclusiones, bloques narrativos y lecciones de bugs.
El fixture sella los 19 ledgers canónicos; las revisiones auxiliares quedan
ligadas por hashes dentro de sus ledgers de procedencia y se vuelven a validar
en los gates focales.

El receipt V2 excluye tanto su propio fichero como la salida capturada. Conserva
argv estructurado, exit code, hash de salida, versión Go, timestamp y la
identidad del conjunto exacto de sujetos. `product/roadmap.json` y el ledger
vivo de bugs del rebuild quedan fuera del candidato estático para permitir que
el progreso causal y nuevas lecciones no falsifiquen la acreditación; sus tests
siguen ejecutándose en el comando completo.

Comando acreditado:

```bash
go test -mod=vendor -count=1 . ./acceptance \
  -run '^(TestTraceabilityRebuild.*|TestProductRoadmap.*|TestAcceptanceV03CanonicalLedgers)$'
```

La verificación posterior del receipt se ejecuta aparte con
`^TestAcceptanceV03CanonicalLedgersReceipt$`; así el receipt no necesita
atestiguarse a sí mismo para poder emitirse.
