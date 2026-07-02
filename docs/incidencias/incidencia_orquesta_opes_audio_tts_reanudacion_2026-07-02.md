# Incidencia: contrato audio/TTS OPES exige reanudacion sin duplicar MP3 validos

Fecha: 2026-07-02

Bug: `BUG-ORQ-20260702-120`

## Hallazgo

El cierre de audio OPES no debe considerar `audio_asset` listo solo por tener
capacidad TTS nominal o MP3 presentes. Para evitar falsos verdes en reintentos,
el contrato debe exigir una de estas dos evidencias:

- proveedor TTS supervisable con heartbeat de progreso, `provider_timeout` y
  ventana maxima sin avance;
- manifest/evidencia de reanudacion que preserve MP3 validos y regenere solo
  faltantes u obsoletos, sin duplicar `audio_ref` por `section_ref`.

## Frontera

`orquesta-domain-work` es el lugar correcto para el perfil neutral de capacidad:
expresa `speech_synthesis` con supervision de proveedor o evidencia generica de
reanudacion/no duplicacion. No conoce MP3, sidecars, rutas, Edge TTS ni OPES.

`orquesta-opes-bridge` es el lugar correcto para el contrato OPES de MP3:
required test `audio_tts_resumable` exige `audio_tts_operational_manifest`,
heartbeat/timeout o `resume_without_duplicate_valid_mp3` antes de `ready` o
`listo_para_revision_operador`.

## Evidencia local

- `TestDomainWorkExternalCapabilityEvaluationV0AceptaAudioConReanudacionSinDuplicar`
- `TestOPESRequiredTestPolicyV0AudioExigeTTSReanudableSinDuplicarMP3Validos`
- `TestOPESRequiredTestPolicyV0FinalTemarioIncluyeAudioTTSReanudable`

Estado: avance parcial de audio/TTS. El bug padre sigue abierto por
disponibilidad Orquesta y QA tests/tutor. El residual RAG/visual se aborda en
`docs/incidencias/incidencia_orquesta_opes_finalpkg_rag_visual_2026-07-02.md`.
