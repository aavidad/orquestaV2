# Indice de funciones Orquesta - 2026-07-10

## Proposito

Inventario no destructivo para la limpieza posterior al cierre del nucleo.
No es una lista de borrado ni sustituye la revision de contratos, wiring,
reflexion, CLI, plugins o adaptadores opt-in.

## Generacion reproducible

```bash
scripts/orquesta_auditoria_codigo.sh \
  --root "$PWD" \
  --json-out /tmp/orquesta-function-index-YYYYMMDD/index.json \
  --no-sqlite
```

El informe detallado se conserva como salida aislada de auditoria, no como
artefacto versionado: contiene una fila por funcion y en este corte pesa
aproximadamente 9,3 MB. El generador, su prueba de contrato y este resumen si
quedan versionados.

## Foto de este corte

| Clasificacion | Funciones |
| --- | ---: |
| indexadas | 25.199 |
| `static_candidate_requires_review` | 1.251 |
| `exported_or_contract_requires_review` | 1.847 |
| `test_only` | 10.582 |
| `unclassified_private` | 11.519 |

El analizador declara `go_function_declaration_lexical_v0`: localiza
declaraciones Go y cuenta referencias textuales de identificador. Es una senal
de triage barata; no resuelve llamadas con receptor, interfaces, reflexion,
generacion, ensamblado ni alcance lexico.

## Uso permitido en la limpieza

1. Elegir una familia o modulo de write-set estrecho.
2. Revisar entradas con `rg`, contratos y wiring de composicion.
3. Clasificar cada candidato como conservar, migrar, deprecar o retirar.
4. Solo retirar con prueba focal y descenso verificable del ratchet.
5. Conservar el commit anterior y registrar en el inventario de bugs cualquier
   falso positivo o patron estructural.

La consolidacion amplia de variables de entorno queda deliberadamente despues
del cierre del nucleo. La primera ola ya tiene ratchet en
`cmd/orquesta-server/server_env_registry_ast_v0_test.go`; no se ampliara
mientras F3/F5/208 y la atestacion independiente sigan pendientes.

## Actualizacion 2026-07-10: retirada minima revisada

La primera pasada posterior a la consolidacion de configuracion retiro dos
wrappers privados sin consumidores de produccion: el lector de politica de
progreso usado solo por una prueba y el lector de limites Codex sin llamadas.
La prueba de politica paso a consultar la configuracion viva equivalente. El
indice reproducible baja de 25.223 a 25.221 funciones y de 1.238 a 1.236
candidatas estaticas. No se borro ningun candidato cubierto por contrato,
produccion o pruebas con semantica propia.

Una segunda revision retiro dos wrappers privados adicionales sin referencias:
el lector de politica desde directorio de proyecto y el adaptador de runtime
Codex desde directorio de proyecto. El indice baja a 25.219 funciones y 1.234
candidatas. Siguen siendo necesarias referencias, contratos y pruebas focales
antes de cualquier retirada posterior.

## Revision de modulos huerfanos 2026-07-10

Los siete modulos marcados por el auditor con `importer_count: 0` fueron
revisados uno a uno. El auditor solo mira imports de composicion y no distingue
un adaptador opt-in de codigo muerto. Se conservan, sin excepcion:

| Modulo | Decision |
| --- | --- |
| `orquesta-data-ingestion` | Contrato de ingesta con adaptadores pendientes de integrar. |
| `orquesta-document-extraction-csv` | Adaptador de exportacion documentado y probado. |
| `orquesta-document-extraction-json` | Adaptador de exportacion documentado y probado. |
| `orquesta-document-extraction-tool-capability` | Adaptador de capacidad pendiente de composicion. |
| `orquesta-domain-work-sql` | Adaptador SQL de referencia deliberadamente sin wiring productivo. |
| `orquesta-presentation-extraction` | Contrato PPTX/ODP con adaptadores reales pendientes. |
| `orquesta-tool-capability-file` | Adaptador durable de referencia, pendiente de wiring. |

Ninguno es candidato a retirada: todos tienen contrato, pruebas y/o tarea de
integracion vigente. Antes de declarar huerfano un modulo se deben consultar
sus tests, README y tarea de composicion, no solo el conteo de imports.

## Refinamiento del indice 2026-07-10

El generador conserva `deadcode_candidates` como señal bruta, pero separa las
funciones privadas marcadas por esa señal en referencias de produccion, solo de
tests o ninguna referencia lexica, excluyendo su propia declaracion. La foto
actual queda en 731 candidatas con referencias de produccion, 41 con referencias
solo de tests y 53 sin referencias lexicas; las 409 exportadas conservan la
clasificacion anterior por contrato potencial. Los nuevos campos JSON/SQLite
son aditivos. Ninguna de estas categorias autoriza borrar: sirven para priorizar
la siguiente revision manual y evitar falsos positivos por tests.

La primera aplicacion de la nueva categoria sin referencias retiro dos wrappers
privados mas: el antiguo lector de allowlist del runner guardian y el acceso
directo al broker de contexto, sustituido hace tiempo por su wiring. Tras
verificar focales, el indice queda en 25.217 funciones y 1.232 candidatas
brutas. No se alteraron contratos ni composicion activa.

## Actualizacion 2026-07-11: base de limpieza estabilizada

Tras consolidar la configuracion OPES y retirar cinco wrappers privados
adicionales de configuracion/runtime/state-file, el auditor reproducible
informa 25.322 funciones indexadas y 1.227 candidatas brutas. La variacion del
total indexado no se interpreta como codigo nuevo o muerto por si sola: el
analizador es lexical y cambia al mover helpers. La prioridad sigue siendo la
categoria privada sin referencias, confirmada con `rg`, contratos y prueba
focal por paquete.

El ratchet de entorno de `cmd/orquesta-server` se cerro en cero antes de seguir
con conectores. Las siguientes extracciones solo se aceptaran por cohesion de
responsabilidades y con regresion focal; no se dividiran archivos solo para
reducir lineas.

## Actualizacion 2026-07-11: wrappers de runtime retirados

La revision manual de adaptadores encontro cinco wrappers privados sin callers:
dos en el paquete goal Codex, uno que era el unico contenido de su fichero de
perfil Codex y uno por cada protocolo de resultado durable Claude/Gemini. Las
rutas activas ya usaban sus builders/localizadores actuales. Se retiraron sin
modificar prompts, perfiles, contratos ni configuracion.

La comprobacion por `rg` no encontro consumidores de codigo y los cuatro
paquetes `orquesta-runtime-codex-goal`, `orquesta-runtime-codex`,
`orquesta-runtime-claude` y `orquesta-runtime-gemini` pasaron `go test` de
forma focal. Este cierre reduce 264 lineas; no autoriza retirada automatica de
otros candidatos.

## Actualizacion 2026-07-11: primer corte de nucleo puro

La auditoria de `orquesta-core-workflow`, `orquesta-goal` y
`orquesta-orchestration-core` encontro dos bloques privados consumidos solo
por tests. `8720dbce6` los mueve a `sensitive_detail_rails_v0_test.go` y los
retira de los archivos de produccion, sin modificar rails, contratos ni
mensajes. La suite completa de `orquesta-core-workflow` queda verde.

La revision separada de estado-vivo, run-control, run-queue y
autoprogramming confirma que `DerivarVeredictoCausalV0` sigue siendo la unica
reconciliacion de estado persistido, terminalidad durable y liveness. Las
proyecciones de fase, cola, control y review son politicas distintas; no se
retiran ni se consolidan por parecido textual.
