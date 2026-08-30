# Corte V26 / TLS-06: registro inmutable de SKILL.md

Fecha: 2026-08-21. Estado: incremento implementado y ejercitado, sin loader
real, activación ni acreditación. `TLS-06` y `AC-V26-TOOLS-SKILLS-SDK`
permanecen en `declared` y `planned`.

## Autoridad y write-set

- capability ID: `TLS-06`; no se atribuyen `TLS-07..12`;
- invariante: el catálogo contiene metadata/ref/hash, nunca el cuerpo mutable de
  una skill ni estado operativo `active`;
- autoridad: `SkillRegistry` read-only; application aún no admite ni activa;
- puertos/adaptadores/stores/procesos: ninguno;
- write-set: `internal/tooling/skills*`,
  `acceptance/v26_skill_registry_test.go` y este documento;
- dependencias: V08, V10, V15, V20-V21 acreditadas y TLS-01 local no
  acreditado;
- presupuesto: 500 LOC productivas, 450 de tests, 120 documentales, cero
  dependencias externas y menos de 1 MiB.

Quedan fuera filesystem/CAS adapter, scope, trust/firma, test attestation,
activación, revocación, rollback, install/upgrade/remove, catálogo curado,
skill creator, SDK/bindings y E2E.

## Read-model, preflight y caracterización

Se consultaron `product/knowledge/tooling_adoption_v1.json` y el inventario
transversal. Ambos mantienen el skill registry como `candidate/adoptar`; el
siguiente gate exige versión, scope, hash, trust, tests, revocación y rollback.

Se ejecutó antes de editar:

```text
scripts/preflight_reutilizacion_legacy.sh --capability TLS-06 \
  --path internal/tooling/skills.go --operation implementar \
  --task 'Registrar SKILL.md versionadas como metadata inmutable con ref/digest, requisitos exactos de tools y carga validada bajo demanda, sin filesystem, activación, trust implícito ni lifecycle paralelo' \
  --function 'NewSkillRegistry/Lookup/List/NewLoadRequest/ValidateLoadedInstructions'
```

Resultado advisory: `characterize`, con 19 fuentes skill `conditional` y
pendientes de scope/trust/activation test. Se abrieron desde el índice Git
congelado `docs/op_121_catalogo_skills_fase1.md`,
`orquesta-mcp-operativo/SKILL.md` y `orquesta-director-agentes/SKILL.md`.

Valor conservado: metadata mínima, orden/versionado, anti-duplicado y bodies
SKILL.md compactos. Se descartan DB/estado `activa`, roles y aliases libres,
equivalencia heurística, herramientas sin versión/hash y CLI/API propias. Las
19 skills históricas siguen condicionales; ninguna se registra o activa. No se
copió código legacy.

## Incremento real

`NewSkillRegistry` valida todos los candidatos antes de publicar estado y:

- exige ID namespaced, revisión numérica y formato `skill_markdown_v1`;
- exige frontmatter mínimo exacto `name`/`description` y heading Markdown;
- limita el cuerpo a 256 KiB, UTF-8 sin CR/controles, y liga bytes a
  `artifact:sha256:<digest>`;
- valida cada tool requerida por ID, revisión y spec digest contra TLS-01;
- exige que permisos declarados sean exactamente la unión mínima de esas
  tools, sin extras ni implícitos;
- ordena skills/requisitos/permisos, rechaza duplicados exactos y produce
  digests deterministas;
- no liga el catálogo a tools no requeridas, evitando churn incidental;
- devuelve copias defensivas y nunca serializa `SkillCandidate.Instructions`.

`NewLoadRequest` publica solo sujeto CAS exacto y límite. La validación de bytes
cargados repite estructura/hash y devuelve copia. No existe reader: este
contrato prepara carga progresiva pero no acredita `TLS-07`.

## Verificación

Verde:

```text
go test -mod=vendor -count=1 ./internal/tooling ./acceptance -run '^(TestSkillRegistry|TestV26TLS06)'
go test -mod=vendor -race -count=1 ./internal/tooling ./acceptance -run '^(TestSkillRegistry|TestV26TLS06)'
go test -mod=vendor -count=1 ./internal/tooling ./internal/application ./sdk/... ./acceptance -run '^(TestRegistry|TestSurfaceCatalog|TestResource|TestToolDiscovery|TestSkillRegistry|TestInvokeTool|TestPublicCatalog|TestPublicDiscovery|TestDecodeSpec|TestV26TLS0|TestV26ToolExecution)'
go test -mod=vendor -count=1 . -run '^(TestRebuildArchitecture|TestProductRoadmapIsExhaustiveAndCausal)$'
GOFLAGS=-mod=vendor go vet ./internal/tooling
git diff --check
```

Los negativos cubren identidad/revisión/formato/frontmatter/ref/hash/tamaño,
tool ausente o divergente, duplicados, permiso faltante/extra, cuerpo mutado,
load request forjado, aliases de memoria y fuga por JSON.

El mínimo transversal dejó verdes tooling, application, SQLite, MCP,
`cmd/orquesta`, SDK y `vet`. Persisten únicamente receipt Codex obsoleto,
script concurrente no versionado, perfiles Codex no disponibles y toolchain
bootstrap inválido; no se tocaron ni regeneraron.

## Cierre honesto

- hecho: registro descriptor versionado y verificación progresiva de SKILL.md;
- invariante restaurado: nombre mutable o body local no sustituyen ref/hash;
- autoridad final: catálogo read-only sin activación;
- tests/E2E: unitarios, aceptación parcial, race y arquitectura verdes; sin E2E;
- receipts/revisión acreditada: ninguno; no existe candidato V26;
- código/legacy retirado: ninguno;
- LOC: 303 productivas y 309 de tests;
- riesgos: sin P0. P1: faltan scope, trust/tests, revocación/rollback y loader;
- siguiente dependencia causal: `TLS-08` debe fijar scope exacto antes de que
  TLS-09 pueda declarar una revisión usable o revocada.

No se atribuye `TLS-06`, V26 ni otra capability como acreditada.
