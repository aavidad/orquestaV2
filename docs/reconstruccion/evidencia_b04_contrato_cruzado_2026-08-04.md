# Evidencia B04: contrato cruzado Orquesta/Agente MicroVM

Fecha: 2026-08-04.

## Resultado

B04 queda `exercised_without_kvm`. No acredita `ORC-28`, B10, B12 ni V38.
Demuestra que un consumidor de Orquesta alcanza un binario versionado real de
Agente MicroVM únicamente mediante `agentmicrovm.local.v1` sobre un socket
Unix privado, sin `replace`, importación cruzada, módulo temporal, base de
datos o filesystem productivo compartido.

Sujetos sellados:

| Sujeto | Commit | Árbol Git | Resumen adicional |
|---|---|---|---|
| Orquesta | `e80b887337fa274fea243cc8df5dbb29d8dea5e0` | `75204b6428887c8439b788f8ba7e4edca9c8348e` | contrato `e096e6c1…c837ec` |
| Agente MicroVM | `da0bd609137d960af358941b634086ccc7a15354` | `0e36cfd350f08655e1d18fee0e9f34790f3c4e54` | binario `faa230b9…56764a` |

El binario release locked mide 8.570.264 bytes y dos builds aislados produjeron
el mismo SHA-256. `--version` publicó `0.1.0` y el protocolo exacto.

## Recorrido y negativos

La prueba construye el binario desde el checkout hermano limpio, verifica su
digest, arranca el servidor con una configuración que solo contiene
`[api].socket`, espera un socket `0600` dentro de un directorio `0700` y
consulta capacidades por HTTP/1.1 Unix. El servidor mínimo publica únicamente
`salud` y `capacidades`, con `maximo_ejecuciones=0`; no abre KVM ni compone
SQLite, Firecracker, Jailer o ejecución física.

Los negativos rechazan:

- protocolo ausente o `agentmicrovm.local.v0`;
- método no permitido y cuerpo superior al límite;
- socket Unix distinto;
- digest de binario distinto;
- dependencia directa o transitiva cruzada en los manifiestos sellados.

La parada usa `SIGTERM` sobre el PID exacto, espera terminación cooperativa y
exige que desaparezca el socket. El inventario final contiene solo el TOML de
prueba. Orquesta, Firecracker, Jailer y KVM no se arrancaron.

## Verificación

Pasaron:

```text
go test -mod=vendor -count=1 ./acceptance -run ^TestV38B04CrossRepositoryContract$ ...
go test -mod=vendor -race -count=1 ./acceptance -run ^TestV38B04CrossRepositoryContract$ ...

cargo fmt --all --check
cargo clippy --locked --workspace --all-targets --all-features -- -D warnings
cargo test --locked --workspace --all-targets --all-features
  221 activas correctas; 2 smokes físicos ignorados

cd conectores/orquesta
GOWORK=off go test ./...
GOWORK=off go test -race ./...
GOWORK=off go vet ./...
```

El manifiesto durable es
`product/evidence/candidates/v38/b04_compatibility_v1.json`, SHA-256
`36ec98654b39c8f55640f8025a1d4cf3f500509b62d2039c30cebdb3967c29ec`.
La fixture de entrada queda separada para evitar un resumen autorreferente.

## Presupuesto y continuación

B04 añade `P=0,V=338`: 336 líneas de aceptación y dos líneas de datos
minificados para fixture/manifiesto. Permanece dentro de `P≤50,V≤350`; las
pruebas ya existentes del hermano se reutilizan y no se reatribuyen.

La siguiente dependencia es B01, una guarda compacta de selección y frontera.
Después pueden empezar B10, B11 y B12. El conector productivo no se anticipa en
B04 y la primera microVM física de agente continúa reservada a B12.
