-- V38-A03: capacidad externa global y reservas causales aisladas por proyecto.
CREATE TABLE agent_capacity_observations (
 ref TEXT PRIMARY KEY CHECK(length(trim(ref))>0),
 source_ref TEXT NOT NULL CHECK(length(trim(source_ref))>0),
 pool_ref TEXT NOT NULL CHECK(length(trim(pool_ref))>0),
 window_ref TEXT NOT NULL CHECK(length(trim(window_ref))>0),
 revision INTEGER NOT NULL CHECK(revision>0),
 expected_revision INTEGER NOT NULL CHECK(expected_revision>=0 AND revision=expected_revision+1),
 status TEXT NOT NULL CHECK(status IN ('available','unavailable')),
 quality TEXT NOT NULL CHECK(quality IN ('unknown','estimated','measured','exact')),
 observed_at INTEGER NOT NULL,expires_at INTEGER NOT NULL CHECK(expires_at>observed_at),
 reset_at INTEGER CHECK(reset_at>observed_at),retry_at INTEGER CHECK(retry_at>observed_at),
 artifact_ref TEXT NOT NULL DEFAULT '' CHECK(artifact_ref='' OR length(trim(artifact_ref))>0),
 slots_applicability TEXT NOT NULL CHECK(slots_applicability IN ('unknown','not_applicable','applicable')),
 slots_limit INTEGER CHECK(slots_limit>=0),slots_remaining INTEGER CHECK(slots_remaining>=0),
 seconds_applicability TEXT NOT NULL CHECK(seconds_applicability IN ('unknown','not_applicable','applicable')),
 seconds_limit INTEGER CHECK(seconds_limit>=0),seconds_remaining INTEGER CHECK(seconds_remaining>=0),
 messages_applicability TEXT NOT NULL CHECK(messages_applicability IN ('unknown','not_applicable','applicable')),
 messages_limit INTEGER CHECK(messages_limit>=0),messages_remaining INTEGER CHECK(messages_remaining>=0),
 tokens_applicability TEXT NOT NULL CHECK(tokens_applicability IN ('unknown','not_applicable','applicable')),
 tokens_limit INTEGER CHECK(tokens_limit>=0),tokens_remaining INTEGER CHECK(tokens_remaining>=0),
 credits_applicability TEXT NOT NULL CHECK(credits_applicability IN ('unknown','not_applicable','applicable')),
 credits_limit INTEGER CHECK(credits_limit>=0),credits_remaining INTEGER CHECK(credits_remaining>=0),
 idempotency_key TEXT NOT NULL CHECK(length(trim(idempotency_key))>0),
 UNIQUE(source_ref,pool_ref,revision),UNIQUE(source_ref,pool_ref,idempotency_key),
 CHECK(slots_applicability='applicable' OR slots_limit IS NULL AND slots_remaining IS NULL),
 CHECK(seconds_applicability='applicable' OR seconds_limit IS NULL AND seconds_remaining IS NULL),
 CHECK(messages_applicability='applicable' OR messages_limit IS NULL AND messages_remaining IS NULL),
 CHECK(tokens_applicability='applicable' OR tokens_limit IS NULL AND tokens_remaining IS NULL),
 CHECK(credits_applicability='applicable' OR credits_limit IS NULL AND credits_remaining IS NULL),
 CHECK(slots_limit IS NULL OR slots_remaining IS NULL OR slots_remaining<=slots_limit),
 CHECK(seconds_limit IS NULL OR seconds_remaining IS NULL OR seconds_remaining<=seconds_limit),
 CHECK(messages_limit IS NULL OR messages_remaining IS NULL OR messages_remaining<=messages_limit),
 CHECK(tokens_limit IS NULL OR tokens_remaining IS NULL OR tokens_remaining<=tokens_limit),
 CHECK(credits_limit IS NULL OR credits_remaining IS NULL OR credits_remaining<=credits_limit)
) STRICT;
CREATE INDEX agent_capacity_observations_current_idx ON agent_capacity_observations(source_ref,pool_ref,revision DESC);
CREATE TRIGGER agent_capacity_observations_revision_guard BEFORE INSERT ON agent_capacity_observations
WHEN NEW.expected_revision<>COALESCE((SELECT MAX(revision) FROM agent_capacity_observations
 WHERE source_ref=NEW.source_ref AND pool_ref=NEW.pool_ref),0)
BEGIN SELECT RAISE(ABORT,'sqlite.agent_capacity_observation_revision_conflict'); END;
CREATE TABLE agent_capacity_reservations (
 ref TEXT PRIMARY KEY CHECK(length(trim(ref))>0),
 observation_ref TEXT NOT NULL REFERENCES agent_capacity_observations(ref) ON DELETE RESTRICT,
 observation_revision INTEGER NOT NULL CHECK(observation_revision>0),
 effect_intent_ref TEXT NOT NULL REFERENCES effect_intents(ref) ON DELETE RESTRICT,
 project_ref TEXT NOT NULL,goal_ref TEXT NOT NULL,work_item_ref TEXT NOT NULL,
 execution_ref TEXT NOT NULL,action_ref TEXT NOT NULL REFERENCES outbox(ref) ON DELETE RESTRICT,
 plan_generation INTEGER NOT NULL CHECK(plan_generation>0),
 work_item_generation INTEGER NOT NULL CHECK(work_item_generation>0),
 fence INTEGER NOT NULL CHECK(fence>0),
 state TEXT NOT NULL DEFAULT 'reserved' CHECK(state IN ('reserved','consumed','released','quarantined')),
 revision INTEGER NOT NULL DEFAULT 1 CHECK(revision>0),
 last_transition_ref TEXT NOT NULL DEFAULT '',last_cause_ref TEXT NOT NULL DEFAULT '',
 updated_at INTEGER NOT NULL,settled_at INTEGER,
 slots INTEGER NOT NULL CHECK(slots>0),seconds INTEGER NOT NULL CHECK(seconds>=0),
 messages INTEGER NOT NULL CHECK(messages>=0),tokens INTEGER NOT NULL CHECK(tokens>=0),
 credits INTEGER NOT NULL CHECK(credits>=0),
 idempotency_key TEXT NOT NULL CHECK(length(trim(idempotency_key))>0),
 reserved_at INTEGER NOT NULL,
 UNIQUE(action_ref,fence),UNIQUE(idempotency_key,fence),UNIQUE(ref,project_ref,fence),
 FOREIGN KEY(goal_ref,project_ref) REFERENCES goals(ref,project_ref) ON DELETE RESTRICT,
 FOREIGN KEY(goal_ref,work_item_ref) REFERENCES work_items(goal_ref,ref) ON DELETE RESTRICT,
 FOREIGN KEY(goal_ref,work_item_ref,execution_ref)
  REFERENCES executions(goal_ref,work_item_ref,ref) ON DELETE RESTRICT
 ,CHECK((revision=1 AND state='reserved' AND last_transition_ref='' AND last_cause_ref=''
   AND updated_at=reserved_at AND settled_at IS NULL) OR revision>1)
) STRICT;
CREATE INDEX agent_capacity_reservations_observation_idx ON agent_capacity_reservations(observation_ref,reserved_at,ref);
CREATE TRIGGER agent_capacity_reservations_guard BEFORE INSERT ON agent_capacity_reservations
WHEN NEW.revision<>1 OR NEW.state<>'reserved' OR NEW.last_transition_ref<>'' OR NEW.last_cause_ref<>'' OR NEW.updated_at<>NEW.reserved_at OR NEW.settled_at IS NOT NULL OR NOT EXISTS (
 SELECT 1 FROM agent_capacity_observations observation
 WHERE observation.ref=NEW.observation_ref AND observation.revision=NEW.observation_revision
  AND observation.revision=(SELECT MAX(current.revision) FROM agent_capacity_observations current
   WHERE current.source_ref=observation.source_ref AND current.pool_ref=observation.pool_ref)
) OR NOT EXISTS (
 SELECT 1 FROM outbox action JOIN effect_intents intent
  ON intent.ref=NEW.effect_intent_ref AND intent.action_ref=action.ref
 WHERE action.ref=NEW.action_ref AND action.kind='launch_agent'
  AND action.goal_ref=NEW.goal_ref AND action.work_item_ref=NEW.work_item_ref
  AND action.execution_ref=NEW.execution_ref AND action.plan_generation=NEW.plan_generation
  AND action.work_item_generation=NEW.work_item_generation AND action.fence=NEW.fence
  AND action.claim_token IS NOT NULL AND action.completed_at IS NULL
  AND intent.kind='agent_launch' AND intent.project_ref=NEW.project_ref
  AND intent.goal_ref=NEW.goal_ref AND intent.work_item_ref=NEW.work_item_ref
  AND intent.execution_ref=NEW.execution_ref AND intent.plan_generation=NEW.plan_generation
) BEGIN SELECT RAISE(ABORT,'sqlite.agent_capacity_reservation_cause_invalid'); END;

-- Hechos inmutables: application decide la transición; SQLite conserva CAS y ligaduras.
CREATE TABLE agent_capacity_transitions (
 ref TEXT PRIMARY KEY CHECK(length(trim(ref))>0),
 reservation_ref TEXT NOT NULL,project_ref TEXT NOT NULL,fence INTEGER NOT NULL CHECK(fence>0),
 expected_revision INTEGER NOT NULL CHECK(expected_revision>0),
 revision INTEGER NOT NULL CHECK(revision=expected_revision+1),
 outcome TEXT NOT NULL CHECK(outcome IN ('consumed','released','quarantined')),
 cause_kind TEXT NOT NULL CHECK(cause_kind IN ('effect_receipt','definitely_not_applied','execution_terminal','unknown_applied','reconciliation')),
 cause_ref TEXT NOT NULL CHECK(length(trim(cause_ref))>0),
 effect_attempt_ref TEXT REFERENCES effect_attempts(ref) ON DELETE RESTRICT,
 effect_receipt_ref TEXT REFERENCES effect_receipts(ref) ON DELETE RESTRICT,
 idempotency_key TEXT NOT NULL UNIQUE CHECK(length(trim(idempotency_key))>0),
 recorded_at INTEGER NOT NULL,
 UNIQUE(reservation_ref,revision),UNIQUE(reservation_ref,outcome,cause_ref),
 FOREIGN KEY(reservation_ref,project_ref,fence)
  REFERENCES agent_capacity_reservations(ref,project_ref,fence) ON DELETE RESTRICT,
 CHECK(effect_receipt_ref IS NULL OR effect_attempt_ref IS NOT NULL)
) STRICT;
CREATE TRIGGER agent_capacity_transitions_guard BEFORE INSERT ON agent_capacity_transitions
WHEN NOT EXISTS (SELECT 1 FROM agent_capacity_reservations reservation
 WHERE reservation.ref=NEW.reservation_ref AND reservation.project_ref=NEW.project_ref
  AND reservation.fence=NEW.fence AND reservation.revision=NEW.expected_revision)
 OR (NEW.effect_attempt_ref IS NOT NULL AND NOT EXISTS (
  SELECT 1 FROM effect_attempts attempt JOIN agent_capacity_reservations reservation
   ON reservation.ref=NEW.reservation_ref WHERE attempt.ref=NEW.effect_attempt_ref
   AND attempt.intent_ref=reservation.effect_intent_ref AND attempt.action_ref=reservation.action_ref
   AND attempt.action_fence=reservation.fence AND attempt.project_ref=reservation.project_ref))
 OR (NEW.effect_receipt_ref IS NOT NULL AND NOT EXISTS (
  SELECT 1 FROM effect_receipts receipt WHERE receipt.ref=NEW.effect_receipt_ref
   AND receipt.attempt_ref=NEW.effect_attempt_ref AND receipt.action_fence=NEW.fence
   AND receipt.status='accepted'))
BEGIN SELECT RAISE(ABORT,'sqlite.agent_capacity_transition_cause_invalid'); END;

CREATE TRIGGER agent_capacity_observations_immutable_update BEFORE UPDATE ON agent_capacity_observations BEGIN SELECT RAISE(ABORT,'sqlite.agent_capacity_observation_immutable'); END;
CREATE TRIGGER agent_capacity_observations_immutable_delete BEFORE DELETE ON agent_capacity_observations BEGIN SELECT RAISE(ABORT,'sqlite.agent_capacity_observation_immutable'); END;
CREATE TRIGGER agent_capacity_reservations_identity_immutable BEFORE UPDATE OF ref,observation_ref,
 observation_revision,effect_intent_ref,project_ref,goal_ref,work_item_ref,execution_ref,action_ref,
 plan_generation,work_item_generation,fence,slots,seconds,messages,tokens,credits,idempotency_key,reserved_at
 ON agent_capacity_reservations BEGIN SELECT RAISE(ABORT,'sqlite.agent_capacity_reservation_identity_immutable'); END;
CREATE TRIGGER agent_capacity_reservations_state_guard BEFORE UPDATE OF state,revision,last_transition_ref,
 last_cause_ref,updated_at,settled_at ON agent_capacity_reservations WHEN NOT EXISTS (
 SELECT 1 FROM agent_capacity_transitions transition WHERE transition.ref=NEW.last_transition_ref
  AND transition.reservation_ref=OLD.ref AND transition.project_ref=OLD.project_ref
  AND transition.fence=OLD.fence AND transition.expected_revision=OLD.revision
  AND transition.revision=NEW.revision AND transition.outcome=NEW.state
  AND transition.cause_ref=NEW.last_cause_ref AND transition.recorded_at=NEW.updated_at
  AND (NEW.settled_at IS NULL OR NEW.settled_at=transition.recorded_at))
BEGIN SELECT RAISE(ABORT,'sqlite.agent_capacity_reservation_state_invalid'); END;
CREATE TRIGGER agent_capacity_reservations_immutable_delete BEFORE DELETE ON agent_capacity_reservations BEGIN SELECT RAISE(ABORT,'sqlite.agent_capacity_reservation_immutable'); END;
CREATE TRIGGER agent_capacity_transitions_immutable_update BEFORE UPDATE ON agent_capacity_transitions BEGIN SELECT RAISE(ABORT,'sqlite.agent_capacity_transition_immutable'); END;
CREATE TRIGGER agent_capacity_transitions_immutable_delete BEFORE DELETE ON agent_capacity_transitions BEGIN SELECT RAISE(ABORT,'sqlite.agent_capacity_transition_immutable'); END;
