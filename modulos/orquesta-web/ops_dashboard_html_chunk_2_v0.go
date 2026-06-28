package orquestaweb

const opsDashboardHTMLChunk2V0 = `    }
    async function ensureRuntimeDetail(runRef) {
      if (!runRef || runtimeDetails[runRef] || runtimeDetailLoading[runRef]) return;
      runtimeDetailLoading[runRef] = true;
      try {
        runtimeDetails[runRef] = await fetchRuntimeDetail(runRef);
      } catch (err) {
        runtimeDetails[runRef] = {estado: 'error', agents: [], issues: [{field: 'runtime_detail', message: err.message || String(err)}]};
      }
      delete runtimeDetailLoading[runRef];
      renderProjects(lastSnapshot.runs || []);
      renderTasks(lastSnapshot.tasks || []);
      renderAgents(lastSnapshot.agents || []);
      renderQueue(lastSnapshot.ranked || []);
      renderSelectedDetail();
    }
    async function refreshAll() {
      const diagnostics = [];
      let server = {};
      let resources = {};
      let auto = {};
      let timeline = {};
      try {
        const result = await Promise.allSettled([
          fetchJSON('/api/v0/server/status'),
          fetchJSON('/api/v0/server/resources'),
          fetchJSON('/api/v0/autoprogramming/status', {
            method: 'POST',
            headers: {'content-type': 'application/json'},
            body: JSON.stringify({
              request_id: 'ops-status-' + Date.now(),
              queue_ref: 'global',
              queue_limit: queueLimit,
              include_process_refs: false,
              include_agent_progress: true,
              include_agent_usage: true
            })
          }),
          fetchJSON('/api/v0/observability/workspace-timeline', {
            method: 'POST',
            headers: {'content-type': 'application/json'},
            body: JSON.stringify({
              schema_version: 'workspace_timeline_query.v0',
              request_id: 'ops-timeline-' + Date.now(),
              correlation_id: 'corr-ops-timeline-' + Date.now(),
              consumer: {module: 'orquesta-web', channel: 'web'},
              locale: 'es-ES',
              scope: 'workspace',
              time_window: {preset: 'last_hour'},
              page: {limit: 12},
              sources: ['director_stats', 'run_queue', 'runtime_progress'],
              include_sources: ['director_stats', 'run_queue', 'runtime_progress'],
              freshness: {max_age_seconds: 30}
            })
          })
        ]);
        if (result[0].status === 'fulfilled') server = result[0].value; else diagnostics.push(result[0].reason.message);
        if (result[1].status === 'fulfilled') resources = result[1].value; else diagnostics.push(result[1].reason.message);
        if (result[2].status === 'fulfilled') auto = result[2].value; else diagnostics.push(result[2].reason.message);
        if (result[3].status === 'fulfilled') timeline = result[3].value; else diagnostics.push(result[3].reason.message);
        const queue = auto.queue || {};
        const ranked = queue.ranked || [];
        const runRefs = ranked.slice(0, maxRuns).map(function(item) { return item.run_ref; }).filter(Boolean);
        const statResults = await Promise.allSettled(runRefs.map(fetchRunStats));
        const stats = statResults.map(function(item, index) {
          const projection = opsStatsProjection(runRefs[index], item);
          if (projection.stats_fetch_status !== 'ok') diagnostics.push(runRefs[index] + ': ' + projection.stats_reason_code);
          return projection;
        });
        render({server: server, resources: resources, auto: auto, timeline: timeline, ranked: ranked, stats: stats, diagnostics: diagnostics});
        markLive(true);
      } catch (err) {
        diagnostics.push(err.message || String(err));
        render({server: server, resources: resources, auto: auto, timeline: timeline, ranked: [], stats: [], diagnostics: diagnostics});
        markLive(false);
      }
    }
    function markLive(ok) {
      byId('live-dot').className = 'dot ' + (ok ? 'ok' : 'bad');
      text('live-text', ok ? 'live' : 'error');
      text('last-refresh', new Date().toLocaleTimeString());
    }
    function render(data) {
      lastServer = data.server || {};
      const queue = data.auto.queue || {};
      const ranked = stableSortByFirstSeen(data.ranked || [], 'queue', function(item) { return item.run_ref; });
      const activeRuns = data.auto.operator && data.auto.operator.active_runs ? data.auto.operator.active_runs : [];
      const safeActions = data.auto.operator && Array.isArray(data.auto.operator.safe_actions) ? data.auto.operator.safe_actions : [];
      const runs = stableSortByFirstSeen(buildRuns(ranked, data.stats || []), 'runs', function(run) { return run.run_ref; });
      const rawAgents = mergeRuntimeAgents(buildAgents(data.stats || []));
      const agents = stableSortByFirstSeen(mergeAgentDisplayCache(rawAgents, runs), 'agents', agentKey);
      const tasks = buildTasks(runs, agents);
      const projectCount = new Set(runs.map(function(run) { return run.app_ref || run.project_ref; }).filter(Boolean)).size;
      const worstDisk = (data.resources.disks || []).reduce(function(acc, disk) {
        return !acc || Number(disk.used_percent || 0) > Number(acc.used_percent || 0) ? disk : acc;
      }, null);
      text('kpi-server', data.server.status || 'unknown');
      text('kpi-server-hint', 'ticks ' + (data.server.supervisor_ticks || 0) + ' · ejecuciones ' + (data.server.supervisor_executions || 0));
      text('kpi-projects', projectCount);
      text('kpi-projects-hint', activeRuns.length + ' runs activos declarados');
      const queueCount = queue.count == null ? ranked.length : queue.count;
      text('kpi-queue', queueCount);
      text('kpi-queue-hint', 'supervisor cola ' + (data.server.last_supervisor_queue_size == null ? '-' : data.server.last_supervisor_queue_size));
      const activeAgentCount = runs.reduce(function(total, run) { return total + Number(run.agents_in_flight || 0); }, 0);
      text('kpi-agents', activeAgentCount || agents.filter(function(a) { return a.in_flight && a.currently_visible !== false; }).length);
      text('kpi-agents-hint', agents.length + ' observados');
      text('kpi-memory', fmtBytes((data.resources.process || {}).rss_bytes || (data.resources.process || {}).go_alloc_bytes));
      text('kpi-memory-hint', 'goroutines ' + ((data.resources.process || {}).num_goroutine || '-'));
      text('kpi-disk', worstDisk ? ((worstDisk.used_percent || 0) + '%') : '-');
      text('kpi-disk-hint', worstDisk ? shortRef(worstDisk.path) : '-');
      renderFlowSummary(ranked, runs, agents, queueCount, activeAgentCount);
      renderDirectorDecision((data.auto || {}).ops_snapshot, ranked, runs, agents, queueCount, activeAgentCount);
      updateCompletedHistory(runs, ranked);
      lastSnapshot = {runs: runs, agents: agents, ranked: ranked, tasks: tasks, safeActions: safeActions, opsSnapshot: (data.auto || {}).ops_snapshot || null};
      if (!selectedRunRef && runs.length) selectedRunRef = runs[0].run_ref;
      renderProjects(runs);
      renderPhaseMatrix(runs);
      renderUsageMatrix(runs, agents);
      renderTasks(tasks);
      renderAgents(agents);
      renderQueue(ranked);
      renderSelectedDetail();
      renderCompletedHistory();
      renderServer(data.server);
      renderAdminConfig(data.server);
      renderResources(data.resources);
      renderWorkspaceSources(data.timeline || {});
      renderDiagnostics((data.auto.diagnostics || []).concat((data.resources.issues || [])).concat((data.timeline || {}).warnings || []).concat((data.diagnostics || []).map(function(message) { return {code: 'fetch_error', message: message}; })));
      saveStableOrders();
      ensureRuntimeDetail(selectedRunRef);
    }
    function renderFlowSummary(ranked, runs, agents, queueCount, activeAgentCount) {
      const readyQueued = (ranked || []).filter(function(item) { return matchesStatus(item.status, 'running'); }).length;
      const activeRuns = (runs || []).filter(function(run) { return matchesStatus(run.status || run.closure_status, 'running') && !isTerminalRun(run); }).length;
      const attentionRuns = (runs || []).filter(runNeedsAttention).length;
      const attentionAgents = (agents || []).filter(agentNeedsAttention).length;
      const terminalRuns = (runs || []).filter(isTerminalRun).length;
      const taskTotals = (runs || []).reduce(function(acc, run) {
        acc.total += Number(run.tasks_total || 0);
        acc.closed += Number(run.tasks_closed || 0);
        return acc;
      }, {total: 0, closed: 0});
      text('flow-queue', queueCount);
      text('flow-queue-hint', readyQueued + ' preparadas o activas');
      text('flow-execution', activeRuns);
      text('flow-execution-hint', (activeAgentCount || 0) + ' agentes en vuelo');
      text('flow-attention', attentionRuns + attentionAgents);
      text('flow-attention-hint', attentionRuns + ' runs · ' + attentionAgents + ' agentes');
      byId('flow-attention-card').className = 'flow-card' + (attentionRuns + attentionAgents > 0 ? ' attention' : '');
      text('flow-closure', terminalRuns);
      text('flow-closure-hint', taskTotals.closed + '/' + taskTotals.total + ' tareas cerradas');
    }
    function directorDecisionSummary(opsSnapshot, ranked, runs, agents, queueCount, activeAgentCount) {
      const snapshotSummary = opsSnapshotDecision(opsSnapshot, runs);
      if (snapshotSummary) return snapshotSummary;
      const attentionRuns = (runs || []).filter(runNeedsAttention);
      const attentionAgents = (agents || []).filter(agentNeedsAttention);
      const activeRuns = (runs || []).filter(function(run) {
        return matchesStatus(run.status || run.closure_status, 'running') && !isTerminalRun(run);
      });
      const terminalRuns = (runs || []).filter(isTerminalRun);
      if (attentionRuns.length || attentionAgents.length) {
        return {
          attention: true,
          decision: 'Revisar y replanificar',
          reason: attentionRuns.length + ' runs y ' + attentionAgents.length + ' agentes requieren atención antes de abrir más trabajo.'
        };
      }
      if (activeRuns.length || Number(activeAgentCount || 0) > 0) {
        return {
          attention: false,
          decision: 'Esperar entregas acotadas',
          reason: activeRuns.length + ' runs activos y ' + Number(activeAgentCount || 0) + ' agentes en vuelo; mantener supervisión y waits por refs.'
        };
      }
      if (Number(queueCount || 0) > 0 || (ranked || []).length > 0) {
        return {
          attention: false,
          decision: 'Lanzar siguiente ola',
          reason: (queueCount || (ranked || []).length) + ' trabajos preparados en cola; el supervisor puede tomar el siguiente lote.'
        };
      }
      if (terminalRuns.length) {
        return {
          attention: false,
          decision: 'Cerrar o preparar nuevo objetivo',
          reason: terminalRuns.length + ' runs terminales observados; revisar evidencia y pendientes futuros.'
        };
      }
      return {
        attention: false,
        decision: 'Sin trabajo activo',
        reason: 'No hay cola ni runs activos publicados por las fuentes actuales.'
      };
    }
    function opsSnapshotDecision(opsSnapshot, runs) {
      const snapshots = [];
      if (opsSnapshot && opsSnapshot.decision) snapshots.push(opsSnapshot);
      (runs || []).forEach(function(run) {
        if (run.ops_snapshot && run.ops_snapshot.decision) snapshots.push(run.ops_snapshot);
      });
      if (!snapshots.length) return null;
      const selected = snapshots.find(function(snapshot) { return !!((snapshot.decision || {}).attention); }) ||
        snapshots.find(function(snapshot) { return String((snapshot.decision || {}).action || '') !== 'idle'; }) ||
        snapshots[0];
      const decision = selected.decision || {};
      if (!decision.action) return null;
      return {
        attention: !!decision.attention,
        decision: opsDecisionLabel(decision.action),
        reason: opsDecisionReason(selected, decision)
      };
    }
    function opsDecisionLabel(action) {
      switch (String(action || '')) {
        case 'review_replan':
          return 'Revisar y replanificar';
        case 'wait_deliveries':
          return 'Esperar entregas acotadas';
        case 'supervise_queue':
          return 'Lanzar siguiente ola';
        case 'observe_goal':
          return 'Observar goal';
        case 'repair_goal_state':
          return 'Reparar estado goal-first';
        case 'close_or_validate':
          return 'Cerrar o validar';
        case 'closed':
          return 'Run cerrada';
        case 'continue_run':
          return 'Continuar run';
        default:
          return 'Sin trabajo activo';
      }
    }
    function opsDecisionReason(snapshot, decision) {
      const queue = (snapshot || {}).queue || {};
      const runs = (snapshot || {}).runs || [];
      const run = runs.find(function(item) { return item.run_ref === decision.run_ref; }) || runs[0] || {};
      const reason = decision.reason_code || 'ops_snapshot';
      if (decision.action === 'supervise_queue') return (queue.count || 0) + ' trabajos preparados en cola; decision del snapshot operativo (' + reason + ').';
      if (decision.action === 'observe_goal') return 'Run goal-first ' + (decision.run_ref || run.run_ref || 'seleccionada') + '; observar Goal en vez de supervision legacy (' + reason + ').';
      if (decision.action === 'repair_goal_state') return 'Run goal-first ' + (decision.run_ref || run.run_ref || 'seleccionada') + ' sin estado completo; reparar GoalWorkStateV0 antes de supervision legacy (' + reason + ').';
      if (decision.action === 'wait_deliveries') return (run.agents_in_flight || 0) + ' agentes en vuelo; mantener waits por refs (' + reason + ').';
      if (decision.action === 'review_replan') return 'Atencion requerida en ' + (decision.run_ref || run.run_ref || 'run') + '; revisar evidencia y replan si procede (' + reason + ').';
      if (decision.action === 'continue_run') return (run.tasks_open || 0) + ' tareas abiertas; continuar supervision causal (' + reason + ').';
      if (decision.action === 'close_or_validate') return 'Cierre listo o validacion final pendiente (' + reason + ').';
      if (decision.action === 'closed') return 'La run aparece cerrada en el snapshot operativo (' + reason + ').';
      return 'Snapshot operativo sin cola ni runs activos (' + reason + ').';
    }
    function renderDirectorDecision(opsSnapshot, ranked, runs, agents, queueCount, activeAgentCount) {
      const summary = directorDecisionSummary(opsSnapshot, ranked, runs, agents, queueCount, activeAgentCount);
      byId('director-callout').className = 'director-callout' + (summary.attention ? ' attention' : '');
      text('director-decision', summary.decision);
      text('director-reason', summary.reason);
      text('director-supervisor-message', supervisorMessage);
    }
    function agentNeedsAttention(agent) {
      const status = String((agent || {}).status || '').toLowerCase();
      const progressStatus = String((agent || {}).progress_status || '').toLowerCase();
      return !!((agent || {}).needs_attention) ||
        progressStatus.includes('loop_detected') ||
        progressStatus.includes('stopped') ||
        progressStatus.includes('failed') ||
        progressStatus.includes('error') ||
        status.includes('loop_detected') ||
        status.includes('stopped') ||
        status.includes('failed') ||
        status.includes('error');
    }
    function runNeedsAttention(run) {
      return !!run.blocked ||
        Number(run.agents_failed || 0) > 0 ||
        Number(run.agents_need_attention || 0) > 0 ||
        Number(run.loop_detected_agents || 0) > 0 ||
        Number(run.stopped_agents || 0) > 0 ||
        Number(run.checkpoint_agents_pending || 0) > 0 ||
        String(run.validation || '').toLowerCase().includes('blocked') ||
        String(run.validation || '').toLowerCase().includes('failed') ||
        String(run.validation || '').toLowerCase().includes('error') ||
        String(run.validation || '').toLowerCase().includes('bloqueada');
    }
    function buildAgents(statsResults) {
      const out = [];
      statsResults.forEach(function(result) {
        const stats = opsStatsObject(result);
        const taskMap = {};
        const taskByAgent = {};
        ((stats.progress || {}).tasks || []).forEach(function(task) {
          if (task.task_ref) taskMap[task.task_ref] = task;
          if (task.agent_request_id && !taskByAgent[task.agent_request_id]) taskByAgent[task.agent_request_id] = task;
        });
        (stats.agents || []).forEach(function(agent) {
          const progress = agent.last_progress || {};
          const task = taskMap[progress.task_ref] || taskByAgent[agent.agent_request_id] || {};
          const usage = agent.usage || {};
          out.push({
            run_ref: stats.run_ref || result.run_ref,
            agent_ref: agent.agent_request_id,
            status: agent.status,
            in_flight: !!agent.in_flight,
            needs_attention: !!agent.needs_attention,
            task_ref: progress.task_ref || task.task_ref || '',
            task_id: taskIDFromRef(progress.task_ref || task.task_ref || agent.agent_request_id),
            task_title: task.summary || progress.summary || titleFromRef(progress.task_ref || agent.agent_request_id),
            progress_status: progress.status || task.progress_status || '',
            no_progress_ticks: progress.no_progress_ticks || 0,
            repeated_action_count: progress.repeated_action_count || 0,
            capacity_level: usage.capacity_level || '',
            quota_status: usage.quota_status || '',
            total_tokens: usage.total_tokens || 0,
            percent: agentPercent(agent, progress)
          });
        });
      });
      return out;
    }
    function buildTasks(runs, agents) {
      const out = [];
      const agentByTask = {};
      (agents || []).forEach(function(agent) {
        if (agent.task_ref && !agentByTask[agent.task_ref]) agentByTask[agent.task_ref] = agent;
      });
      (runs || []).forEach(function(run) {
        (run.tasks || []).forEach(function(task) {
          const agent = agentByTask[task.task_ref] || {};
          out.push({
            run_ref: run.run_ref,
            app_ref: run.app_ref,
            task_ref: task.task_ref,
            task_id: taskIDFromRef(task.task_ref || run.run_ref),
            title: task.summary || titleFromRef(task.task_ref || run.run_ref),
            status: task.status || 'unknown',
            agent_ref: task.agent_request_id || agent.agent_ref || '',
            progress_status: task.progress_status || agent.progress_status || '',
            no_progress_ticks: task.no_progress_ticks || agent.no_progress_ticks || 0
          });
        });
      });
      return out;
    }
    function agentKey(agent) {
      const runRef = String((agent || {}).run_ref || '');
      const agentRef = String((agent || {}).agent_ref || '');
      if (!runRef || !agentRef) return '';
      return runRef + '|' + agentRef;
    }
    function mergeRuntimeAgents(agents) {
      const byKey = {};
      (agents || []).forEach(function(agent) {
        const key = agentKey(agent);
        if (key) byKey[key] = Object.assign({}, agent);
      });
      Object.keys(runtimeDetails || {}).forEach(function(runRef) {
        const detail = runtimeDetails[runRef] || {};
        (detail.agents || []).forEach(function(runtimeAgent) {
          const agentRef = runtimeAgent.agent_ref || '';
          if (!agentRef) return;
          const packetTask = runtimeAgent.task || {};
          const ackFile = ((runtimeAgent.files || []).find(function(item) { return item.name === 'agent_ack.json'; }) || {});
          const ackStatus = ((ackFile.extracts || {}).status) || '';
          const key = runRef + '|' + agentRef;
          byKey[key] = Object.assign({}, byKey[key] || {}, {
            run_ref: runRef,
            agent_ref: agentRef,
            status: (byKey[key] || {}).status || ackStatus || 'runtime_observed',
            in_flight: (byKey[key] || {}).in_flight || false,
            needs_attention: (byKey[key] || {}).needs_attention || false,
            task_ref: (byKey[key] || {}).task_ref || packetTask.task_ref || '',
            task_id: (byKey[key] || {}).task_id || taskIDFromRef(packetTask.task_ref || agentRef),
            task_title: (byKey[key] || {}).task_title || packetTask.title || packetTask.objective_summary || titleFromRef(packetTask.task_ref || agentRef),
            progress_status: (byKey[key] || {}).progress_status || 'runtime',
            no_progress_ticks: (byKey[key] || {}).no_progress_ticks || 0,
            repeated_action_count: (byKey[key] || {}).repeated_action_count || 0,
            capacity_level: (byKey[key] || {}).capacity_level || '',
            quota_status: (byKey[key] || {}).quota_status || '',
            total_tokens: (byKey[key] || {}).total_tokens || 0,
            percent: (byKey[key] || {}).percent || (ackStatus === 'completed' ? 100 : 0)
          });
        });
      });
      return Object.values(byKey);
    }
    function mergeAgentDisplayCache(freshAgents, runs) {
      const graceMs = 8000;
      const nowMs = Date.now();
      const now = new Date().toLocaleTimeString();
      const freshKeys = new Set();
      (freshAgents || []).forEach(function(agent) {
        const key = agentKey(agent);
        if (!key) return;
        freshKeys.add(key);
        agentDisplayCache[key] = Object.assign({}, agentDisplayCache[key] || {}, agent, {`
