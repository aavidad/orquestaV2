# Contrato V35–V37: composición profesional de videojuegos

Fecha: 2026-07-22.

Estado: propuesta ejecutable posterior a V34. No acredita código ni modifica el
catálogo vigente de 257 capacidades.

## Frontera y decisión

`/home/alberto/Trabajo/juegos` es una aplicación externa: contiene los juegos,
assets, toolchain y emuladores de su dominio. Orquesta solo dirige Goals y
efectos mediante un `DomainPlugin` externo por MCP/HTTP y refs opacas. No hay
imports, DB compartida, lectura de filesystem interno, mount permanente ni
adaptador Neo Geo, ngdevkit, MAME o GnGeo dentro del core.

La composición reutiliza `EXT-00`, `EXT-01`, `EXT-09`, `EXT-20`, `EXT-21` y
V29; no crea una capability canónica nueva. `OPE-14` sigue rechazado: excluye
juegos/actividades como producto OPES, no esta composición de videojuegos
externa e independiente.

Un plugin declara refs como `game-project`, `game-build`, `game-toolchain` y
`game-release`; su significado queda en el plugin. Orquesta conserva solo
solicitud autorizada, identidad causal, artifact refs y receipts.

## V35 — composición y build reproducible

Dependencias: V26, V28, V29 y V32–V34 acreditadas.

El plugin inventaría un proyecto externo y ejecuta un build aislado con
toolchain identificado por digest. Recibe el workspace o snapshot autorizado y
devuelve refs inmutables para ROM, manifiesto de hashes, log y metadatos de
build. No publica ni ejecuta emuladores en esta vertical.

Write-set futuro: repositorio del plugin/fixture temporal de videojuegos y sus
tests. El núcleo no cambia. Una eventual entrada de roadmap, fixture de
aceptación o evidencia en Orquesta requiere write-set separado y autorización
explícita.

E2E V35: mediante comando público se crea un Goal de build para un fixture
temporal; el plugin realiza preparación y build, y el cliente recupera ROM,
manifiesto y log por artifact refs. Deben fallar un toolchain no sellado, scope
de proyecto incorrecto o intento de leer estado/FS interno de Orquesta.

## V36 — QA reproducible

Dependencias: V35, V17–V18 y V27 acreditadas.

Un tool externo y autorizado arranca el artefacto V35 en un emulador aislado,
aplica entradas guionizadas y registra checks de arranque, ROM, presupuesto de
assets y capturas de frame/audio como artefactos. Las decisiones de revisión se
ligan al digest exacto del build; no se infiere calidad por texto ni se crea un
motor de gameplay en Orquesta.

Write-set futuro: plugin/runner de QA, fixture y pruebas externas. No se añade
un tipo de dominio de videojuego al core ni un scheduler, store o lifecycle.

E2E V36: desde el build sellado de V35, la ejecución emulada produce las
capturas y métricas esperadas. Cambiar ROM, assets, script de entrada o digest
de toolchain invalida el resultado; una aprobación de QA no puede promocionar
un build diferente.

## V37 — promoción gobernada

Dependencias: V36, V29 y V31–V34 acreditadas.

La promoción crea un manifiesto externo que liga fuente, plugin/toolchain,
ROM, QA y destino. Cualquier copia a una cabina, repositorio de distribución o
target equivalente es un efecto: principal autorizado, aprobación, scope
exacto, idempotency key y receipt. Sin aprobación queda como intención; el
rollback/demote conserva receipt y no reescribe evidencia.

Write-set futuro: plugin de promoción, target temporal, fixture y pruebas de
efectos. Ningún target concreto entra en bootstrap ni en core.

E2E V37: un target temporal recibe una promoción aprobada una sola vez y el
replay devuelve el mismo resultado causal. Sin aprobación o con scope distinto
no existe publicación; rollback deja el target y receipt observables.

## Sellado P/S/E y seguridad común

Cada vertical sigue el mismo sello:

1. `P`: candidato con plugin, fixture y pruebas, sin autoacreditación.
2. `S`: digests inmutables de árbol Orquesta, plugin, fixture, toolchain,
   configuración efectiva redactada y artefactos de entrada.
3. `E`: desde `detached_clean` de `S`, el argv público ejecuta el E2E y produce
   un receipt V3 externo al candidato.

El build/QA usa sandbox y permisos mínimos; herramientas, egress y mounts se
declaran de forma exacta. Credenciales y producción no viajan al plugin salvo
por el puerto autorizado. Los outputs grandes son artefactos, no prompts.

## Simplicidad y diferidos

Hay un único `DomainPlugin`, los mismos Goal/DAG/receipts de Orquesta y una
composición externa; no se añade core de videojuegos, proveedor privilegiado,
cola privada, catálogo paralelo ni integración con el runtime histórico.

Quedan diferidos hasta decisión separada: fabricación física/firma propietaria
de cartuchos, storefronts, publicación pública real, soporte multiplataforma y
evaluación subjetiva de diversión. Si uno exige una frontera no cubierta por
los contratos existentes, se propone primero como target/plugin externo; solo
una necesidad demostrada justifica ampliar el catálogo de 257.
