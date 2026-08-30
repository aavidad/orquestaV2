# Corte V26 / TLS-09: disposición gobernada de revisiones de skill

Fecha: 2026-08-21. Estado: incremento implementado y ejercitado, sin
activación, snapshot de ejecución ni acreditación. `TLS-09` y
`AC-V26-TOOLS-SKILLS-SDK` permanecen en `declared` y `planned`.

## Autoridad y write-set

- capability ID: `TLS-09`; no se atribuyen `TLS-06..08` ni `TLS-10..14`;
- invariante: una revisión solo se resuelve si review aprobado y test PASS
  pertenecen al sujeto exacto; en una cadena `Refresh`, una revocación firmada
  nunca se omite, reescribe ni resucita;
- autoridad: `SkillReleaseCatalog` es un read-model inmutable; application
  conserva autorización, admisión, snapshot de ExecutionRef y lifecycle;
- puertos, adaptadores, stores, configuración y procesos: ninguno;
- write-set: `internal/tooling/skill_governance*`,
  `acceptance/v26_skill_governance_test.go` y este documento;
- dependencias: V08, V10, V15, V20-V21 acreditadas y catálogo TLS-06/TLS-08
  local aún no acreditado;
- presupuesto: 450 LOC productivas, 500 de tests, 140 documentales, cero
  dependencias externas, hasta 300k tokens, 90 minutos y menos de 200 MiB.

Quedan fuera trust-root store/config y rotación explícita, firma privada,
ejecutor de tests, workflow de review, snapshot por ExecutionRef,
admisión/activación, instalación, upgrade/remove, loader CAS, persistencia y
restart, plugins, rulepacks, bindings públicos y E2E.

## Preflight y caracterización

Se ejecutó antes de editar:

```text
scripts/preflight_reutilizacion_legacy.sh --capability TLS-09 \
  --path internal/tooling/skill_governance.go --operation reauditar-corregir \
  --task 'Reauditar y corregir gobernanza del sujeto exacto, trust, revocación monotónica, rollback, fence/replay, copias y límites sin autorizar, cargar, ejecutar ni persistir skills' \
  --function 'NewSkillReleaseClaim/NewSkillReleaseCatalog/Refresh/NewSkillRevocationClaim/ResolveRollback'
```

Resultado advisory: `reimplement` con una lección contractual, sin función
legacy exacta ni verde histórico. Se abrió desde el índice congelado
`docs/informe_ecosistema_orquestacion_2026-03-29.md`, MEJORA-2.

Se conserva versión/hash y snapshot exacto como condición futura. Se descarta
la tabla mutable `session_skills_snapshot`, la skill activa por nombre y toda
presunción de firma, permisos, tests, revocación o rollback: el propio ledger
dice que nunca fueron demostrados. Ninguna de las 19 skills condicionales se
registra, revisa o activa.

## Incremento real

`NewSkillReleaseClaim` produce un sujeto determinista que liga:

- ID, versión y digest completo del registro, que ya cubre instrucciones,
  scopes, tools exactas y unión mínima de permisos;
- ref/digest de review y veredicto máquina `approved`;
- `AttestationRef`, policy digest, sujeto de test derivado y resultado `pass`;
- signer ref y digest del sujeto criptográfico.

`NewSkillReleaseCatalog` verifica Ed25519 contra roots públicas explícitas,
rechaza claims divergentes y duplicados y conserva firma y key digest
auditables. Rechaza UTF-8 ambiguo, controles, digests no canónicos, más de 64
roots o 1024 releases y sustitución de signer/key entre versiones del mismo ID.
Roots no usadas no provocan churn sobre el catálogo. No hay rotación implícita:
un protocolo de rotación futuro necesita contrato y autoridad propios.

La revocación liga releases origen y rollback exactos —registration, review,
test, signer y firma—, reason code y destino inferior ya revisado. Solo el
signer/key que publica ese ID puede revocarlo; refs de revocación no se
reutilizan. `Refresh` acepta únicamente el conjunto completo ya observado,
exige el digest previo, conserva tombstones, rechaza stale, omisión y
reescritura, y hace idempotente el replay exacto. Es una transición pura local:
no serializa ramas ni acredita CAS durable, persistencia, restore o restart.

`ResolveReviewed` y `ResolveRollback` revalidan ambos scopes TLS-08. El match
solo expone metadata: no autoriza permisos, carga bytes ni ejecuta o persiste.

## Verificación

Verde:

```text
go test -mod=vendor -count=1 ./internal/tooling ./acceptance -run '^(TestSkillRegistry|TestSkillScope|TestResolveScoped|TestSkillRelease|TestSkillRevocation|TestV26TLS0[689])'
go test -mod=vendor -race -count=1 ./internal/tooling ./acceptance -run '^(TestSkillRegistry|TestSkillScope|TestResolveScoped|TestSkillRelease|TestSkillRevocation|TestV26TLS0[689])'
go test -mod=vendor -count=1 ./internal/tooling ./acceptance
GOFLAGS=-mod=vendor go vet ./internal/tooling ./acceptance
go test -mod=vendor -count=1 . -run '^(TestRebuildArchitecture|TestProductRoadmapIsExhaustiveAndCausal)$'
go test -mod=vendor -run '^$' ./...
git diff --check
gofmt -d internal/tooling/skill_governance.go internal/tooling/skill_governance_test.go acceptance/v26_skill_governance_test.go
! rg -n '[[:blank:]]+$|\r$' internal/tooling/skill_governance.go internal/tooling/skill_governance_test.go acceptance/v26_skill_governance_test.go docs/reconstruccion/corte_v26_tls09_gobernanza_skills_2026-08-21.md
```

Los negativos cubren review/test inválidos, UTF-8/controles y digest no
canónico; claims, firmas, roots, límites y sustitución de trust; cruce de scope;
rollback ascendente, target ausente/revocado y evidencia trasplantada;
revocador ajeno, revocation ref duplicada, stale, replay y resurrección por
omisión. Las copias y claves de entrada no aliasan el catálogo.

El worktree recibido tenía 85 entradas ajenas y los cuatro targets sin
seguimiento. Los gates amplios compilan código ajeno; sus resultados se
declaran abajo sin atribuir sus cambios a TLS-09.

## Cierre honesto

- hecho: claims exactos y transición local monotónica de revocación/rollback;
- invariante restaurado: nombre o versión no sustituyen digest, firma y tests;
- autoridad final: read-model inmutable, sin writer operativo;
- tests/E2E: unitarios, aceptación parcial, race y arquitectura; sin E2E;
- receipts/revisión acreditada: fixtures sintéticos; no hay candidato V26;
- código/legacy retirado: ninguno;
- LOC: 448 productivas, 466 de tests y 119 documentales; menos de 51 KiB, sin
  loops residentes ni store;
- riesgos: sin P0 contractual conocido. P1: faltan roots/rotación compuestas,
  review/test reales, persistencia/CAS/restart, snapshot por ExecutionRef y
  admisión gobernada;
- siguiente dependencia causal: `TLS-07` debe materializar carga progresiva y
  snapshot exacto antes de instalación o activación.

No se atribuye `TLS-09`, V26 ni otra capability como acreditada.
