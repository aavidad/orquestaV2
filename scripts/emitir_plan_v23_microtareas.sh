#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd -- "${script_dir}/.." && pwd)"
manifest="${repo_root}/docs/reconstruccion/worksets/v23_agent_microtasks_v1.json"

jq --sort-keys '
  {
    phases: [
      .phases[] |
      {
        ref,
        key,
        template_ref,
        input_refs: [],
        criterion_refs: []
      }
    ],
    work_items: [
      .tasks[] |
      {
        key: .work_item_key,
        objective,
        phase,
        role,
        handoff_required: (
          .role == "role:reviewer" or
          .role == "role:adversarial-reviewer" or
          .role == "role:integrator"
        ),
        dependencies,
        write_set,
        council_policy: "auto",
        required_tests,
        skill_refs: [],
        tool_refs: ([.required_tests[].tool_ref] | unique),
        capability_refs: [
          .capability_ids[] |
          "capability:" + ascii_downcase
        ],
        output_contract: "evidence_bundle",
        security_criticality,
        reasoning_effort
      }
    ]
  }
' "${manifest}"
