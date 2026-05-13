# Orquesta Como Plataforma De Agentes

Fecha: 2026-05-13.

## Decision

Orquesta debe evolucionar como nucleo de orquestacion de agentes. La creacion de
software, OPES y futuras apps son modulos o dominios consumidores, no parte del
nucleo.

## Capas

Nucleo de Orquesta:

- runs, tareas, fases, artefactos, revisiones y evidencias;
- scheduler, director, agentes, capacidad, modelos, cuotas y leases;
- sesiones, runtime, handoff, watchdogs, estadisticas y shutdown controlado;
- REST/MCP/web como adaptadores sobre puertos.

Dominios consumidores:

- programacion: crear, modificar, auditar, desplegar o refactorizar software;
- OPES: crear temarios, bloques, revisiones y fuentes;
- otros dominios futuros con sus propios contratos.

Conectores:

- traducen un contrato de dominio a trabajo orquestable;
- devuelven artefactos al dominio propietario;
- no comparten DB ni filesystem interno.

## Regla De Migracion

No se mueve codigo funcional de golpe. Primero se anaden contratos genericos,
luego los modulos actuales los adoptan por adaptadores finos. Si un corte falla,
se revierte solo ese commit sin perder el nucleo actual.

## Primer Contrato Comun

`modulos/orquesta-domain-work` define jobs y artefactos de dominio externos. Su
objetivo es que OPES, programacion u otra app puedan pedir trabajo a Orquesta
sin conocer Codex, Claude, Gemini, tmux, sesiones, cuotas ni runtime.
