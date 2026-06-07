---
name: orquesta-revision-consejo-votacion
description: Organizar debate, critica, voto y seleccion entre agentes para elegir la mejor solucion sin convertir la votacion en un rail automatico.
---

# Orquesta Revision Consejo Votacion

Usa esta skill cuando varias soluciones, candidatos o revisiones deban
compararse antes de que el Director cierre una decision.

## Roles

- propuesta: crea una opcion con evidencia y tradeoffs;
- critica: busca fallos, riesgos, regresiones y coste;
- voto: compara opciones con criterios declarados;
- director: selecciona, fusiona, pide rework o conserva como borrador.

## Contrato de candidato

Cada candidato debe traer:

- `candidate_ref`;
- objetivo;
- artefactos o cambios;
- evidencia;
- riesgos;
- pruebas o validacion;
- partes reutilizables.

## Votacion

Cada voto debe indicar:

- ranking;
- razon principal;
- riesgos no resueltos;
- si conviene fusionar candidatos;
- rework recomendado.

## Reglas

- La votacion informa; no decide sola.
- No se descarta trabajo recuperable por texto, alias o formato reparable.
- El Director conserva candidatos no elegidos como evidencia, borrador o insumo.
- Si hay empate o discrepancia fuerte, pedir rework acotado con refs causales.

## Entrega

Devuelve matriz con:

- candidatos;
- votos;
- decision del Director;
- accepted_refs;
- merged_refs;
- rework_refs;
- discarded_as_draft_refs.
