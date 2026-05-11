# orquesta-cli

Responsabilidad: cliente secundario para diagnostico, recuperacion y automatizacion.

Incluye:

- comandos de inspeccion;
- operaciones de emergencia;
- compatibilidad;
- smoke local.

CLI no es la fuente de verdad ni el camino primario de autonomia.

## Corte inicial

`orquesta-cli` arranca como adaptador fino sobre contratos publicos de V2. El
primer contrato compartido autorizado para la CLI es `SolicitarNuevaApp v0`,
propiedad de `orquesta-factory`, expuesto inicialmente por REST como
`POST /api/v0/apps/spec`.

La CLI no decide negocio, no persiste estado, no arranca runtime, no selecciona
modelos, no asigna agentes y no interpreta tablas. Si un comando necesita esas
capacidades, debe consumir un puerto publico versionado o quedar bloqueado con
`CONSULTA AL DIRECTOR`.

## Reglas de diseno

- Server-first: los comandos operativos hablan con API/servicios publicos.
- Automatizable: todo comando debe soportar salida estable, preferentemente
  `--json`, y codigos de salida deterministas.
- Contratos primero: opciones, DTOs, errores y envelopes salen de contratos
  compartidos o de contratos locales de adaptador.
- i18n por defecto: textos visibles y errores publicos se enlazan a claves
  localizables cuando exista implementacion.
- Sin fallback local: no hay lectura/escritura directa de DB ni filesystem como
  sustituto del servidor. Las herramientas de rescate futuras requieren contrato
  aislado.

## Alcance v0 propuesto

- `app spec solicitar`: cliente fino de `SolicitarNuevaApp v0`.
- `contratos funcion listar|ver`: cliente read-only de `FunctionContract v0`
  sobre rutas REST candidatas de `orquesta-core`; `registrar` sigue bloqueado.
- `gobernanza catalogo listar|ver`: lectura secundaria de
  `GovernanceCatalog v0` sobre ruta REST compacta asumida.
- `doctor contratos`: smoke local de configuracion y disponibilidad de
  contratos, sin inspeccionar internals.
- `compat v1 inventario`: documentacion de opciones reutilizables, sin copiar
  codigo de V1 ni reactivar comandos locales.

## Inventario V1 reutilizable

La revision de V1 se usa solo como inventario de UX y automatizacion:

- familias V1 reutilizables ya respaldadas por contrato V2:
  - `app spec solicitar --input --json --correlation-id --timeout`
    sobre `SolicitarNuevaApp v0`;
  - `doctor contratos --json --correlation-id --timeout`
    sobre `OperationalStatusQuery v0`.
  - `contratos funcion listar|ver` sobre transporte REST candidato
    read-only de `FunctionContract v0`;
  - `gobernanza catalogo listar|ver` sobre transporte REST compacto
    read-only de `GovernanceCatalog v0`.
- familias V1 traducibles pero todavia no ejecutables:
  - futuros comandos read-only cuando exista contrato publico compartido.
- flags V1 reutilizables solo como patron de UX si el contrato consumido los
  soporta: `--json`, `--tsv`, `--limit`, `--estado`, `--proyecto`,
  `--agente`, `--desde`, `--dry-run`, `--input`, `--correlation-id`,
  `--timeout`.
- superficies en cuarentena para V2: `runtime *`, `agente *`, `tarea *`,
  `repo *`, `pool *`, `modelo *`, `worktree *`, `deploy *`,
  `persistencia *`, `respaldo *`, `server *`, `serve *`, `microprogramacion *`
  y cualquier comando V1 que lanzaba procesos, escribia DB, manipulaba
  worktrees, purgaba runtime, sembraba modelos o hacia fallback local.
- criterio de reutilizacion: conservar nombres y flags solo cuando no
  contradigan contratos V2 ni oculten decisiones de negocio en la CLI.

## Bloqueos transversales

CONSULTA AL DIRECTOR: `orquesta-cli` necesita confirmar si V2 expondra un
puerto publico de diagnostico/progreso compacto para cubrir la responsabilidad
de diagnostico y recuperacion sin acceder a internals de `orquesta-core`,
`orquesta-runtime`, `orquesta-observability` o `orquesta-persistence`.
