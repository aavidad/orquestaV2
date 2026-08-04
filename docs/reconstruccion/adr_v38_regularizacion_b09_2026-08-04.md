# ADR V38: regularización del corte B09

Fecha: 2026-08-04.

## Contexto

El presupuesto prospectivo de B09 (`P=600,V=550`) se fijó antes de implementar
el intermediario de egreso completo. El corte lógico `7c9ef00..fde9654` del
repositorio `agente_microvm` añadió 3.836 líneas y retiró 181, pero todavía no
incluía la corrección de portabilidad musl ni los activos reconstruidos. Usar
ese corte como candidato habría conservado un falso verde: las pruebas GNU
pasaban y el huésped no compilaba para su target real.

El candidato reproducible es `7c9ef00..bb90346`. `df6469c` corrige el tipo de
las peticiones `ioctl` para musl y convierte el build cruzado locked en guarda;
`bb90346` liga el SBOM y la prueba B05 al nuevo huésped.

## Medición

La métrica normativa cuenta líneas brutas añadidas y retiradas por clase. En
ficheros Rust mixtos se clasifican por cada lado del diff los módulos
`#[cfg(test)]`, incluidos imports y helpers, como verificación. No se atribuye
el fichero entero a producto. Documentación queda fuera de `P/V`:

| Clase | Añadidas | Retiradas | Neto |
|---|---:|---:|---:|
| producto, contratos, configuración y scripts (`P`) | 2.683 | 117 | 2.566 |
| pruebas, fixtures y módulos `cfg(test)` (`V`) | 1.179 | 69 | 1.110 |
| documentación (`D`) | 255 | 17 | 238 |

`P+V+D` reconcilia exactamente las 4.117 inserciones y 203 retiradas de Git.

## Decisión

1. B09 sustituye su techo histórico por `P=2683,V=1179`, equivalente a las
   líneas brutas añadidas del candidato reproducible ya implementado.
2. La cifra regulariza un hecho pasado. No constituye una bolsa disponible:
   cualquier producto o prueba posterior necesita tarea, write-set y
   presupuesto propios.
3. No se reutiliza la retirada compensatoria de B08. Aquella retirada cerró
   autoridades HMAC/red duplicadas y no puede financiar otro frente.
4. B09 queda `exercised` sin KVM: acredita contrato, política, persistencia,
   puente huésped y construcción reproducible. La ejecución privilegiada se
   reserva a B12.
5. La traducción productiva de concesiones y receipts dentro de Orquesta se
   implementa en el adaptador B10. B09 define y prueba el contrato neutral; no
   introduce anticipadamente un segundo conector.
6. El total V38 pasa de `P=7200,V=10083` a `P=9283,V=10712`. Es una corrección
   contable, no una promoción de `ORC-28` ni de V38.

## Evidencia reproducible

Dos builds aislados produjeron los mismos artefactos:

- huésped: `449417530d194924cbbe8f20094b80a94135f7af48dcf0fecf9040836c01fde2`;
- initramfs: `837a38ed9efaca72c065eb3330762d9b622d882bbcafacd7ed68541c54b6445e`;
- perfil ext4: `04954ea400573419d7bdea7486caeb550c555c80701cef7e625270c9b7584885`.

Pasaron 221 pruebas Rust activas, dos smokes físicos quedaron ignorados de
forma explícita, Clippy estricto quedó verde y el test focal B05 validó el SBOM.
No se arrancaron Orquesta, Firecracker, Jailer ni KVM para esta evidencia.

## Siguiente dependencia

B04 debe sellar la compatibilidad entre candidatos versionados reales sin
`replace`, importación cruzada ni almacén compartido. Después siguen B01, B10,
B11 y B12 en el orden causal del plan. Ninguno hereda holgura de esta
regularización.
