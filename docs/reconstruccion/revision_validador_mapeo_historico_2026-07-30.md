# Revisión del validador de mapeo histórico — 2026-07-30

Este documento conserva dos revisiones independientes del candidato exacto del
validador auxiliar. Es un registro de coordinación de arranque con Orquesta
parada; no es un recibo productivo ni cambia el estado de ninguna capacidad.

## Ficha

```text
capability IDs: GOV-16
invariante: un revisor solo acepta los mismos bytes que inspeccionó
autoridad que escribe: ninguna; este documento no escribe producto
puertos afectados: ninguno
adaptadores afectados: ninguno
write-set: este documento, legacy_physical_mapping_review_record_test.go y
  legacy_physical_mapping_subject_security_test.go
dependencias causales: candidato comprometido y sujeto agregado exacto
código antiguo que permitirá retirar: ninguno
test de contrato: TestPhysicalMappingIndependentReviewRecord y sujeto físico
negativo/mutación: JSON ambiguo, promoción, subdirectorio, enlace o FIFO
E2E/gate: no aplica; el registro no acredita una compuerta productiva
presupuesto: documento <=180 líneas; cada prueba <=180 líneas
```

Excepción de arranque declarada: Codex directo registra las revisiones porque
Orquesta está parada. La consulta de lecciones de `GOV-16` devolvió
`Sin patrones`.

## Sujeto inmutable revisado

- aplicación: `scripts/legacy_physical_mapping`;
- commit: `08063318512682b7c6e5212affc1986afa363da3`
  (abreviado `08063318`);
- sujeto agregado:
  `7e85ac617f396023d76c1751f4122151bf45926d059972b4d67fe2bd00bb2a71`;
- universo: el conjunto nominal exacto de 14 ficheros regulares directos de la
  aplicación, sin recursión;
- orden: nombres de ruta en orden de bytes C;
- línea por fichero: `sha256_en_hexadecimal`, dos espacios, ruta y LF;
- agregado: SHA-256 de la concatenación exacta de esas líneas, igual que la
  salida encadenada de `sha256sum`.

La prueba asociada reproduce este cálculo con la biblioteca estándar de Go. No
ejecuta shell ni confía en el orden implícito del sistema de ficheros.
Cualquier nombre ausente o adicional, subdirectorio, enlace simbólico, FIFO,
entrada no regular o tipo dudoso hace fallar el cálculo. `Lstat` y la
información de directorio deben coincidir conservadoramente antes de leer.

## Revisiones independientes

```json
{"schema":"orquesta.bootstrap-independent-review-record.v1","subject_sha256":"7e85ac617f396023d76c1751f4122151bf45926d059972b4d67fe2bd00bb2a71","productive_receipt":false,"reviews":[{"reviewer":"revisor_mapeo_semantico","verdict":"ACEPTAR","p0":0,"p1":0,"p2":0},{"reviewer":"revisor_bloqueo_app13","verdict":"ACEPTAR","p0":0,"p1":0,"p2":0}]}
```

Ambos revisores inspeccionaron el mismo agregado y devolvieron
`ACEPTAR`, con `P0=0`, `P1=0` y `P2=0`. Sus identidades son distintas y
ninguno es el constructor del candidato.

El bloque es JSON canónico: compacto, con orden fijo, sin claves duplicadas,
desconocidas ni valores posteriores. `productive_receipt` debe ser `false`.

## Verificación reproducida

Los siguientes comandos pasaron sobre el candidato comprometido:

```bash
go test -mod=vendor -count=1 ./scripts/legacy_physical_mapping
go test -mod=vendor -count=1 -race ./scripts/legacy_physical_mapping
GOFLAGS=-mod=vendor go vet ./scripts/legacy_physical_mapping
go test -mod=vendor -count=1 . -run '^TestLegacyInventoryApplicationsDeclareTheirContract$'
git diff --check 08063318^ 08063318
```

El cuarto comando es la comprobación local de manifiesto y cabeceras de la
aplicación; no acredita APP-13 universal.

Métricas del sujeto:

- código productivo Go: 847 líneas;
- pruebas Go: 847 líneas;
- `README.md`: 137 líneas;
- fichero mayor: 203 líneas;
- función mayor: 59 líneas;
- total del sujeto: 1.831 líneas en 14 ficheros.

## Síntesis de fronteras revisadas

- `main.go` limita la línea de órdenes, abre tres entradas privadas sin seguir
  enlaces y reserva `stdout` para un veredicto completo;
- `json.go` impone tamaño, profundidad, elementos léxicos, JSON canónico y
  huellas con dominio;
- `model.go` mantiene contratos tipados sin rutas ni identidad física cruda;
- `base.go` liga por bytes el V3 y el universo y deriva los sujetos lógicos;
- `candidate.go` separa presencia histórica de presencia declarada en la vista;
- `ownership.go` normaliza por vinculación física, conserva aliases lógicos y
  valida dueño, solapamiento, poda y ausencia de ciclos;
- las pruebas ejercitan mutaciones de base, contexto, tipo, alias, propiedad,
  poda, límites JSON, permisos, enlaces y escritura parcial.

La utilidad valida únicamente forma, ligadura por bytes e integridad interna.
No abre una vista estable, no observa presencia física, no autentica evidencias
de reapertura, no descubre solapamientos omitidos y no decide autoridad
productiva.

## Límite explícito de este registro

Este registro bootstrap no es un receipt productivo y no acredita el mapeo, la
vista estable, el cercado, la presencia o ausencia real, los recibos del censo
ni ninguna compuerta. Tampoco es una atestación independiente emitida por
Orquesta o evidencia suficiente para promover `GOV-16`.

Queda prohibida cualquier frase afirmativa que atribuya a este registro la
acreditación de GOV-16, del mapeo, de la vista, de los recibos o el cierre de
una compuerta. La prueba conserva una lista explícita de esas promociones y
demuestra que cada una se rechaza.

Las dos aceptaciones solo significan que los dos revisores no encontraron P0,
P1 ni P2 en los bytes identificados. Cambiar cualquier fichero produce otro
sujeto y obliga a repetir pruebas y revisiones.

## Cierre

```text
hecho: dos revisiones independientes quedan ligadas al sujeto exacto
invariante restaurado: ningún veredicto se atribuye a bytes distintos
autoridad final: ninguna; el roadmap permanece intacto
tests: agregado, revisores, severidades y caveat comprobados
receipts y revisión acreditada: registro bootstrap; no receipt productivo
código o decisión retirados: ninguno
legacy retirado o bloqueo de retirada: no se retira nada por esta revisión
LOC netas y complejidad: solo documento y prueba de registro
riesgos/P0/P1: falsa promoción si se omite el límite no productivo
siguiente dependencia causal: decidir y acreditar la composición física
```
