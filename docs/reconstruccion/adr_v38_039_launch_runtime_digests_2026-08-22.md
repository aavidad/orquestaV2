# ADR V38/039: digests físicos en la preparación de launch

## Decisión

El launch nuevo persiste `plan_sha256`, `concession_sha256`, `kernel_sha256`,
`initramfs_sha256` y `profile_sha256` como suplemento 1:1 de la autoridad v32.
El adaptador deriva `plan_sha256` y `concession_sha256` únicamente de la
extracción canónica de la solicitud firmada; deriva los otros tres del
`Compilation.Plan` ya validado y ligado a esa misma solicitud. SQLite inserta
primero el suplemento con FK diferida y después la autoridad v32 dentro de la
misma transacción, antes de cruzar `client.Lanzar`.

La migración 039 toma una instantánea explícita de todas las claves v32
preexistentes en `microvm_host_launch_runtime_digest_legacy_exemptions`, fija
el epoch 39 y, desde ese corte, rechaza cualquier autoridad v32 nueva sin
suplemento. Recovery exige una
partición XOR exacta: cada clave pertenece al snapshot legacy o posee los cinco
digests, nunca a ambos ni a ninguno. Epoch y exemptions quedan cerradas tras
la migración; los suplementos solo admiten inserción causal y son inmutables.

## Límite honesto

El snapshot legacy demuestra únicamente pertenencia pre-039.
No es una prueba criptográfica de plan, concesión, kernel, initramfs o perfil
y no permite reconstruirlos. Por eso el lookup runtime de una clave exenta
falla cerrado; la proyección histórica completa queda fuera de este corte. Este cambio no
acredita 040, Preserve, B12, Gate B, las olas físicas C, ORC-28 ni V38.

## Presupuesto

El corte excede el techo preliminar de schema-only porque ese corte habría
dejado una ventana indistinguible entre migración y writer. Se acepta el coste
del puerto compuesto, writer transaccional y adaptación productiva como una
sola unidad causal; no incluye Preserve ni la proyección histórica posterior.

Baseline: `4b60ad14`. Medida actual contra ese baseline, incluyendo archivos
no rastreados como adiciones completas: `P=358`, `V=414`, `D=42`; borrados
separados: producto 6, tests 29 y documentación 0. Esta medida no acredita el
candidato.

Permanecen pendientes el resultado final de `git diff --check`, las suites
focales normal y race, SQLite completo, `go vet` proporcional y la revisión
independiente del diff exacto. Hasta que esos gates terminen verdes, 039 no se
presenta como corte publicable ni como evidencia de Gate A. Ningún resultado
de este ADR sustituye receipts ni revisión independiente.
