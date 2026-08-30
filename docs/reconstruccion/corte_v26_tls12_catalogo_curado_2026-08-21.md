# Corte V26 / TLS-12: catálogo curado exacto

Fecha: 2026-08-21. Estado: incremento implementado y ejercitado, sin
instalación, firma propia de curación ni acreditación. `TLS-12` y
`AC-V26-TOOLS-SKILLS-SDK` permanecen en `declared` y `planned`.

## Autoridad y write-set

- capability ID: `TLS-12`; no se atribuyen `TLS-11`, `TLS-13` ni `TLS-14`;
- invariante: estar registrado nunca implica estar seleccionado, cargado o
  instalable;
- autoridad: `CuratedCatalog` es una allowlist read-only; application conserva
  autorización, aprobación, instalación y lifecycle;
- puertos, adaptadores, stores, configuración y procesos: ninguno;
- write-set: `internal/tooling/curation*`,
  `acceptance/v26_curated_catalog_test.go` y este documento;
- dependencias: TLS-01 y TLS-06..10 locales, aún no acreditadas;
- presupuesto: 400 LOC productivas, 400 de tests, 120 documentales, cero
  dependencias externas y menos de 1 MiB.

Quedan fuera downloader/loader, install/upgrade/disable/remove, trust root de
la propia lista curada, aprobación application-side, persistencia, refresh,
configuración, UI/SDK y E2E.

## Bitácora, preflight y caracterización

La bitácora vigente mantiene V38 prioritaria, pero su siguiente write-set cruza
el repositorio hermano expresamente fuera de alcance. V26 permanece libre y
todo su owner sigue `declared`; `AC-V26-TOOLS-SKILLS-SDK` sigue `planned`.

TLS-12 precede causalmente TLS-11: ninguna operación de instalación debe poder
usar el catálogo completo como fuente implícita.

Se ejecutó antes de editar:

```text
scripts/preflight_reutilizacion_legacy.sh --capability TLS-12 \
  --path internal/tooling/curation.go --operation implementar \
  --task 'Construir una vista curada e inmutable que seleccione por ID/version/digest solo tools, skills revisadas y plugins exactos, sin wildcard, auto-discovery, carga ni instalación implícita' \
  --function 'NewCuratedCatalog/ResolveTool/ResolveSkill/ResolvePlugin'
```

Resultado advisory: `reimplement`, con una idea y ninguna función exacta o
verde histórico. Se abrió desde el índice congelado
`docs/plan_mejora_continua_orquesta_2026-07-04.md`, MEJ-TASK-206.

Se conserva curación, propuesta/review, carga progresiva y scope. Se descartan
goal idle, backlog/auto-commit, coincidencia textual, popularidad como trust y
el store file-based legacy. No se copió código.

## Incremento real

`CuratedCatalogSpec` liga ID/revisión/i18n, review ref/digest, capabilities y
scopes exactos con tres listas tipadas:

- tool por ID/revisión/spec digest TLS-01;
- skill por ID/revisión/registration y release digest TLS-09 no revocado;
- plugin por ID/revisión/plugin digest TLS-10.

No admite wildcard, capability vacía/duplicada, selector implícito, “latest” ni
list-all. La resolución exige `CapabilityRef` y `SkillScopeContext` construidos
explícitamente. Skills deben pasar además su scope y trust; plugins verifican
todas las skills empaquetadas en el mismo contexto o no se resuelven. Tools no
seleccionadas siguen invisibles aunque se añadan al registry, y esa adición no
cambia el digest curado.

Review ref/digest son metadata ligada, no prueba criptográfica autónoma. TLS-11
debe exigir autorización/aprobación verificadas antes de mutar una disposición.

## Verificación

Verde:

```text
go test -mod=vendor -count=1 ./internal/tooling ./acceptance -run '^(TestCurated|TestV26TLS12)'
go test -mod=vendor -race -count=1 ./internal/tooling ./acceptance -run '^(TestCurated|TestV26TLS12)'
go test -mod=vendor -count=1 . -run '^TestRebuildArchitecture$'
GOFLAGS=-mod=vendor go vet ./internal/tooling
git diff --check
```

Los negativos cubren wildcard, capability/scope/contexto divergentes, revisión
malformada, selección vacía, ID/revisión/digest ausentes o forjados, duplicados,
versión no seleccionada, cruce de proyecto, plugin con skill invisible,
registries nil, alias de memoria y ausencia de list-all.

En el mínimo transversal quedaron verdes toda la aceptación, tooling,
application, SQLite, MCP, `cmd/orquesta`, SDK, ambos guards y `vet`. El test
raíz completo conserva únicamente el receipt Codex obsoleto y el script
concurrente no versionado; `internal/...` conserva únicamente los perfiles
Codex no disponibles y el toolchain bootstrap inválido. Son los mismos
bloqueos ambientales previos y este write-set no los tocó.

## Cierre honesto

- hecho: allowlist curada exacta para tools, skills y plugins;
- invariante restaurado: catálogo amplio no equivale a selección;
- autoridad final: read-model, sin instalador ni grant;
- tests/E2E: unitarios, aceptación parcial, race y arquitectura; sin E2E;
- receipts/revisión acreditada: fixtures sintéticos; no hay candidato V26;
- código/legacy retirado: ninguno;
- LOC: 367 productivas y 328 de tests; sin store ni loop;
- riesgos: sin P0. P1: review de curación aún no está firmado/admitido y falta
  toda mutación auditable;
- siguiente dependencia causal: `TLS-11` debe consumir exclusivamente este
  digest curado para install/upgrade/disable/remove.

No se atribuye `TLS-12`, V26 ni otra capability como acreditada.
