# ADR V38: frontera de trabajo del agente antes del conector B10

Fecha: 2026-08-04

Estado: aceptada para ejecutar; no acredita B10 ni arranca KVM.

## Hallazgo

El cliente público `agentmicrovm.local.v1` ya crea, observa, ordena, detiene,
preserva y cierra una ejecución física. Sin embargo, `Lanzar` solo admite la
microVM y `Observar` solo devuelve su ciclo de vida físico. El protocolo no
expone aún una sesión de agente, su resultado terminal, uso de recursos o un
comprobante causal del trabajo. Por tanto, `disponible` no significa
`AgentRunning` y `detenida`, `preservada` o `cerrada` no significan
`AgentCompleted`.

También faltan en la frontera Orquesta los hechos durables necesarios para
preparar una concesión repetible: autorización, aprobación e intento de efecto,
cerca, instante de emisión, referencias de artefactos/MCP/buzón y descriptor
inmutable del perfil físico. Inventar esas referencias o firmar otra concesión
con el reloj vivo durante un replay rompería idempotencia y recuperación.

## Decisión

B10 se divide de dentro hacia fuera:

1. `B10.0`, contrato público hermano: descriptor de perfil y sesión/resultado
   de trabajo neutrales, con claves de idempotencia, cerca y comprobantes.
2. `B10.1`, autoridad de lanzamiento: Orquesta transporta los hechos durables
   ya decididos por aplicación; el adaptador no los deriva por aproximación.
3. `B10.2`, dependencia pública: se fija una revisión del módulo Go que ya
   contenga `B10.0`, sin `replace` local ni segundo cliente HTTP.
4. `B10.3`, conector: negociación, lanzamiento firmado, observación de trabajo
   y traducción de errores; nunca lee SQLite, PID, CID, cgroup o rutas hermanas.
5. `B10.4`, composición: configuración canónica, secreto por referencia y
   selección explícita `provider=codex`, `isolation=microvm`, sin fallback a
   proceso.
6. `B10.5`, recuperación: concesión byte a byte, respuesta perdida, socket
   perdido, cerca obsoleta, reinicio y carrera.

`Shutdown` del adaptador solo libera conexiones. La parada exacta pertenece a
B11 y la preservación a B12. El contrato físico continúa siendo autoridad
exclusiva de `agente_microvm`; Orquesta conserva la autoridad de Goal, efectos,
presupuesto, colocación y ciclo de vida del trabajo.

## Acreditación

B10 termina como `wired/exercised_without_kvm`: candidato negociado, contrato
traducido y recuperación probada con servidor real sin KVM o dobles de puerto.
El primer Codex real dentro de una microVM, junto con orden posterior, salida,
parada, sello, inventario y limpieza física, sigue reservado a B12.4.

No acreditan B10: un `POST /v1/ejecuciones` correcto sin sesión de agente, un
receipt fabricado con el reloj local, estados físicos reinterpretados como
estados de trabajo, referencias sentinela, concesiones solo en memoria ni un
fake de KVM.

## Presupuesto

Los techos originales `P=280,V=430` eran una estimación sobre una API supuesta.
Se conservan como referencia histórica, no como permiso para recortar el
contrato. Antes de cerrar B10 se medirá el delta causal de `B10.0` y se
regularizará el techo con el mismo método usado en B09.
