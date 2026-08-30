# Corte V26 / TLS-10: descriptores inmutables de plugins

Fecha: 2026-08-21. Estado: fundación descriptor local implementada y
ejercitada, sin wiring ni acreditación. `TLS-10` sigue `declared` y
`AC-V26-TOOLS-SKILLS-SDK` sigue `planned` en el roadmap.

## Autoridad, contrato y presupuesto

- capability: `TLS-10`; no se atribuyen `TLS-11..14`, `EXT-15` ni V28;
- invariante: un plugin liga metadata y componentes exactos, pero no instala,
  carga, ejecuta, autoriza, persiste, concede permisos ni decide lifecycle;
- autoridad: `PluginCatalog` es un descriptor read-only compuesto en memoria;
  no prueba que application, un marketplace o un instalador lo consuman;
- dependencias base: registros locales TLS-01 y TLS-06, todavía no
  acreditados. Una composición posterior puede aportar una release TLS-09,
  pero TLS-10 no exige ni atribuye firma/trust de plugin o publisher;
- puertos, adaptadores, stores, config global, procesos y efectos: ninguno;
- write-set exacto: `internal/tooling/plugins.go`,
  `internal/tooling/plugins_test.go`, `acceptance/v26_plugins_test.go` y este
  documento;
- presupuesto previo: producto <=450 LOC, verificación <=500, documento <=140,
  <=300k tokens, <=90 min y <200 MiB persistentes.

Firma de plugin, trust de publisher, paquete físico, downloader, marketplace,
install/upgrade/disable/remove, secrets, endpoints, handlers, procesos,
catálogo curado, UI/SDK y E2E quedan fuera. TLS-11 conserva las operaciones;
TLS-12 conserva selección y scope efectivo; application conserva autorización.

## Preflight y caracterización

Antes de editar se ejecutó:

```text
scripts/preflight_reutilizacion_legacy.sh --capability TLS-10 \
  --path internal/tooling/plugins.go --operation reauditar-corregir \
  --task 'Reauditar y corregir el catalogo inmutable de descriptores de plugins: version, digest, scope, capabilities, config, skills y tools exactos; duplicados, refs, limites y copias; sin instalacion, carga, ejecucion, autorizacion, persistencia ni efectos' \
  --function 'NewPluginCatalog/Lookup/List'
```

Resultado advisory: `characterize`, sin candidato, lección, pista, fuente ni
función legacy. No procedía abrir la copia legacy y no se copió código.
`plugins_rulepacks` continúa `candidate/adoptar` en el read-model; su gate
completo no pertenece a este corte descriptor.

## Fundación corregida

El formato TLS-10 es `plugin_descriptor_v1`; el digest queda separado por
`orquesta.tooling.plugin-descriptor.v1`. Con la dependencia TLS-06 directa,
formato ausente o distinto falla cerrado. Por compatibilidad, cualquier caller
que aporte `SkillReleaseCatalog` puede omitirlo en la entrada: se normaliza a
v1 antes de publicar/digerir y esa excepción deberá retirarse al migrarlo.

Cada descriptor puede ligar, sin convertirlos en autoridad:

- capabilities opacas exactas, sin wildcard, ordenadas y sin duplicados;
- scopes tipados canónicos; su coincidencia futura no concede autorización;
- un contrato externo de configuración por `artifact:sha256`, digest y tamaño,
  nunca keys, valores, defaults, credenciales o otro registro de config;
- conectores por ID/revisión, contrato CAS exacto y permisos explícitos;
- tools por ID/revisión/spec digest exactos contra TLS-01;
- skills por ID/revisión/registration digest exactos contra TLS-06.

Una skill obliga a incluir también cada tool exacta que declara. Si el caller
aporta deliberadamente `SkillReleaseCatalog`, se comprueban release digest y
revocación para compatibilidad con la curation posterior; eso no firma el
plugin ni introduce trust propio. Ausencia de capabilities, scopes o config
declara ausencia de esa metadata y nunca amplía autoridad.

El catálogo valida forma y límites antes de copiar/ordenar: 128 plugins, 128
componentes por clase, 64 capabilities, 32 scopes, 64 permisos, contrato de
config de hasta 1 MiB y descriptor serializado de hasta 32 KiB. Los SHA-256
solo admiten hexadecimal minúsculo canónico. Identidades duplicadas se
rechazan por clase; la misma identidad entre connector/tool/skill conserva su
tipo. Permisos son la unión exacta, sin grants implícitos.

Construcción, `Lookup` y `List` devuelven copias profundas de slices y config.
El catálogo no retiene bodies de skills, config, handlers o ubicaciones de
código y no observa registries ajenos no referenciados por sus digests.

## Verificación de esta revisión

Verdes durante autoría:

```text
go test -mod=vendor -count=1 ./internal/tooling \
  -run '^(TestPluginCatalog|TestPluginComponentKinds)'
go test -mod=vendor -count=1 ./acceptance \
  -run '^TestV26TLS10PluginDescriptor'
```

Los negativos cubren formato/ID/revisión, wildcard/duplicados/scope, config,
refs/digests y SHA mayúsculo, límites, componentes/permissions, tool/skill
ausente o divergente, cierre skill→tool, scope de skill alterado, catálogo
atómico, release revocada, descriptor sobredimensionado y copias de metadata
por input/Lookup/List. La allowlist fija campos y tags JSON exactos.

También quedaron verdes: diez repeticiones focales, focal `-race`, todo
`internal/tooling`, todo `acceptance` (última pasada: 76.140 s), compilación `./...`, guards de
arquitectura/roadmap y `go vet ./internal/... ./cmd/orquesta`.

Los mínimos transversales no están globalmente verdes por estado ajeno:

- `go test -mod=vendor -count=1 .`: ancla de
  `internal/application/ports.go` divergente, receipts Codex obsoletos y
  `scripts/continuar_orquestav2_home_codex.sh` no trackeado;
- `go test -mod=vendor -count=1 ./internal/... ./cmd/orquesta`: perfiles Codex
  no disponibles y `bootstrap.codex_go_toolchain_invalid`; tooling, SQLite,
  application y `cmd/orquesta` sí pasaron en esa invocación.

`git diff --check` y el check explícito de los cuatro ficheros nuevos no
emitieron diagnósticos. Ningún bloqueo autorizó tocar rutas ajenas.

## Cierre honesto

- hecho: fundación descriptor versionada, acotada, determinista e inmutable;
- autoridad final: catálogo local read-only, no wiring productivo;
- receipts/revisión acreditada: ninguno; no existe candidato V26;
- código o legacy retirado: ninguno; el hueco legacy permanece caracterizado;
- E2E, persistencia, marketplace y efectos: inexistentes en este corte;
- LOC: 447 producto, 398 verificación y 126 documento; menos de 0,05 MiB persistentes;
- riesgos P0/P1 TLS-10: ninguno observado tras tres contraauditorías; P2:
  retirar la normalización de formato al migrar callers `SkillReleaseCatalog`;
- siguiente dependencia causal: TLS-11/12 deberán consumir el digest exacto sin
  convertir instalación, curation o autorización en responsabilidad TLS-10.

No se atribuye `TLS-10`, V26 ni ninguna otra capability como acreditada.
