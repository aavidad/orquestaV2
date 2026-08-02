-- V38-A04: una reserva por lanzamiento y replay con un intento cercado posterior.
CREATE UNIQUE INDEX agent_capacity_reservations_action_unique_idx
 ON agent_capacity_reservations(action_ref);

DROP TRIGGER agent_capacity_transitions_guard;
CREATE TRIGGER agent_capacity_transitions_guard BEFORE INSERT ON agent_capacity_transitions
WHEN NOT EXISTS (SELECT 1 FROM agent_capacity_reservations reservation
 WHERE reservation.ref=NEW.reservation_ref AND reservation.project_ref=NEW.project_ref
  AND reservation.fence=NEW.fence AND reservation.revision=NEW.expected_revision)
 OR (NEW.effect_attempt_ref IS NOT NULL AND NOT EXISTS (
  SELECT 1 FROM effect_attempts attempt JOIN agent_capacity_reservations reservation
   ON reservation.ref=NEW.reservation_ref WHERE attempt.ref=NEW.effect_attempt_ref
   AND attempt.intent_ref=reservation.effect_intent_ref AND attempt.action_ref=reservation.action_ref
   AND attempt.action_fence>=reservation.fence AND attempt.project_ref=reservation.project_ref))
 OR (NEW.effect_receipt_ref IS NOT NULL AND NOT EXISTS (
  SELECT 1 FROM effect_receipts receipt JOIN effect_attempts attempt ON attempt.ref=receipt.attempt_ref
   WHERE receipt.ref=NEW.effect_receipt_ref AND attempt.ref=NEW.effect_attempt_ref
    AND receipt.action_fence=attempt.action_fence AND receipt.status='accepted'))
BEGIN SELECT RAISE(ABORT,'sqlite.agent_capacity_transition_cause_invalid'); END;
