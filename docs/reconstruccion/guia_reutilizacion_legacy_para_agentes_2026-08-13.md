# Guía de reutilización del legado para agentes

Fecha: 2026-08-13.

Estado: read-model advisory cerrado para el snapshot Git visible. No modifica
roadmap, no crea trabajo, no acredita capacidades y no autoriza imports,
arranque ni copia de paquetes del runtime antiguo.

## Pregunta que debe responder el inventario

Antes de programar una conducta que ya pudo existir, un agente debe poder
responder:

1. qué problema intentaba resolver V1;
2. qué parte fue descrita como verde y con qué procedencia;
3. qué parte falló o quedó sin demostrar;
4. cuál era el mecanismo, incluida la autoridad que decidía;
5. qué valor conserva para V2;
6. si debe reutilizar implementación V2, contrato, idea o lección negativa;
7. qué evidencia V2 manda ahora.

`legacy` significa «sin autoridad de runtime», no «sin valor». Tampoco significa
que una frase histórica como `cerrado`, `validado` o `verde` sea un receipt.

## Dos capas: catálogo general y decisión OrquestaV2

El inventario no está encerrado en OrquestaV2:

1. El **catálogo general portable de V1** busca funciones desde cualquier
   aplicación o directorio. Devuelve identidad, firma, procedencia, mecanismo,
   aciertos declarados, fallos y límites, tests observados y condiciones de
   adopción. No recibe capability, ruta objetivo ni estado V2.
2. El **assessment específico de OrquestaV2** cruza esa información con el
   roadmap, la capability, aceptación y evidencia vigentes para decidir el
   siguiente paso dentro de esta arquitectura.

Ejemplo de la primera capa desde otra aplicación:

```bash
cd /ruta/de/otra-app
/home/alberto/Trabajo/orquestaV2/scripts/consultar_catalogo_funciones_legacy_v1.sh \
  --name HandleCommandV0 --json
```

También admite `--text`, `--path`, `--package`, `--kind`,
`--occurrence-ref` y `--summary`. La ruta del catálogo se resuelve desde el
propio script, no desde el `cwd` consumidor. Su salida omite capabilities,
estado y recomendaciones V2. Conserva `contains_body=false`,
`consumer_application_decision=null` y `code_reuse_authorized=false`.

El repositorio no declara una licencia o autoría reutilizable por cada función
V1. El catálogo lo expresa como
`not_declared_per_function_requires_source_review`: no inventa MIT ni otra
licencia. Antes de copiar código hay que establecer procedencia/propiedad,
licencia compatible, equivalencia de conducta, dependencias, seguridad y tests
en la aplicación destino. Mientras falte cualquiera, solo se reutilizan
contratos, mecanismos o lecciones, nunca cuerpos.

## Preflight que usa el agente

El consumidor es otro agente de Orquesta V2, no una persona revisando el
inventario. Antes de diseñar o programar una función/tarea ejecuta:

```bash
scripts/preflight_reutilizacion_legacy.sh \
  --capability ORC-13 \
  --path internal/application/processing.go \
  --operation outbox \
  --task 'entregar outbox sin duplicar el efecto' \
  --function ClaimNextAction
```

La salida es un único JSON compacto. Combina:

- los ratchets transversales de `consultar_lecciones_legacy.sh`;
- las conductas V1 ya caracterizadas para la capability;
- qué se declaró verde, qué falló y por qué;
- el mecanismo y la autoridad que decidía;
- qué conservar y qué evitar;
- si V2 ya tiene solución acreditada;
- la acción concreta: reutilizar V2, contrato, idea o negativo;
- candidatos del censo estructural completo cuando el nombre o la identidad
  de función existen en V1.

Cada candidato incluye una decisión explícita:

- `reuse`: usar el owner y evidencia ya acreditados de V2;
- `reimplement`: trasladar contrato o idea a la arquitectura V2;
- `characterize`: falta resolver la brecha o el encaje exactos;
- `reject`: conservar el fallo como negativo y no adoptar el mecanismo.

Son decisiones advisory de implementación, no cierres ni permisos para copiar
código. `decision_summary.primary` señala el candidato mejor ordenado y cada
ficha conserva resultado histórico, limitaciones, tests y enlaces exactos.

La descripción de tarea/función solo ordena candidatos mediante coincidencia
léxica y nunca acredita equivalencia. Una ficha puede incluir enlaces AST
revisados y aun así declarar `exact_function_match=false`: significa que la
consulta entró por intención, no por identidad. Si el agente ya conoce una
`occurrence_ref`, puede exigir la relación exacta:

```bash
scripts/preflight_reutilizacion_legacy.sh \
  --capability ORC-13 \
  --path internal/application/processing.go \
  --operation outbox \
  --task 'cerrar una entrega confirmada' \
  --legacy-function-ref \
  sha256:dcc8f47db4bb4273329ca49c4926e6a846cf9e8fe5e18699f0eddcd2c3733bf5
```

Solo ese filtro puede producir `exact_function_match=true`. Si la capability
aún no está caracterizada, devuelve un hueco explícito en vez de inventar una
respuesta.

No usa IA, red, base de datos ni el runtime antiguo. Para investigación más
profunda, el agente puede consultar una ficha exacta:

```bash
scripts/consultar_reutilizacion_legacy.sh \
  --behavior recuperacion_outbox_sin_duplicar_efecto --json
```

También puede buscar cualquier función o método del snapshot, aunque todavía
no tenga valoración semántica:

```bash
scripts/consultar_funciones_legacy.sh --name HandleCommandV0 --json
scripts/consultar_funciones_legacy.sh --text 'outbox dispatch' --limit 10
scripts/consultar_funciones_legacy.sh --summary
```

`semantic_state=structural_only_not_semantically_assessed` significa «existe y
su identidad es estable», no «aporta valor» ni «puede copiarse». Si incluye
`semantic_links`, cada vínculo lleva conducta, resultado histórico, relación,
evidencia y recomendación V2.

Todas las funciones incluyen además `module_context`: regla de retirada,
capabilities relacionadas y razón arquitectónica del módulo. Es contexto de
alcance, no equivalencia de la función. `module_semantic_leads` añade hasta
cinco pistas documentales del mismo módulo con la misma cautela.

Si todavía no existe una ficha profunda, el agente baja primero a una pista o
a la fuente exacta, sin cargar todo el corpus:

```bash
scripts/consultar_pistas_legacy.sh --capability ORC-12 --limit 20 --json
scripts/consultar_pistas_legacy.sh --entry-ref TASKENTRY-... --json
scripts/consultar_fuentes_legacy.sh --capability ORC-12 --json
scripts/consultar_fuentes_legacy.sh --source RUTA/EXACTA.md --json
```

Una `ledger_semantic_lead` ya conserva capability, razón, rango y hashes, pero
todavía no afirma acierto, fallo ni función. Una `source_disposition_lead`
localiza documentos accionables; tampoco afirma que un bug siga vivo. El
agente abre solo esas refs y crea una caracterización profunda antes de tomar
una decisión de reutilización.

## Cómo leer el verde histórico

Estados conservadores de la contrarrevisión:

| Estado | Significado |
| --- | --- |
| `documented_green_unverified` | La fuente describe resultados favorables, pero el inventario no conserva un receipt independiente. |
| `partial_or_mixed` | Hay mecanismos o pruebas favorables y también huecos o fallos relevantes. |
| `failed_or_negative` | El problema central no quedó resuelto; el valor principal es el contrato o la regresión negativa. |
| `unknown` | Hay diseño o propuesta, pero no base suficiente para clasificar un resultado. |

La salida separa `claims`, `limitations` y el dictamen. Nunca convierte una
afirmación documental en `verified_test_green`. Para elevar una función a verde
verificado harán falta test o receipt recuperable, sujeto exacto, revisión y
procedencia comprobada.

## Decisión de reutilización

Cada ficha tiene un tipo primario:

- `v2_existing`: buscar primero contratos, implementación y evidencia V2;
- `contract`: conservar entradas, salidas, invariantes, errores y negativos;
- `idea`: conservar el enfoque, reimplementándolo dentro de V2;
- `negative_lesson`: conservar el fallo para no repetirlo.

La reutilización de código queda siempre en
`not_assessed_requires_function_provenance_license_and_architecture_fit` hasta
que el censo por función compruebe identidad, variante, licencia/autoría,
dependencias, seguridad, equivalencia contractual y ausencia de una solución
V2 mejor. No se almacenan cuerpos de función en el catálogo.

Si la capability está `accredited`, el agente no debe crear otra implementación
por defecto. Debe abrir primero `acceptance_contracts` y `evidence_refs`, buscar
el owner actual y comparar el detalle semántico. Si está `declared`, la ficha es
material para caracterización y diseño; no es autorización de implementación
ni cierre.

## Cobertura actual

No copies cifras congeladas desde este documento. El estado vigente se obtiene
de los read-models validados:

```bash
scripts/consultar_reutilizacion_legacy.sh --summary
scripts/consultar_funciones_legacy.sh --summary
scripts/consultar_catalogo_funciones_legacy_v1.sh --summary
scripts/consultar_pistas_legacy.sh --summary
scripts/consultar_fuentes_legacy.sh --summary
```

Hay cinco porcentajes distintos y el agente debe nombrar siempre cuál usa:

| Capa | Denominador | Qué significa un 100 % |
| --- | --- | --- |
| fuentes Markdown | fuentes clasificadas del snapshot | toda fuente está marcada como accionable o solo contexto |
| pistas del ledger | `TASKENTRY-*` únicas | toda pista tiene capability, razón y procedencia revisadas |
| censo Go | declaraciones productivas del índice Git | toda función/método tiene identidad, ruta y firma consultables |
| amplitud profunda | capabilities del ledger con al menos una conducta | cada familia de problema V1 tiene una ficha rápida o un hueco explícito |
| detalle profundo | `TASKENTRY-*` enlazadas a conductas | esas pistas concretas ya fueron agrupadas por problema, aciertos, fallos, mecanismo, valor V2 y funciones exactas |

La amplitud por capability es el porcentaje de avance principal de este
inventario: responde si un agente encontrará una ficha al entrar por problema.
El detalle por `TASKENTRY` mide cuánto material fuente ha sido además agrupado
en fichas profundas; no necesita llegar a 100 % cuando varias entradas repiten
el mismo backlog o mecanismo. Ninguna capa es porcentaje del producto V2. El
censo físico de historia y raíces externas sigue sujeto a sus compuertas y no
se mezcla con el snapshot Git.

El cierre vigente clasifica 840/840 fuentes, revisa 2.108/2.108 pistas y da una
ficha profunda a 184/184 capabilities del ledger. Es 100 % de amplitud para el
problema que consulta el agente; las 642 pistas agrupadas (30,45 %) son detalle
no redundante y no otra deuda de amplitud.

El ratchet estructural contiene 15.543 declaraciones en dos snapshots separados,
sin cuerpos: `modulos/**` aporta 10.822 funciones y 2.382 métodos; el `cmd/**`
legacy, excluyendo el producto nuevo `cmd/orquesta/**`, aporta 1.960 funciones y
379 métodos. `cmd/**` permanece estructural y no se le atribuye utilidad ni verde
por el mero censo. `function_behavior_links` cuenta relaciones;
`unique_exact_function_occurrences` cuenta apariciones físicas distintas. No se
divide una cifra por otra para fabricar avance: una conducta puede usar varias
funciones y la misma función puede explicar varias conductas.

Los enlaces exactos se recalculan contra el índice Git: ruta, blob, declaración,
variante, firma, AST, cuerpo por hash y líneas deben coincidir. El catálogo no
guarda cuerpos. Cambiar una firma, implementación, ruta o identidad rompe el
test en vez de dejar una referencia silenciosamente obsoleta. Los tres grupos
de tests legacy observados prueban únicamente su alcance unitario; no elevan
una conducta completa a verde ni acreditan V2.

Los fixtures originales todavía conservan su estado pendiente de
contrarrevisión porque son inmutables en este write-set. El fichero
`product/knowledge/legacy_reuse_assessments_v1.jsonl` registra la contrarrevisión
advisory del valor de reutilización; no cambia su disposición canónica.

## Regla para no reinventar la rueda

```text
capability acreditada -> buscar owner/test/evidence V2 -> reutilizar V2
capability declarada  -> conservar contrato/idea V1 -> caracterizar -> reimplementar
fallo histórico       -> convertir en negativo/ratchet
código V1             -> no copiar hasta procedencia, licencia y fit acreditados
```

En todos los casos permanecen vigentes: un Goal, un writer, un scheduler, una
fuente de estado, configuración canónica, permisos antes del efecto y cero
dependencia runtime del legado.

## Verificación focal

```bash
scripts/test_consultar_reutilizacion_legacy.sh
scripts/test_preflight_reutilizacion_legacy.sh
scripts/test_consultar_funciones_legacy.sh
scripts/test_consultar_catalogo_funciones_legacy_v1.sh
scripts/test_consultar_pistas_legacy.sh
scripts/test_consultar_fuentes_legacy.sh
go test -mod=vendor -count=1 ./scripts/legacy_function_inventory
go test -mod=vendor -count=1 . \
  -run '^TestLegacyFunctionSnapshotIndexMatchesGitIndex$'
```

El test fija conteos, unicidad, estados advisory, separación entre verde
histórico y acreditación, mapeo ORC-13 a V06, el caso aún declarado AGT-12, el
ranking advisory por intención, la identidad exacta y la respuesta explícita
cuando no hay candidatos. El test Go además rechaza drift y metadata
manipulada contra el índice Git.
