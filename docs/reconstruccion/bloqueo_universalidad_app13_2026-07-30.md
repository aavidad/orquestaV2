# Bloqueo de universalidad de APP-13 — 2026-07-30

Este documento registra por qué Orquesta todavía no puede afirmar que todas
las aplicaciones que crea o modifica llevan manifiestos y cabeceras que
expliquen qué hacen. No implementa APP-13 ni sustituye al roadmap.

## Ficha del corte

```text
capability IDs: APP-13
invariante: la guía no amplía el contrato superior
autoridad que escribe: ninguna; este documento solo registra un bloqueo
puertos afectados: TestAttestor y futuro registro gobernado de herramientas
adaptadores afectados: futuros atestadores neutrales; no se modifican aquí
write-set: este documento y legacy_app13_universality_block_test.go
dependencias causales: Wizard/AppSpec, TLS-01, atestación, ruta total y AC-V33
código antiguo que permitirá retirar: ninguno en este corte
test de contrato: TestAPP13UniversalityRemainsNoGo
negativo/mutación: roadmap insuficiente o ApplicationTarget productivo oculto
E2E/gate: bloqueado hasta que el roadmap ordene los cuatro escenarios
presupuesto: documento <=220 líneas; prueba <=190 líneas; cero producto
```

Excepción de arranque declarada: Codex directo coordina esta documentación
porque Orquesta está parada. La consulta previa de lecciones para `APP-13`
devolvió `Sin patrones`.

## Decisión honesta

**Estado: NO-GO.**

Los hechos de autoridad actuales son:

- `APP-13` está `declared`;
- `AC-V33-GENERATED-APPS` está `planned`;
- el corte acreditado de `product/capabilities.json` no contiene APP-13 ni
  AC-V33;
- sus aserciones exigen una aplicación Go y una no Go, el mismo árbol o imagen
  en el E2E público y que los ficheros generados aislados no cuenten;
- esas aserciones no fijan por separado creación Go, modificación Go, creación
  no Go y modificación no Go.

La guía de reconstrucción desea esos cuatro casos y enumera negativos útiles.
Sin embargo, la guía no amplía el contrato superior. La discrepancia debe
resolverse de forma atómica en `product/roadmap.json`,
`product_roadmap_test.go`, `docs/reconstruccion/ruta_total_100.md` y la
aceptación V33 antes de convertir ese deseo en aceptación ejecutable. La ruta
total vigente solo exige dos aplicaciones creadas, no los cuatro escenarios.
Roadmap y ruta total están actualmente ocupados por V38. Este conjunto de
escritura no toma esa decisión ni acredita conducta universal.

## Alcance universal que queda pendiente

Una vez resuelta la contradicción e implantada y acreditada APP-13, la regla
aplicará a **toda aplicación creada o modificada por Orquesta**, con
independencia de su lenguaje. No dependerá de que el operador lo recuerde, de
una frase del encargo ni de una convención exclusiva de Go.

Esa universalidad aún no existe. Las pruebas locales de aplicaciones
auxiliares solo acreditan esas aplicaciones concretas.

## Conducto genérico que sí existe

El producto ya ofrece piezas reutilizables, pero ninguna equivale a APP-13:

1. un trabajo escritor debe declarar `RequiredTests`;
2. la huella del plan liga referencia, herramienta, argumentos y directorio;
3. `compilePlan` admite el plan inicial y `compilePlanExtension` las
   ampliaciones y replanificaciones;
4. el cambio queda limitado por su conjunto de escritura;
5. la atestación liga el candidato, generaciones, cambio, árbol, pruebas y
   política;
6. la integración rechaza un candidato sin las pruebas obligatorias aprobadas.

Por tanto, no hace falta otro ciclo de vida ni otra compuerta de integración.
Faltan el hecho durable que identifica la aplicación y la política que
incorpora la prueba concreta APP-13.

La ausencia actual de un tipo productivo llamado `ApplicationTarget` es solo
un canario nominal del hueco; no demuestra por sí misma que falte o exista la
semántica completa.

## Lo que falta

- `ApplicationTarget` durable dentro de `AppSpec`, ligado por su hash, snapshot,
  restauración y persistencia;
- modo tipado `create | modify`, identidad opaca, raíz relativa canónica,
  perfil de calidad y referencia/hash del contrato i18n durable del dossier;
- proyección de la decisión durable del Wizard, sin inferirla desde texto;
- política central, determinista e idempotente que incorpore APP-13 a cada
  escritor que alcance la raíz;
- conservación de esa política en el plan inicial, dossier y replanificación;
- `TLS-01` y un atestador neutral que no acepte únicamente `tool:go`;
- herramienta registrada con versión, permisos, coste, hash y recibo;
- informe estructurado de hallazgos, no solo código de salida y hash del texto;
- ligadura explícita de proyecto, aplicación y destino en el sujeto;
- comprobador del manifiesto tipado completo, cabeceras, i18n, exclusiones y
  secretos;
- cuatro E2E sobre el mismo árbol o imagen que usan las superficies públicas.

## Contratos tipados que no pueden reducirse a texto libre

El manifiesto de cada aplicación debe contener secciones tipadas y no vacías
para:

1. propósito y usuarios;
2. alcance y exclusiones;
3. entradas y salidas;
4. arquitectura y módulos;
5. autoridades escritoras;
6. datos, permisos, secretos y efectos;
7. arranque, diagnóstico, recuperación y parada;
8. contratos y pruebas.

Son negativos obligatorios una sección ausente, vacía o mal tipada. La
comprobación de cabeceras respeta la sintaxis y convenciones del lenguaje; no
convierte una frase repetitiva en documentación válida.

La política i18n permanece como contrato durable y tipado del dossier. Declara
configuración regional predeterminada, fallback, catálogos, plurales, formatos
y superficies visibles. `ApplicationTarget` conserva solo su referencia y
hash; `AppSpec` los liga causalmente sin duplicar esa política ni reducirla a
un campo de idioma.

## Cadena causal mínima

```text
ApplicationTarget durable en AppSpec
-> política central idempotente en compilePlan + compilePlanExtension + dossier
-> RequiredTests y atestación existente
-> TLS-01 y atestador neutral
-> comprobador APP-13
-> cuatro E2E: crear Go, modificar Go, crear no Go, modificar no Go
-> acreditación AC-V33 sobre el candidato exacto
```

El motor repone la prueba canónica si falta y rechaza una declaración con la
misma referencia pero herramienta, argumentos o directorio diferentes. El
Director no puede retirarla mediante una replanificación.

## Raíz sin barrido ni registro paralelo

El Wizard confirma una raíz relativa al repositorio. Para creación puede no
existir todavía; para modificación debe pertenecer al árbol base exacto.
`ProjectRef` más la raíz canónica producen una identidad opaca ligada en
`AppSpec`; no se crea otro registro de aplicaciones.

La política compara segmentos de ruta. Un escritor alcanza la aplicación si su
ámbito es la raíz, un descendiente o un ancestro; la raíz `.` alcanza todo
escritor del repositorio. Un ámbito hermano no la alcanza. El control Git ya
impide cambiar rutas fuera del conjunto declarado.

El comprobador recibe la raíz y recorre solo ese subárbol del candidato. No
busca `README`, `go.mod`, `package.json` u otros marcadores por todo el
repositorio. Enlaces simbólicos, recorridos fuera de raíz y exclusiones
implícitas fallan cerrados.

## Microconjuntos de escritura, en orden

A. **Contrato atómico**: `product/roadmap.json`, `product_roadmap_test.go`,
`docs/reconstruccion/ruta_total_100.md` y aceptación V33 con sus fixtures.
Conflicto actual: roadmap y ruta total V38.

B. **Tipo puro**: nuevos `internal/goal/application_target.go` y su prueba.

C. **AppSpec**: `app_spec.go`, `snapshot.go`, `restore.go` y pruebas en
`internal/goal`. Conflicto: AppSpec y snapshots V23.

D. **SQLite**: siguiente migración libre, `read.go`, `write.go`,
`app_specs_test.go` y `repository_test.go`. Conflicto: migraciones V23.

E. **Wizard y dossier**: extracción tipada, dossier durable, superficie de
comandos y confirmación en cuatro tareas separadas. Conflicto: dossier V23.

F. **Política central**: nuevo `app_profile_plan_policy.go` y prueba; cableado
estrecho en `planning.go` y preparación del dossier. Conflicto: planificador
V23.

G. **Sujeto e informe**: contratos versionados de `TestAttestor` y ligadura de
proyecto, aplicación, raíz e informe estructurado.

H. **Herramienta gobernada**: después de TLS-01, adaptar Bubblewrap y después
Firecracker. Conflicto actual: Firecracker/V38; no tocar mientras esté ocupado.

I. **Comprobador pequeño**: paquete cohesivo `internal/appprofile` sin almacén,
bucle, ciclo de vida ni barrido global.

J. **Compuerta y E2E**: prueba focal de integración y
`acceptance/v33_generated_apps_test.go` con los cuatro escenarios. Evidencia
solo después del pase real.

Cada tarea debe terminar verde y no introducir un registro, escritor,
planificador o aplicación mastodóntica alternativos.

## Cierre de este corte

```text
hecho: contradicción, NO-GO, cadena causal y tareas A-J documentados
invariante restaurado: la guía no amplía el contrato superior
autoridad final: product/roadmap.json permanece sin cambios
tests: prueba focal estructural del bloqueo
receipts y revisión acreditada: no aplican; no se acredita APP-13
código o decisión retirados: ninguno
legacy retirado o bloqueo de retirada: no se inspeccionó ni modificó legacy
LOC netas y complejidad: solo documento y prueba de bloqueo
riesgos/P0/P1: P0 falsa universalidad; P1 bypass por plan o replanificación
siguiente dependencia causal: resolver AC-V33 en roadmap tras liberar V38
```
