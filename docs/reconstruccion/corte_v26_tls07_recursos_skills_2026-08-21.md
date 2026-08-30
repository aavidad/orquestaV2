# Corte V26 / TLS-07: recursos progresivos de skill

Fecha: 2026-08-21. Estado: incremento implementado y ejercitado, sin loader
CAS real, snapshot de ejecución ni acreditación. `TLS-07` y
`AC-V26-TOOLS-SKILLS-SDK` permanecen en `declared` y `planned`.

## Autoridad y write-set

- capability ID: `TLS-07`; no se atribuyen `TLS-06`, `TLS-08` ni `TLS-09`;
- invariante: catálogo y prompt contienen metadata/ref/hash, nunca bytes de un
  recurso de skill no solicitado;
- autoridad: `SkillRegistry` descriptor read-only; application conserva
  admisión, autorización, snapshot y lifecycle;
- puertos, adaptadores, stores, configuración y procesos: ninguno;
- write-set: `internal/tooling/skill_resources.go`,
  `internal/tooling/skill_resources_test.go`,
  `acceptance/v26_skill_resources_test.go` y este documento;
- dependencias: catálogo TLS-06 y scopes TLS-08 locales, aún no acreditados;
- presupuesto: `P<=450`, `V<=500`, documentación `<=140`, cero dependencias
  externas, menos de 200 MiB de disco, 300k tokens y 90 minutos.

Quedan fuera reader/filesystem/CAS, resources MCP generales, streaming, red,
snapshot por ExecutionRef, activación, instalación, plugins, rulepacks,
bindings públicos y E2E.

## Preflight y decisión

Se ejecutó antes de editar:

```text
scripts/preflight_reutilizacion_legacy.sh --capability TLS-07 \
  --path internal/tooling/skill_resources.go --operation implementar \
  --task 'Completar carga progresiva de skills con metadata de recursos inmutable y requests exactas ref/digest/tamaño, validando bytes bajo demanda sin filesystem, loader concreto ni activación' \
  --function 'ListResources/NewResourceLoadRequest/ValidateLoadedResource'
```

La reauditoría repitió el preflight con `LoadSkillResource` y obtuvo
`characterize`, sin candidato semántico, ficha, pista de ledger ni función
estructural. El hueco queda registrado; no se copió código legacy ni se
materializó el árbol congelado.

Se reutilizaron únicamente invariantes del protocolo TLS-03 ya local: identidad
versionada, media type canónico, límites y copia defensiva. Un resource MCP
paginado y un blob CAS perteneciente a una skill conservan tipos distintos.

## Incremento real

`SkillSpec.Resources` forma parte del digest firmado por TLS-09 y conserva por
recurso ID local namespaced, media type, `artifact:sha256`, digest y tamaño.
Durante construcción se valida el set completo de bytes y después se descarta:

- orden determinista y rechazo de IDs/specs/contents duplicados;
- correspondencia uno-a-uno sin metadata o contenido huérfanos;
- media type minúsculo sin parámetros;
- ref/digest/tamaño exactos y máximo de 4 MiB por recurso;
- máximo de 1024 recursos por skill y páginas de metadata de 32 por defecto y
  128 como máximo;
- cero body en `SkillRegistration` o JSON de `SkillCandidate`.

`ListResources` entrega solo metadata. `NewResourceLoadRequest` exige skill,
versión y resource ID exactos y liga el request al registration digest y al
digest del conjunto canónico de scopes. `NewResourcePageRequest` y
`ListResourcePage` entregan páginas deterministas con cursor derivado del
sujeto exacto; un cursor, digest o scope obsoleto no sirve tras freshness.
`ValidateLoadedResource` vuelve a comparar todos los campos, tamaño y hash, y
devuelve copia. No existe búsqueda latest, ruta local ni inferencia de scope.
El digest de scopes acredita identidad declarativa, pero no visibilidad,
autorización ni permiso de ejecución.

Metadata e instrucciones progresivas ya estaban ejercitadas por TLS-06; este
corte completa el tercer nivel contractual, pero sin adapter de carga no se
afirma cierre de TLS-07.

## Verificación

Verde:

```text
go test -mod=vendor -count=1 ./internal/tooling ./acceptance -run '^(TestSkillResource|TestSkillRegistryRejectsMalformedOrDivergentResources|TestV26TLS07)'
go test -mod=vendor -race -count=1 ./internal/tooling ./acceptance -run '^(TestSkillResource|TestSkillRegistryRejectsMalformedOrDivergentResources|TestV26TLS07)'
go test -mod=vendor -count=1 . -run '^TestRebuildArchitecture$'
GOFLAGS=-mod=vendor go vet ./internal/... ./cmd/orquesta
go test -mod=vendor -run '^$' ./...
git diff --check
```

Los negativos cubren metadata/contenido ausente, extra o duplicado; ID, media
type, ref, hash, tamaño, body y máximo divergentes; request forjado en cada
campo; skill/resource/revisión inexistente; traversal, límite de cantidad y de
página, cursor cruzado, scope/freshness obsoletos, replay, mutación y alias.

Los focales TLS-07 normal/race, arquitectura, compilación global y `vet`
quedaron verdes. La suite transversal `./internal/... ./cmd/orquesta` conserva
rojos ajenos por perfiles Codex no disponibles, toolchain bootstrap y
governance (`tooling.skill_spec_invalid:scopes`). La raíz conserva además
anclas/receipts obsoletos y el script no rastreado ya presente. Quedan fuera
del write-set; no se ocultaron ni tocaron.

## Cierre honesto

- hecho: progresión metadata→instrucciones→recursos, paginación y bytes CAS
  validados bajo demanda por sujeto exacto;
- invariante restaurado: ningún body de resource entra en catálogo/listado;
- autoridad final: catálogo descriptor, no loader ni activador;
- tests/E2E: unitarios, aceptación parcial, race y arquitectura; sin E2E;
- receipts/revisión acreditada: fixtures sintéticos; no hay candidato V26;
- código/legacy retirado: ninguno;
- LOC del write-set: `P=229`, `V=407`, documentación `=115`; delta de sesión
  `+270/-19`; sin store ni loop residente;
- riesgos: sin P0. P1: faltan loader real, autorización y snapshot ExecutionRef;
  el rojo governance ajeno impide afirmar suite tooling completa;
- siguiente dependencia causal para elevar TLS-07: reader CAS por puerto y
  admisión/snapshot de ejecución en application, en write-set independiente.

No se atribuye `TLS-07`, V26 ni otra capability como acreditada.
