# ADR V38: baseline publicado de Agente MicroVM

Fecha: 2026-08-02.

## Contexto

El plan V38 presupuestó B03 antes de que el operador ordenase crear y probar
Agente MicroVM como producto independiente. El repositorio publicado
`https://github.com/aavidad/agente_microvm`, revisión `bdba503`, ya no es un
esqueleto atribuible solo a B03: contiene conductas que el plan distribuye
entre B02, B03, B05, B06, B07, B08, B11 y B12.

La caracterización local, sin arrancar KVM ni Firecracker, obtuvo:

- árbol limpio y sincronizado con `origin/main` en `bdba503`;
- 29.221 líneas físicas en fuentes Rust y Go, incluidas pruebas embebidas;
- 31.621 inserciones y 78 retiradas acumuladas desde el commit raíz;
- 164 pruebas Rust descubiertas: 162 verdes y dos smokes físicos ignorados;
- cliente Go, formato y Clippy verdes;
- 6.916 líneas físicas en la superficie transversal seleccionada de B03,
  incluyendo contrato, API, CLI, clientes, arranque y documentación.

Estas cifras no son una separación fiable `P/V`: Rust mantiene muchas pruebas
en módulos `cfg(test)` dentro de los mismos ficheros. Sí demuestran que la frase
«B03 incluye toda la base del nuevo repositorio» y su techo prospectivo
`P=250,V=300` no pueden coexistir con el candidato vivo.

La base conserva resultados útiles ya probados físicamente: microVM real,
recuperación tras caída, órdenes por vsock, sincronización sellada, perfil de
agente, preservación, cliente Go y Jailer. Reescribirla o borrarla para hacer
cuadrar una estimación destruiría evidencia sin reducir autoridades.

## Decisión

1. El candidato publicado se trata como **baseline externo caracterizado**, no
   como implementación automática ni evidencia suficiente de ninguna tarea
   V38.
2. El presupuesto `P/V` de cada tarea B gobierna únicamente su cambio neto
   posterior a `bdba503`. El baseline físico se informa por separado y no se
   oculta dentro de B03 ni se vuelve a contar como delta V38.
3. Cada conducta existente se acredita solo al atravesar el gate de la tarea
   que la posee. Código existente sin contrato, composición o prueba requerida
   sigue siendo trabajo pendiente aunque compile.
4. B03 congela la frontera independiente, el protocolo y los contratos
   neutrales. Solo anuncia en capacidades las operaciones realmente
   implementadas. `Events` se activa con B05 y `Recover` con B07/B10; hasta
   entonces no hay stub exitoso, fallback ni afirmación de disponibilidad.
5. Toda línea nueva se carga a la tarea causal correspondiente. Superar su
   techo residual exige otro ADR y retirada compensatoria nombrada.
6. La compensación arquitectónica obligatoria es retirar del producto Orquesta
   los puertos y adaptadores físicos solapados de asignación CID, autorización
   de lanzamiento y red Firecracker al completar B02/B07/B08/B09/B10. El
   atestador Firecracker permanece separado porque acredita pruebas, no agentes.

## Consecuencias

- B03 deja de fingir que una aplicación ya publicada puede reconstruirse en
  550 líneas y conserva un write-set pequeño de caracterización y contrato.
- El total prospectivo `P=7.200,V=10.083` sigue midiendo el cambio V38 desde el
  corte de planificación; el ledger debe mostrar además este baseline externo
  para que la medida no se interprete como tamaño total del producto compuesto.
- La existencia de una ruta documentada no equivale a capacidad. El servidor,
  la CLI y los clientes deben negociar la operación antes de usarla.
- No se importa código, base de datos, filesystem, configuración o tipos del
  repositorio hermano en el núcleo Orquesta.
- La decisión no promociona `ORC-28`, no emite evidencia V38 y no autoriza
  arrancar Orquesta, Firecracker o tareas conservadas.

## Gate de retirada de esta excepción

En C04/C05 el manifiesto final debe informar por separado:

- digest y revisión del baseline `bdba503`;
- delta neto por cada tarea B;
- símbolos físicos retirados de Orquesta;
- digest final de ambos repositorios y de sus binarios/configuración;
- comprobantes A+B+C sobre esos mismos sujetos.

Si alguno falta, el baseline sigue siendo útil pero V38 no queda acreditada.
