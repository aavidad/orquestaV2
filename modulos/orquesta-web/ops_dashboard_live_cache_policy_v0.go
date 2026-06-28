package orquestaweb

const opsDashboardHTMLLiveCachePolicyV0 = `    function opsClampPercent(value) {
      const n = Number(value);
      if (!Number.isFinite(n)) return 0;
      return Math.max(0, Math.min(100, Math.round(n)));
    }
    function opsStatsProjection(runRef, settled) {
      if (!settled || settled.status !== 'fulfilled') {
        return {
          run_ref: runRef,
          stats_fetch_status: 'error',
          freshness: 'stats_unavailable',
          stats_reason_code: 'stats_fetch_failed',
          error_message: settled && settled.reason ? (settled.reason.message || String(settled.reason)) : 'stats_fetch_failed'
        };
      }
      const payload = settled.value || {};
      const stats = payload.stats || payload.director_stats || {};
      if (!stats.run_ref) {
        return {
          run_ref: payload.run_ref || runRef,
          result: payload,
          stats_fetch_status: 'empty',
          freshness: 'stats_unavailable',
          stats_reason_code: 'stats_missing'
        };
      }
      return {
        run_ref: stats.run_ref || payload.run_ref || runRef,
        result: payload,
        stats_fetch_status: 'ok',
        freshness: 'live',
        stats_reason_code: '',
        progress_source: ((stats.progress || {}).source_status || 'stats_loaded')
      };
    }
    function opsStatsPayload(projection) {
      return (projection && projection.result) || projection || {};
    }
    function opsStatsObject(projection) {
      const payload = opsStatsPayload(projection);
      return payload.stats || payload.director_stats || {};
    }
    function opsHasLiveSignals(run) {
      return Number(run.agents_in_flight || 0) > 0 ||
        Number(run.agents_started || 0) > 0 ||
        Number(run.tasks_delivered || 0) > 0 ||
        Number(run.progressing_agents || 0) > 0 ||
        Number(run.tasks_observed || 0) > 0;
    }
    function opsFallbackPercent(run) {
      const explicit = opsClampPercent(run.percent_complete);
      if (explicit > 0) return explicit;
      if (isTerminalRun(run)) return 100;
      return opsHasLiveSignals(run) ? 1 : 0;
    }
    function queueGlobalStatusItems(globalStatus) {
      return Array.isArray((globalStatus || {}).items) ? (globalStatus || {}).items : [];
    }
    function queueGlobalStatusByRun(globalStatus) {
      const byRun = {};
      queueGlobalStatusItems(globalStatus).forEach(function(item) {
        const runRef = String((item || {}).run_ref || '');
        if (!runRef) return;
        byRun[runRef] = item;
      });
      return byRun;
    }
    function queueGlobalStatusQueueItem(globalStatus) {
      return queueGlobalStatusItems(globalStatus).find(function(item) {
        return !String((item || {}).run_ref || '') && (queueRecommendedAction(item) || queueNoActionReason(item));
      }) || null;
    }
    function opsMergeEvidenceRefs(left, right) {
      const seen = {};
      return ([]).concat(left || [], right || []).filter(function(value) {
        const key = String(value || '').trim();
        if (!key || seen[key]) return false;
        seen[key] = true;
        return true;
      });
    }
    function enrichQueueItemWithGlobalStatus(item, globalItem, globalLoaded, fallbackRank) {
      const merged = Object.assign({}, item || {});
      if (fallbackRank != null && merged.rank == null) merged.rank = fallbackRank;
      merged.global_status_loaded = Boolean(globalLoaded);
      if (!globalLoaded) {
        if (!queueRecommendedAction(merged) && !queueNoActionReason(merged)) merged.no_action_reason = 'fallback_autoprogramming_status';
        return merged;
      }
      if (!globalItem) {
        if (!queueRecommendedAction(merged) && !queueNoActionReason(merged)) merged.no_action_reason = 'global_status_item_missing';
        return merged;
      }
      merged.run_ref = merged.run_ref || globalItem.run_ref || '';
      merged.app_ref = merged.app_ref || globalItem.app_ref || '';
      merged.status = merged.status || globalItem.status || 'unknown';
      merged.global_status = globalItem.status || '';
      merged.needs_action = Boolean(globalItem.needs_action);
      merged.global_status_evidence_refs = globalItem.evidence_refs || [];
      merged.evidence_refs = opsMergeEvidenceRefs(merged.evidence_refs, globalItem.evidence_refs);
      if (queueRecommendedAction(globalItem)) merged.recommended_action = globalItem.recommended_action;
      if (queueNoActionReason(globalItem)) merged.no_action_reason = globalItem.no_action_reason;
      if (!queueRecommendedAction(merged) && !queueNoActionReason(merged)) merged.no_action_reason = 'no_operator_action_required';
      return merged;
    }
    function enrichQueueWithGlobalStatus(ranked, globalStatus) {
      const globalLoaded = Array.isArray((globalStatus || {}).items);
      const byRun = queueGlobalStatusByRun(globalStatus);
      const seen = {};
      const out = (ranked || []).map(function(item, index) {
        const runRef = String((item || {}).run_ref || '');
        if (runRef) seen[runRef] = true;
        return enrichQueueItemWithGlobalStatus(item, byRun[runRef], globalLoaded, index + 1);
      });
      if (globalLoaded) {
        queueGlobalStatusItems(globalStatus).forEach(function(item) {
          const runRef = String((item || {}).run_ref || '');
          if (!runRef || seen[runRef]) return;
          seen[runRef] = true;
          out.push(enrichQueueItemWithGlobalStatus({
            rank: out.length + 1,
            run_ref: item.run_ref,
            app_ref: item.app_ref,
            status: item.status || 'unknown',
            priority_score: 0
          }, item, true));
        });
      }
      return out;
    }
    function opsQueueRunProjection(item) {
      const run = Object.assign({}, item, {
        validation: validationState(item),
        task_id: taskIDFromRef(item.run_ref),
        task_title: titleFromRef(item.run_ref),
        tasks_delivered: Number(item.tasks_delivered || 0),
        tasks_observed: Number(item.tasks_observed || 0),
        agents_in_flight: Number(item.agents_in_flight || 0),
        agents_started: Number(item.agents_started || 0),
        agents_failed: Number(item.agents_failed || 0),
        agents_need_attention: Number(item.agents_need_attention || 0),
        loop_detected_agents: Number(item.loop_detected_agents || 0),
        stopped_agents: Number(item.stopped_agents || 0),
        checkpoint_agents_pending: Number(item.checkpoint_agents_pending || 0),
        recommended_action: queueRecommendedAction(item),
        no_action_reason: queueNoActionReason(item),
        needs_action: Boolean(item.needs_action),
        global_status: item.global_status || '',
        global_status_evidence_refs: item.global_status_evidence_refs || [],
        progress_source: 'queue_snapshot',
        freshness: 'stats_unavailable',
        stats_fetch_status: 'not_requested',
        stats_reason_code: item.run_ref ? 'stats_not_loaded_yet' : 'run_ref_missing'
      });
      run.percent_complete = opsFallbackPercent(run);
      return run;
    }
    function opsRunUsageSummary(run) {
      const summary = run.usage_summary || {};
      if (summary.quota_status) return summary;
      if (run.stats_fetch_status && run.stats_fetch_status !== 'ok') {
        return Object.assign({}, summary, {quota_status: 'unavailable', quota_reason_code: run.stats_reason_code || 'stats_unavailable'});
      }
      if (opsHasLiveSignals(run)) {
        return Object.assign({}, summary, {quota_status: 'unknown', quota_reason_code: 'usage_report_missing'});
      }
      return summary;
    }
    function opsApplyStatsProjection(existing, projection) {
      const payload = opsStatsPayload(projection);
      const stats = opsStatsObject(projection);
      const runRef = stats.run_ref || payload.run_ref || projection.run_ref;
      if (!runRef) return existing;
      if (!stats.run_ref && projection.stats_fetch_status !== 'ok') {
        const failed = Object.assign(existing || {}, {
          run_ref: runRef,
          stats_fetch_status: projection.stats_fetch_status || 'error',
          freshness: projection.freshness || 'stats_unavailable',
          stats_reason_code: projection.stats_reason_code || 'stats_unavailable'
        });
        failed.percent_complete = opsFallbackPercent(failed);
        failed.usage_summary = opsRunUsageSummary(failed);
        return failed;
      }
      const counts = stats.counts || {};
      const progress = stats.progress || {};
      const closure = stats.closure || {};
      const tasks = progress.tasks || [];
      const firstTask = tasks.find(function(task) { return task.summary; }) || tasks[0] || {};
      const merged = Object.assign(existing || {}, {
        run_ref: runRef,
        app_ref: stats.project_ref || (existing || {}).app_ref,
        status: stats.status || payload.estado || (existing || {}).status,
        current_phase: stats.current_phase,
        summary: firstTask.summary || (existing || {}).summary || '',
        task_id: (existing || {}).task_id || taskIDFromRef(runRef || firstTask.task_ref),
        task_title: firstTask.summary || titleFromRef(firstTask.task_ref || runRef),
        tasks: tasks,
        evidence_refs: ((existing || {}).evidence_refs || []).concat((stats.refs && stats.refs.validations) || []),
        percent_complete: opsClampPercent(progress.percent_complete),
        tasks_total: progress.tasks_total || counts.tasks_total || 0,
        tasks_closed: progress.tasks_closed || counts.tasks_closed || 0,
        tasks_delivered: counts.tasks_delivered || (existing || {}).tasks_delivered || 0,
        tasks_observed: progress.tasks_observed || (existing || {}).tasks_observed || 0,
        agents_in_flight: counts.agents_in_flight || 0,
        agents_started: counts.agents_started || 0,
        agents_failed: counts.agents_failed || 0,
        agents_need_attention: counts.agents_need_attention || 0,
        progressing_agents: progress.progressing_agents || 0,
        stalled_agents: progress.stalled_agents || 0,
        loop_detected_agents: progress.loop_detected_agents || 0,
        stopped_agents: progress.stopped_agents || 0,
        checkpoint_agents_pending: stats.checkpoint_agents_pending || counts.checkpoint_agents_pending || 0,
        progress_source: progress.source_status || projection.progress_source || 'stats_loaded',
        freshness: projection.freshness || 'live',
        stats_fetch_status: projection.stats_fetch_status || 'ok',
        stats_reason_code: projection.stats_reason_code || '',
        closure_status: closure.status,
        goal: payload.goal || (existing || {}).goal,
        director_execution_mode: ((payload.goal || {}).director_execution_mode) || (existing || {}).director_execution_mode,
        usage_summary: stats.usage_summary || (existing || {}).usage_summary,
        ops_snapshot: payload.ops_snapshot || (existing || {}).ops_snapshot,
        recommended_action: (existing || {}).recommended_action,
        no_action_reason: (existing || {}).no_action_reason,
        needs_action: Boolean((existing || {}).needs_action),
        global_status: (existing || {}).global_status || '',
        global_status_evidence_refs: (existing || {}).global_status_evidence_refs || [],
        blocked: closure.blocked,
        validation: (existing || {}).validation || 'pendiente'
      });
      merged.percent_complete = opsFallbackPercent(merged);
      merged.usage_summary = opsRunUsageSummary(merged);
      return merged;
    }
    function buildRuns(ranked, statsResults) {
      const byRun = {};
      (ranked || []).forEach(function(item) {
        if (!item.run_ref) return;
        byRun[item.run_ref] = opsQueueRunProjection(item);
      });
      (statsResults || []).forEach(function(projection) {
        const runRef = projection.run_ref;
        if (!runRef) return;
        byRun[runRef] = opsApplyStatsProjection(byRun[runRef] || {run_ref: runRef}, projection);
      });
      return Object.values(byRun).map(function(run) {
        if (!run.freshness) run.freshness = run.observed_at ? 'completed_snapshot' : 'stats_unavailable';
        run.usage_summary = opsRunUsageSummary(run);
        return run;
      });
    }`
