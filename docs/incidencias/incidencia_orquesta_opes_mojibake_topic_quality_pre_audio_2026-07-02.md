# Incidencia: Mojibake OPES debe bloquear calidad de tema antes de audio

Fecha: 2026-07-02.

Relacionado con: `BUG-ORQ-20260630-046` y
`external/opes/INCIDENCIA_OPES_VALIDACION_TEXTO_PRE_AUDIO_NO_DETECTA_MOJIBAKE_2026-06-30.md`.

## Sintoma

El preflight de audio ya bloquea `generate_audio_asset` cuando el job trae texto
publicable corrupto en campos textuales. Quedaba un hueco anterior: un artefacto
de tema podia declarar `topic_quality_status=passed` y proponer `ready` aunque
su `topic_text`, `public_text`, `markdown` o contenido equivalente tuviera
mojibake como `mÃ`, `Ã`, `Â`, `�` o `â€`.

## Causa

El contrato `OPESTopicQualityContractV0` detectaba extension, contador canonico,
metacomentarios, andamiaje interno, contaminacion estructural y visual didactico,
pero no marcaba corrupcion de codificacion como issue propio de texto publico.
Eso dejaba la reparacion demasiado tarde, en la fase de audio.

## Cierre

Se anade el issue `opes_public_text_mojibake` al contrato de calidad de tema. El
registro OPES proyecta ese issue como `topic_quality_issue_refs`, conserva
`topic-quality-needs-rework` y materializa rework causal antes de promover el
tema a `ready`.

Pruebas focales:

```bash
go test -count=1 ./modulos/orquesta-opes-director -run 'TestValidateOPESTopicQualityContractV0DetectaMojibakePreAudioV0|TestProduceOPESCausalJobsV0BloqueaRegistroPorMojibakePreAudioV0'
```

## Residual

Si el contenido publicable llega solo como referencia a fichero o artefacto
externo, Orquesta no inspecciona aun ese fichero desde este contrato y depende
de evidencias `official_text_qa_pass` o del preflight de audio. El cierre
arquitectonico completo requiere resolver refs de contenido publicable o exigir
un informe determinista de encoding antes de `ready`.
