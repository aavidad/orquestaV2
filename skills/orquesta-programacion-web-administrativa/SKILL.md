---
name: orquesta-programacion-web-administrativa
description: Crear y revisar webs administrativas densas: portales de empleado, bolsas, VEC, backoffice, expedientes, meritos, autobaremacion, listados, alegaciones, auditoria y pantallas con muchos datos visibles.
---

# Orquesta Programacion Web Administrativa

Usa esta skill cuando una tarea web requiera interfaz administrativa, portal de
empleado, bolsa de empleo, VEC, expediente, meritos, autobaremacion, listados,
alegaciones, notificaciones, auditoria o revision de muchos registros.

Esta skill especializa `orquesta-programacion-web-app`: prima densidad,
trazabilidad y lectura operativa sobre composicion promocional.

## Resultado Por Defecto

- La primera pantalla es el workspace real, no una landing.
- Deben verse accion pendiente, estado legal/operativo, evidencia, seleccion,
  filtros, plazos, puntuaciones y cambios recientes sin navegar en exceso.
- Usa tablas, filtros, contadores, panel de detalle y timeline cuando el dominio
  lo soporte.
- No uses hero, marketing, tarjetas decorativas grandes ni espacio vacio
  dominante.

## Estructura Recomendada

- Barra superior: contexto, rol, busqueda, notificaciones y sesion.
- Navegacion lateral: modulos administrativos estables.
- Zona principal: contadores compactos, filtros visibles, tabla o listado denso.
- Panel derecho o drawer: expediente seleccionado, baremo, documentos,
  comunicaciones, auditoria y acciones.
- Estado inferior si aporta valor: ultima sincronizacion, recibos, borrador,
  jobs pendientes o validaciones.

## Densidad De Datos

- Muestra el maximo dato util sin ocultar estados criticos tras hover.
- Ordena columnas de forma predecible: identidad, procedimiento, estado, plazo,
  puntuacion, evidencia y accion.
- Incluye estados de carga, vacio, error, filtrado, seleccionado, guardado,
  presentado, bloqueado y pendiente cuando el flujo pueda alcanzarlos.
- En movil conserva el mismo modelo de informacion: estado, plazo, puntuacion y
  siguiente accion deben seguir visibles.

## Colores Semanticos

Usa base neutral y acentos por significado, nunca color solo como senal:

- identidad/perfil: azul;
- convocatoria/procedimiento: indigo;
- meritos, puntuacion o ranking: violeta o teal;
- documentos/evidencia/registro: cyan;
- comunicaciones/notificaciones: ambar;
- plazos, riesgo o accion requerida: naranja;
- aceptado, registrado o valido: verde;
- rechazado, caducado o bloqueo: rojo;
- auditoria/historial/metadatos: gris o slate.

Cada estado debe combinar texto, icono o forma ademas del color.

## Modulos Esperables

Si el dominio se parece a Bolsa, VEC o portal de empleado, contempla:

- dashboard de acciones pendientes, plazos y notificaciones;
- perfil, datos de contacto, representacion y verificaciones;
- meritos/CV/RUM con evidencia, estado y contribucion al baremo;
- solicitudes por convocatoria con borrador, presentada, provisional,
  alegaciones, definitiva y ranking;
- documentos, firmas, CSV/HCV, justificantes y versionado;
- autobaremacion con desglose, simulacion, advertencias y recibo;
- alegaciones/subsanaciones con plazo, evidencia, resolucion y trazabilidad;
- administracion con colas, asignacion, bulk actions, auditoria y exportacion.

## Acciones Juridicas

- Separa edicion en borrador de presentacion, firma, registro, retirada,
  aceptacion, rechazo o resolucion.
- Las acciones irreversibles necesitan confirmacion con expediente, registros
  afectados y estado resultante.
- Si un estado es provisional, simulado, pendiente o consultivo, etiquetalo como
  tal. No inventes certeza juridica desde la UI.

## Accesibilidad

- HTML semantico, foco visible, navegacion por teclado y contraste AA.
- Iconos con etiqueta accesible.
- Color siempre acompanado por texto, icono o estado escrito.
- Textos y chips deben caber en desktop, laptop y movil sin solaparse.

## Uso En Orquesta

La tarea debe transportar una ref opaca, por ejemplo
`skill-ref-orquesta-programacion-web-administrativa-v0`, y la composicion debe
materializar la instruccion compacta para el runtime. No metas rutas de skills,
HOME, proveedor, modelo ni contenido completo de esta skill en el core.

En el ACK, deja nota breve si la skill influyo en la UI, por ejemplo:
`skill_ref: skill-ref-orquesta-programacion-web-administrativa-v0`.
