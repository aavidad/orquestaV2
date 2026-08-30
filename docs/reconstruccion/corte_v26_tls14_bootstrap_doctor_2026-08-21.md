# Corte V26 / TLS-14: bootstrap y doctor de disposición inmutable

Fecha: 2026-08-21. Estado: rework local de contrato, sin instalación física,
persistencia, autorización application-side ni acreditación. `TLS-14` y
`AC-V26-TOOLS-SKILLS-SDK` permanecen
`declared`/`planned` en sus autoridades, que este corte no modifica.

## Autoridad y write-set

- capability ID: `TLS-14`;
- invariante: el bootstrap solo admite sujetos exactos ya seleccionados por
  `ID+version+digest`; observar un catálogo no instala ni activa nada;
- autoridad: `ToolingBootstrapDoctor` conserva un snapshot in-process de
  disposición deseada. No escribe Goal, estado durable ni sistemas externos;
- puertos, adaptadores, stores, configuración, procesos y comandos: ninguno;
- write-set: `internal/tooling/bootstrap_doctor.go`, su test,
  `acceptance/v26_bootstrap_doctor_test.go` y este documento;
- dependencias causales: V08, V10, V15 y V21 acreditadas; contratos locales
  TLS-01..TLS-13 presentes pero todavía no acreditados;
- presupuesto: hasta 450 LOC productivas, 500 de tests, 140 de documento,
  menos de 200 MiB persistentes, 300k tokens, 90 minutos y
  cero dependencias nuevas.

## Preflight y hueco legacy

Antes de diseñar o editar se volvió a ejecutar el preflight exigido:

```text
scripts/preflight_reutilizacion_legacy.sh --capability TLS-14 \
  --path internal/tooling/bootstrap_doctor.go \
  --operation reaudit_bootstrap_doctor \
  --task 'Reauditar y corregir un doctor fail-closed ligado a snapshots y digests exactos que no confunda diagnostico o proyeccion con wiring, boot o restart real' \
  --function 'BootstrapDoctor'
```

Terminó con código cero y JSON advisory `characterize`: cero candidatos,
funciones exactas o pistas semánticas. Es un hueco, no permiso para copiar; no
se consultó código legacy.

El read-model `tooling_bootstrap_doctor` sigue en `debt/adoptar`; su gate total
pide manifiesto de versiones/licencias/permisos, smoke y estado seguro. Este
corte aporta la parte de identidad, digest, diagnóstico y defaults; no afirma
licencias ni permisos que los descriptores fuente no demuestren por sí solos.

## Incremento real

`NewToolingBootstrapDoctor` recibe los catálogos inmutables ya construidos y
un manifiesto versionado. El manifest incluye capability `TLS-14`, producto,
proyecto, rol y Goal del contexto de selección, y el constructor exige que
coincidan con un `CuratedContext` validado cuya capability sea exactamente
`TLS-14`; además valida ese contexto contra el catálogo curado antes de mirar
sujetos, por lo que un manifest solo-rulepack no elude capability/scope. Tools, skills y
plugins deben resolverse por el `CuratedCatalog` bajo ese contexto exacto. Los
rulepacks se obtienen mediante `RulePackCatalog.Resolve` con el mismo
`SkillScopeContext`; `List` global no participa y un pack activo de otro scope
se rechaza. Cada sujeto se fija por tipo, ID, versión
entera canónica y digest. Wildcards, `latest`, duplicados, sujetos no curados,
digest divergente y catálogo sustituido fallan atómicamente.

El digest canónico no depende del orden de entrada. Inputs, manifest y snapshot
se clonan para que una mutación del llamador no reescriba la disposición. El
snapshot registra capability TLS-14, versión de bootstrap, digest del manifest,
contexto de selección, ambos catálogos, sujetos exactos y las disposiciones de superficies. Es únicamente
una proyección del deseo aceptado: no contiene path, URL, bytes, download,
autorización, intento, efecto, persistencia, instalación ni activación.

Antes de ordenar o calcular manifest/snapshot se valida UTF-8 estricto en todos
sus campos textuales, incluidos contexto y sujetos. Los helpers de digest
fallan cerrados ante bytes inválidos o error de JSON; no aceptan la sustitución
silenciosa de `encoding/json`.

Broker queda con disposición `disabled_default` salvo `BrokerOptIn=true`, que
solo registra `requested_opt_in`; no arranca broker alguno. UI y puertos
externos solo admiten la disposición `disabled_default`; este write-set no
implementa esas superficies.

`Diagnose` rehasha una proyección completa entregada por el llamador, incluidos
contexto y digests de ambos catálogos. Solo emite `projection_match` o
`projection_drift` bajo scope `caller_supplied_projection`; nunca health,
wiring, boot o restart. Reporta de forma determinista versión/digest divergentes, sujeto
ausente, inesperado, duplicado o con digest distinto, y cualquier violación de
los defaults. UTF-8/digests inválidos fallan cerrados sin eco ambiguo. No
inspecciona filesystem, red ni procesos y por ello no prueba estado físico.

## Verificación y cierre honesto

Quedaron verdes:

```text
go test -mod=vendor -count=1 ./internal/tooling
go test -mod=vendor -count=1 ./acceptance
go test -mod=vendor -race -count=1 ./internal/tooling -run '^TestToolingBootstrapDoctor'
go test -mod=vendor -race -count=1 acceptance/v26_bootstrap_doctor_test.go acceptance/v26_rulepacks_test.go -run '^TestV26TLS14'
go test -mod=vendor -run '^$' ./...
go test -mod=vendor -count=1 . -run '^TestRebuildArchitecture$'
GOFLAGS=-mod=vendor go vet ./internal/tooling ./acceptance
GOFLAGS=-mod=vendor go vet ./internal/... ./cmd/orquesta
git diff --check # más no-index --check para los cuatro ficheros no rastreados
```

El mínimo transversal con tests no quedó verde por fallos ajenos al carril:
locks `codex.account_profile_unavailable` y toolchain de bootstrap inválido en
`internal/adapters/agent/codex` e `internal/bootstrap`. La suite raíz se
interrumpió cooperativamente tras varios minutos sin salida; no quedó proceso
propio. No se parcheó ninguna de esas rutas.

Los tests del carril cubren match de proyección, capability TLS-14 exacta, catálogo
curado, rulepack efectivo bajo producto/proyecto/rol/Goal exactos, rechazo de
pack activo de otro scope y bypass solo-rulepack, orden canónico, broker opt-in,
defaults seguros, aliasing, mutación de inputs/salidas, UTF-8 inválido antes de
manifest/snapshot/proyección, drift exacto, contexto divergente, wildcard/`latest`, duplicados, digest falso,
sujeto no curado y ausencia explícita de campos que aparenten efectos físicos.

```text
hecho: manifest, snapshot y doctor de proyección exacta para tool/skill/plugin/rulepack
invariante restaurado: proyección coincidente no equivale a instalación, wiring, boot o restart
autoridad final: snapshot in-process read-only; application conserva autorización y lifecycle
receipts y revisión acreditada: ninguno; ninguna evidencia de release
código o decisión retirados: ninguno
legacy retirado o bloqueo de retirada: hueco characterize sin candidato; no existe equivalencia que autorice retirada
LOC netas y complejidad: 420 productivas, 496 de tests y 126 de documento; +48/+98/+10 sobre el baseline recibido; cero writers, stores, procesos o loops residentes
riesgos/P0/P1: P0 ninguno observado; P1 faltan writer/puerto durable, autorización, adapter físico, licencias/permisos agregados, smoke/restart/rollback y receipt de efecto
siguiente dependencia causal: composición application-side transaccional y adapter de instalación/staging antes de pretender el gate V26; OPS-15 completa operación física
```

No se atribuye `TLS-14`, V26 ni otra capability como acreditada.
