<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Documentacion de Orquesta v1

## Documentos principales

- `orquesta_v1_vision.md`
  Vision funcional y arquitectonica completa.

- `op_049_matriz_voto.md`
  Preguntas y criterios para votar la arquitectura objetivo.

- `op_050_orquestador_jerarquico.md`
  Nota de decision sobre orquestador de orquestadores y presupuesto de sesion.

- `op_050_matriz_voto.md`
  Opciones concretas para votar cómo medir y reaccionar al presupuesto de sesión/token.

- `op_052_control_presupuesto_y_relevo.md`
  Matriz específica para decidir qué métrica manda en el control de cuota, cuándo entrar en handoff y cómo reaccionar con telemetría parcial.

- `orquesta_v1_roadmap.md`
  Orden de implementacion por bloques.

- `diseno_microprogramacion_dirigida_agentes.md`
  Decision de producto y arquitectura para pasar a microprogramacion dirigida con agentes.

- `operacion_agentes_manuales.md`
  Operativa de sesiones manuales, worktrees y Terminator.
  Incluye el wrapper `scripts/inicio_agente.sh` como vía manual de compatibilidad y recuperación mientras el servicio termina de absorber el arranque completo.

- `uso_actual_app_orquesta.md`
  Resumen del uso actual con política servidor-primero: servicio/daemon como fuente de verdad y CLI/web como clientes.

- `estado_persistencia_portabilidad.md`
  Estado real de portabilidad de persistencia y regla vigente: SQLite no es backend operativo canónico.

- `politica_acceso_persistencia_es.md`
  Política operativa de acceso a persistencia, incluyendo la prohibición de asumir SQLite como dialecto universal.

- `handoff_2026-03-22_codex1.md`
  Estado de relevo con decisiones, deuda abierta y siguiente bloque recomendado.

- `diseno_minimo_pools_y_presupuestos.md`
  Esquema mínimo recomendado para pools de capacidad, modelos y presupuestos de sesión.

- `matriz_modelo_y_razonamiento.md`
  Criterio para que Orquesta decida modelo, razonamiento y perfil por tarea.

- `diseno_control_activo_agentes.md`
  Diseño mínimo para enviar instrucciones, pausar, continuar y hacer handoff sobre agentes vivos.

- `diseno_app_escritorio_orquesta.md`
  Diseño de la futura app de escritorio como cliente rico sobre la API de Orquesta, sin acceso directo a persistencia.

- `diseno_orquestador_jerarquico_y_presupuesto.md`
  Documento integrador que une pools de capacidad, presupuesto de sesión y handoff preventivo dentro del mismo plano de control.

- `plan_traslado_orquestador_a_trabajo.md`
  Plan operativo para mover el repo a `~/Trabajo/orquestador` sin romper sesiones, rutas ni automatizaciones.

- `diseno_observabilidad_pasiva_runtimes_es.md`
  Diseño mínimo en castellano para árbol de runtimes, agentes hijos y telemetría pasiva del panel.

- `diseno_observabilidad_pasiva_runtimes_en.md`
  Minimal design in English for runtime trees, child agents, and passive panel telemetry.

## Documento resumen

- `../ARQUITECTURA.md`
  Resumen breve con enlaces a los documentos anteriores.
