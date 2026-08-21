# Corte V25 ORC-26: scoring de provider/modelo

Fecha: 2026-08-21.

## Alcance y autoridad

Este corte implementa una consulta de `application` para ordenar
provider/modelo. No modifica lifecycle, no persiste scorecards, no fabrica
fallback y no conecta una fuente productiva de métricas.

`ORC-26` permanece `declared`, sin `evidence_refs`, y
`AC-V25-PROVIDER-ADAPTERS` permanece `planned`. El incremento es
`implemented_query_not_wired_not_accredited`: no acredita adapters reales,
E2E, telemetría, persistencia, bindings públicos ni V25.

`internal/application` conserva la autoridad de ranking y routing. El
`ProviderScoreEvidenceReader` solo resuelve policy y el corpus admitido; no
decide rutas ni escribe estado. Los fakes son contractuales, no adapters
productivos.

## Invariante restaurado

El request contiene únicamente `PolicyRef`, `ProfileRef`, capabilities,
esfuerzo y la opción explícita de fallback. No acepta métricas ni refs de
receipts, por lo que el caller no puede escoger un subconjunto favorable.

La secuencia autoritativa es:

1. `application` normaliza el sujeto acotado y resuelve la policy por ref;
2. cruza `ProfileRef`, capabilities y esfuerzo con la policy nuevamente
   sellada;
3. pide al reader el corpus completo y canónico de esa policy y sujeto,
   imponiendo el máximo de observaciones;
4. falla cerrado si el reader no acredita completitud y vuelve a comprobar
   límite, orden, duplicados, sujeto y hash completo de cada observación;
5. solo entonces lee una vez el reloj y calcula ranking y routing.

Así, una observación que caduca durante las lecturas queda `score_stale`. Un
corpus incompleto o vacío es `evidence_unavailable`; un corpus sobredimensionado,
no canónico o divergente es `evidence_divergent`. Receipts válidos pero ligados
a otra policy o perfil también se rechazan.

Los límites privados y no configurables son 64 capabilities, 256 observaciones
y 256 bytes por ref genérica. Los conteos se comprueban antes de copiar,
ordenar, reservar mapas o recorrer contenido. El reader recibe el límite y
`application` lo vuelve a validar; nunca se trunca.

El desempate determinista sigue siendo score, confianza, muestras, provider y
modelo, como dimensiones separadas. Policy y evidence refs conservan el SHA-256
completo y son content-addressed; el hash liga contenido, no autentica al
productor. La autenticidad corresponde al reader futuro admitido por
composición.

## Preflight y revisiones bootstrap

El preflight local de `ORC-26` recomendó `reimplement`: no encontró función
legacy exacta ni permiso de copia. Conservó score, confianza y muestras
separados, perfil explícito, observaciones reales y decisión en `application`.

Se crearon exactamente tres subagentes de solo lectura y se esperaron sus tres
informes: autoridad de corpus/policy/perfil; reloj/frescura/límites; y
refactor/compresión. Ninguno editó. Codex directo actuó como excepción
bootstrap acotada al write-set declarado.

## Presupuesto, LOC y complejidad

Los cinco archivos eran nuevos frente a `HEAD`, por lo que el delta íntegro es
su contenido completo:

- producto: `P=425` líneas en `provider_scoring.go`, límite `P<=450`;
- verificación: `V=500` líneas, 304 unitarias y 196 de aceptación, límite
  `V<=500`;
- fixture: 51 líneas; este documento: 99 líneas tras el cierre.

Complejidad operativa añadida: dos lecturas externas acotadas y una lectura de
reloj por consulta; cero writers, stores, goroutines, loops residentes,
configuración o bindings HTTP/MCP/CLI/web. Los únicos recorridos nuevos quedan
acotados por 64 capabilities o 256 observaciones.

## Verificación del candidato

Verdes en esta revisión:

- focales ORC-26 de `internal/application` y aceptación, normales y con
  `-race`;
- `GOFLAGS=-mod=vendor go vet ./internal/application ./acceptance`;
- `go test -mod=vendor -count=1 . -run 'TestRebuildArchitecture$'`;
- compile gate completo `go test -mod=vendor -run '^$' ./...`;
- `gofmt`, parseo JSON, presupuestos y `git diff --check`, incluido el
  contenido todavía no registrado.

## Diferidos y dependencia siguiente

Sigue un adapter durable y autenticado para policies y corpus, su composición
y bindings; después, contrato neutral, E2E real e aislamiento entre Goals de
`AC-V25-PROVIDER-ADAPTERS`.

No se retira legacy: no existe equivalencia acreditada ni censo de consumidores
que autorice retirada.
