# ADR V38: intermediario B08 y retirada compensatoria

Fecha: 2026-08-02.

## Contexto

B08 presupuestó su intermediario físico como `P=120,V=100`. Al implementar el
contrato real de Firecracker, esa división no podía conservar simultáneamente:

- el listener Unix que Firecracker crea conceptualmente como
  `<uds_path>_<puerto>` para cada servicio `vsock`;
- identidad exacta del proceso Firecracker mediante PID, UID e instante ya
  cercados por la ejecución;
- identidad exacta del servicio anfitrión mediante tipo de socket, propietario,
  permisos y resumen SHA-256 de su ejecutable;
- separación causal de dos ejecuciones que usan el mismo puerto;
- parada bloqueante sin sondeo de CPU, cierre de conexiones activas y retirada
  de sockets propios;
- recuperación tras reinicio que distingue socket huérfano propio, listener
  vivo, enlace simbólico y recurso ajeno, tanto con Jailer como sin él.

Reducirlo a la estimación habría eliminado controles del contrato o repartido
una misma autoridad entre paquetes artificiales. Ninguna de esas opciones
reduce complejidad real.

La consulta previa del read-model para
`ORC-28 + registrar-excepcion-presupuesto-b08` no encontró patrones. El hueco
es advisory y no se convirtió en otra autoridad.

## Medida

La medida física neta del repositorio Agente MicroVM entre `e663bd3` y
`2f65254`, excluyendo documentación y ejemplos y separando módulos
`cfg(test)`, es:

```text
intermediario Firecracker             P=477  V=328
integración con el motor              P=109  V=14
firmante neutral del conector Go      P=234  V=127
contratos, configuración y composición P=108 V=43
total Agente MicroVM                  P=928  V=512
```

La retirada causal en Orquesta, revisiones `59f84d98` y `0e1abc67`, elimina
una segunda autorización física, el desafío HMAC y la política de red sin
consumidor productivo:

```text
producto retirado en Orquesta         P=-1146
verificación retirada/ajustada        V=-1248
delta causal combinado B08            P=-218 V=-736
límite del padre B08                   P<=550 V<=450
```

Los cambios de fixtures, roadmap, inventario y documentación se informan, pero
no se reclasifican como producto para alterar estas cifras.

## Decisión

1. Se acepta la medida real de B08.1 y B08.3 y se redistribuye el presupuesto
   únicamente dentro del padre B08.
2. La retirada de `agent_microvm_launch_auth`,
   `firecracker/networkauth` y `agent_microvm_network` es su compensación
   nombrada. No puede descontarse de otra tarea.
3. El contrato final usa una concesión Ed25519 neutral, pública y de un solo
   uso. No cruza `CredentialStore`, clave privada, secreto HMAC, tipo de
   atestador ni referencia interna de Orquesta.
4. La cabecera causal `agentmicrovm.servicio-host.v1` identifica ejecución,
   `RunRef`, CID, plan, concesión, cerca, servicio y proceso Firecracker antes
   de reenviar bytes del huésped. No concede lifecycle a ninguno de los lados.
5. El proxy de salida permanece rechazado hasta B09. B08 no puede usar este ADR
   para adelantar red, credenciales de proveedor o acceso directo a Internet.
6. B08 queda como máximo `exercised` sin KVM. Solo B12 puede demostrar que la
   VM física exacta consume el servicio y que no quedan recursos huérfanos.

## Consecuencias

- Desaparecen 2.444 líneas físicas de rutas duplicadas en Orquesta y no queda
  bridge, dual write ni fallback HMAC.
- El conector puede firmar para Codex, Claude, Gemini u otro proveedor porque
  solo traduce contexto autorizado a referencias neutrales; las credenciales
  del proveedor no pertenecen a Agente MicroVM.
- El exceso de los hijos queda visible. El balance negativo del padre no crea
  una bolsa para B09 o tareas posteriores.
- B08 no acredita `ORC-28`, V38 ni ejecución física. B09 es la siguiente
  dependencia abierta antes del contrato cruzado B04.

## Gate de retirada de la excepción

B12 debe ejecutar sobre los mismos digests candidatos al menos:

- conexión desde la microVM correcta y rechazo de otra identidad;
- dos ejecuciones con el mismo puerto sin cruce de `RunRef` o CID;
- reinicio, reconexión, parada cooperativa y forzada;
- inventario final sin sockets, procesos, cgroups o microVM propios huérfanos.

Si falta cualquiera de estos hechos, el intermediario sigue ejercitado pero no
acreditado.
