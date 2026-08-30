# Corte V26 / TLS-12: fundación pura del catálogo curado

Fecha: 2026-08-21. Estado: incremento local implementado y ejercitado, sin
wiring, receipt ni acreditación. `TLS-12` permanece `declared` y
`AC-V26-TOOLS-SKILLS-SDK` permanece `planned`.

## Autoridad, frontera y presupuesto

- capability: `TLS-12`; no se atribuyen TLS-09, TLS-11, TLS-13 ni TLS-14;
- invariante: presencia en un catálogo amplio nunca equivale a selección,
  carga, autorización, instalación ni ejecución;
- autoridad: `CuratedCatalog` es un read-model inmutable de selección y
  visibilidad; no escribe lifecycle ni disposición;
- dependencias canónicas del roadmap: `credentials`,
  `identity_projects_rbac`, `budgets_effects` y `command_registry`;
- insumos locales: snapshots TLS-01, TLS-09 y TLS-10 todavía no acreditados;
  no se inventa una arista nueva en el roadmap;
- puertos, adaptadores, stores, configuración, procesos y efectos: ninguno;
- write-set exacto: `internal/tooling/curation.go`,
  `internal/tooling/curation_test.go`,
  `acceptance/v26_curated_catalog_test.go` y este documento;
- presupuesto previo: P<=450, V<=500, documento<=140 líneas, <=300k tokens,
  <=90 min y <200 MiB.

Tres revisiones bootstrap, todas en solo lectura, cubrieron contrato/legacy,
adversarial/seguridad y presupuesto/gates. El padre fue el único writer. No se
tocaron roadmap, evidence, application, ports, adapters, bootstrap, rulepacks,
tool execution, operaciones de plugins, legacy ni `agente_microvm`.

## Preflight y decisión

Antes de inspeccionar o editar se ejecutó:

```text
scripts/preflight_reutilizacion_legacy.sh --capability TLS-12 \
  --path internal/tooling/curation.go --operation reauditar-corregir \
  --task 'Reauditar y corregir una vista curada pura e inmutable que selecciona exactamente catálogo snapshot contexto scope digest y review; falla cerrada ante límites duplicados y refs; sin autorización ejecución instalación persistencia ni wiring' \
  --function 'NewCuratedCatalog/ResolveTool/ResolveSkill/ResolvePlugin'
```

Resultado advisory: `reimplement`; una pista léxica, ninguna función exacta ni
verde histórico y ningún permiso de copia. Se conservan curación, review
referenciada, carga progresiva y scope. Se descartan goal idle, backlog,
autocommit, popularidad como trust y cualquier store o runtime legacy.

## Fundación cerrada

`CuratedCatalogSpec` liga ID y revisión exactos, clave i18n, review ref/digest,
capabilities, scopes y selecciones tipadas:

- tool: ID, revisión y spec digest TLS-01;
- skill: ID, revisión, registration digest y release digest TLS-09 no revocado;
- plugin: ID, revisión y descriptor digest TLS-10, con cierre de tools y skills
  cotejado contra los mismos snapshots aportados a la curación.

El digest del catálogo usa dominio `orquesta.tooling.curated-catalog.v1` y el
spec canónico. Ref de review, capabilities y refs de scope son UTF-8, sin
controles ni wildcard; los SHA-256 son minúsculos canónicos. Review ref/digest
son metadata ligada al snapshot, no firma, aprobación ni receipt de review.

Capabilities, scopes y selecciones se copian, ordenan y rechazan duplicados.
Los límites son 64 capabilities, 32 scopes, 128 selecciones por clase y 32 KiB
serializados. `Registration` devuelve copias defensivas de todas las listas y
no existe list-all.

`CuratedContext` solo nace de capability y scope tipados explícitos. El catálogo
aplica su capability/scope y, para plugins, también la metadata efectiva de
capability/scope del descriptor. Cada resolución vuelve a cotejar el digest
seleccionado; drift, mezcla de snapshots, componente ausente, skill revocada o
scope divergente fallan cerrados.

El digest curado no incluye entradas amplias no seleccionadas: añadir una tool
ajena no cambia el snapshot curado. El receiver elige el snapshot; ligar su
digest a una admisión o comando pertenece al consumidor y no se atribuye aquí.

TLS-12 no ofrece refresh ni revoke mutables. Consume la disposición inmutable
de TLS-09 y rechaza una revocación presente en ese snapshot. Una disposición
nueva exige recomponer y revisar otro catálogo; no hay watcher, store o writer
oculto.

## Separación negativa

El catálogo no autoriza ni aprueba principals, no reserva presupuesto, no
ejecuta connectors, no carga bodies, no instala/actualiza/deshabilita/elimina,
no persiste, no emite receipts y no decide lifecycle. El write-set ajeno de
ejecución gobernada no forma parte de este corte y no aporta cifras, causalidad
ni evidencia a TLS-12.

## Verificación de esta revisión

Verde:

```text
go test -mod=vendor -count=1 ./internal/tooling ./acceptance -run '^(TestCurated|TestV26TLS12)'
go test -mod=vendor -race -count=1 ./internal/tooling ./acceptance -run '^(TestCurated|TestV26TLS12)'
go test -mod=vendor -count=1 ./internal/tooling ./acceptance
GOFLAGS=-mod=vendor go vet ./internal/tooling ./acceptance
go test -mod=vendor -count=1 . -run '^TestRebuildArchitecture$'
go test -mod=vendor -run '^$' ./...
GOFLAGS=-mod=vendor go vet ./internal/... ./cmd/orquesta
git diff --check
git diff --no-index --check /dev/null <cada fichero untracked del write-set>
```

Los negativos cubren límites y tamaño, duplicados, wildcard, controles y UTF-8
inválido, SHA no canónico, refs/digests divergentes, aliasing, scope/capability,
drift, mezcla de snapshots, cierre plugin, revocación y ausencia de métodos de
autorización, ejecución, instalación, persistencia, refresh o revoke.

## Cierre honesto

- hecho: allowlist exacta, acotada, inmutable y fail-closed;
- autoridad final: read-model local; ningún writer nuevo;
- tests: unitarios, aceptación parcial, race, arquitectura, compile y vet; sin
  E2E público;
- receipts/revisión acreditada: ninguno; las refs sintéticas no prueban review;
- código/legacy retirado: ninguno; no existe permiso para copiar legacy;
- LOC brutas porque los cuatro ficheros siguen untracked: P=444, V=500 y este
  documento <=140 líneas; no son LOC netas contra un baseline Git;
- riesgos: P0=0, P1=0 locales; permanece la integración/acreditación V26;
- siguiente consumidor causal: TLS-11 puede usar únicamente este digest curado
  bajo su propia autorización y contrato, sin convertirlo en dependencia
  canónica nueva.

No se atribuye `TLS-12`, V26 ni otra capability como acreditada.
