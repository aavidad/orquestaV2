# Decisiones

## Alcance blando de busqueda OPES

Fecha: 2026-06-27.

Los jobs OPES de temario deben arrancar la busqueda por `course_id`,
`topic_id`, programa oficial, canon OPES y materiales reutilizables del temario.
Backups, paquetes historicos, snapshots, runtime, `orquesta_state`,
`runtime_orquesta`, `bin` y ficheros de control quedan fuera del alcance por
defecto salvo auditoria global explicita.

Motivo: las incidencias OPES mostraron agentes investigando demasiado amplio y
mezclando materiales historicos o de runtime con el contenido canonico del
curso. Esa senal debe orientar al agente y al Director, no convertirse en rail
duro.

Impacto: el bridge inyecta la regla en `opes_temario_agent_rules_2026_06_04` y
en los criterios de `research_exam_precedents`. Un hallazgo fuera de alcance se
conserva como evidencia blanda, nota de revision o insumo recuperable; no
bloquea ni descarta trabajo util por si solo. La regla vive en el adaptador OPES
y no en el nucleo Orquesta.

## T12: bloqueo real por entorno, no por contrato

Fecha: 2026-05-27.

Los intentos cerrados de T12 validan el contrato local del bridge y el wrapper
fake hasta `assemble_topic -> assembled_topic`. Eso no cierra el smoke real:
solo una instancia OPES temporal con confirmacion de efectos, servidor Orquesta
temporal y cuota/modelo confirmados puede producir la evidencia faltante.

Nota 2026-06-02: la secuencia vigente de derivados anade investigacion externa,
`generate_question_bank -> question_bank`, `generate_audio_asset ->
audio_asset`, `generate_tutor_assets -> tutor_bot_package` y
`generate_html_site -> local_html_site`. Nota posterior 2026-06-02: el cierre
OPES anade tambien `generate_help_manual_assets -> help_manual_package` para
manuales graficos de ayuda USO derivados del HTML local. Esta extension no
cambia la decision T12 ni convierte motores de audio, infografia, tutor, HTML o
manuales de ayuda en contrato del nucleo Orquesta: proveedores y adaptadores
quedan en OPES/composicion.

Decision operativa: dejar T12 como `bloqueado verificable` y no como
`pendiente` generico. El backlog no debe volver a crear una implementacion
padre para repetir pruebas fake; debe esperar el entorno opt-in o una tarea
nueva con refs de ejecucion reales.

Fronteras conservadas:

- OPES sigue siendo consumidor por API publica;
- bridge y conector no leen DB, filesystem ni colas productivas;
- `worktree_ref` y `branch_ref` se conservan como refs opacas, no como rutas ni
  nombres Git;
- el nucleo no recibe reglas OPES ni transporte REST concreto.
