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
          })
        ]);
        if (result[0].status === 'fulfilled') server = result[0].value; else diagnostics.push(result[0].reason.message);
        if (result[1].status === 'fulfilled') resources = result[1].value; else diagnostics.push(result[1].reason.message);
        if (result[2].status === 'fulfilled') auto = result[2].value; else diagnostics.push(result[2].reason.message);
        const queue = auto.queue || {};
        const ranked = queue.ranked || [];
        const runRefs = ranked.slice(0, maxRuns).map(function(item) { return item.run_ref; }).filter(Boolean);
        const statResults = await Promise.allSettled(runRefs.map(fetchRunStats));
        const stats = statResults.map(function(item, index) {
          if (item.status === 'fulfilled') return item.value;
          diagnostics.push(runRefs[index] + ': ' + item.reason.message);
          return null;
        }).filter(Boolean);
        render({server: server, resources: resources, auto: auto, ranked: ranked, stats: stats, diagnostics: diagnostics});
        markLive(true);
      } catch (err) {
        diagnostics.push(err.message || String(err));
        render({server: server, resources: resources, auto: auto, ranked: [], stats: [], diagnostics: diagnostics});
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
      updateCompletedHistory(runs, ranked);
      lastSnapshot = {runs: runs, agents: agents, ranked: ranked, tasks: tasks};
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
      renderDiagnostics((data.auto.diagnostics || []).concat((data.resources.issues || [])).concat((data.diagnostics || []).map(function(message) { return {code: 'fetch_error', message: message}; })));
      saveStableOrders();
      ensureRuntimeDetail(selectedRunRef);
    }
    function buildRuns(ranked, statsResults) {
      const byRun = {};
      ranked.forEach(function(item) {
        byRun[item.run_ref] = Object.assign({}, item, {
          validation: validationState(item),
          task_id: taskIDFromRef(item.run_ref),
          task_title: titleFromRef(item.run_ref)
        });
      });
      statsResults.forEach(function(result) {
        const stats = result.stats || result.director_stats || {};
        const runRef = stats.run_ref || result.run_ref;
        if (!runRef) return;
        const counts = stats.counts || {};
        const progress = stats.progress || {};
        const closure = stats.closure || {};
        const tasks = progress.tasks || [];
        const firstTask = tasks.find(function(task) { return task.summary; }) || tasks[0] || {};
        byRun[runRef] = Object.assign(byRun[runRef] || {}, {
          run_ref: runRef,
          app_ref: stats.project_ref || (byRun[runRef] || {}).app_ref,
          status: stats.status || result.estado || (byRun[runRef] || {}).status,
          current_phase: stats.current_phase,
          summary: firstTask.summary || (byRun[runRef] || {}).summary || '',
          task_id: (byRun[runRef] || {}).task_id || taskIDFromRef(runRef || firstTask.task_ref),
          task_title: firstTask.summary || titleFromRef(firstTask.task_ref || runRef),
          tasks: tasks,
          evidence_refs: ((byRun[runRef] || {}).evidence_refs || []).concat((stats.refs && stats.refs.validations) || []),
          percent_complete: progress.percent_complete || 0,
          tasks_total: progress.tasks_total || counts.tasks_total || 0,
          tasks_closed: progress.tasks_closed || counts.tasks_closed || 0,
          agents_in_flight: counts.agents_in_flight || 0,
          agents_started: counts.agents_started || 0,
          agents_failed: counts.agents_failed || 0,
          agents_need_attention: counts.agents_need_attention || 0,
          progressing_agents: progress.progressing_agents || 0,
          stalled_agents: progress.stalled_agents || 0,
          closure_status: closure.status,
          usage_summary: stats.usage_summary || (byRun[runRef] || {}).usage_summary,
          blocked: closure.blocked,
          validation: (byRun[runRef] || {}).validation || 'pendiente'
        });
      });
      return Object.values(byRun);
    }
    function renderFlowSummary(ranked, runs, agents, queueCount, activeAgentCount) {
      const readyQueued = (ranked || []).filter(function(item) { return matchesStatus(item.status, 'running'); }).length;
      const activeRuns = (runs || []).filter(function(run) { return matchesStatus(run.status || run.closure_status, 'running') && !isTerminalRun(run); }).length;
      const attentionRuns = (runs || []).filter(runNeedsAttention).length;
      const attentionAgents = (agents || []).filter(function(agent) {
        return agent.needs_attention || Number(agent.no_progress_ticks || 0) > 0 || String(agent.progress_status || '').toLowerCase().includes('stall');
      }).length;
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
    function runNeedsAttention(run) {
      return !!run.blocked ||
        Number(run.stalled_agents || 0) > 0 ||
        Number(run.agents_failed || 0) > 0 ||
        Number(run.agents_need_attention || 0) > 0 ||
        String(run.validation || '').toLowerCase().includes('bloqueada');
    }
    function buildAgents(statsResults) {
      const out = [];
      statsResults.forEach(function(result) {
        const stats = result.stats || result.director_stats || {};
        const taskMap = {};
        ((stats.progress || {}).tasks || []).forEach(function(task) {
          if (task.task_ref) taskMap[task.task_ref] = task;
        });
        (stats.agents || []).forEach(function(agent) {
          const progress = agent.last_progress || {};
          const task = taskMap[progress.task_ref] || {};
          const usage = agent.usage || {};
          out.push({
            run_ref: stats.run_ref || result.run_ref,
            agent_ref: agent.agent_request_id,
            status: agent.status,
            in_flight: !!agent.in_flight,
            needs_attention: !!agent.needs_attention,
            task_ref: progress.task_ref || '',
            task_id: taskIDFromRef(progress.task_ref || agent.agent_request_id),
            task_title: task.summary || progress.summary || titleFromRef(progress.task_ref || agent.agent_request_id),
            progress_status: progress.status || '',
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
          const packetTask = (((runtimeAgent.agent_packet || {}).task) || {});
          const key = runRef + '|' + agentRef;
          byKey[key] = Object.assign({}, byKey[key] || {}, {
            run_ref: runRef,
            agent_ref: agentRef,
            status: (byKey[key] || {}).status || (runtimeAgent.ack || {}).status || 'runtime_observed',
            in_flight: (byKey[key] || {}).in_flight || false,
            needs_attention: (byKey[key] || {}).needs_attention || false,
            task_ref: (byKey[key] || {}).task_ref || packetTask.task_ref || '',
            task_id: (byKey[key] || {}).task_id || taskIDFromRef(packetTask.task_ref || agentRef),
            task_title: (byKey[key] || {}).task_title || packetTask.title || packetTask.objective || titleFromRef(packetTask.task_ref || agentRef),
            progress_status: (byKey[key] || {}).progress_status || 'runtime',
            no_progress_ticks: (byKey[key] || {}).no_progress_ticks || 0,
            repeated_action_count: (byKey[key] || {}).repeated_action_count || 0,
            capacity_level: (byKey[key] || {}).capacity_level || '',
            quota_status: (byKey[key] || {}).quota_status || '',
            total_tokens: (byKey[key] || {}).total_tokens || 0,
            percent: (byKey[key] || {}).percent || (((runtimeAgent.ack || {}).status === 'completed') ? 100 : 0)
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
