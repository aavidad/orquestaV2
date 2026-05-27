package orquestaweb

const opsDashboardHTMLChunk3RuntimeDetailV0 = `    function runtimeFileByName(agent, fileName) {
      return ((agent || {}).files || []).find(function(item) { return item.name === fileName; }) || null;
    }
    function runtimeFileSummary(file) {
      if (!file) return 'no observado';
      const reasons = (file.reason_codes || []).join(', ') || '-';
      return (file.file_kind || 'runtime_file') + ' · ' +
        (file.exists ? fmtBytes(file.size_bytes || 0) : 'no disponible') +
        ' · redaccion ' + (file.redaction_level || 'metadata_only') +
        ' · ref ' + (file.content_ref || '-') +
        ' · razones ' + reasons;
    }
    function runtimeFileBlock(agent, fileName, key) {
      const file = runtimeFileByName(agent, fileName);
      if (!file) return '';
      const safeKey = String(key || fileName || '');
      const open = openDetailKeys.has(safeKey) ? ' open' : '';
      const preview = file.preview ? '<pre>' + esc(file.preview) + '</pre>' : '';
      const extracts = file.extracts ? '<pre>' + esc(JSON.stringify(file.extracts, null, 2)) + '</pre>' : '';
      return '<details data-detail-key="' + esc(safeKey) + '"' + open + '><summary>' + esc(fileName) + '</summary>' +
        '<div class="kv"><div class="k">Envelope</div><div class="mono">' + esc(runtimeFileSummary(file)) + '</div></div>' +
        preview + extracts +
      '</details>';
    }
    function runtimeDetailHTML(run) {
      const loading = runtimeDetailLoading[run.run_ref];
      const detail = runtimeDetails[run.run_ref];
      if (loading && !detail) return '<div class="detail-section"><div class="sub">Cargando detalle runtime...</div></div>';
      if (!detail) return '<div class="detail-section"><div class="sub">Detalle runtime pendiente.</div></div>';
      if (detail.estado === 'error') {
        const issue = ((detail.issues || [])[0] || {}).message || 'detalle runtime no disponible';
        return '<div class="detail-section"><div class="error">' + esc(issue) + '</div></div>';
      }
      const agent = runtimeAgentForRun(run.run_ref);
      if (!agent) return '<div class="detail-section"><div class="sub">Sin runtime de agente para este run.</div></div>';
      const task = agent.task || {};
      const skills = agent.skills || {};
      const logs = ['codex_last_message.txt', 'codex_stdout.log', 'codex_stderr.log', 'director_decisions.json'].map(function(name) {
        return runtimeFileBlock(agent, name, detailKey(run.run_ref, agent.agent_ref, name));
      }).join('');
      return '<div class="detail-section">' +
        '<div class="kv"><div class="k">Qué se consigue</div><div>' + esc(task.objective_summary || task.title || runTitle(run)) + '</div></div>' +
        '<div class="kv"><div class="k">Agente seleccionado</div><div class="mono">' + esc(agent.agent_ref || '-') + '</div></div>' +
        '<div class="kv"><div class="k">Runtime</div><div class="mono">' + esc(agent.runtime_ref || '-') + '</div></div>' +
        '<div class="kv"><div class="k">Write-set</div><div>' + listInline(task.write_set || []) + '</div></div>' +
        '<div class="kv"><div class="k">Tests requeridos</div><div>' + listInline(task.required_tests || []) + '</div></div>' +
        '<div class="kv"><div class="k">Skills/policies</div><div>' +
          '<div>Caveman: ' + esc(skills.caveman_requested ? 'si' : 'no') + ' · compacto: ' + esc(skills.compact_protocol ? 'si' : 'no') + ' · subagentes max: ' + esc(skills.max_child_agents || '-') + '</div>' +
          listInline(skills.policies || []) +
        '</div></div>' +
        runtimeFileBlock(agent, 'agent_prompt.txt', detailKey(run.run_ref, agent.agent_ref, 'prompt')) +
        runtimeFileBlock(agent, 'agent_packet.json', detailKey(run.run_ref, agent.agent_ref, 'agent_packet.json')) +
        runtimeFileBlock(agent, 'agent_ack.json', detailKey(run.run_ref, agent.agent_ref, 'agent_ack.json')) +
        logs +
      '</div>';
    }`
