package orquestaweb

const opsDashboardHTMLChunk5V0 = `    }
    function buildPhaseSummary(runs) {
      const byPhase = {};
      (runs || []).forEach(function(run) {
        const phase = String(run.current_phase || 'sin_fase');
        const row = byPhase[phase] || {
          phase: phase,
          runs: 0,
          percent_total: 0,
          attention: 0,
          tasks_total: 0,
          tasks_closed: 0,
          tasks_delivered: 0,
          agents_in_flight: 0,
          stale: 0,
          completed_snapshot: 0
        };
        row.runs += 1;
        row.percent_total += Number(run.percent_complete || 0);
        if (runNeedsAttention(run)) row.attention += 1;
        row.tasks_total += Number(run.tasks_total || 0);
        row.tasks_closed += Number(run.tasks_closed || 0);
        row.tasks_delivered += Number(run.tasks_delivered || 0);
        row.agents_in_flight += Number(run.agents_in_flight || 0);
        if (run.freshness === 'stats_unavailable' || run.freshness === 'stale') row.stale += 1;
        if (run.freshness === 'completed_snapshot') row.completed_snapshot += 1;
        byPhase[phase] = row;
      });
      return Object.values(byPhase).sort(function(a, b) {
        if (b.attention !== a.attention) return b.attention - a.attention;
        if (b.runs !== a.runs) return b.runs - a.runs;
        return a.phase.localeCompare(b.phase);
      });
    }
	    function renderPhaseMatrix(runs) {
	      const rows = buildPhaseSummary(runs);
      if (!rows.length) {
        byId('phase-map').innerHTML = '<div class="empty">Sin fases observadas</div>';
        byId('phase-body').innerHTML = '<tr><td colspan="5" class="empty">Sin fases observadas</td></tr>';
        return;
      }
      byId('phase-map').innerHTML = rows.map(function(row) {
        const avg = row.runs ? Math.round(row.percent_total / row.runs) : 0;
        const bad = row.attention > 0;
        return '<div class="phase-node ' + (bad ? 'attention' : '') + '">' +
          '<div class="phase-title" title="' + esc(row.phase) + '">' + esc(row.phase) + '</div>' +
          bar(avg, bad) +
          '<div class="phase-meta"><span>' + esc(String(avg)) + '% medio</span><span>' + esc(String(row.runs)) + ' runs</span></div>' +
          '<div class="phase-meta"><span>' + esc(String(row.tasks_closed)) + '/' + esc(String(row.tasks_delivered)) + '/' + esc(String(row.tasks_total)) + ' cierre/entrega/total</span><span>' + esc(String(row.agents_in_flight)) + ' agentes vuelo</span></div>' +
          '<div class="phase-meta"><span>' + esc(row.attention ? (row.attention + ' atención') : 'sin atención') + '</span><span>' + esc(row.stale ? (row.stale + ' sin stats frescas') : 'stats frescas') + '</span></div>' +
        '</div>';
      }).join('');
      byId('phase-body').innerHTML = rows.map(function(row) {
        const avg = row.runs ? Math.round(row.percent_total / row.runs) : 0;
        const bad = row.attention > 0;
        return '<tr>' +
          tableCell('Fase', esc(row.phase), 'title="' + esc(row.phase) + '"') +
          tableCell('Runs', esc(String(row.runs))) +
          tableCell('Progreso medio', bar(avg, bad) + '<span class="sub">' + esc(String(avg)) + '%</span>') +
          tableCell('Atención', statusPill(row.attention ? (row.attention + ' con atención') : 'sin atención')) +
          tableCell('Tareas cerradas', esc(String(row.tasks_closed)) + ' / ' + esc(String(row.tasks_total)) + '<span class="task-subtitle">' + esc(String(row.tasks_delivered)) + ' entregadas · ' + esc(String(row.agents_in_flight)) + ' agentes en vuelo · freshness ' + (row.stale ? 'stale' : 'live') + '</span>') +
        '</tr>';
      }).join('');
    }
	    function renderUsageMatrix(runs, agents) {
	      const agentsByRun = {};
	      (agents || []).forEach(function(agent) {
	        if (!agent.run_ref) return;
	        const row = agentsByRun[agent.run_ref] || {
	          run_ref: agent.run_ref,
	          tokens: 0,
	          agents: 0,
	          attention: 0,
	          quota: ''
	        };
	        row.agents += 1;
	        row.tokens += Number(agent.total_tokens || 0);
	        if (agent.needs_attention || Number(agent.no_progress_ticks || 0) > 0 || String(agent.progress_status || '').toLowerCase().includes('stall')) row.attention += 1;
	        if (!row.quota && agent.quota_status) row.quota = agent.quota_status;
	        agentsByRun[agent.run_ref] = row;
	      });
	      const rows = (runs || []).map(function(run) {
	        const summary = run.usage_summary || {};
	        const usage = agentsByRun[run.run_ref] || {tokens: 0, agents: 0, attention: 0, quota: ''};
	        const quota = usage.quota || summary.quota_status || (opsHasLiveSignals(run) ? 'unknown' : '-');
	        return {
	          run: run,
	          quota: quota,
	          quota_reason_code: summary.quota_reason_code || run.stats_reason_code || '',
	          tokens: usage.tokens || Number(summary.total_tokens || 0),
	          prompt_tokens: Number(summary.prompt_tokens || 0),
	          completion_tokens: Number(summary.completion_tokens || 0),
	          agents: usage.agents || Number(summary.agents_observed || run.agents_started || 0),
	          attention: usage.attention || 0
	        };
	      }).filter(function(row) {
	        return row.tokens > 0 || row.agents > 0 || row.quota !== '-' || row.run.stats_fetch_status !== 'ok';
	      }).sort(function(a, b) {
	        if (b.attention !== a.attention) return b.attention - a.attention;
	        if (b.tokens !== a.tokens) return b.tokens - a.tokens;
	        return String(a.run.run_ref || '').localeCompare(String(b.run.run_ref || ''));
	      });
	      if (!rows.length) {
	        byId('usage-body').innerHTML = '<tr><td colspan="5" class="empty">Sin uso publicado por stats</td></tr>';
	        return;
	      }
	      byId('usage-body').innerHTML = rows.map(function(row) {
	        const tokenDetail = row.prompt_tokens || row.completion_tokens ?
	          '<span class="task-subtitle">prompt ' + esc(String(row.prompt_tokens)) + ' · completion ' + esc(String(row.completion_tokens)) + '</span>' :
	          '';
	        const selected = row.run.run_ref === selectedRunRef ? ' class="selected"' : '';
	        return '<tr data-run-ref="' + esc(row.run.run_ref || '') + '"' + selected + ' onclick="selectRun(\'' + jsArg(row.run.run_ref || '') + '\')">' +
	          tableCell('Tarea', taskTitleCell(runTitle(row.run), row.run.app_ref || '-', row.run.task_id || taskIDFromRef(row.run.run_ref))) +
	          tableCell('Cuota', statusPill(row.quota) + '<span class="task-subtitle">' + esc(row.quota_reason_code || row.run.stats_fetch_status || 'live') + '</span>') +
	          tableCell('Tokens', esc(formatTokens(row.tokens)) + tokenDetail) +
	          tableCell('Agentes', esc(String(row.agents))) +
	          tableCell('Atención', statusPill(row.attention ? (row.attention + ' agentes') : 'sin atención')) +
	        '</tr>';
	      }).join('');
	    }
	    refreshAll();
    timer = setInterval(refreshAll, refreshMs);
  </script>
</body>
</html>`
