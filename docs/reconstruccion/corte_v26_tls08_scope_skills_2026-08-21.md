# Corte V26 / TLS-08: scope explícito de skills

Fecha: 2026-08-21. Estado: fundación local corregida y ejercitada, sin
admisión, activación, autorización ni acreditación. `TLS-08` sigue `declared` y
`AC-V26-TOOLS-SKILLS-SDK` sigue `planned` en el roadmap.

## Autoridad, alcance y presupuesto

- capability: `TLS-08`; no se atribuyen `TLS-06`, `TLS-07` ni `TLS-09`;
- invariante: una revisión declara scopes canónicos acotados y solo se expone
  por ID, versión y contexto exactos; visibilidad nunca concede permisos;
- autoridad local: `SkillRegistry`, read-model puro; application conserva
  actor, membership, autorización, admisión, lifecycle y efectos;
- puertos, adapters, stores, configuración, procesos y wiring: ninguno;
- write-set exacto: `internal/tooling/skill_scope.go`, su test,
  `acceptance/v26_skill_scope_test.go` y este documento;
- presupuesto: P<=450, V<=500, doc<=140 líneas, <=300k tokens, <=90 minutos y
  <200 MiB; sin dependencias externas.

Quedan fuera registry/resources/governance, plugins, curation, rulepacks,
application, architecture, persistencia, membership, trust, test attestations,
revocación, rollback, instalación, loader CAS, bindings y E2E.

## Preflight y revisión independiente

Antes de abrir la implementación se ejecutó:

```text
scripts/preflight_reutilizacion_legacy.sh --capability TLS-08 \
  --path internal/tooling/skill_scope.go --operation reauditar_corregir \
  --task 'Reauditar y corregir el resolvedor puro de scope de skills para global, product, project, role y Goal exactos; refs opacas, precedencia no ambigua, actor y contexto explícitos, deny-by-default, copias, límites y replay; sin autorización, membership, persistencia ni wiring' \
  --function 'ResolveScopedSkills'
```

Resultado advisory: `characterize`, sin candidato semántico, función exacta,
pista o fuente accionable. No se copió código ni se tocó el runtime legacy.

Tres subagentes de solo lectura revisaron contrato/legacy, seguridad adversarial
y presupuesto/gates. Coincidieron en cero P0 y detectaron límites ausentes,
UTF-8 no fiel, cobertura incompleta, falta de replay directo y la frontera de
actor fuera del contrato actual.

## Contrato corregido

- scopes exactos: `global`, `product`, `project`, `role` y `goal`;
- `global` es exclusivo; los scopes no globales forman una unión explícita sin
  override ni precedencia de autorización;
- el orden canónico de serialización es global/product/project/role/goal y no
  depende del orden de entrada;
- máximo 32 scopes y 200 bytes por selector; todos exigen UTF-8 válido, trim
  exacto y cero caracteres de control;
- project/Goal se revalidan como refs opacas; Goal lleva project conjuntamente;
- role y Goal de contexto exigen project explícito y rol canónico;
- el zero value del contexto falla cerrado; un contexto sin coordenadas creado
  explícitamente solo puede ver `global`, que sigue siendo visibilidad, no una
  sesión anónima autorizada;
- ID, versión, producto, proyecto, rol y Goal comparan exactamente: no hay
  normalización, wildcard, latest ni fallback;
- registro, scopes y permisos permanecen inmutables ante mutación del resultado;
  repetir la misma consulta devuelve la misma copia y digest.

La validación temprana de UTF-8 elimina dos sujetos distintos que
`encoding/json` habría proyectado al mismo U+FFFD y, por tanto, evita un replay
no fiel antes de publicar el descriptor.

## Frontera de actor y P1

`SkillScopeContext` no transporta `ActorRef`: el actor autenticado y su
membership pertenecen a application y un scope no puede probarlos. Exigir actor
y project también para `global`/`product` obliga a cambiar consumidores actuales
en curation, governance, rulepacks y application, todos fuera del write-set.
No se añadió sentinel, membership local, sobrecarga permisiva ni wiring ficticio.

Permanece un P1 causal: abrir un write-set de integración para hacer explícito
el actor en la frontera consumidora y retirar la ruta antigua de forma atómica.
Hasta entonces este corte acredita solo filtrado descriptor, nunca autorización
ni admisión de una skill.

La validación temprana expone otro P1 de integración: un test ajeno de TLS-09
espera aceptar `product:\xff` en el registry y rechazarlo tarde en governance.
No se debilitó TLS-08 ni se editó governance; ese test debe migrarse en su propio
write-set para aceptar el rechazo temprano.

## Gates ejecutados

Verdes:

```text
focal TLS-08 normal
focal TLS-08 -race
focal TLS-08 -count=20
go test -mod=vendor -count=1 ./acceptance
go test -mod=vendor -count=1 . -run '^TestRebuildArchitecture$'
go test -mod=vendor -run '^$' ./...
GOFLAGS=-mod=vendor go vet ./internal/... ./cmd/orquesta ./acceptance
```

El transversal `go test ./internal/... ./cmd/orquesta` pasó todos los paquetes
salvo `internal/tooling`, únicamente por el test TLS-09 descrito arriba. El test
raíz completo conserva rojos ajenos por anclas de `ports.go`, receipts Codex
obsoletos y el script no versionado `continuar_orquestav2_home_codex.sh`; no se
tocaron ni se atribuyen a TLS-08. `git diff --check`, check de untracked y
formato quedaron limpios; el censo final no encontró procesos propios vivos.

## Cierre honesto

- hecho: scopes exactos, acotados, deterministas, fail-closed y con replay/copia;
- autoridad final: descriptor read-only; application sigue siendo autoridad;
- tests: unitarios, aceptación, race, repetición, arquitectura, compile y vet;
  sin E2E, receipt ni candidato sellado;
- código/legacy retirado: ninguno; preflight sin candidato reutilizable;
- LOC físicas: P=216, V=330 y doc=118; delta de sesión +50/+89/+9 y 29.401
  bytes en el write-set, dentro de todos los límites;
- riesgos: P0=0; P1=actor consumidor explícito y migración del negativo TLS-09;
- siguiente dependencia causal: integrar actor/autorización application-side y
  después resolver governance TLS-09 sin convertir scope en membership.

No se atribuye `TLS-08`, V26 ni otra capability como acreditada.
