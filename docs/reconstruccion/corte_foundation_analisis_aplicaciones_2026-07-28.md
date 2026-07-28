# Corte foundation: análisis de aplicaciones — 2026-07-28

## Estado honesto

`FOUNDATION-APP-ANALYSIS` acredita una base interna aislada, con estado
`foundation_only_unwired`. No acredita una capacidad del roadmap, no crea
evidencia en `product/evidence/` y no adelanta `AC-V28-DOMAIN-PLUGINS`, que
sigue `planned`. `WIZ-10` y `EXT-00` permanecen relacionados y `declared`, sin
evidencia: este corte no los promociona.

## Qué aporta Analizador y qué aporta esta foundation

El proyecto externo `windows-analyzer` ejecuta y observa una aplicación
Windows. Produce evidencia de procesos, GUI y efectos del sistema. No clona,
no refactoriza y no comprende por sí solo el código fuente o la arquitectura
interna. Su salida puede servir como requisitos conductuales para una
reimplementación limpia o como baseline parcial de regresión para un refactor.

La foundation de Orquesta implementada aquí recibe y conserva un análisis
verificable de cualquier productor. Usa refs opacas de sujeto, proveedor,
productor, intake, repositorio y artefactos; exige autorización, snapshot de
intake por proyecto, revisión aceptada ligada al manifiesto canónico y recibos
de procedencia de artefacto. Sus cuatro puertos son el verificador de intake,
resolvedor de artefactos, verificador de revisión y store de adjuntos. El
digest, la idempotencia y la restauración detectan sustituciones causales.

La foundation no analiza un checkout, no ejecuta herramientas, no abre una
conexión a la aplicación externa y no conoce su DB ni filesystem. Tampoco tiene
bootstrap, adaptador concreto, comando público canónico, persistencia integrada
ni E2E. El conector independiente que se desarrolla junto a
`windows-analyzer` no forma parte de este corte ni se usa como evidencia. Por
eso este trabajo no cierra todavía el Wizard de `WIZ-10` ni un conector de
dominio de `EXT-00`.

## Foundation implementada y pruebas

La superficie se limita a `internal/application/app_analysis.go`,
`app_analysis_service.go` y sus dos ficheros de prueba. Mantiene las fronteras
hexagonales: no importa adaptadores, comandos ni legado. La aceptación carga el
fixture estricto, verifica los cuatro ficheros, los puertos, las fuentes
causales, las fronteras de importación y la ausencia de wiring público,
persistencia o evidencia. Además inventaría y ejecuta los tests `TestAppAnalysis`
de `internal/application`; un “no tests to run” no cuenta como verde.

## Riesgos P2 y secuencia V28

P2: al cablear puertos se puede convertir una ref opaca en acceso directo a
repositorio, DB o filesystem; también se puede promocionar erróneamente la base
unitaria como E2E. Los adaptadores futuros deben garantizar que los
verificadores y el resolvedor devuelven receipts inmutables y estables para la
misma petición causal.

La siguiente secuencia es: definir el contrato público y la autorización de
`WIZ-10`; implementar adaptadores concretos aislados y persistencia
transaccional; componer comandos HTTP/MCP/CLI sin segundo writer; ejecutar E2E
con una aplicación externa temporal y receipts; solo entonces evaluar el
contrato `AC-V28-DOMAIN-PLUGINS` y la evidencia de la misma revisión.
