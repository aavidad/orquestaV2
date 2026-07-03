## Source: scripts/docs/context_compression_benchmark_sample_noncritical_t284.md

# Non Critical Context Compression Sample T284

This sample is documentary and disposable. It exists only to produce durable
benchmark evidence for CTX-TASK-801D.

Refs that must survive any compression pass:

- scan-ref-backlog-d438fd99b7a6
- task-instance-ref-backlog-f43e8f42
- docs/autoprogramacion_orquesta_pendientes_2026-05-23.md:line:16:sha256:49ddfa8cf3f609e42a9cec0342e28231a4bce588da6c8af8ba14f046c9e78d7a
- CTX-TASK-801D
- T284

The rest of this document is intentionally low priority narrative text. The
benchmark should reduce it while preserving the refs above. It is not used as a
runtime prompt, does not carry protected policy, and does not decide any product
behavior.

Line 001: context compression can be useful only when the material is clearly
non critical and selected by an operator.
Line 002: a benchmark result is evidence for discussion, not permission to add a
runtime adapter.
Line 003: local tools can change wording, reorder details, or drop identifiers,
so refs must be measured after every run.
Line 004: token estimates in this benchmark are approximate and are meant for
relative comparison between source and compressed text.
Line 005: byte counts are exact for the UTF-8 text processed by the script.
Line 006: elapsed time measures only the local compression step.
Line 007: an external command can be supplied by an operator when it reads stdin
and writes compressed text to stdout.
Line 008: the default baseline is deterministic and uses only Python standard
library behavior behind an opt-in shell script.
Line 009: this paragraph repeats narrative detail so that a compression ratio
has something safe to remove.
Line 010: this paragraph repeats narrative detail so that a compression ratio
