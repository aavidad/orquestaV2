---
name: orquesta-director-agentes
description: Dirigir agentes y subagentes con Orquesta: repartir trabajo, elegir topologia, pedir revisiones, conservar entregas, resolver discrepancias y cerrar con evidencia sin rails genericos bloqueantes.
---

# Orquesta Director Agentes

Usa esta skill cuando el agente actue como Director de un trabajo con varios
agentes, subagentes, revisores o candidatos alternativos.

## Responsabilidad

El Director no hace todo a mano. Decide:

- que tareas se paralelizan;
- que agente o rol conviene;
- que contexto minimo recibe cada agente;
- que entregas se aceptan;
- que se fusiona;
- que se manda a rework;
- que queda como borrador/evidencia;
- cuando se cierra.

## Topologia

- Para muchas piezas homogeneas, agrupa en padres con hasta 6 subagentes cuando
  ahorre contexto.
- Para piezas aisladas o de alto riesgo, usa un padre por pieza.
- No pongas limites artificiales globales. Documenta solo limites duros de
  proveedor, runtime, sistema operativo o instruccion expresa del operador.
- Pide comunicacion compacta y, si existe, modo `caveman` para ahorrar tokens.

## Encargos

Cada agente debe recibir:

- objetivo concreto;
- write-set o alcance;
- refs de contexto;
- criterios de aceptacion;
- pruebas o validacion esperada;
- formato de respuesta breve;
- regla de conservar trabajo recuperable.

## Revision

Usa revisores independientes cuando haya riesgo de calidad:

- un revisor tecnico;
- un revisor de producto/dominio;
- un revisor de tests o validacion;
- un revisor visual si hay UI o assets.

Cuando haya alternativas, usa consejo y votacion:

- candidatos separados;
- votos razonados;
- matriz de discrepancias;
- decision final del Director.

La votacion no decide sola. El Director selecciona, fusiona o pide rework.

## Rework

No tirar trabajo por:

- alias;
- nombre cercano;
- formato recuperable;
- contexto omitido por presupuesto;
- texto imperfecto;
- pruebas incompletas pero reparables.

Convertirlo en:

- normalizacion;
- rework causal;
- borrador;
- insumo;
- evidencia;
- tarea derivada.

Solo cortar fuerte por seguridad real, causalidad rota, refs imposibles, datos
sensibles efectivos o efectos externos no autorizados.

## Cierre

Antes de cerrar:

- integrar entregas;
- pasar pruebas/capturas/validaciones necesarias;
- revisar worktree;
- documentar decisiones;
- archivar artefactos generados fuera del repo cuando no sean codigo;
- dejar pendientes futuros al final, no mezclados con lo cerrado.

## Salida del Director

Responder con:

- agentes lanzados;
- entregas aceptadas;
- entregas fusionadas;
- rework pendiente;
- pruebas ejecutadas;
- bloqueo real si existe;
- siguiente accion.
