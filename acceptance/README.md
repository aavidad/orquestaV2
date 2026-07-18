# Gates de aceptación del rebuild

## V01: integración de fuentes

`AC-V01-SOURCE-INTEGRATION` ejecuta todos los tests `TestProductRoadmap.*` y
acredita el catálogo causal solo mediante receipt V3. `V01` identifica la
vertical; `V3` identifica la versión sellada del formato de evidencia.

El candidato exacto contiene el soporte común de receipts, el fixture y test
V01, `product/roadmap.json` y `product_roadmap_test.go`. Receipt y salida quedan
fuera del candidato. Fixture, rutas, modos y contenidos se leen de blobs del
commit sellado: un cambio posterior del roadmap no reescribe la evidencia
histórica. Una nueva versión del contrato V01 exige otro candidato, ejecución y
receipt; no se modifica el anterior en silencio:

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

## Receipt V3: candidato Git sellado

Todo receipt V3 apunta a un commit Git completo, existente y ancestro de
`HEAD`, descendiente del mínimo confiable del contrato. La ejecución declara ese
mismo commit como fuente. El validador obtiene árbol, fixture y sujetos mediante
`ls-tree`/`cat-file`; nunca usa sus bytes del worktree vivo.

Emisión tiene dos commits. Primero se integra candidato `C`, sin modificar
receipt/salida. Los gates se ejecutan en un worktree temporal `--detach C` cuyo
`HEAD` coincide exactamente con `C` y cuyo `git status --porcelain=v1` está
vacío. Receipt declara `source_git_commit_oid=C`, estado `detached_clean` y el
SHA-256 de status vacío; `executed_at` no puede preceder al commit. Después se
integra commit de evidencia `E`, que solo añade/renueva salidas y receipts. No se
ejecuta desde el worktree principal sucio ni se cambia candidato entre `C` y la
ejecución.

Los contratos que además sellan un delta `base..P` usan la forma equivalente
de tres commits: `P` contiene producto, tests, roadmap y documentación; `S`
solo liga el fixture y su constante a `P` y puede añadir un receipt placeholder
regular cuando el roadmap exige que la ruta exista; el argv corre desde `S`
detached y limpio; `E` sustituye el placeholder y añade la salida. Los
`candidate_subjects` son exactamente el delta `base..P`, nunca receipt/salida
ni los cambios de enlace exclusivos de `S`.

El fixture sellado es la única lista de `candidate_subjects`. Cada sujeto debe
ser blob regular `100644` o `100755`; rutas absolutas, escapes, duplicados,
symlinks, árboles y submódulos fallan cerrados. El digest enmarca ruta, modo y
contenido. Receipt y salida quedan fuera del conjunto. El receipt conserva
argv, exit code, salida, versión Go, timestamp, commit, tree, blob del fixture y
digests. Cambios futuros pueden evolucionar los mismos paths sin invalidar el
hecho histórico acreditado en aquel commit.

V3 acredita reproducibilidad e integridad dentro del historial Git; no aporta
autenticidad frente a un escritor capaz de reescribir todo el repositorio. Un
despliegue que necesite esa garantía debe añadir tag firmado, Sigstore/in-toto o
CAS externo sin cambiar el contrato del candidato.

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
`product/evidence/v02_authority_rules.json` sea válido. El fixture del commit
sellado enumera los ficheros exactos, excluye receipt/salida y liga ruta, modo y
contenido. El candidato contiene `AGENTS.md` y este paquete de aceptación
—guía, soporte común, fixture y test—. Cambiar ahora autoridad, gate, fixture o
guía requiere un nuevo candidato; no altera el receipt histórico ya sellado.

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

El receipt V3 excluye tanto su propio fichero como la salida capturada. Conserva
argv estructurado, exit code, hash de salida, versión Go, timestamp y la
identidad Git sellada del conjunto exacto de sujetos. `product/roadmap.json` y el ledger
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
